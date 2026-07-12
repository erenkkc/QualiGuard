package config

import "testing"

func TestBrandWithDefaults(t *testing.T) {
	b := BrandConfig{Name: "CodeShield"}.WithDefaults()
	if b.Name != "CodeShield" {
		t.Fatalf("name = %q", b.Name)
	}
	if b.Tagline == "" || b.Accent == "" {
		t.Fatalf("defaults not applied: %+v", b)
	}
}

func TestBrandInvalidAccent(t *testing.T) {
	b := BrandConfig{Accent: "red"}.WithDefaults()
	if b.Accent != "#4d9fff" {
		t.Fatalf("accent = %q", b.Accent)
	}
}

func TestBrandHTMLReplacements(t *testing.T) {
	reps := BrandConfig{Name: "Test<Co"}.HTMLReplacements("qg_x")
	if reps["__QG_BRAND_NAME__"] != "Test&lt;Co" {
		t.Fatalf("escaped name = %q", reps["__QG_BRAND_NAME__"])
	}
	if reps["__QG_TOKEN__"] != "qg_x" {
		t.Fatalf("token missing")
	}
}
