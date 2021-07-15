package git_test

import (
	"database/sql"
	"os"
	"testing"
	"time"
)

func TestSelectAllCommits(t *testing.T) {
	db := Connect(t, Memory)
	repo, ref := "https://github.com/augmentable-dev/askgit", "HEAD"

	rows, err := db.Query("SELECT * FROM commits(?, ?) LIMIT 5", repo, ref)
	if err != nil {
		t.Fatalf("failed to execute query: %v", err.Error())
	}
	defer rows.Close()

	for rows.Next() {
		var hash, message string
		var authorName, authorEmail, authorWhen string
		var committerName, committerEmail, committerWhen string
		var parents int
		err = rows.Scan(&hash, &message, &authorName, &authorEmail, &authorWhen, &committerName, &committerEmail, &committerWhen, &parents)
		if err != nil {
			t.Fatalf("failed to scan resultset: %v", err)
		}
		t.Logf("commit: hash=%q author=\"%s <%s>\" committer=\"%s <%s>\" parents=%d", hash, authorName, authorEmail, committerName, committerEmail, parents)
	}

	if err = rows.Err(); err != nil {
		t.Fatalf("failed to fetch results: %v", err.Error())
	}
}

func TestSelectCommitByHash(t *testing.T) {
	db := Connect(t, Memory)
	repo, ref := "https://github.com/augmentable-dev/askgit", "HEAD"
	hash := "5ce802c851d3bedb5bb4a0f749093cae9a34818b"

	var message, email string
	var when time.Time
	err := db.QueryRow("SELECT message, committer_email, committer_when FROM commits(?, ?) WHERE hash = ?",
		repo, ref, hash).Scan(&message, &email, &when)
	if err != nil {
		t.Fatalf("failed to execute query: %v", err.Error())
	}

	t.Logf("commit: %q by %q on %q", hash, email, when.Format(time.RFC3339))
}

func TestDateFilterOnCommit(t *testing.T) {
	db := Connect(t, Memory)
	repo := "https://github.com/augmentable-dev/askgit"

	rows, err := db.Query("SELECT hash, committer_email, committer_when FROM commits(?)"+
		"	WHERE committer_when > DATE(?) AND committer_when < DATE(?) ORDER BY committer_when DESC",
		repo, time.Date(2021, 01, 01, 00, 00, 00, 00, time.UTC), time.Date(2021, 04, 30, 00, 00, 00, 00, time.UTC))
	if err != nil {
		t.Fatalf("failed to execute query: %v", err.Error())
	}
	defer rows.Close()

	for rows.Next() {
		var hash, email string
		var when time.Time
		if err = rows.Scan(&hash, &email, &when); err != nil {
			t.Fatalf("failed to scan resultset: %v", err)
		}
		t.Logf("commit: %q by %q on %q", hash, email, when.Format(time.RFC3339))
	}

	if err = rows.Err(); err != nil {
		t.Fatalf("failed to fetch results: %v", err.Error())
	}
}

func TestDefaultCases(t *testing.T) {
	db := Connect(t, Memory)
	repo := "https://github.com/augmentable-dev/askgit"

	var q = func(row *sql.Row) (hash, email string, when time.Time, err error) {
		err = row.Scan(&hash, &email, &when)
		return
	}

	t.Run("should use current working directory as default repository", func(t *testing.T) {
		if ci, ok := os.LookupEnv("CI"); ok && ci == "true" {
			t.Skip("skipping test as current working directory cannot be set in ci environment")
		}

		_, _, _, err := q(db.QueryRow("SELECT hash, committer_email, committer_when FROM commits LIMIT 1"))
		if err != nil {
			t.Error(err)
			t.Fail()
		}
	})

	t.Run("should use head as the default reference", func(t *testing.T) {
		_, _, _, err := q(db.QueryRow("SELECT hash, committer_email, committer_when FROM commits(?) LIMIT 1", repo))
		if err != nil {
			t.Error(err)
			t.Fail()
		}
	})
}
