package baseline

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/qualiguard/qualiguard/internal/model"
)

type File struct {
	Fingerprints []string `json:"fingerprints"`
}

func Load(path string) (File, error) {
	if path == "" {
		return File{}, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return File{}, fmt.Errorf("read baseline: %w", err)
	}
	var f File
	if err := json.Unmarshal(data, &f); err != nil {
		return File{}, fmt.Errorf("parse baseline: %w", err)
	}
	return f, nil
}

func (f File) Filter(issues []model.Issue) []model.Issue {
	if len(f.Fingerprints) == 0 {
		return issues
	}
	skip := make(map[string]struct{}, len(f.Fingerprints))
	for _, fp := range f.Fingerprints {
		skip[fp] = struct{}{}
	}
	out := make([]model.Issue, 0, len(issues))
	for _, issue := range issues {
		if _, ok := skip[issue.Fingerprint]; ok {
			continue
		}
		out = append(out, issue)
	}
	return out
}

func Save(path string, issues []model.Issue) error {
	fps := make([]string, 0, len(issues))
	seen := map[string]struct{}{}
	for _, issue := range issues {
		if issue.Fingerprint == "" {
			continue
		}
		if _, ok := seen[issue.Fingerprint]; ok {
			continue
		}
		seen[issue.Fingerprint] = struct{}{}
		fps = append(fps, issue.Fingerprint)
	}
	data, err := json.MarshalIndent(File{Fingerprints: fps}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}
