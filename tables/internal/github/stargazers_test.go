package github_test

import (
	"testing"
)

func TestStargazers(t *testing.T) {
	cleanup := newRecorder(t)
	defer cleanup()

	db := Connect(t, Memory)

	rows, err := db.Query("SELECT login FROM github_stargazers('askgitdev/askgit') LIMIT 500")
	if err != nil {
		t.Fatalf("failed to execute query: %v", err.Error())
	}
	defer rows.Close()

	for rows.Next() {
		var login string
		if err = rows.Scan(&login); err != nil {
			t.Fatalf("failed to scan resultset: %v", err)
		}
		t.Logf("ref: login=%s", login)

		if login == "" {
			t.Fatalf("expected a login")
		}
	}

	if err = rows.Err(); err != nil {
		t.Fatalf("failed to fetch results: %v", err.Error())
	}
}
