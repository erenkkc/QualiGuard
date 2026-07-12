package rules

import "testing"

func TestFixJSNoVar(t *testing.T) {
	got := fixJSNoVar("  var hamburger = document.querySelector('.hamburger');")
	if !containsAll(got, "const", "Önce:", "Sonra") {
		t.Fatalf("expected detailed fix, got: %s", got)
	}
}

func TestFixSyntaxErrorJSHint(t *testing.T) {
	got := fixSyntaxError("unexpected indent", "var x = 1;")
	if !containsAll(got, "JavaScript", "Dosya Yükle") {
		t.Fatalf("expected JS hint, got: %s", got)
	}
}

func containsAll(s string, parts ...string) bool {
	for _, p := range parts {
		if !contains(s, p) {
			return false
		}
	}
	return true
}

func contains(s, sub string) bool {
	return len(sub) == 0 || (len(s) >= len(sub) && indexOf(s, sub) >= 0)
}
