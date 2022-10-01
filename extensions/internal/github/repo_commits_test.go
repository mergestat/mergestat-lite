package github_test

import (
	"testing"

	"github.com/mergestat/mergestat-lite/extensions/internal/tools"
)

func TestRepoCommits(t *testing.T) {
	cleanup := newRecorder(t)
	defer cleanup()

	db := Connect(t, Memory)

	rows, err := db.Query("SELECT * FROM github_repo_commits('mergestat/mergestat') LIMIT 5")
	if err != nil {
		t.Fatalf("failed to execute query: %v", err.Error())
	}
	defer rows.Close()

	colCount, _, err := tools.RowContent(rows)
	if err != nil {
		t.Fatalf("failed to retrieve row contents: %v", err.Error())
	}

	if expected := 12; colCount != expected {
		t.Fatalf("expected %d columns, got: %d", expected, colCount)
	}

	rows, err = db.Query("SELECT * FROM github_repo_commits('mergestat/mergestat','main') LIMIT 5")
	if err != nil {
		t.Fatalf("failed to execute query: %v", err.Error())
	}
	defer rows.Close()

	colCount, _, err = tools.RowContent(rows)
	if err != nil {
		t.Fatalf("failed to retrieve row contents: %v", err.Error())
	}

	if expected := 12; colCount != expected {
		t.Fatalf("expected %d columns, got: %d", expected, colCount)
	}
}
