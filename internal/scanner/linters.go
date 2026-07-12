package scanner

import (
	"context"
	"strings"

	"github.com/qualiguard/qualiguard/internal/linter"
	"github.com/qualiguard/qualiguard/internal/model"
	"github.com/qualiguard/qualiguard/internal/rules"
)

func appendExternalLinterIssues(ctx context.Context, lang, filePath, relative string, issues []model.Issue) []model.Issue {
	switch strings.ToLower(lang) {
	case "javascript":
		eslintIssues, err := linter.RunESLint(ctx, filePath, relative)
		if err == nil && len(eslintIssues) > 0 {
			return append(issues, eslintIssues...)
		}

		styleIssues := rules.JSStyleIssues(filePath)
		for i := range styleIssues {
			styleIssues[i].File = relative
		}
		return append(issues, styleIssues...)

	case "python":
		ruffIssues, err := linter.RunRuff(ctx, filePath, relative)
		if err == nil && len(ruffIssues) > 0 {
			return append(issues, ruffIssues...)
		}
	}

	return issues
}
