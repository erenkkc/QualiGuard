package scanner

import (
	"context"
	"time"

	"github.com/qualiguard/qualiguard/internal/gate"
	"github.com/qualiguard/qualiguard/internal/metrics"
	"github.com/qualiguard/qualiguard/internal/model"
)

// RescanStoredSource re-analyzes a project's saved single file or zip manifest.
func RescanStoredSource(ctx context.Context, projectKey, projectName, filename, source, version string) (*model.Report, error) {
	if arch, ok := model.ParseArchiveManifest(source); ok {
		return rescanArchive(ctx, arch, projectKey, projectName, version)
	}
	return AnalyzeSourceFile(ctx, source, filename, version, projectKey, projectName)
}

func rescanArchive(ctx context.Context, arch *model.ArchiveSource, projectKey, projectName, version string) (*model.Report, error) {
	start := time.Now()
	var (
		allIssues  []model.Issue
		analyses   []model.FileAnalysis
		totalMeas  model.Measures
	)

	for _, f := range arch.Files {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		sub, err := AnalyzeSourceFile(ctx, f.Text, f.Filename, version, projectKey, projectName)
		if err != nil {
			continue
		}
		for i := range sub.Issues {
			sub.Issues[i].File = f.Filename
			allIssues = append(allIssues, sub.Issues[i])
		}
		totalMeas.Files += sub.Measures.Files
		totalMeas.Ncloc += sub.Measures.Ncloc
		totalMeas.Complexity += sub.Measures.Complexity
		totalMeas.CognitiveComplexity += sub.Measures.CognitiveComplexity
		if len(sub.Issues) > 0 {
			analyses = append(analyses, model.FileAnalysis{File: f.Filename})
		}
	}

	if totalMeas.Files == 0 {
		totalMeas.Files = len(arch.Files)
	}
	meas := metrics.FromAnalyses(analyses, allIssues)
	if meas.Ncloc == 0 {
		meas = totalMeas
		meas.Bugs = countByType(allIssues, model.TypeBug)
		meas.Vulnerabilities = countByType(allIssues, model.TypeVulnerability)
		meas.CodeSmells = countByType(allIssues, model.TypeCodeSmell)
	}

	gateResult := gate.ToModelResult(gate.Evaluate(gate.InputFromIssues(allIssues)))
	zipName := arch.ZipName
	if zipName == "" {
		zipName = "archive.zip"
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
		Archive:  arch,
		Issues:   allIssues,
		Measures: meas,
		Gate:     &gateResult,
	}, nil
}

func countByType(issues []model.Issue, t model.IssueType) int {
	n := 0
	for _, i := range issues {
		if i.Type == t {
			n++
		}
	}
	return n
}
