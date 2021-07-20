package git_test

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
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

func TestSelectKnownBlame(t *testing.T) {
	db := Connect(t, Memory)
	repo, hash := "https://github.com/askgitdev/askgit", "2359c9a9ba0ba8aa694601ff12538c4e74b82cd5"

	rows, err := db.Query("SELECT line_no, commit_hash FROM blame(?, ?, 'README.md')", repo, hash)
	if err != nil {
		t.Fatalf("failed to execute query: %v", err.Error())
	}
	defer rows.Close()

	fixtureFileName := "README.md.blame.csv.testdata"
	f, err := os.Open(fmt.Sprintf("./testdata/%s/%s", hash, fixtureFileName))
	if err != nil {
		t.Fatalf("failed to open fixture file %s: %v", fixtureFileName, err.Error())
	}

	r := csv.NewReader(f)
	fixtureRecords, err := r.ReadAll()
	if err != nil {
		t.Fatalf("failed to parse CSV in fixture file %s: %v", fixtureFileName, err.Error())
	}

	i := 0
	for rows.Next() {
		var commitHash string
		var lineNo int
		err = rows.Scan(&lineNo, &commitHash)
		if err != nil {
			t.Fatalf("failed to scan resultset: %v", err.Error())
		}
		t.Logf("blame: line_no=%d commit_hash=%s", lineNo, commitHash)

		i++
		fixtureRow := fixtureRecords[i]
		fixtureLine, fixtureHash := fixtureRow[0], fixtureRow[1]
		if strconv.Itoa(lineNo) != fixtureLine {
			t.Fatalf("expected line: %s, got: %d", fixtureLine, lineNo)
		}

		if commitHash != fixtureHash {
			t.Fatalf("expected hash: %s, got: %d", fixtureLine, lineNo)
		}
	}

	if err = rows.Err(); err != nil {
		t.Fatalf("failed to fetch results: %v", err.Error())
	}
}
