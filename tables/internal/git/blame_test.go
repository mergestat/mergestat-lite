package git_test

import (
	"testing"
)

func TestSelectBlameREADMELines(t *testing.T) {
	db := Connect(t, Memory)
	repo := "https://github.com/askgitdev/askgit"

	rows, err := db.Query("SELECT line_no, commit_hash FROM blame(?, '', 'README.md') LIMIT 10", repo)
	if err != nil {
		t.Fatalf("failed to execute query: %v", err.Error())
	}
	defer rows.Close()

	for rows.Next() {
		var commitHash string
		var lineNo int
		err = rows.Scan(&lineNo, &commitHash)
		if err != nil {
			t.Fatalf("failed to scan resultset: %v", err)
		}
		t.Logf("blame: line_no=%d commit_hash=%s", lineNo, commitHash)
	}

	if err = rows.Err(); err != nil {
		t.Fatalf("failed to fetch results: %v", err.Error())
	}
}
