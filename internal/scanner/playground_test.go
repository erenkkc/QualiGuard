package scanner_test

import (
	"context"
	"strings"
	"testing"

	"github.com/qualiguard/qualiguard/internal/scanner"
)

const sampleBadCode = `import os

DB_PASSWORD = "secret123"

def risky(user_id):
    unused = 1
    try:
        pass
    except:
        pass
    eval("1")
    cursor = None
    cursor.execute(f"SELECT * FROM users WHERE id = {user_id}")
`

func TestAnalyzePythonSource(t *testing.T) {
	report, err := scanner.AnalyzePythonSource(context.Background(), sampleBadCode, "test")
	if err != nil {
		if strings.Contains(err.Error(), "Python") {
			t.Skip("python not available")
		}
		t.Fatal(err)
	}

	if len(report.Issues) == 0 {
		t.Fatal("expected issues")
	}

	withFix := 0
	withSnippet := 0
	for _, issue := range report.Issues {
		if issue.FixSuggestion != "" {
			withFix++
		}
		if issue.Snippet != "" {
			withSnippet++
		}
		if issue.Line <= 0 {
			t.Fatalf("invalid line for %s", issue.RuleKey)
		}
	}

	if withFix == 0 {
		t.Fatal("expected fix suggestions")
	}
	if withSnippet == 0 {
		t.Fatal("expected snippets")
	}

	keys := map[string]bool{}
	for _, issue := range report.Issues {
		keys[issue.RuleKey] = true
	}
	for _, expected := range []string{
		"python:sql-injection",
		"python:bare-except",
		"python:hardcoded-password",
	} {
		if !keys[expected] {
			t.Fatalf("missing rule %s", expected)
		}
	}
}

const sampleSecurityCode = `PASSWORD = "admin123"

def run(user_input):
    os.system(user_input)
    db.execute("DELETE FROM users WHERE name = '" + user_input + "'")
    eval(user_input)
`

func TestAnalyzePythonSourceSecurity(t *testing.T) {
	report, err := scanner.AnalyzePythonSource(context.Background(), sampleSecurityCode, "test")
	if err != nil {
		if strings.Contains(err.Error(), "Python") {
			t.Skip("python not available")
		}
		t.Fatal(err)
	}

	keys := map[string]bool{}
	for _, issue := range report.Issues {
		keys[issue.RuleKey] = true
	}
	for _, expected := range []string{
		"python:hardcoded-password",
		"python:command-injection",
		"python:sql-injection",
		"python:eval-usage",
	} {
		if !keys[expected] {
			t.Fatalf("missing rule %s", expected)
		}
	}
}

func TestAnalyzePythonSourceWithSyntaxTail(t *testing.T) {
	code := sampleSecurityCode + "\ngfjşfhj4\nj ghjgy\n"
	report, err := scanner.AnalyzePythonSource(context.Background(), code, "test")
	if err != nil {
		if strings.Contains(err.Error(), "Python") {
			t.Skip("python not available")
		}
		t.Fatal(err)
	}

	keys := map[string]bool{}
	for _, issue := range report.Issues {
		keys[issue.RuleKey] = true
	}
	if !keys["python:syntax-error"] {
		t.Fatal("expected syntax-error issue")
	}
	for _, expected := range []string{
		"python:hardcoded-password",
		"python:command-injection",
		"python:sql-injection",
		"python:eval-usage",
	} {
		if !keys[expected] {
			t.Fatalf("missing rule %s with trailing syntax error", expected)
		}
	}
}

func TestAnalyzePythonSourceClean(t *testing.T) {
	clean := `def add(a, b):
    return a + b
`
	report, err := scanner.AnalyzePythonSource(context.Background(), clean, "test")
	if err != nil {
		if strings.Contains(err.Error(), "Python") {
			t.Skip("python not available")
		}
		t.Fatal(err)
	}
	if len(report.Issues) != 0 {
		t.Fatalf("expected no issues, got %d", len(report.Issues))
	}
}
