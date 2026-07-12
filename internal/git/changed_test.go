package git_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/qualiguard/qualiguard/internal/git"
)

func TestChangedFiles(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}

	dir := t.TempDir()
	runGit(t, dir, "init")
	runGit(t, dir, "config", "user.email", "test@test.com")
	runGit(t, dir, "config", "user.name", "Test")

	writeFile(t, dir, "a.py", "x = 1\n")
	writeFile(t, dir, "b.py", "y = 2\n")
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "initial")

	writeFile(t, dir, "b.py", "y = 3\n")
	runGit(t, dir, "add", "b.py")
	runGit(t, dir, "commit", "-m", "change b")

	changed, err := git.ChangedFiles(dir, "HEAD~1")
	if err != nil {
		t.Fatal(err)
	}
	if len(changed) != 1 || changed[0] != "b.py" {
		t.Fatalf("expected [b.py], got %v", changed)
	}
}

func TestFilterByChanged(t *testing.T) {
	root := t.TempDir()
	a := filepath.Join(root, "src", "a.py")
	b := filepath.Join(root, "src", "b.py")
	filtered := git.FilterByChanged(root, []string{a, b}, []string{"src/a.py"})
	if len(filtered) != 1 || filtered[0] != a {
		t.Fatalf("unexpected filter result: %v", filtered)
	}
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
