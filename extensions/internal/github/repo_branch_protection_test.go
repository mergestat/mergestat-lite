package github_test

import (
	"testing"

	"github.com/askgitdev/askgit/extensions/internal/tools"
)

func TestRepoProtections(t *testing.T) {
	cleanup := newRecorder(t)
	defer cleanup()

	db := Connect(t, Memory)

	rows, err := db.Query("SELECT * FROM github_protections('askgitdev','askgit') LIMIT 10")
	if err != nil {
		t.Fatalf("failed to execute query: %v", err.Error())
	}
	defer rows.Close()

	colCount, _ /*content*/, err := tools.RowContent(rows)
	if err != nil {
		t.Fatalf("failed to retrieve row contents: %v", err.Error())
	}

	if colCount != 18 {
		t.Fatalf("expected 18 columns, got: %d", colCount)
	}
	// TODO Find a repo that returns 10+ rows so we can limit it
	// if len(content) != 10 {
	// 	t.Fatalf("expected 10 rows, got: %d", len(content))
	// }
}
