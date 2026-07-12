package git

import (
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

const defaultBaseRef = "main"

// ChangedFiles returns repo-relative paths changed between baseRef and HEAD.
func ChangedFiles(workDir, baseRef string) ([]string, error) {
	root, err := RepoRoot(workDir)
	if err != nil {
		return nil, err
	}

	baseRef = strings.TrimSpace(baseRef)
	if baseRef == "" {
		baseRef = defaultBaseRef
	}

	base, err := resolveRef(root, baseRef)
	if err != nil {
		return nil, fmt.Errorf("resolve base ref %q: %w", baseRef, err)
	}

	mergeBase, err := runGit(root, "merge-base", "HEAD", base)
	if err != nil {
		return nil, fmt.Errorf("merge-base HEAD %s: %w", base, err)
	}

	out, err := runGit(root, "diff", "--name-only", "--diff-filter=ACMR", mergeBase, "HEAD")
	if err != nil {
		return nil, fmt.Errorf("git diff: %w", err)
	}

	var paths []string
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		paths = append(paths, filepath.ToSlash(line))
	}
	return paths, nil
}

// RepoRoot finds the git repository root for workDir.
func RepoRoot(workDir string) (string, error) {
	if !IsRepo(workDir) {
		return "", fmt.Errorf("not a git repository (incremental scan requires git)")
	}
	out, err := runGit(workDir, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", err
	}
	return filepath.Clean(out), nil
}

// IsRepo reports whether workDir is inside a git repository.
func IsRepo(workDir string) bool {
	_, err := runGit(workDir, "rev-parse", "--git-dir")
	return err == nil
}

func resolveRef(root, ref string) (string, error) {
	candidates := []string{ref}
	if !strings.Contains(ref, "/") {
		candidates = append(candidates, "origin/"+ref)
	}
	if ref == "main" {
		candidates = append(candidates, "master", "origin/master")
	}
	if ref == "master" {
		candidates = append(candidates, "main", "origin/main")
	}

	seen := make(map[string]struct{}, len(candidates))
	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		if _, ok := seen[candidate]; ok {
			continue
		}
		seen[candidate] = struct{}{}

		if _, err := runGit(root, "rev-parse", "--verify", candidate); err == nil {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("ref not found: %s", ref)
}

func runGit(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("%s", msg)
	}
	return strings.TrimSpace(stdout.String()), nil
}

// FilterByChanged keeps only files whose path relative to repoRoot is in changed.
func FilterByChanged(repoRoot string, files, changed []string) []string {
	if len(changed) == 0 {
		return nil
	}
	set := make(map[string]struct{}, len(changed))
	for _, p := range changed {
		set[filepath.ToSlash(p)] = struct{}{}
	}

	var out []string
	for _, file := range files {
		rel, err := filepath.Rel(repoRoot, file)
		if err != nil {
			continue
		}
		rel = filepath.ToSlash(rel)
		if _, ok := set[rel]; ok {
			out = append(out, file)
		}
	}
	return out
}
