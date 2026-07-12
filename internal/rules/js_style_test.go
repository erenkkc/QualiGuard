package rules

import (
	"os"
	"path/filepath"
	"testing"
)

func TestJSStyleIssues(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.js")
	content := "var x = 1;\nif (a == b) {}\nconsole.log('hi');\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	issues := JSStyleIssues(path)
	if len(issues) < 3 {
		t.Fatalf("expected at least 3 issues, got %d", len(issues))
	}
}
