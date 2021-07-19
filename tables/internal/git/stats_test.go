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
		var hash, filePath string
		var additions, deletions int
		err = rows.Scan(&hash, &filePath, &additions, &deletions)
		if err != nil {
			t.Fatalf("failed to scan resultset: %v", err)
		}
		t.Logf("stat: hash=%q file_path=%s additions=%d deletions=%d", hash, filePath, additions, deletions)
	}

	if err = rows.Err(); err != nil {
		t.Fatalf("failed to fetch results: %v", err.Error())
	}
}

func TestInitialCommitStats(t *testing.T) {
	db := Connect(t, Memory)
	repo, initialCommit := "https://github.com/askgitdev/askgit", "a4562d2d5a35536771745b0aa19d705eb47234e7"

	var filesChanged, additions, deletions int
	err := db.QueryRow("SELECT count(DISTINCT file_path), sum(additions), sum(deletions) FROM stats(?, ?)", repo, initialCommit).
		Scan(&filesChanged, &additions, &deletions)
	if err != nil {
		t.Fatalf("failed to execute query: %v", err.Error())
	}

	t.Logf("stat: files_changed=%d additions=%d deletions=%d", filesChanged, additions, deletions)

	expectedFilesChanged, expectedAdditions, expectedDeletions := 13, 1612, 0

	if filesChanged != expectedFilesChanged {
		t.Fatalf("expected %d files changed, got %d", expectedFilesChanged, filesChanged)
	}

	if additions != expectedAdditions {
		t.Fatalf("expected %d additions, got %d", expectedAdditions, additions)
	}

	if deletions != expectedDeletions {
		t.Fatalf("expected %d deletions, got %d", expectedDeletions, deletions)
	}
}

func TestFromAndToStats(t *testing.T) {
	db := Connect(t, Memory)
	repo, fromHash, toHash := "https://github.com/askgitdev/askgit", "2359c9a9ba0ba8aa694601ff12538c4e74b82cd5", "d65736fd08fab5a64027f0c050ee148d88549406"

	var filesChanged, additions, deletions int
	err := db.QueryRow("SELECT count(DISTINCT file_path), sum(additions), sum(deletions) FROM stats(?, ?, ?)", repo, fromHash, toHash).
		Scan(&filesChanged, &additions, &deletions)
	if err != nil {
		t.Fatalf("failed to execute query: %v", err.Error())
	}

	t.Logf("stat: files_changed=%d additions=%d deletions=%d", filesChanged, additions, deletions)

	expectedFilesChanged, expectedAdditions, expectedDeletions := 5, 13, 13

	if filesChanged != expectedFilesChanged {
		t.Fatalf("expected %d files changed, got %d", expectedFilesChanged, filesChanged)
	}

	if additions != expectedAdditions {
		t.Fatalf("expected %d additions, got %d", expectedAdditions, additions)
	}

	if deletions != expectedDeletions {
		t.Fatalf("expected %d deletions, got %d", expectedDeletions, deletions)
	}
}
