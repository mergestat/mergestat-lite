package git_test

import (
	"database/sql"
	"testing"
)

func TestSelectAllRefs(t *testing.T) {
	db := Connect(t, Memory)
	repo := "https://github.com/mergestat/mergestat-lite"

	rows, err := db.Query("SELECT * FROM refs(?)", repo)
	if err != nil {
		t.Fatalf("failed to execute query: %v", err.Error())
	}
	defer rows.Close()

	for rows.Next() {
		var name, _type, remote sql.NullString
		var fullName, hash, target sql.NullString
		if err = rows.Scan(&name, &_type, &remote, &fullName, &hash, &target); err != nil {
			t.Fatalf("failed to scan resultset: %v", err)
		}
		t.Logf("ref: name=%q type=%s fullName=%q hash=%q remote=%s target=%s",
			name.String, _type.String, fullName.String, hash.String, remote.String, target.String)
	}

	if err = rows.Err(); err != nil {
		t.Fatalf("failed to fetch results: %v", err.Error())
	}
}
