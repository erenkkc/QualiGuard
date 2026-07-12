package rules

import (
	"strings"

	"github.com/qualiguard/qualiguard/internal/fingerprint"
	"github.com/qualiguard/qualiguard/internal/model"
)

func (e *Engine) AnalyzeGo(path string, analysis model.FileAnalysis) []model.Issue {
	var issues []model.Issue
	issues = append(issues, e.goHardcodedSecrets(path, analysis)...)
	issues = append(issues, e.goWeakCrypto(path, analysis)...)
	issues = append(issues, e.goSQLFormat(path, analysis)...)
	for i := range issues {
		fingerprint.Annotate(&issues[i])
	}
	return issues
}

func (e *Engine) AnalyzeForLanguage(lang, path string, analysis model.FileAnalysis) []model.Issue {
	switch lang {
	case "go":
		return e.AnalyzeGo(path, analysis)
	case "javascript":
		return e.AnalyzeJS(path, analysis)
	case "java":
		return e.AnalyzeJava(path, analysis)
	case "csharp":
		return e.AnalyzeCSharp(path, analysis)
	default:
		return e.Analyze(path, analysis)
	}
}

func (e *Engine) AnalyzeJava(path string, analysis model.FileAnalysis) []model.Issue {
	var issues []model.Issue
	issues = append(issues, e.javaSQL(path, analysis)...)
	issues = append(issues, e.javaWeakCrypto(path, analysis)...)
	issues = append(issues, e.javaEval(path, analysis)...)
	issues = append(issues, e.javaSecrets(path, analysis)...)
	for i := range issues {
		fingerprint.Annotate(&issues[i])
	}
	return issues
}

func (e *Engine) AnalyzeCSharp(path string, analysis model.FileAnalysis) []model.Issue {
	var issues []model.Issue
	issues = append(issues, e.csharpSQL(path, analysis)...)
	issues = append(issues, e.csharpWeakCrypto(path, analysis)...)
	issues = append(issues, e.csharpSecrets(path, analysis)...)
	for i := range issues {
		fingerprint.Annotate(&issues[i])
	}
	return issues
}

func (e *Engine) goHardcodedSecrets(path string, analysis model.FileAnalysis) []model.Issue {
	var issues []model.Issue
	for _, s := range analysis.Secrets {
		issues = append(issues, model.Issue{
			RuleKey:  "go:hardcoded-secret",
			Severity: model.SeverityCritical,
			Type:     model.TypeVulnerability,
			Message:  "Potential hardcoded secret in '" + s.Name + "'",
			File:     path,
			Line:     s.Line,
			EffortMin: 20,
		})
	}
	return issues
}

func (e *Engine) goWeakCrypto(path string, analysis model.FileAnalysis) []model.Issue {
	var issues []model.Issue
	for _, call := range analysis.Calls {
		name := strings.ToLower(call.Func)
		if strings.Contains(name, "md5") || strings.Contains(name, "sha1") {
			issues = append(issues, model.Issue{
				RuleKey:  "go:weak-crypto",
				Severity: model.SeverityMajor,
				Type:     model.TypeVulnerability,
				Message:  "Avoid MD5/SHA1 for security-sensitive hashing",
				File:     path,
				Line:     call.Line,
				EffortMin: 15,
			})
		}
	}
	return issues
}

func (e *Engine) goSQLFormat(path string, analysis model.FileAnalysis) []model.Issue {
	var issues []model.Issue
	for _, call := range analysis.Calls {
		name := strings.ToLower(call.Func)
		if !strings.Contains(name, "query") && !strings.Contains(name, "exec") {
			continue
		}
		if !call.VariableArg && !call.DynamicSQL {
			continue
		}
		issues = append(issues, model.Issue{
			RuleKey:  "go:sql-format",
			Severity: model.SeverityBlocker,
			Type:     model.TypeVulnerability,
			Message:  "Use parameterized queries instead of fmt.Sprintf in SQL",
			File:     path,
			Line:     call.Line,
			EffortMin: 30,
		})
	}
	return issues
}

func (e *Engine) AnalyzeJS(path string, analysis model.FileAnalysis) []model.Issue {
	var issues []model.Issue
	issues = append(issues, e.jsEval(path, analysis)...)
	issues = append(issues, e.jsInnerHTML(path, analysis)...)
	issues = append(issues, e.jsDocWrite(path, analysis)...)
	issues = append(issues, e.jsSecrets(path, analysis)...)
	for i := range issues {
		fingerprint.Annotate(&issues[i])
	}
	return issues
}

func (e *Engine) javaSQL(path string, analysis model.FileAnalysis) []model.Issue {
	var issues []model.Issue
	for _, call := range analysis.Calls {
		if call.Func != "Statement.execute" {
			continue
		}
		issues = append(issues, model.Issue{
			RuleKey:  "java:sql-concat",
			Severity: model.SeverityBlocker,
			Type:     model.TypeVulnerability,
			Message:  "Use PreparedStatement instead of string concatenation in SQL",
			File:     path,
			Line:     call.Line,
			EffortMin: 30,
		})
	}
	return issues
}

func (e *Engine) javaWeakCrypto(path string, analysis model.FileAnalysis) []model.Issue {
	var issues []model.Issue
	for _, call := range analysis.Calls {
		if call.Func != "MessageDigest.getInstance" {
			continue
		}
		issues = append(issues, model.Issue{
			RuleKey:  "java:weak-crypto",
			Severity: model.SeverityMajor,
			Type:     model.TypeVulnerability,
			Message:  "Avoid MD5/SHA1 for security-sensitive hashing",
			File:     path,
			Line:     call.Line,
			EffortMin: 15,
		})
	}
	return issues
}

