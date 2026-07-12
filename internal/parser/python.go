package parser

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/qualiguard/qualiguard/internal/model"
)

//go:embed python_analyzer.py
var pythonAnalyzerScript string

type pythonRunner struct {
	bin  string
	args []string
}

type Python struct {
	runner pythonRunner
}

func (p *Python) Language() string { return "python" }

func NewPython() *Python {
	return &Python{runner: resolvePython()}
}

func resolvePython() pythonRunner {
	if custom := strings.TrimSpace(os.Getenv("QUALIGUARD_PYTHON")); custom != "" {
		return pythonRunner{bin: custom}
	}

	if runtime.GOOS == "windows" {
		if path, err := exec.LookPath("py"); err == nil && pythonWorks(path, "-3") {
			return pythonRunner{bin: path, args: []string{"-3"}}
		}
	}

	for _, candidate := range []string{"python3", "python"} {
		path, err := exec.LookPath(candidate)
		if err != nil {
			continue
		}
		if isWindowsStoreStub(path) {
			continue
		}
		if pythonWorks(path) {
			return pythonRunner{bin: path}
		}
	}

	if runtime.GOOS == "windows" {
		for _, candidate := range windowsPythonCandidates() {
			if pythonWorks(candidate) {
				return pythonRunner{bin: candidate}
			}
		}
	}

	return pythonRunner{bin: "python"}
}

func windowsPythonCandidates() []string {
	localAppData := os.Getenv("LOCALAPDATA")
	if localAppData == "" {
		return nil
	}
	base := filepath.Join(localAppData, "Programs", "Python")
	entries, err := os.ReadDir(base)
	if err != nil {
		return nil
	}
	var paths []string
	for _, entry := range entries {
		if !entry.IsDir() || !strings.HasPrefix(entry.Name(), "Python") {
			continue
		}
		paths = append(paths, filepath.Join(base, entry.Name(), "python.exe"))
	}
	return paths
}

func isWindowsStoreStub(path string) bool {
	lower := strings.ToLower(filepath.Clean(path))
	return strings.Contains(lower, `\windowsapps\`) || strings.Contains(lower, `\microsoft\windowsapps\`)
}

func pythonWorks(bin string, extraArgs ...string) bool {
	args := append(append([]string{}, extraArgs...), "-c", "import sys; print(sys.version_info[0])")
	cmd := exec.Command(bin, args...)
	cmd.Env = os.Environ()
	out, err := cmd.CombinedOutput()
	return err == nil && strings.TrimSpace(string(out)) != ""
}

func (p *Python) commandArgs(scriptPath, targetPath string) []string {
	args := append([]string{}, p.runner.args...)
	args = append(args, scriptPath, targetPath)
	return args
}

func (p *Python) AnalyzeFile(path string) (model.FileAnalysis, error) {
	scriptPath, err := p.writeTempScript()
	if err != nil {
		return model.FileAnalysis{}, err
	}
	defer os.Remove(scriptPath)

	cmd := exec.Command(p.runner.bin, p.commandArgs(scriptPath, path)...)
	cmd.Env = os.Environ()
	out, err := cmd.CombinedOutput()
	textOut := strings.TrimSpace(string(out))
	if err != nil {
		msg := pythonErrorMessage(textOut, err)
		return model.FileAnalysis{}, fmt.Errorf("analyze %s: %s", filepath.Base(path), msg)
	}

	var analysis model.FileAnalysis
	if err := json.Unmarshal(out, &analysis); err != nil {
		return model.FileAnalysis{}, fmt.Errorf("decode analysis for %s: %w", filepath.Base(path), err)
	}

	analysis.File = path
	return analysis, nil
}

func pythonErrorMessage(output string, err error) string {
	lower := strings.ToLower(output)
	if strings.Contains(lower, "python bulunamad") || strings.Contains(lower, "not found") {
		return "Python kurulu değil veya PATH'te yok. Python 3 yükleyin veya QUALIGUARD_PYTHON ortam değişkenini ayarlayın."
	}
	if output != "" {
		if len(output) > 240 {
			output = output[:240] + "..."
		}
		return output
	}
	return fmt.Sprintf("%v (Python 3 gerekli — QUALIGUARD_PYTHON ile yol verebilirsiniz)", err)
}

func (p *Python) writeTempScript() (string, error) {
	file, err := os.CreateTemp("", "qualiguard-analyzer-*.py")
	if err != nil {
		return "", err
	}
	if _, err := file.WriteString(pythonAnalyzerScript); err != nil {
		file.Close()
		return "", err
	}
	if err := file.Close(); err != nil {
		return "", err
	}
	return file.Name(), nil
}

func (p *Python) Available() bool {
	return pythonWorks(p.runner.bin, p.runner.args...)
}

func (p *Python) Interpreter() string {
	if len(p.runner.args) == 0 {
		return p.runner.bin
	}
	return p.runner.bin + " " + strings.Join(p.runner.args, " ")
}
