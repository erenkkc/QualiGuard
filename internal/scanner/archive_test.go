package scanner_test

import (
	"archive/zip"
	"bytes"
	"context"
	"testing"

	"github.com/qualiguard/qualiguard/internal/scanner"
)

func TestAnalyzeZipArchive(t *testing.T) {
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	f, err := w.Create("src/app.js")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.Write([]byte("var x = 1;\n")); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}

	report, err := scanner.AnalyzeZipArchive(context.Background(), buf.Bytes(), "demo.zip", "1.0.0", "demo", "Demo")
	if err != nil {
		t.Fatal(err)
	}
	if report.Archive == nil || len(report.Archive.Files) != 1 {
		t.Fatalf("expected 1 archived file, got %+v", report.Archive)
	}
	if len(report.Issues) == 0 {
		t.Fatal("expected issues from var usage")
	}
}

func TestAnalyzeZipArchiveNestedRoot(t *testing.T) {
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	entries := []struct {
		name    string
		content string
	}{
		{"23903045_Mehmet_Akif_Buyukdag/assets/app.js", "var x = 1;\n"},
		{"23903045_Mehmet_Akif_Buyukdag/assets/css/style.css", "body { color: red; }\n"},
		{"23903045_Mehmet_Akif_Buyukdag/index.html", "<html></html>\n"},
	}
	for _, e := range entries {
		f, err := w.Create(e.name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := f.Write([]byte(e.content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}

	report, err := scanner.AnalyzeZipArchive(context.Background(), buf.Bytes(), "mehmet.zip", "1.0.0", "demo", "Demo")
	if err != nil {
		t.Fatal(err)
	}
	if report.Archive == nil || len(report.Archive.Files) == 0 {
		t.Fatalf("expected archived files, got %+v", report.Archive)
	}
}

func TestAnalyzeZipArchiveBackslashPaths(t *testing.T) {
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	f, err := w.Create("root\\assets\\app.js")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.Write([]byte("let a = 1;\n")); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}

	report, err := scanner.AnalyzeZipArchive(context.Background(), buf.Bytes(), "demo.zip", "1.0.0", "demo", "Demo")
	if err != nil {
		t.Fatal(err)
	}
	if report.Archive == nil || len(report.Archive.Files) != 1 {
		t.Fatalf("expected 1 archived file, got %+v", report.Archive)
	}
}
