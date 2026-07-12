package reporter

import (
	"strings"
	"testing"

	"github.com/qualiguard/qualiguard/internal/model"
)

func TestRenderHTMLReportContainsProjectAndIssue(t *testing.T) {
	gate := &model.GateResult{Status: "FAIL", StatusTR: "Kaldı"}
	issues := []model.Issue{
		{
			Severity: model.SeverityCritical,
			Type:     model.TypeVulnerability,
			File:     "app.py",
			Line:     12,
			Message:  "SQL injection",
			Status:   "OPEN",
		},
	}
	measures := map[string]float64{"files": 1, "ncloc": 42}

	html := RenderHTMLReport("Demo Proje", "demo-key", gate, issues, measures)
	for _, want := range []string{"Demo Proje", "demo-key", "Kaldı", "SQL injection", "app.py", "QualiGuard"} {
		if !strings.Contains(html, want) {
			t.Fatalf("expected HTML to contain %q", want)
		}
	}
	if strings.Contains(html, "<script") {
		t.Fatal("report should not contain script tags")
	}
	if !strings.Contains(html, "window.print") {
		t.Fatal("report should include print action")
	}
}
