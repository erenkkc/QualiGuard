package discovery_test

import (
	"path/filepath"
	"testing"

	"github.com/qualiguard/qualiguard/internal/discovery"
)

func TestDiscoverPythonFiles(t *testing.T) {
	root, err := filepath.Abs("../../testdata/sample_project")
	if err != nil {
		t.Fatal(err)
	}

	files, err := discovery.Discover(root, []string{"python"}, []string{"**/__pycache__/**"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) < 3 {
		t.Fatalf("expected at least 3 python files, got %d", len(files))
	}
}

func TestMatchGlobstar(t *testing.T) {
	if !discovery.MatchesAny("project/node_modules/pkg/index.js", []string{"**/node_modules/**"}) {
		t.Fatal("expected node_modules path to match")
	}
}
