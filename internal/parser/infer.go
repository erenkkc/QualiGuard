package parser

import (
	"path/filepath"
	"strings"
)

// InferLanguage picks a language from filename extension or source heuristics.
func InferLanguage(filename, source string) string {
	if lang := LanguageForFilename(filename); lang != "" {
		return lang
	}
	return inferLanguageFromSource(source)
}

func inferLanguageFromSource(source string) string {
	sample := strings.ToLower(source)
	if strings.Contains(sample, "package main") || strings.Contains(sample, "func ") && strings.Contains(sample, "fmt.") {
		return "go"
	}
	if strings.Contains(sample, "public class ") || strings.Contains(sample, "public static void main") {
		return "java"
	}
	if strings.Contains(sample, "namespace ") && strings.Contains(sample, "class ") {
		return "csharp"
	}
	if strings.Contains(sample, "const ") || strings.Contains(sample, "let ") || strings.Contains(sample, "function ") || strings.Contains(sample, "=>") {
		return "javascript"
	}
	if strings.Contains(sample, "def ") || strings.Contains(sample, "import ") || strings.Contains(sample, "except:") || strings.Contains(sample, "elif ") {
		return "python"
	}
	return "python"
}

// CanonicalFilename ensures analyzers receive a recognizable extension.
func CanonicalFilename(filename, lang string) string {
	base := filepath.Base(filename)
	if base == "" || base == "." {
		base = "upload"
	}
	if LanguageForFilename(base) != "" {
		return base
	}
	stem := strings.TrimSuffix(base, filepath.Ext(base))
	if stem == "" {
		stem = "upload"
	}
	switch lang {
	case "go":
		return stem + ".go"
	case "javascript":
		return stem + ".js"
	case "java":
		return stem + ".java"
	case "csharp":
		return stem + ".cs"
	default:
		return stem + ".py"
	}
}
