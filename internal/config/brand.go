package config

import (
	"fmt"
	"html"
	"regexp"
)

var hexColor = regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)

// BrandConfig — panel ve landing white-label ayarları.
type BrandConfig struct {
	Name    string `yaml:"name"`
	Tagline string `yaml:"tagline"`
	Eyebrow string `yaml:"eyebrow"`
	Accent  string `yaml:"accent"`
	Accent2 string `yaml:"accent2"`
}

func DefaultBrand() BrandConfig {
	return BrandConfig{
		Name:    "QualiGuard",
		Tagline: "Kod kalitesi platformu",
		Eyebrow: "Statik analiz · Kalite kapısı",
		Accent:  "#4d9fff",
		Accent2: "#7c6cff",
	}
}

func (b BrandConfig) WithDefaults() BrandConfig {
	d := DefaultBrand()
	if b.Name == "" {
		b.Name = d.Name
	}
	if b.Tagline == "" {
		b.Tagline = d.Tagline
	}
	if b.Eyebrow == "" {
		b.Eyebrow = d.Eyebrow
	}
	if !hexColor.MatchString(b.Accent) {
		b.Accent = d.Accent
	}
	if !hexColor.MatchString(b.Accent2) {
		b.Accent2 = d.Accent2
	}
	return b
}

func (b BrandConfig) ThemeCSS() string {
	b = b.WithDefaults()
	return fmt.Sprintf(
		`<style>:root{--accent:%s;--accent-2:%s;--accent-soft:color-mix(in srgb,%s 12%%,transparent);}</style>`,
		b.Accent, b.Accent2, b.Accent,
	)
}

func (b BrandConfig) HTMLReplacements(token string) map[string]string {
	b = b.WithDefaults()
	reps := map[string]string{
		"__QG_BRAND_NAME__":    html.EscapeString(b.Name),
		"__QG_BRAND_TAGLINE__": html.EscapeString(b.Tagline),
		"__QG_BRAND_EYEBROW__": html.EscapeString(b.Eyebrow),
		"__QG_THEME_CSS__":     b.ThemeCSS(),
	}
	if token != "" {
		reps["__QG_TOKEN__"] = token
	}
	return reps
}
