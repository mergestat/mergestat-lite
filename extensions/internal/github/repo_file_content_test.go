package github_test

import (
	"testing"
)

func TestRepoFileContent(t *testing.T) {
	cleanup := newRecorder(t)
	defer cleanup()

	db := Connect(t, Memory)

	row := db.QueryRow("SELECT github_repo_file_content('askgitdev/askgit', 'cfdcd109d8582ac7cb5b69c48bb426f31f39e948:README.md')")
	if err := row.Err(); err != nil {
		t.Fatalf("failed to execute query: %v", err.Error())
	}

	var content string
	err := row.Scan(&content)
	if err != nil {
		t.Fatalf("could not scan row: %v", err.Error())
	}

	if len(content) < 100 {
		t.Fatal("expected some file content")
	}
}

func TestRepoFileContentMultiArg(t *testing.T) {
	cleanup := newRecorder(t)
	defer cleanup()

	db := Connect(t, Memory)

	row := db.QueryRow("SELECT github_repo_file_content('askgitdev', 'askgit', 'cfdcd109d8582ac7cb5b69c48bb426f31f39e948:README.md')")
	if err := row.Err(); err != nil {
		t.Fatalf("failed to execute query: %v", err.Error())
	}

	var content string
	err := row.Scan(&content)
	if err != nil {
		t.Fatalf("could not scan row: %v", err.Error())
	}

	if len(content) < 100 {
		t.Fatal("expected some file content")
	}
}
