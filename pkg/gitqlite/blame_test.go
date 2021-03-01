package gitqlite

import (
	"context"
	"io"
	"strconv"
	"testing"

	"github.com/augmentable-dev/tickgit/pkg/blame"
)

func TestBlameDistinctFiles(t *testing.T) {

	rows, err := fixtureDB.Query("SELECT count(distinct file_path) from blame")
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
func TestBlameCommitID(t *testing.T) {
	iterator, err := NewBlameIterator(fixtureRepo)
	if err != nil {
		t.Fatal(err)
	}
	rows, err := fixtureDB.Query("SELECT commit_id from blame limit 100")
	if err != nil {
		t.Fatal(err)
	}
	_, lines, err := GetRowContents(rows)
	if err != nil {
		t.Fatal(err)
	}
	for i, line := range lines {
		cont, err := iterator.Next()
		if err != nil {
			t.Fatal(err)
		}
		results, err := blame.Exec(context.Background(), cont.File, &blame.Options{})
		if err != nil {
			t.Fatal(err)
		}
		if !(line[0] == results[i].SHA) {
			t.Fatalf("expected %s content in blame, got %s", cont.Content, line[0])
		}
	}
}
