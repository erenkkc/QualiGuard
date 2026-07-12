package github_test

import (
	"strings"
	"testing"

	"github.com/qualiguard/qualiguard/internal/github"
	"github.com/qualiguard/qualiguard/internal/model"
)

func TestBuildPRCommentPassWithStyleIssues(t *testing.T) {
	report := &model.Report{
		ScannerVersion: "1.0.0",
		Project:        model.Project{Key: "demo"},
		Measures:       model.Measures{Files: 1},
		Gate: &model.GateResult{
			NameTR:   "QualiGuard Yolu",
			Status:   "PASS",
			StatusTR: "Geçti",
			Conditions: []model.GateCondition{
				{LabelTR: "Engelleyici sorun sayısı", Actual: 0, Threshold: 0, Passed: true},
				{LabelTR: "Hata (bug) sayısı", Actual: 0, Threshold: 0, Passed: true},
			},
		},
		Issues: []model.Issue{
			{RuleKey: "javascript:no-var", Severity: model.SeverityMinor, Type: model.TypeCodeSmell, Message: "var kullanmayın", File: "script.js", Line: 3},
			{RuleKey: "javascript:no-var", Severity: model.SeverityMinor, Type: model.TypeCodeSmell, Message: "var kullanmayın", File: "script.js", Line: 7},
		},
	}

	body := github.BuildPRComment(report)
	if !strings.Contains(body, github.CommentMarker) {
		t.Fatal("missing comment marker")
	}
	if !strings.Contains(body, "**2**") {
		t.Fatalf("expected total count in table, got:\n%s", body)
	}
	if !strings.Contains(body, "✅") || !strings.Contains(body, "Geçti") {
		t.Fatalf("expected pass status, got:\n%s", body)
	}
	if !strings.Contains(body, "Stil uyarısı") {
		t.Fatalf("expected style note, got:\n%s", body)
	}
	if !strings.Contains(body, "`script.js`") {
		t.Fatalf("expected issue table row, got:\n%s", body)
	}
}

func TestBuildPRCommentFail(t *testing.T) {
	report := &model.Report{
		ScannerVersion: "1.0.0",
		Gate: &model.GateResult{
			Status:   "FAIL",
			StatusTR: "Kaldı",
			Conditions: []model.GateCondition{
				{LabelTR: "Kritik sorun sayısı", Actual: 2, Threshold: 0, Passed: false},
			},
		},
		Issues: []model.Issue{
			{RuleKey: "python:sql-injection", Severity: model.SeverityCritical, Type: model.TypeVulnerability, Message: "SQL injection riski", File: "app.py", Line: 12},
		},
	}

	body := github.BuildPRComment(report)
	if !strings.Contains(body, "❌") || !strings.Contains(body, "Kaldı") {
		t.Fatalf("expected fail status, got:\n%s", body)
	}
	if !strings.Contains(body, "geçemedi") {
		t.Fatalf("expected failure note, got:\n%s", body)
	}
}
