package ghqlite

import (
	"testing"
)

func TestOrgReposTable(t *testing.T) {
	rows, err := DB.Query("SELECT * FROM org_repos LIMIT 5")
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

func TestUserReposTable(t *testing.T) {
	rows, err := DB.Query("SELECT * FROM user_repos LIMIT 5")
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
