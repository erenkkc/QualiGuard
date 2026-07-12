package store_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/qualiguard/qualiguard/internal/model"
	"github.com/qualiguard/qualiguard/internal/processor"
	"github.com/qualiguard/qualiguard/internal/store"
)

func TestDeleteProjectRemovesAllData(t *testing.T) {
	dir := t.TempDir()
	st, err := store.Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	ctx := context.Background()
	p := processor.New(st)
	report := &model.Report{
		Project: model.Project{Key: "del-me", Name: "Delete Me"},
		Issues: []model.Issue{
			{
				RuleKey:  "python:eval",
				Severity: model.SeverityCritical,
				Type:     model.TypeVulnerability,
				Message:  "eval kullanımı",
				File:     "bad.py",
				Line:     1,
			},
		},
	}
	if _, err := p.ProcessReport(ctx, report); err != nil {
		t.Fatal(err)
	}

	project, err := st.GetProjectByKey(ctx, "del-me")
	if err != nil || project == nil {
		t.Fatal("project not created")
	}
	issues, err := st.ListIssuesByProject(ctx, project.ID)
	if err != nil || len(issues) == 0 {
		t.Fatal("expected issues before delete")
	}

	if err := st.DeleteProject(ctx, project.ID); err != nil {
		t.Fatal(err)
	}

	got, err := st.GetProjectByKey(ctx, "del-me")
	if err == nil && got != nil {
		t.Fatal("project still exists after delete")
	}
	issues, err = st.ListIssuesByProject(ctx, project.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(issues) != 0 {
		t.Fatalf("expected 0 issues after delete, got %d", len(issues))
	}
}

func TestDeleteProjectNotFound(t *testing.T) {
	dir := t.TempDir()
	st, err := store.Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	err = st.DeleteProject(context.Background(), "missing-id")
	if err == nil {
		t.Fatal("expected error for missing project")
	}
}
