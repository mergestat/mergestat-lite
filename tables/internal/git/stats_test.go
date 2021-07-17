package git_test

import (
	"testing"
)

func TestSelectLast5CommitStats(t *testing.T) {
	db := Connect(t, Memory)
	repo := "https://github.com/askgitdev/askgit"

	rows, err := db.Query("SELECT commits.hash, file_path, additions, deletions FROM commits($1), stats($1, commits.hash) LIMIT 5", repo)
	if err != nil {
		t.Fatalf("failed to execute query: %v", err.Error())
	}
	defer rows.Close()

	for rows.Next() {
		var hash, file_path string
		var additions, deletions int
		err = rows.Scan(&hash, &file_path, &additions, &deletions)
		if err != nil {
			t.Fatalf("failed to scan resultset: %v", err)
		}
		t.Logf("commit: hash=%q file_path=%s additions=%d deletions=%d", hash, file_path, additions, deletions)
	}

	if err = rows.Err(); err != nil {
		t.Fatalf("failed to fetch results: %v", err.Error())
	}
}

func TestSummarizeLast5CommitStats(t *testing.T) {
	db := Connect(t, Memory)
	repo := "https://github.com/askgitdev/askgit"

	rows, err := db.Query("SELECT hash, count(DISTINCT file_path), sum(additions), sum(deletions) FROM commits($1), stats($1, commits.hash) GROUP BY hash ORDER BY commits.committer_when DESC LIMIT 5", repo)
	if err != nil {
		t.Fatalf("failed to execute query: %v", err.Error())
	}
	defer rows.Close()

	for rows.Next() {
		var hash string
		var filesChanged, additions, deletions int
		err = rows.Scan(&hash, &filesChanged, &additions, &deletions)
		if err != nil {
			t.Fatalf("failed to scan resultset: %v", err)
		}
		t.Logf("commit: hash=%q, files_changed=%d additions=%d deletions=%d", hash, filesChanged, additions, deletions)
	}

	if err = rows.Err(); err != nil {
		t.Fatalf("failed to fetch results: %v", err.Error())
	}
}
