package parser

import (
	"os"
	"regexp"
	"strings"

	"github.com/qualiguard/qualiguard/internal/model"
)

type JavaScript struct{}

func NewJavaScript() *JavaScript { return &JavaScript{} }

func (j *JavaScript) Language() string { return "javascript" }

func (j *JavaScript) Available() bool { return true }

var (
	jsEvalPattern      = regexp.MustCompile(`\b(eval|Function)\s*\(`)
	jsInnerHTMLPattern = regexp.MustCompile(`\.(innerHTML|outerHTML)\s*=|dangerouslySetInnerHTML`)
	jsDocWritePattern  = regexp.MustCompile(`\bdocument\.write\s*\(`)
	jsSecretPattern    = regexp.MustCompile(`(?i)(password|secret|api[_-]?key|token)\s*[:=]\s*['"][^'"]{4,}['"]`)
)

func (j *JavaScript) AnalyzeFile(path string) (model.FileAnalysis, error) {
	source, err := os.ReadFile(path)
	if err != nil {
		return model.FileAnalysis{}, err
	}
	return j.AnalyzeSource(path, string(source)), nil
}

func (j *JavaScript) AnalyzeSource(path, source string) model.FileAnalysis {
	result := model.FileAnalysis{
		File:  path,
		Ncloc: countGoNcloc(source),
	}

	lines := strings.Split(source, "\n")
	for i, line := range lines {
		lineNo := i + 1
		trim := strings.TrimSpace(line)
		if trim == "" || strings.HasPrefix(trim, "//") {
			continue
		}

		if jsEvalPattern.MatchString(line) {
			result.Calls = append(result.Calls, model.CallInfo{Func: "eval", Line: lineNo})
		}
		if jsInnerHTMLPattern.MatchString(line) {
			dynamic := isDynamicInnerHTML(lines, i)
			result.Calls = append(result.Calls, model.CallInfo{
				Func:         "innerHTML",
				Line:         lineNo,
				HasUserInput: dynamic,
			})
		}
		if jsDocWritePattern.MatchString(line) {
			result.Calls = append(result.Calls, model.CallInfo{Func: "document.write", Line: lineNo})
		}
		if loc := jsSecretPattern.FindStringSubmatchIndex(line); loc != nil {
			name := strings.TrimSpace(line[:loc[0]])
			if idx := strings.LastIndexAny(name, " \t:="); idx >= 0 {
				name = strings.TrimSpace(name[idx+1:])
			}
			if name == "" {
				name = "secret"
			}
			result.Secrets = append(result.Secrets, model.SecretInfo{Name: name, Line: lineNo})
		}
	}
	return result
}

// isDynamicInnerHTML checks whether innerHTML assignment may include untrusted/dynamic content.
func isDynamicInnerHTML(lines []string, start int) bool {
	var block strings.Builder
	for i := start; i < len(lines) && i < start+8; i++ {
		block.WriteString(lines[i])
		if strings.Contains(lines[i], ";") {
			break
		}
	}
	text := block.String()
	if strings.Contains(text, "${") {
		return true
	}
	if regexp.MustCompile(`\.(innerHTML|outerHTML)\s*=\s*[A-Za-z_$][\w$]*`).MatchString(text) {
		return true
	}
	for _, part := range strings.Split(text, "+") {
		trim := strings.TrimSpace(part)
		if trim == "" {
			continue
		}
		if strings.Contains(trim, "innerHTML") || strings.Contains(trim, "outerHTML") {
			continue
		}
		if !strings.HasPrefix(trim, "'") && !strings.HasPrefix(trim, "\"") && !strings.HasPrefix(trim, "`") {
			return true
		}
	}
	return false
}
