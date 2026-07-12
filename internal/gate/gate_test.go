package gate_test

import (
	"testing"

	"github.com/qualiguard/qualiguard/internal/gate"
	"github.com/qualiguard/qualiguard/internal/model"
)

func TestEvaluatePass(t *testing.T) {
	result := gate.Evaluate(gate.Input{})
	if result.Status != gate.StatusPass {
		t.Fatalf("expected PASS, got %s", result.Status)
	}
	if result.StatusTR != "Geçti" {
		t.Fatalf("expected Geçti, got %s", result.StatusTR)
	}
}

func TestEvaluateFailOnBlocker(t *testing.T) {
	result := gate.Evaluate(gate.Input{BlockerIssues: 1})
	if result.Status != gate.StatusFail {
		t.Fatalf("expected FAIL, got %s", result.Status)
	}
}

func TestInputFromIssues(t *testing.T) {
	issues := []model.Issue{
		{Severity: model.SeverityBlocker, Type: model.TypeVulnerability},
		{Severity: model.SeverityCritical, Type: model.TypeBug},
		{Severity: model.SeverityMajor, Type: model.TypeCodeSmell},
	}
	in := gate.InputFromIssues(issues)
	if in.BlockerIssues != 1 || in.CriticalIssues != 1 {
		t.Fatalf("unexpected severity counts: %+v", in)
	}
	if in.Vulnerabilities != 1 || in.Bugs != 1 || in.CodeSmells != 1 {
		t.Fatalf("unexpected type counts: %+v", in)
	}
}
