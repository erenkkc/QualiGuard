package metrics

import (
	"github.com/qualiguard/qualiguard/internal/model"
)

func FromAnalyses(analyses []model.FileAnalysis, issues []model.Issue) model.Measures {
	measures := model.Measures{
		Files: len(analyses),
	}

	for _, analysis := range analyses {
		measures.Ncloc += analysis.Ncloc
		for _, fn := range analysis.Functions {
			measures.Complexity += fn.Complexity
		}
	}

	for _, issue := range issues {
		switch issue.Type {
		case model.TypeBug:
			measures.Bugs++
		case model.TypeVulnerability:
			measures.Vulnerabilities++
		case model.TypeCodeSmell:
			measures.CodeSmells++
		case model.TypeSecurityHotspot:
			measures.SecurityHotspots++
		}
	}

	return measures
}

func FromIssues(issues []model.Issue, files, ncloc, complexity int) model.Measures {
	measures := model.Measures{
		Files:      files,
		Ncloc:      ncloc,
		Complexity: complexity,
	}
	for _, issue := range issues {
		switch issue.Type {
		case model.TypeBug:
			measures.Bugs++
		case model.TypeVulnerability:
			measures.Vulnerabilities++
		case model.TypeCodeSmell:
			measures.CodeSmells++
		case model.TypeSecurityHotspot:
			measures.SecurityHotspots++
		}
	}
	return measures
}
