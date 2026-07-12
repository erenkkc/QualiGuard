package rules

import (
	"path/filepath"
	"strings"

	"github.com/qualiguard/qualiguard/internal/fingerprint"
	"github.com/qualiguard/qualiguard/internal/model"
)

type Engine struct{}

func NewEngine() *Engine {
	return &Engine{}
}

func (e *Engine) Analyze(relativePath string, analysis model.FileAnalysis) []model.Issue {
	var issues []model.Issue

	issues = append(issues, e.syntaxError(relativePath, analysis)...)

	issues = append(issues, e.unusedImports(relativePath, analysis)...)
	issues = append(issues, e.unusedVariables(relativePath, analysis)...)
	issues = append(issues, e.bareExcept(relativePath, analysis)...)
	issues = append(issues, e.emptyExcept(relativePath, analysis)...)
	issues = append(issues, e.complexFunctions(relativePath, analysis)...)
	issues = append(issues, e.longFunctions(relativePath, analysis)...)
	issues = append(issues, e.evalUsage(relativePath, analysis)...)
	issues = append(issues, e.commandInjection(relativePath, analysis)...)
	issues = append(issues, e.sqlInjection(relativePath, analysis)...)
	issues = append(issues, e.hardcodedSecrets(relativePath, analysis)...)
	issues = append(issues, e.pickleUsage(relativePath, analysis)...)
	issues = append(issues, e.weakHash(relativePath, analysis)...)
	issues = append(issues, e.debugBreakpoint(relativePath, analysis)...)
	issues = append(issues, e.assertUsage(relativePath, analysis)...)

	for i := range issues {
		fingerprint.Annotate(&issues[i])
	}
	return issues
}

func (e *Engine) syntaxError(path string, analysis model.FileAnalysis) []model.Issue {
	if analysis.ParseError == nil {
		return nil
	}
	issue := model.Issue{
		RuleKey:   "python:syntax-error",
		Severity:  model.SeverityBlocker,
		Type:      model.TypeBug,
		Message:   "Unable to parse file: " + analysis.ParseError.Message,
		File:      path,
		Line:      analysis.ParseError.Line,
		Column:    analysis.ParseError.Column,
		EffortMin: 30,
	}
	fingerprint.Annotate(&issue)
	return []model.Issue{issue}
}

func (e *Engine) unusedImports(path string, analysis model.FileAnalysis) []model.Issue {
	var issues []model.Issue
	for _, imp := range analysis.Imports {
		if imp.Used {
			continue
		}
		name := imp.Name
		if imp.Alias != "" {
			name = imp.Alias
		}
		issues = append(issues, model.Issue{
			RuleKey:  "python:unused-import",
			Severity: model.SeverityMinor,
			Type:     model.TypeCodeSmell,
			Message:  "Remove unused import '" + name + "'",
			File:     path,
			Line:     imp.Line,
			EffortMin: 1,
		})
	}
	return issues
}

func (e *Engine) unusedVariables(path string, analysis model.FileAnalysis) []model.Issue {
	var issues []model.Issue
	for _, assign := range analysis.Assignments {
		if assign.Used || assign.Name == "_" {
			continue
		}
		issues = append(issues, model.Issue{
			RuleKey:  "python:unused-variable",
			Severity: model.SeverityMinor,
			Type:     model.TypeCodeSmell,
			Message:  "Remove unused variable '" + assign.Name + "'",
			File:     path,
			Line:     assign.Line,
			EffortMin: 2,
		})
	}
	return issues
}

func (e *Engine) bareExcept(path string, analysis model.FileAnalysis) []model.Issue {
	var issues []model.Issue
	for _, block := range analysis.ExceptBlocks {
		if !block.Bare {
			continue
		}
		issues = append(issues, model.Issue{
			RuleKey:  "python:bare-except",
			Severity: model.SeverityCritical,
			Type:     model.TypeBug,
			Message:  "Avoid bare except; catch specific exceptions",
			File:     path,
			Line:     block.Line,
			EffortMin: 5,
		})
	}
	return issues
}

