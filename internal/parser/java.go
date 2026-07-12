package parser

import (
	"os"
	"regexp"
	"strings"

	"github.com/qualiguard/qualiguard/internal/model"
)

type Java struct{}

func NewJava() *Java { return &Java{} }

func (j *Java) Language() string  { return "java" }
func (j *Java) Available() bool   { return true }

var (
	javaSQLPattern    = regexp.MustCompile(`(?i)(executeQuery|executeUpdate|execute)\s*\([^)]*\+`)
	javaMD5Pattern    = regexp.MustCompile(`(?i)getInstance\s*\(\s*["']MD5["']|getInstance\s*\(\s*["']SHA-?1["']`)
	javaEvalPattern   = regexp.MustCompile(`(?i)\.eval\s*\(|ScriptEngine.*eval`)
	javaSecretPattern = regexp.MustCompile(`(?i)(password|secret|api[_-]?key|token)\s*=\s*["'][^"']{4,}["']`)
)

func (j *Java) AnalyzeFile(path string) (model.FileAnalysis, error) {
	source, err := os.ReadFile(path)
	if err != nil {
		return model.FileAnalysis{}, err
	}
	return j.analyze(path, string(source)), nil
}

func (j *Java) analyze(path, source string) model.FileAnalysis {
	result := model.FileAnalysis{File: path, Ncloc: countGoNcloc(source)}
	lines := strings.Split(source, "\n")
	for i, line := range lines {
		lineNo := i + 1
		trim := strings.TrimSpace(line)
		if trim == "" || strings.HasPrefix(trim, "//") {
			continue
		}
		if javaSQLPattern.MatchString(line) {
			result.Calls = append(result.Calls, model.CallInfo{Func: "Statement.execute", Line: lineNo, DynamicSQL: true, VariableArg: true})
		}
		if javaMD5Pattern.MatchString(line) {
			result.Calls = append(result.Calls, model.CallInfo{Func: "MessageDigest.getInstance", Line: lineNo})
		}
		if javaEvalPattern.MatchString(line) {
			result.Calls = append(result.Calls, model.CallInfo{Func: "eval", Line: lineNo})
		}
		if javaSecretPattern.MatchString(line) {
			result.Secrets = append(result.Secrets, model.SecretInfo{Name: "secret", Line: lineNo})
		}
	}
	return result
}
