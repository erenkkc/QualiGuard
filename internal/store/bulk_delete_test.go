package store_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/qualiguard/qualiguard/internal/model"
	"github.com/qualiguard/qualiguard/internal/processor"
	"github.com/qualiguard/qualiguard/internal/store"
)

func TestDeleteProjectsByKeys(t *testing.T) {
	dir := t.TempDir()
	st, err := store.Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	ctx := context.Background()
	p := processor.New(st)
	for _, key := range []string{"bulk-a", "bulk-b"} {
		if _, err := p.ProcessReport(ctx, &model.Report{
			Project: model.Project{Key: key, Name: key},
		}); err != nil {
			t.Fatal(err)
		}
	}

	deleted, notFound, err := st.DeleteProjectsByKeys(ctx, []string{"bulk-a", "bulk-b", "missing"})
	if err != nil {
		t.Fatal(err)
	}
	if len(deleted) != 2 || len(notFound) != 1 {
		t.Fatalf("deleted=%v notFound=%v", deleted, notFound)
	}
}