func (e *Engine) emptyExcept(path string, analysis model.FileAnalysis) []model.Issue {
	var issues []model.Issue
	for _, block := range analysis.ExceptBlocks {
		if !block.Empty {
			continue
		}
		issues = append(issues, model.Issue{
			RuleKey:  "python:empty-except",
			Severity: model.SeverityMajor,
			Type:     model.TypeBug,
			Message:  "Except block should not be empty",
			File:     path,
			Line:     block.Line,
			EffortMin: 5,
		})
	}
	return issues
}

func (e *Engine) complexFunctions(path string, analysis model.FileAnalysis) []model.Issue {
	var issues []model.Issue
	for _, fn := range analysis.Functions {
		if fn.Complexity <= 10 {
			continue
		}
		issues = append(issues, model.Issue{
			RuleKey:  "python:complex-function",
			Severity: model.SeverityMajor,
			Type:     model.TypeCodeSmell,
			Message:  "Function '" + fn.Name + "' has cyclomatic complexity " + itoa(fn.Complexity) + " (threshold: 10)",
			File:     path,
			Line:     fn.Line,
			EffortMin: 15,
		})
	}
	return issues
}

func (e *Engine) longFunctions(path string, analysis model.FileAnalysis) []model.Issue {
	var issues []model.Issue
	for _, fn := range analysis.Functions {
		length := fn.EndLine - fn.Line + 1
		if length <= 50 {
			continue
		}
		issues = append(issues, model.Issue{
			RuleKey:  "python:long-function",
			Severity: model.SeverityMajor,
			Type:     model.TypeCodeSmell,
			Message:  "Function '" + fn.Name + "' is " + itoa(length) + " lines long (threshold: 50)",
			File:     path,
			Line:     fn.Line,
			EffortMin: 20,
		})
	}
	return issues
}

func (e *Engine) evalUsage(path string, analysis model.FileAnalysis) []model.Issue {
	var issues []model.Issue
	for _, call := range analysis.Calls {
		base := filepath.Base(strings.ReplaceAll(call.Func, ".", "/"))
		if base != "eval" && base != "exec" && call.Func != "eval" && call.Func != "exec" {
			continue
		}
		issues = append(issues, model.Issue{
			RuleKey:  "python:eval-usage",
			Severity: model.SeverityCritical,
			Type:     model.TypeVulnerability,
			Message:  "Avoid using eval/exec; this can lead to code injection",
			File:     path,
			Line:     call.Line,
			EffortMin: 30,
		})
	}
	return issues
}

func (e *Engine) commandInjection(path string, analysis model.FileAnalysis) []model.Issue {
	var issues []model.Issue
	for _, call := range analysis.Calls {
		if !isShellCommand(call.Func) {
			continue
		}
		if !call.VariableArg && !call.HasUserInput {
			continue
		}
		issues = append(issues, model.Issue{
			RuleKey:   "python:command-injection",
			Severity:  model.SeverityBlocker,
			Type:      model.TypeVulnerability,
			Message:   "Avoid passing variables to os.system/subprocess — command injection risk",
			File:      path,
			Line:      call.Line,
			EffortMin: 30,
		})
	}
	return issues
}

func (e *Engine) sqlInjection(path string, analysis model.FileAnalysis) []model.Issue {
	var issues []model.Issue
	for _, call := range analysis.Calls {
		if !isSQLExecute(call.Func) {
			continue
		}
		if call.IsFString || call.HasUserInput || call.DynamicSQL {
			issues = append(issues, model.Issue{
				RuleKey:  "python:sql-injection",
				Severity: model.SeverityBlocker,
				Type:     model.TypeVulnerability,
				Message:  "Use parameterized queries instead of string formatting in SQL",
				File:     path,
				Line:     call.Line,
				EffortMin: 30,
			})
		}
	}
	return issues
}

func (e *Engine) hardcodedSecrets(path string, analysis model.FileAnalysis) []model.Issue {
	var issues []model.Issue
	for _, secret := range analysis.Secrets {
		issues = append(issues, model.Issue{
			RuleKey:  "python:hardcoded-password",
			Severity: model.SeverityCritical,
			Type:     model.TypeVulnerability,
			Message:  "Potential hardcoded secret in '" + secret.Name + "'",
			File:     path,
			Line:     secret.Line,
			EffortMin: 20,
		})
	}
	return issues
}

