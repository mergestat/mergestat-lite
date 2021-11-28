package github_test

import (
	"testing"

	"github.com/mergestat/mergestat/extensions/internal/tools"
)

func TestRepoIssues(t *testing.T) {
	cleanup := newRecorder(t)
	defer cleanup()

	db := Connect(t, Memory)

	rows, err := db.Query("SELECT * FROM github_repo_issues('mergestat/mergestat') LIMIT 10")
	if err != nil {
		t.Fatalf("failed to execute query: %v", err.Error())
	}
	defer rows.Close()

	colCount, content, err := tools.RowContent(rows)
	if err != nil {
		t.Fatalf("failed to retrieve row contents: %v", err.Error())
	}

	if colCount != 22 {
		t.Fatalf("expected 22 columns, got: %d", colCount)
	}

	if len(content) != 10 {
		t.Fatalf("expected 10 rows, got: %d", len(content))
	}
}
