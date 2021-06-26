package ghqlite

import (
	"testing"
)

func TestOrgReposTable(t *testing.T) {
	rows, err := DB.Query("SELECT * FROM github_org_repos('askgitdev') LIMIT 5")
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
	rows, err := DB.Query("SELECT * FROM github_user_repos('patrickdevivo') LIMIT 5")
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
