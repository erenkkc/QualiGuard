package linter

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/qualiguard/qualiguard/internal/fingerprint"
	"github.com/qualiguard/qualiguard/internal/model"
)

type ruffRunner struct {
	bin  string
	args []string
}

type ruffDiagnostic struct {
	Code     string `json:"code"`
	Message  string `json:"message"`
	Filename string `json:"filename"`
	Location struct {
		Row    int `json:"row"`
		Column int `json:"column"`
	} `json:"location"`
}

// RuffAvailable reports whether ruff can be invoked.
func RuffAvailable() bool {
	_, err := resolveRuff()
	return err == nil
}

// RunRuff runs Ruff on a single Python file.
// Returns nil issues without error when Ruff is not installed.
func RunRuff(ctx context.Context, filePath, relativePath string) ([]model.Issue, error) {
	runner, err := resolveRuff()
	if err != nil {
		return nil, nil
	}

	args := append(append([]string{}, runner.args...),
		"check",
		"--output-format", "json",
		"--no-fix",
		filePath,
	)

	runCtx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()

	cmd := exec.CommandContext(runCtx, runner.bin, args...)
	cmd.Env = os.Environ()
	out, runErr := cmd.Output()
	if runErr != nil {
		var exitErr *exec.ExitError
		if errors.As(runErr, &exitErr) && exitErr.ExitCode() == 1 {
			// exit 1 = lint issues found
		} else if runCtx.Err() != nil {
			return nil, nil
		} else {
			return nil, nil
		}
	}

	if len(strings.TrimSpace(string(out))) == 0 {
		return nil, nil
	}

	return parseRuffOutput(out, displayPath(relativePath, filePath))
}

func parseRuffOutput(data []byte, file string) ([]model.Issue, error) {
	var diagnostics []ruffDiagnostic
	if err := json.Unmarshal(data, &diagnostics); err != nil {
		return nil, fmt.Errorf("parse ruff json: %w", err)
	}

	var issues []model.Issue
	for _, diag := range diagnostics {
		if diag.Code == "" || diag.Location.Row <= 0 {
			continue
		}
		issue := model.Issue{
			RuleKey:   "ruff:" + diag.Code,
			Severity:  mapRuffSeverity(diag.Code),
			Type:      mapRuffType(diag.Code),
			Message:   diag.Message,
			File:      file,
			Line:      diag.Location.Row,
			Column:    diag.Location.Column,
			EffortMin: 5,
		}
		fingerprint.Annotate(&issue)
		issues = append(issues, issue)
	}
	return issues, nil
}

func mapRuffSeverity(code string) model.Severity {
	if isRuffSecurityRule(code) {
		return model.SeverityCritical
	}
	switch {
	case strings.HasPrefix(code, "B"),
		strings.HasPrefix(code, "E9"),
		strings.HasPrefix(code, "F8"):
		return model.SeverityMajor
	default:
		return model.SeverityMinor
	}
}

func mapRuffType(code string) model.IssueType {
	if isRuffSecurityRule(code) {
		return model.TypeVulnerability
	}
	if strings.HasPrefix(code, "B") {
		return model.TypeBug
	}
	return model.TypeCodeSmell
}

func isRuffSecurityRule(code string) bool {
	return strings.HasPrefix(code, "S")
}

func resolveRuff() (ruffRunner, error) {
	candidates := []string{"ruff"}
	if runtime.GOOS == "windows" {
		candidates = append(candidates, "ruff.exe")
	}
	for _, name := range candidates {
		if path, err := exec.LookPath(name); err == nil {
			return ruffRunner{bin: path}, nil
		}
	}

	pythonBins := []struct {
		bin  string
		args []string
	}{
		{"python3", nil},
		{"python", nil},
		{"py", []string{"-3"}},
	}
	for _, py := range pythonBins {
		if path, err := exec.LookPath(py.bin); err == nil {
			testArgs := append(append([]string{}, py.args...), "-m", "ruff", "--version")
			if exec.Command(path, testArgs...).Run() == nil {
				moduleArgs := append(append([]string{}, py.args...), "-m", "ruff")
				return ruffRunner{bin: path, args: moduleArgs}, nil
			}
		}
	}

	return ruffRunner{}, fmt.Errorf("ruff not found")
}

// RuffCommand returns a human-readable hint for installing Ruff.
func RuffCommand() string {
	if runtime.GOOS == "windows" {
		return "pip install ruff"
	}
	return "pip install ruff  # veya: curl -LsSf https://astral.sh/ruff/install.sh | sh"
}
