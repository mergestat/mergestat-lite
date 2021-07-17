package git_test

import (
	"testing"
)

func TestSelect10FilesHEAD(t *testing.T) {
	db := Connect(t, Memory)
	repo := "https://github.com/askgitdev/askgit"

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
