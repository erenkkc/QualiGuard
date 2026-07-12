package processor_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/qualiguard/qualiguard/internal/fingerprint"
	"github.com/qualiguard/qualiguard/internal/model"
	"github.com/qualiguard/qualiguard/internal/processor"
	"github.com/qualiguard/qualiguard/internal/store"
)

func TestSuppressedIssueIgnoredOnRescan(t *testing.T) {
	dir := t.TempDir()
	st, err := store.Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	p := processor.New(st)
	ctx := context.Background()

	issue := model.Issue{
		RuleKey:  "javascript:no-var",
		Severity: model.SeverityMinor,
		Type:     model.TypeCodeSmell,
		Message:  "var kullanmayın",
		File:     "script.js",
		Line:     3,
	}
	fingerprint.Annotate(&issue)

	report := &model.Report{
		Project: model.Project{Key: "fp-test", Name: "FP Test"},
		Issues:  []model.Issue{issue},
	}
	if _, err := p.ProcessReport(ctx, report); err != nil {
		t.Fatal(err)
	}
	projectID := mustProjectID(t, st, "fp-test")

	issues, err := st.ListIssuesByProject(ctx, projectID)
	if err != nil || len(issues) != 1 {
		t.Fatalf("expected 1 issue, got %d err=%v", len(issues), err)
	}
	if err := st.ResolveIssue(ctx, projectID, issues[0].ID, model.ResolutionFalsePositive); err != nil {
		t.Fatal(err)
	}

	if _, err := p.ProcessReport(ctx, report); err != nil {
		t.Fatal(err)
	}

	issues, err = st.ListIssuesByProject(ctx, projectID)
	if err != nil {
		t.Fatal(err)
	}
	if len(issues) != 1 {
		t.Fatalf("expected 1 issue row, got %d", len(issues))
	}
	if issues[0].Status != model.StatusClosed || issues[0].Resolution != model.ResolutionFalsePositive {
		t.Fatalf("expected suppressed closed issue, got status=%s resolution=%s", issues[0].Status, issues[0].Resolution)
	}

	open, err := st.CountOpenIssues(ctx, projectID)
	if err != nil {
		t.Fatal(err)
	}
	if open != 0 {
		t.Fatalf("expected 0 open issues after suppression, got %d", open)
	}
}
