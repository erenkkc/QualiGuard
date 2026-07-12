package rules

import (
	"os"
	"regexp"
	"strings"

	"github.com/qualiguard/qualiguard/internal/fingerprint"
	"github.com/qualiguard/qualiguard/internal/model"
)

var (
	jsVarPattern      = regexp.MustCompile(`\bvar\s+[A-Za-z_$]`)
	jsLooseEqPattern  = regexp.MustCompile(`[^!=<>]==[^=]|[^!=<>]!=[^=]`)
	jsConsolePattern  = regexp.MustCompile(`\bconsole\.(log|debug|info)\s*\(`)
)

// JSStyleIssues adds built-in JavaScript quality checks when ESLint is unavailable.
func JSStyleIssues(path string) []model.Issue {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	text := strings.ReplaceAll(string(data), "\r\n", "\n")
	lines := strings.Split(text, "\n")

	var issues []model.Issue
	for i, line := range lines {
		trim := strings.TrimSpace(line)
		if trim == "" || strings.HasPrefix(trim, "//") || strings.HasPrefix(trim, "*") {
			continue
		}
		lineNo := i + 1

		if jsVarPattern.MatchString(line) {
			issues = append(issues, makeJSStyleIssue(path, lineNo, "javascript:no-var",
				"Use 'let' or 'const' instead of 'var'"))
		}
		if jsLooseEqPattern.MatchString(line) {
			issues = append(issues, makeJSStyleIssue(path, lineNo, "javascript:eqeqeq",
				"Use '===' and '!==' instead of '==' and '!='"))
		}
		if jsConsolePattern.MatchString(line) {
			issues = append(issues, makeJSStyleIssue(path, lineNo, "javascript:no-console",
				"Avoid console logging in production code"))
		}
	}

	return issues
}

func makeJSStyleIssue(path string, line int, ruleKey, message string) model.Issue {
	issue := model.Issue{
		RuleKey:   ruleKey,
		Severity:  model.SeverityMinor,
		Type:      model.TypeCodeSmell,
		Message:   message,
		File:      path,
		Line:      line,
		EffortMin: 2,
	}
	fingerprint.Annotate(&issue)
	return issue
}