func (e *Engine) javaEval(path string, analysis model.FileAnalysis) []model.Issue {
	var issues []model.Issue
	for _, call := range analysis.Calls {
		if call.Func != "eval" {
			continue
		}
		issues = append(issues, model.Issue{
			RuleKey:  "java:script-eval",
			Severity: model.SeverityCritical,
			Type:     model.TypeVulnerability,
			Message:  "Avoid dynamic script evaluation",
			File:     path,
			Line:     call.Line,
			EffortMin: 20,
		})
	}
	return issues
}

func (e *Engine) javaSecrets(path string, analysis model.FileAnalysis) []model.Issue {
	var issues []model.Issue
	for _, s := range analysis.Secrets {
		issues = append(issues, model.Issue{
			RuleKey:  "java:hardcoded-secret",
			Severity: model.SeverityCritical,
			Type:     model.TypeVulnerability,
			Message:  "Potential hardcoded secret in source code",
			File:     path,
			Line:     s.Line,
			EffortMin: 20,
		})
	}
	return issues
}

func (e *Engine) csharpSQL(path string, analysis model.FileAnalysis) []model.Issue {
	var issues []model.Issue
	for _, call := range analysis.Calls {
		if call.Func != "SqlCommand" {
			continue
		}
		issues = append(issues, model.Issue{
			RuleKey:  "csharp:sql-concat",
			Severity: model.SeverityBlocker,
			Type:     model.TypeVulnerability,
			Message:  "Use parameterized SqlCommand instead of string concatenation",
			File:     path,
			Line:     call.Line,
			EffortMin: 30,
		})
	}
	return issues
}

func (e *Engine) csharpWeakCrypto(path string, analysis model.FileAnalysis) []model.Issue {
	var issues []model.Issue
	for _, call := range analysis.Calls {
		if call.Func != "MD5.Create" {
			continue
		}
		issues = append(issues, model.Issue{
			RuleKey:  "csharp:weak-crypto",
			Severity: model.SeverityMajor,
			Type:     model.TypeVulnerability,
			Message:  "Avoid MD5/SHA1 for security-sensitive hashing",
			File:     path,
			Line:     call.Line,
			EffortMin: 15,
		})
	}
	return issues
}

func (e *Engine) csharpSecrets(path string, analysis model.FileAnalysis) []model.Issue {
	var issues []model.Issue
	for _, s := range analysis.Secrets {
		issues = append(issues, model.Issue{
			RuleKey:  "csharp:hardcoded-secret",
			Severity: model.SeverityCritical,
			Type:     model.TypeVulnerability,
			Message:  "Potential hardcoded secret in source code",
			File:     path,
			Line:     s.Line,
			EffortMin: 20,
		})
	}
	return issues
}

func (e *Engine) jsEval(path string, analysis model.FileAnalysis) []model.Issue {
	var issues []model.Issue
	for _, call := range analysis.Calls {
		if call.Func != "eval" {
			continue
		}
		issues = append(issues, model.Issue{
			RuleKey:  "javascript:eval-usage",
			Severity: model.SeverityCritical,
			Type:     model.TypeVulnerability,
			Message:  "Avoid eval()/Function() — code injection and XSS risk",
			File:     path,
			Line:     call.Line,
			EffortMin: 20,
		})
	}
	return issues
}

func (e *Engine) jsInnerHTML(path string, analysis model.FileAnalysis) []model.Issue {
	var issues []model.Issue
	for _, call := range analysis.Calls {
		if call.Func != "innerHTML" {
			continue
		}
		if call.HasUserInput {
			issues = append(issues, model.Issue{
				RuleKey:   "javascript:innerhtml-xss",
				Severity:  model.SeverityCritical,
				Type:      model.TypeVulnerability,
				Message:   "Avoid assigning dynamic HTML — use textContent or sanitize input",
				File:      path,
				Line:      call.Line,
				EffortMin: 20,
			})
			continue
		}
		issues = append(issues, model.Issue{
			RuleKey:   "javascript:innerhtml-static",
			Severity:  model.SeverityMinor,
			Type:      model.TypeCodeSmell,
			Message:   "Static innerHTML works but createElement is safer and clearer",
			File:      path,
			Line:      call.Line,
			EffortMin: 10,
		})
	}
	return issues
}

func (e *Engine) jsDocWrite(path string, analysis model.FileAnalysis) []model.Issue {
	var issues []model.Issue
	for _, call := range analysis.Calls {
		if call.Func != "document.write" {
			continue
		}
		issues = append(issues, model.Issue{
			RuleKey:  "javascript:document-write",
			Severity: model.SeverityMajor,
			Type:     model.TypeVulnerability,
			Message:  "document.write can enable DOM-based XSS",
			File:     path,
			Line:     call.Line,
			EffortMin: 15,
		})
	}
	return issues
}

func (e *Engine) jsSecrets(path string, analysis model.FileAnalysis) []model.Issue {
	var issues []model.Issue
	for _, s := range analysis.Secrets {
		issues = append(issues, model.Issue{
			RuleKey:  "javascript:hardcoded-secret",
			Severity: model.SeverityCritical,
			Type:     model.TypeVulnerability,
			Message:  "Hardcoded secret in client-side code",
			File:     path,
			Line:     s.Line,
			EffortMin: 20,
		})
	}
	return issues
}
