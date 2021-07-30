package github_test

import (
	"testing"
)

func TestStarredRepos(t *testing.T) {
	cleanup := newRecorder(t)
	defer cleanup()

	db := Connect(t, Memory)

	rows, err := db.Query("SELECT name FROM github_starred_repos('patrickdevivo') LIMIT 50")
	if err != nil {
		t.Fatalf("failed to execute query: %v", err.Error())
	}
	defer rows.Close()

	for rows.Next() {
		var name string
		if err = rows.Scan(&name); err != nil {
			t.Fatalf("failed to scan resultset: %v", err)
		}
		t.Logf("ref: name=%s", name)

		if name == "" {
			t.Fatalf("expected a name")
		}
	}

	if err = rows.Err(); err != nil {
		t.Fatalf("failed to fetch results: %v", err.Error())
	}
}
