package github_test

import (
	"encoding/json"
	"testing"
)

func TestRepoInfo(t *testing.T) {
	cleanup := newRecorder(t)
	defer cleanup()

	db := Connect(t, Memory)

	row := db.QueryRow("SELECT github_repo('mergestat/mergestat')")
	if err := row.Err(); err != nil {
		t.Fatalf("failed to execute query: %v", err.Error())
	}

	var repoInfoJSON []byte
	err := row.Scan(&repoInfoJSON)
	if err != nil {
		t.Fatalf("could not scan row: %v", err.Error())
	}

	repoInfo := make(map[string]interface{})
	if err := json.Unmarshal(repoInfoJSON, &repoInfo); err != nil {
		t.Fatal(err)
	}

	if n, ok := repoInfo["name"]; !ok || n != "mergestat" {
		t.Fatalf("unexpected result for repo name: %s", n)
	}
}
