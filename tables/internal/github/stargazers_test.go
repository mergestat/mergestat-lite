package github_test

import (
	"testing"

	"github.com/askgitdev/askgit/tables/internal/tools"
)

func TestStargazers(t *testing.T) {
	cleanup := newRecorder(t)
	defer cleanup()

	db := Connect(t, Memory)

	rows, err := db.Query("SELECT * FROM github_stargazers('askgitdev/askgit') LIMIT 500")
	if err != nil {
		t.Fatalf("failed to execute query: %v", err.Error())
	}
	defer rows.Close()

	colCount, content, err := tools.RowContent(rows)
	if err != nil {
		t.Fatalf("failed to retrieve row contents: %v", err.Error())
	}

	if colCount != 12 {
		t.Fatalf("expected 12 columns, got: %d", colCount)
	}

	if len(content) != 500 {
		t.Fatalf("expected 500 rows, got: %d", len(content))
	}
}
