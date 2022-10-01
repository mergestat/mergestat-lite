package github_test

import (
	"testing"

	"github.com/mergestat/mergestat-lite/extensions/internal/tools"
)

func TestRepoBranchProtections(t *testing.T) {
	cleanup := newRecorder(t)
	defer cleanup()

	db := Connect(t, Memory)

	rows, err := db.Query("SELECT * FROM github_repo_branch_protections('askgitdev', 'askgit') LIMIT 10")
	if err != nil {
		t.Fatalf("failed to execute query: %v", err.Error())
	}
	defer rows.Close()

	colCount, content, err := tools.RowContent(rows)
	if err != nil {
		t.Fatalf("failed to retrieve row contents: %v", err.Error())
	}

	if colCount != 18 {
		t.Fatalf("expected 18 columns, got: %d", colCount)
	}

	// TODO(patrickdevivo) setup a fixture repo for more branch protection rules?
	if len(content) != 1 {
		t.Fatalf("expected 1 rows, got: %d", len(content))
	}
}
