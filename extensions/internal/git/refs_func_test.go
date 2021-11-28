package git_test

import (
	"database/sql"
	"testing"
)

func TestCommitFromTagFn(t *testing.T) {
	db := Connect(t, Memory)
	repo := "https://github.com/mergestat/mergestat"

	rows, err := db.Query("SELECT name, full_name, COMMIT_FROM_TAG(tag) FROM refs(?) WHERE type = 'tag'", repo)
	if err != nil {
		t.Fatalf("failed to execute query: %v", err.Error())
	}
	defer rows.Close()

	for rows.Next() {
		var name, fullName, hash sql.NullString
		if err = rows.Scan(&name, &fullName, &hash); err != nil {
			t.Fatalf("failed to scan resultset: %v", err)
		}
		t.Logf("ref: name=%q fullName=%q hash=%q", name.String, fullName.String, hash.String)
	}

	if err = rows.Err(); err != nil {
		t.Fatalf("failed to fetch results: %v", err.Error())
	}
}
