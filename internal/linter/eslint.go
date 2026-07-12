package linter

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/qualiguard/qualiguard/internal/fingerprint"
	"github.com/qualiguard/qualiguard/internal/model"
)

//go:embed eslintrc.json
var eslintConfigJSON string

type eslintResult struct {
	FilePath string          `json:"filePath"`
	Messages []eslintMessage `json:"messages"`
}

type eslintMessage struct {
	RuleID   string `json:"ruleId"`
	Severity int    `json:"severity"`
	Message  string `json:"message"`
	Line     int    `json:"line"`
	Column   int    `json:"column"`
}

// ESLintAvailable reports whether npx/eslint can be invoked.
func ESLintAvailable() bool {
	_, err := resolveNPX()
	return err == nil
}

// RunESLint runs ESLint on a single JavaScript/TypeScript file.
// Returns nil issues without error when ESLint is not installed.
func RunESLint(ctx context.Context, filePath, relativePath string) ([]model.Issue, error) {
	if !ESLintAvailable() {
		return nil, nil
	}

	configPath, err := writeTempESLintConfig()
	if err != nil {
		return nil, err
	}
	defer os.Remove(configPath)

	npx, err := resolveNPX()
	if err != nil {
		return nil, nil
	}

	args := []string{
		"--yes", "eslint@8.57.1",
		filePath,
		"--format", "json",
		"--no-eslintrc",
		"--config", configPath,
		"--no-error-on-unmatched-pattern",
	}

	runCtx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()

	cmd := exec.CommandContext(runCtx, npx, args...)
	cmd.Env = os.Environ()
	out, runErr := cmd.Output()
	if runErr != nil {
		var exitErr *exec.ExitError
		if errors.As(runErr, &exitErr) && exitErr.ExitCode() == 1 {
			// exit 1 = lint issues found; stdout still valid
		} else if runCtx.Err() != nil {
			return nil, nil
		} else {
			return nil, nil
		}
	}

	if len(strings.TrimSpace(string(out))) == 0 && runErr != nil {
		return nil, nil
	}

	return parseESLintOutput(out, displayPath(relativePath, filePath))
}

func displayPath(relativePath, filePath string) string {
	if strings.TrimSpace(relativePath) != "" {
		return filepath.ToSlash(relativePath)
	}
	return filepath.ToSlash(filepath.Base(filePath))
}

func parseESLintOutput(data []byte, file string) ([]model.Issue, error) {
	var results []eslintResult
	if err := json.Unmarshal(data, &results); err != nil {
		return nil, fmt.Errorf("parse eslint json: %w", err)
	}

	var issues []model.Issue
	for _, result := range results {
		for _, msg := range result.Messages {
			if msg.Line <= 0 || msg.RuleID == "" {
				continue
			}
			issue := model.Issue{
				RuleKey:   "eslint:" + msg.RuleID,
				Severity:  mapESLintSeverity(msg.RuleID, msg.Severity),
				Type:      mapESLintType(msg.RuleID),
				Message:   msg.Message,
				File:      file,
				Line:      msg.Line,
				Column:    msg.Column,
				EffortMin: 5,
			}
			fingerprint.Annotate(&issue)
			issues = append(issues, issue)
		}
	}
	return issues, nil
}

func mapESLintSeverity(ruleID string, severity int) model.Severity {
	if isESLintSecurityRule(ruleID) {
		return model.SeverityCritical
	}
	if severity >= 2 {
		return model.SeverityMajor
	}
	return model.SeverityMinor
}

func mapESLintType(ruleID string) model.IssueType {
	if isESLintSecurityRule(ruleID) {
		return model.TypeVulnerability
	}
	return model.TypeCodeSmell
}

func isESLintSecurityRule(ruleID string) bool {
	switch ruleID {
	case "no-eval", "no-implied-eval", "no-new-func", "no-script-url":
		return true
	default:
		return false
	}
}

func writeTempESLintConfig() (string, error) {
	f, err := os.CreateTemp("", "qualiguard-eslint-*.json")
	if err != nil {
		return "", err
	}
	if _, err := f.WriteString(eslintConfigJSON); err != nil {
		f.Close()
		return "", err
	}
	if err := f.Close(); err != nil {
		return "", err
	}
	return f.Name(), nil
}

func resolveNPX() (string, error) {
	if runtime.GOOS == "windows" {
		for _, name := range []string{"npx.cmd", "npx.exe", "npx"} {
			if path, err := exec.LookPath(name); err == nil {
				return path, nil
			}
		}
		return "", fmt.Errorf("npx not found")
	}
	return exec.LookPath("npx")
}
