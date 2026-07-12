package parser

import "testing"

func TestInferLanguageFromTxt(t *testing.T) {
	src := "def hello():\n    pass\n"
	if got := InferLanguage("python.txt", src); got != "python" {
		t.Fatalf("expected python, got %s", got)
	}
}

func TestInferLanguageJS(t *testing.T) {
	src := "const x = 1;\nfunction foo() {}\n"
	if got := InferLanguage("code.txt", src); got != "javascript" {
		t.Fatalf("expected javascript, got %s", got)
	}
}

func TestCanonicalFilename(t *testing.T) {
	if got := CanonicalFilename("data.txt", "python"); got != "data.py" {
		t.Fatalf("unexpected canonical name: %s", got)
	}
}
