package github_test

import (
	"testing"

	"github.com/mergestat/mergestat-lite/extensions/internal/tools"
)

func TestBranches(t *testing.T) {
	cleanup := newRecorder(t)
	defer cleanup()

	db := Connect(t, Memory)

	rows, err := db.Query("SELECT * FROM github_repo_branches('mergestat/mergestat') LIMIT 10")
	if err != nil {
		t.Fatalf("failed to execute query: %v", err.Error())
	}
	defer rows.Close()

	colCount, _, err := tools.RowContent(rows)
	if err != nil {
		t.Fatalf("failed to retrieve row contents: %v", err.Error())
	}

	if expected := 4; colCount != expected {
		t.Fatalf("expected %d columns, got: %d", expected, colCount)
	}

	rows, err = db.Query("SELECT * FROM github_repo_branches('askgitdev', 'askgit') LIMIT 10")
	if err != nil {
		t.Fatalf("failed to execute query: %v", err.Error())
	}
	defer rows.Close()

	colCount, _, err = tools.RowContent(rows)
	if err != nil {
		t.Fatalf("failed to retrieve row contents: %v", err.Error())
	}

	if expected := 4; colCount != expected {
		t.Fatalf("expected %d columns, got: %d", expected, colCount)
	}

}