func (e *Engine) pickleUsage(path string, analysis model.FileAnalysis) []model.Issue {
	var issues []model.Issue
	for _, call := range analysis.Calls {
		if !isPickleCall(call.Func) {
			continue
		}
		issues = append(issues, model.Issue{
			RuleKey:   "python:pickle-usage",
			Severity:  model.SeverityCritical,
			Type:      model.TypeVulnerability,
			Message:   "Avoid pickle with untrusted data — arbitrary code execution risk",
			File:      path,
			Line:      call.Line,
			EffortMin: 30,
		})
	}
	return issues
}

func (e *Engine) weakHash(path string, analysis model.FileAnalysis) []model.Issue {
	var issues []model.Issue
	for _, call := range analysis.Calls {
		if !isWeakHashCall(call.Func) {
			continue
		}
		issues = append(issues, model.Issue{
			RuleKey:   "python:weak-hash",
			Severity:  model.SeverityMajor,
			Type:      model.TypeVulnerability,
			Message:   "Avoid MD5/SHA1 for security-sensitive hashing",
			File:      path,
			Line:      call.Line,
			EffortMin: 15,
		})
	}
	return issues
}

func (e *Engine) debugBreakpoint(path string, analysis model.FileAnalysis) []model.Issue {
	var issues []model.Issue
	for _, call := range analysis.Calls {
		if !isDebugCall(call.Func) {
			continue
		}
		issues = append(issues, model.Issue{
			RuleKey:   "python:debug-breakpoint",
			Severity:  model.SeverityMajor,
			Type:      model.TypeCodeSmell,
			Message:   "Remove debug breakpoint before production",
			File:      path,
			Line:      call.Line,
			EffortMin: 2,
		})
	}
	return issues
}

func (e *Engine) assertUsage(path string, analysis model.FileAnalysis) []model.Issue {
	var issues []model.Issue
	for _, call := range analysis.Calls {
		if call.Func != "assert" {
			continue
		}
		issues = append(issues, model.Issue{
			RuleKey:   "python:assert-usage",
			Severity:  model.SeverityMinor,
			Type:      model.TypeCodeSmell,
			Message:   "Do not rely on assert for runtime validation (stripped with -O)",
			File:      path,
			Line:      call.Line,
			EffortMin: 5,
		})
	}
	return issues
}

func isShellCommand(funcName string) bool {
	name := strings.ToLower(funcName)
	switch name {
	case "os.system", "os.popen", "subprocess.call", "subprocess.run", "subprocess.popen":
		return true
	}
	return strings.HasSuffix(name, ".system") || strings.HasSuffix(name, ".popen")
}

func isSQLExecute(funcName string) bool {
	name := strings.ToLower(funcName)
	return strings.HasSuffix(name, ".execute") || name == "execute" || strings.HasSuffix(name, "executemany")
}

func isPickleCall(funcName string) bool {
	name := strings.ToLower(funcName)
	switch name {
	case "pickle.load", "pickle.loads", "_pickle.load", "_pickle.loads",
		"cPickle.load", "cPickle.loads", "pickle.Unpickler":
		return true
	}
	return strings.HasSuffix(name, ".load") && strings.Contains(name, "pickle")
}

func isWeakHashCall(funcName string) bool {
	name := strings.ToLower(funcName)
	switch name {
	case "hashlib.md5", "hashlib.sha1", "md5", "sha1":
		return true
	}
	return strings.HasSuffix(name, ".md5") || strings.HasSuffix(name, ".sha1")
}

func isDebugCall(funcName string) bool {
	name := strings.ToLower(funcName)
	switch name {
	case "breakpoint", "pdb.set_trace", "pdb.post_mortem", "pdb.pm", "ipdb.set_trace":
		return true
	}
	return strings.HasSuffix(name, ".set_trace")
}

func itoa(v int) string {
	if v == 0 {
		return "0"
	}
	digits := []byte{}
	n := v
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}
