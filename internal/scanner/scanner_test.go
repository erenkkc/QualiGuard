package scanner_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/qualiguard/qualiguard/internal/config"
	"github.com/qualiguard/qualiguard/internal/parser"
	"github.com/qualiguard/qualiguard/internal/scanner"
)

func TestScanSampleProject(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}

	cfg, err := config.Load(filepath.Join(root, "qualiguard.yaml"))
	if err != nil {
		t.Fatal(err)
	}

	if !parser.NewPython().Available() {
		t.Skip("python not available")
	}

	report, err := scanner.New(cfg, root, "test").Scan(context.Background(), scanner.ScanOptions{})
	if err != nil {
		t.Fatal(err)
	}

	if report.Measures.Files < 3 {
		t.Fatalf("expected at least 3 files, got %d", report.Measures.Files)
	}
	if len(report.Issues) == 0 {
		t.Fatal("expected issues in sample project")
	}
	if report.Measures.Vulnerabilities == 0 {
		t.Fatal("expected vulnerabilities in bad.py")
	}
}
