package scanner

import (
	"context"
	"os"
	"path/filepath"
	"time"

	"github.com/qualiguard/qualiguard/internal/gate"
	"github.com/qualiguard/qualiguard/internal/metrics"
	"github.com/qualiguard/qualiguard/internal/model"
	"github.com/qualiguard/qualiguard/internal/parser"
	"github.com/qualiguard/qualiguard/internal/rules"
	"github.com/qualiguard/qualiguard/internal/snippet"
)

func AnalyzePythonSource(ctx context.Context, source, version string) (*model.Report, error) {
	return AnalyzePlaygroundSource(ctx, source, version)
}

// AnalyzePlaygroundSource detects language from source and analyzes accordingly.
func AnalyzePlaygroundSource(ctx context.Context, source, version string) (*model.Report, error) {
	lang := parser.InferLanguage("playground.txt", source)
	filename := "playground.py"
	switch lang {
	case "javascript":
		filename = "playground.js"
	case "go":
		filename = "playground.go"
	case "java":
		filename = "playground.java"
	case "csharp":
		filename = "playground.cs"
	}
	return AnalyzeSourceFile(ctx, source, filename, version, "playground", "Canlı Analiz")
}

func AnalyzePythonFile(ctx context.Context, source, filename, version, projectKey, projectName string) (*model.Report, error) {
	return AnalyzeSourceFile(ctx, source, filename, version, projectKey, projectName)
}

func AnalyzeSourceFile(ctx context.Context, source, filename, version, projectKey, projectName string) (*model.Report, error) {
	start := time.Now()

	if filename == "" {
		filename = "source.py"
	}
	originalName := filepath.Base(filename)
	if originalName == "." || originalName == "" {
		originalName = "source.py"
	}

	lang := parser.InferLanguage(originalName, source)
	canonicalName := parser.CanonicalFilename(originalName, lang)

	tmpDir, err := os.MkdirTemp("", "qualiguard-import-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmpDir)

	filePath := filepath.Join(tmpDir, canonicalName)
	if err := os.WriteFile(filePath, []byte(source), 0o644); err != nil {
		return nil, err
	}

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	registry := parser.NewRegistry()
	analysis, err := registry.AnalyzeFile(lang, filePath)
	if err != nil {
		return nil, err
	}

	engine := rules.NewEngine()
	issues := engine.AnalyzeForLanguage(lang, originalName, analysis)
	issues = appendExternalLinterIssues(ctx, lang, filePath, originalName, issues)

	for i := range issues {
		if snip, err := snippet.FromFile(filePath, issues[i].Line, 3); err == nil {
			issues[i].Snippet = snip
		}
		if line, err := snippet.LineAt(filePath, issues[i].Line); err == nil {
			issues[i].FixSuggestion = rules.BuildFix(issues[i], line)
		}
		issues[i].File = originalName
	}

	analyses := []model.FileAnalysis{analysis}
	gateResult := gate.ToModelResult(gate.Evaluate(gate.InputFromIssues(issues)))
	if projectKey == "" {
		projectKey = "import"
	}
	if projectName == "" {
		projectName = projectKey
	}
	return &model.Report{
		SchemaVersion:  model.SchemaVersion,
		ScannerVersion: version,
		Project: model.Project{
			Key:  projectKey,
			Name: projectName,
		},
		Analysis: model.AnalysisMeta{
			Timestamp:  time.Now().UTC(),
			DurationMS: time.Since(start).Milliseconds(),
		},
		Source: &model.FileSource{
			Filename: originalName,
			Text:     source,
			Language: lang,
		},
		Issues:   issues,
		Measures: metrics.FromAnalyses(analyses, issues),
		Gate:     &gateResult,
	}, nil
}

func errPythonRequired() error {
	return &PlaygroundError{Message: "Python 3 bulunamadı. Python yükleyin veya QUALIGUARD_PYTHON ortam değişkenini ayarlayın."}
}

type PlaygroundError struct {
	Message string
}

func (e *PlaygroundError) Error() string {
	return e.Message
}
