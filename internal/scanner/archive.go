package scanner

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/qualiguard/qualiguard/internal/discovery"
	"github.com/qualiguard/qualiguard/internal/gate"
	"github.com/qualiguard/qualiguard/internal/metrics"
	"github.com/qualiguard/qualiguard/internal/model"
	"github.com/qualiguard/qualiguard/internal/parser"
	"github.com/qualiguard/qualiguard/internal/rules"
	"github.com/qualiguard/qualiguard/internal/snippet"
)

const maxZipBytes = 32 << 20 // 32 MB
const maxZipFiles = 400

var defaultArchiveExclusions = []string{
	"**/__pycache__/**",
	"**/node_modules/**",
	"**/.git/**",
	"**/venv/**",
	"**/.venv/**",
	"**/dist/**",
	"**/build/**",
	"**/vendor/**",
	"**/*.min.js",
}

var defaultArchiveLanguages = []string{"python", "javascript", "go", "java", "csharp"}

// AnalyzeZipArchive scans all supported source files inside a zip archive.
func AnalyzeZipArchive(ctx context.Context, zipData []byte, zipName, version, projectKey, projectName string) (*model.Report, error) {
	if len(zipData) == 0 {
		return nil, fmt.Errorf("zip arşivi boş")
	}
	if len(zipData) > maxZipBytes {
		return nil, fmt.Errorf("zip çok büyük (limit %d MB)", maxZipBytes/(1<<20))
	}

	start := time.Now()
	tmpDir, err := os.MkdirTemp("", "qualiguard-zip-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmpDir)

	extracted, err := extractZipSafe(zipData, tmpDir)
	if err != nil {
		return nil, err
	}
	if extracted == 0 {
		return nil, fmt.Errorf("zip içinde dosya bulunamadı")
	}

	files, err := discovery.Discover(tmpDir, defaultArchiveLanguages, defaultArchiveExclusions, nil)
	if err != nil {
		return nil, err
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("zip içinde desteklenen kaynak dosyası yok (.py, .js, .go …)")
	}
	if len(files) > maxZipFiles {
		return nil, fmt.Errorf("çok fazla dosya (limit %d)", maxZipFiles)
	}

	registry := parser.NewRegistry()
	engine := rules.NewEngine()

	var (
		issues   []model.Issue
		analyses []model.FileAnalysis
		archFiles []model.FileSource
	)

	for _, absPath := range files {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		rel, err := filepath.Rel(tmpDir, absPath)
		if err != nil {
			rel = filepath.Base(absPath)
		}
		rel = filepath.ToSlash(rel)

		content, err := os.ReadFile(absPath)
		if err != nil {
			continue
		}

		lang := discovery.LanguageForPath(absPath)
		if lang == "" {
			lang = parser.InferLanguage(rel, string(content))
		}

		analysis, err := registry.AnalyzeFile(lang, absPath)
		if err != nil {
			continue
		}

		fileIssues := engine.AnalyzeForLanguage(lang, rel, analysis)
		fileIssues = appendExternalLinterIssues(ctx, lang, absPath, rel, fileIssues)

		for i := range fileIssues {
			if sn, err := snippet.FromFile(absPath, fileIssues[i].Line, 3); err == nil {
				fileIssues[i].Snippet = sn
			}
			if line, err := snippet.LineAt(absPath, fileIssues[i].Line); err == nil {
				fileIssues[i].FixSuggestion = rules.BuildFix(fileIssues[i], line)
			}
			fileIssues[i].File = rel
		}

		issues = append(issues, fileIssues...)
		analyses = append(analyses, analysis)
		archFiles = append(archFiles, model.FileSource{
			Filename: rel,
			Text:     string(content),
			Language: lang,
		})
	}

	if projectKey == "" {
		projectKey = projectKeyFromZipName(zipName)
	}
	if projectName == "" {
		projectName = strings.TrimSuffix(filepath.Base(zipName), filepath.Ext(zipName))
	}

	gateResult := gate.ToModelResult(gate.Evaluate(gate.InputFromIssues(issues)))

	return &model.Report{
		SchemaVersion:  model.SchemaVersion,
		ScannerVersion: version,
		Project: model.Project{
			Key:  projectKey,
			Name: projectName,
		},
		Analysis: model.AnalysisMeta{
			Timestamp:    time.Now().UTC(),
			DurationMS:   time.Since(start).Milliseconds(),
			ScannedFiles: len(files),
		},
		Archive: &model.ArchiveSource{
			ZipName: filepath.Base(zipName),
			Files:   archFiles,
		},
		Issues:   issues,
		Measures: metrics.FromAnalyses(analyses, issues),
		Gate:     &gateResult,
	}, nil
}

func projectKeyFromZipName(name string) string {
	base := strings.TrimSuffix(filepath.Base(name), filepath.Ext(filepath.Base(name)))
	base = strings.ToLower(base)
	var b strings.Builder
	for _, ch := range base {
		if (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') || ch == '-' || ch == '_' {
			b.WriteRune(ch)
		} else if ch == ' ' || ch == '.' {
			b.WriteRune('-')
		}
	}
	key := strings.Trim(b.String(), "-")
	if key == "" {
		return "zip-projesi"
	}
	return key
}
