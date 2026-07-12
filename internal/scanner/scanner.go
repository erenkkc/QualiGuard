package scanner

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/qualiguard/qualiguard/internal/config"
	"github.com/qualiguard/qualiguard/internal/discovery"
	"github.com/qualiguard/qualiguard/internal/gate"
	"github.com/qualiguard/qualiguard/internal/git"
	"github.com/qualiguard/qualiguard/internal/metrics"
	"github.com/qualiguard/qualiguard/internal/model"
	"github.com/qualiguard/qualiguard/internal/parser"
	"github.com/qualiguard/qualiguard/internal/rules"
	"github.com/qualiguard/qualiguard/internal/snippet"
)

type Scanner struct {
	cfg            config.Config
	workDir        string
	scannerVersion string
	analyzers      *parser.Registry
	rules          *rules.Engine
}

// ScanOptions controls optional scan behavior.
type ScanOptions struct {
	Incremental bool
	BaseRef     string
}

func New(cfg config.Config, workDir, scannerVersion string) *Scanner {
	return &Scanner{
		cfg:            cfg,
		workDir:        workDir,
		scannerVersion: scannerVersion,
		analyzers:      parser.NewRegistry(),
		rules:          rules.NewEngine(),
	}
}

func (s *Scanner) Scan(ctx context.Context, opts ScanOptions) (*model.Report, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	start := time.Now()
	sources, err := s.cfg.ResolveSources(s.workDir)
	if err != nil {
		return nil, err
	}

	if s.needsPython() {
		if py, ok := s.analyzers.Get("python"); !ok || !py.Available() {
			return nil, fmt.Errorf("python is required for Python analysis but was not found in PATH")
		}
	}

	var files []string
	for _, source := range sources {
		found, err := discovery.Discover(source, s.cfg.Languages, s.cfg.Exclusions, s.cfg.Inclusions)
		if err != nil {
			return nil, fmt.Errorf("discover files in %s: %w", source, err)
		}
		files = append(files, found...)
	}

	var repoRoot string
	if opts.Incremental {
		repoRoot, err = git.RepoRoot(s.workDir)
		if err != nil {
			return nil, err
		}
		changed, err := git.ChangedFiles(s.workDir, opts.BaseRef)
		if err != nil {
			return nil, err
		}
		files = git.FilterByChanged(repoRoot, files, changed)
	}

	var (
		analyses []model.FileAnalysis
		issues   []model.Issue
	)

	for _, file := range files {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		relative, err := filepath.Rel(s.workDir, file)
		if err != nil {
			relative = filepath.Base(file)
		}
		relative = filepath.ToSlash(relative)

		analysis, fileIssues, err := s.analyzePath(ctx, file, relative)
		if err != nil {
			return nil, err
		}
		analyses = append(analyses, analysis)
		issues = append(issues, fileIssues...)
	}

	report := &model.Report{
		SchemaVersion:  model.SchemaVersion,
		ScannerVersion: s.scannerVersion,
		Project:        s.cfg.Project,
		Analysis: model.AnalysisMeta{
			Timestamp:     time.Now().UTC(),
			DurationMS:    time.Since(start).Milliseconds(),
			Incremental:   opts.Incremental,
			BaseRef:       opts.BaseRef,
			ScannedFiles:  len(files),
		},
		Issues:   issues,
		Measures: metrics.FromAnalyses(analyses, issues),
	}
	gateResult := gate.ToModelResult(gate.Evaluate(gate.InputFromIssues(issues)))
	report.Gate = &gateResult
	return report, nil
}

func (s *Scanner) needsPython() bool {
	for _, lang := range s.cfg.Languages {
		if strings.EqualFold(lang, "python") {
			return true
		}
	}
	return false
}

func (s *Scanner) analyzePath(ctx context.Context, filePath, relative string) (model.FileAnalysis, []model.Issue, error) {
	if err := ctx.Err(); err != nil {
		return model.FileAnalysis{}, nil, err
	}

	lang := discovery.LanguageForPath(filePath)
	analysis, err := s.analyzers.AnalyzeFile(lang, filePath)
	if err != nil {
		return model.FileAnalysis{}, nil, err
	}
	if lang == "" {
		lang = parser.LanguageForFilename(filePath)
	}
	issues := s.rules.AnalyzeForLanguage(lang, relative, analysis)
	issues = appendExternalLinterIssues(ctx, lang, filePath, relative, issues)
	return analysis, s.enrichIssues(filePath, relative, issues), nil
}

func (s *Scanner) enrichIssues(filePath, relative string, fileIssues []model.Issue) []model.Issue {
	for i := range fileIssues {
		if snip, err := snippet.FromFile(filePath, fileIssues[i].Line, 1); err == nil {
			fileIssues[i].Snippet = snip
		}
		if line, err := snippet.LineAt(filePath, fileIssues[i].Line); err == nil {
			fileIssues[i].FixSuggestion = rules.BuildFix(fileIssues[i], line)
		}
		fileIssues[i].File = relative
	}
	return fileIssues
}
