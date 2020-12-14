package ghqlite

import (
	"testing"
)

func TestPullRequestsTable(t *testing.T) {
	rows, err := DB.Query("SELECT * FROM github_pull_requests('augmentable-dev', 'askgit') WHERE state = 'closed' LIMIT 5")
	if err != nil {
		t.Fatal(err)
	}

	_, contents, err := GetRowContents(rows)
	if err != nil {
		t.Fatal(err)
	}

	if len(contents) != 5 {
		t.Fatalf("expected: 5 rows, got: %d rows", len(contents))
	}

}
