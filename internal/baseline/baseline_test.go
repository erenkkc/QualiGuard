package baseline_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/qualiguard/qualiguard/internal/baseline"
	"github.com/qualiguard/qualiguard/internal/model"
)

func TestFilter(t *testing.T) {
	bl := baseline.File{Fingerprints: []string{"fp1"}}
	issues := []model.Issue{
		{Fingerprint: "fp1", Message: "skip"},
		{Fingerprint: "fp2", Message: "keep"},
	}
	out := bl.Filter(issues)
	if len(out) != 1 || out[0].Fingerprint != "fp2" {
		t.Fatalf("unexpected filter result: %+v", out)
	}
}

func TestSaveLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "baseline.json")
	issues := []model.Issue{{Fingerprint: "abc"}, {Fingerprint: "abc"}, {Fingerprint: "def"}}
	if err := baseline.Save(path, issues); err != nil {
		t.Fatal(err)
	}
	bl, err := baseline.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(bl.Fingerprints) != 2 {
		t.Fatalf("expected 2 fingerprints, got %d", len(bl.Fingerprints))
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
}
