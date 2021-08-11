package github_test

import (
	"testing"
)

func TestStargazersCount(t *testing.T) {
	cleanup := newRecorder(t)
	defer cleanup()

	db := Connect(t, Memory)

	row := db.QueryRow("SELECT github_stargazer_count('askgitdev/askgit')")
	if err := row.Err(); err != nil {
		t.Fatalf("failed to execute query: %v", err.Error())
	}

	var count int
	err := row.Scan(&count)
	if err != nil {
		t.Fatalf("could not scan row: %v", err.Error())
	}

	if count < 100 {
		t.Fatalf("expected at least 100 stars")
	}
}
