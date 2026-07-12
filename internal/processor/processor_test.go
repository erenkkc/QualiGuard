package processor_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/qualiguard/qualiguard/internal/model"
	"github.com/qualiguard/qualiguard/internal/processor"
	"github.com/qualiguard/qualiguard/internal/store"
)

func TestIssueMergeAndFixFields(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	st, err := store.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	p := processor.New(st)
	ctx := context.Background()

	report := &model.Report{
		Project: model.Project{Key: "test-merge", Name: "Test"},
		Issues: []model.Issue{
			{
				RuleKey:       "python:sql-injection",
				Severity:      model.SeverityBlocker,
				Type:          model.TypeVulnerability,
				Message:       "SQL injection",
				File:          "app.py",
				Line:          10,
				Snippet:       "> 10 | bad",
				FixSuggestion: "cursor.execute(...)",
			},
		},
	}

	r1, err := p.ProcessReport(ctx, report)
	if err != nil {
		t.Fatal(err)
	}
	if r1.IssuesNew != 1 {
		t.Fatalf("expected 1 new issue, got %d", r1.IssuesNew)
	}

	issues, err := st.ListIssuesByProject(ctx, mustProjectID(t, st, "test-merge"))
	if err != nil {
		t.Fatal(err)
	}
	if len(issues) != 1 {
		t.Fatalf("expected 1 issue in db, got %d", len(issues))
	}
	if issues[0].FixSuggestion == "" {
		t.Fatal("fix suggestion not stored")
	}
	if issues[0].Snippet == "" {
		t.Fatal("snippet not stored")
	}

	report.Issues = nil
	r2, err := p.ProcessReport(ctx, report)
	if err != nil {
		t.Fatal(err)
	}
	if r2.IssuesClosed != 1 {
		t.Fatalf("expected 1 closed, got %d", r2.IssuesClosed)
	}
}

func mustProjectID(t *testing.T, st *store.Store, key string) string {
	t.Helper()
	p, err := st.GetProjectByKey(context.Background(), key)
	if err != nil || p == nil {
		t.Fatal("project not found")
	}
	return p.ID
}
