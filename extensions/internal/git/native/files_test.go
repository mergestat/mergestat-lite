package native_test

import (
	"fmt"
	"os"
	"testing"
)

func TestSelect10FilesHEAD(t *testing.T) {
	db := Connect(t, Memory)
	repo := "https://github.com/mergestat/mergestat"

	rows, err := db.Query("SELECT path, executable, contents FROM files(?) LIMIT 10", repo)
	if err != nil {
		t.Fatalf("failed to execute query: %v", err.Error())
	}
	defer rows.Close()

	for rows.Next() {
		var path, contents string
		var executable int
		err = rows.Scan(&path, &executable, &contents)
		if err != nil {
			t.Fatalf("failed to scan resultset: %v", err)
		}
		t.Logf("file: path=%s executable=%d contents_len=%d", path, executable, len(contents))
	}

	if err = rows.Err(); err != nil {
		t.Fatalf("failed to fetch results: %v", err.Error())
	}
}

func TestSelectKnownContents(t *testing.T) {
	db := Connect(t, Memory)
	repo, hash := "https://github.com/mergestat/mergestat", "2359c9a9ba0ba8aa694601ff12538c4e74b82cd5"

	rows, err := db.Query("SELECT path, contents FROM files(?, ?) WHERE path LIKE 'Makefile' OR path LIKE 'go.mod'", repo, hash)
	if err != nil {
		t.Fatalf("failed to execute query: %v", err.Error())
	}
	defer rows.Close()

	for rows.Next() {
		var path, contents string
		err = rows.Scan(&path, &contents)
		if err != nil {
			t.Fatalf("failed to scan resultset: %v", err)
		}
		t.Logf("file: path=%s contents_len=%d", path, len(contents))

		expectedFileContents, err := os.ReadFile(fmt.Sprintf("./testdata/%s/%s.testdata", hash, path))
		if err != nil {
			t.Fatalf("failed to load fixture file: %v", err)
		}

		if contents != string(expectedFileContents) {
			t.Fatalf("contents of %s files do not match", path)
		}
	}

	if err = rows.Err(); err != nil {
		t.Fatalf("failed to fetch results: %v", err.Error())
	}
}
