package ai_test

import (
	"context"
	"testing"

	"github.com/qualiguard/qualiguard/internal/ai"
	"github.com/qualiguard/qualiguard/internal/model"
)

func TestTemplateExplain(t *testing.T) {
	exp := ai.NewExplainer(false)
	out := exp.Explain(context.Background(), model.Issue{
		RuleKey:  "python:sql-injection",
		Severity: model.SeverityBlocker,
		Type:     model.TypeVulnerability,
		Message:  "SQL injection",
	})
	if out.SummaryTR == "" || out.RiskTR == "" {
		t.Fatal("expected template explanation")
	}
	if out.Source != "template" {
		t.Fatalf("expected template source, got %s", out.Source)
	}
}

func TestExplainOnDemandRequiresQuestion(t *testing.T) {
	exp := ai.NewExplainer(false)
	out := exp.ExplainOnDemand(context.Background(), model.Issue{
		RuleKey: "javascript:no-var",
		Message: "Unexpected var",
		Line:    10,
	}, "var x = 1;", "")
	if out.Source != "hint" {
		t.Fatalf("expected hint when question empty, got %s", out.Source)
	}
}

func TestExplainOnDemandWithoutLLM(t *testing.T) {
	exp := ai.NewExplainer(false)
	out := exp.ExplainOnDemand(context.Background(), model.Issue{
		RuleKey: "javascript:no-var",
		Message: "Unexpected var",
		Line:    10,
	}, "var x = 1;", "Bu ne demek?")
	if out.Source != "template" || out.SummaryTR == "" {
		t.Fatalf("expected offline message, got %+v", out)
	}
}
