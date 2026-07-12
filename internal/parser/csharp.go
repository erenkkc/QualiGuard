package parser

import (
	"os"
	"regexp"
	"strings"

	"github.com/qualiguard/qualiguard/internal/model"
)

type CSharp struct{}

func NewCSharp() *CSharp { return &CSharp{} }

func (c *CSharp) Language() string { return "csharp" }
func (c *CSharp) Available() bool  { return true }

var (
	csSQLPattern    = regexp.MustCompile(`(?i)(SqlCommand|ExecuteReader|ExecuteNonQuery)[^;]*\+`)
	csMD5Pattern    = regexp.MustCompile(`(?i)MD5\.Create\s*\(|SHA1\.Create\s*\(|CreateHash\s*\(\s*["']MD5`)
	csSecretPattern = regexp.MustCompile(`(?i)(password|secret|api[_-]?key|token)\s*=\s*"[^"]{4,}"`)
)

func (c *CSharp) AnalyzeFile(path string) (model.FileAnalysis, error) {
	source, err := os.ReadFile(path)
	if err != nil {
		return model.FileAnalysis{}, err
	}
	return c.analyze(path, string(source)), nil
}

func (c *CSharp) analyze(path, source string) model.FileAnalysis {
	result := model.FileAnalysis{File: path, Ncloc: countGoNcloc(source)}
	lines := strings.Split(source, "\n")
	for i, line := range lines {
		lineNo := i + 1
		trim := strings.TrimSpace(line)
		if trim == "" || strings.HasPrefix(trim, "//") {
			continue
		}
		if csSQLPattern.MatchString(line) {
			result.Calls = append(result.Calls, model.CallInfo{Func: "SqlCommand", Line: lineNo, DynamicSQL: true, VariableArg: true})
		}
		if csMD5Pattern.MatchString(line) {
			result.Calls = append(result.Calls, model.CallInfo{Func: "MD5.Create", Line: lineNo})
		}
		if csSecretPattern.MatchString(line) {
			result.Secrets = append(result.Secrets, model.SecretInfo{Name: "secret", Line: lineNo})
		}
	}
	return result
}
