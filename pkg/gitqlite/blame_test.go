package gitqlite

import (
	"io"
	"strconv"
	"testing"
)

func TestBlameDistinctFiles(t *testing.T) {

	rows, err := fixtureDB.Query("SELECT count(distinct path) from blame")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	_, contents, err := GetRowContents(rows)
	if err != nil {
		t.Fatal(err)
	}

	gotFileCount, err := strconv.Atoi(contents[0][0])
	if err != nil {
		t.Fatal(err)
	}

	head, err := fixtureRepo.Head()
	if err != nil {
		t.Fatal(err)
	}
	defer head.Free()

	iter, err := NewCommitFileIter(fixtureRepo, &commitFileIterOptions{head.Target().String()})
	if err != nil {
		t.Fatal(err)
	}

	var expectedFileCount int
	for {
		_, err := iter.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			t.Fatal(err)
		}
		expectedFileCount++
	}

	if gotFileCount != expectedFileCount {
		t.Fatalf("expected %d distinct file paths in blame, got %d", expectedFileCount, gotFileCount)
	}
}
