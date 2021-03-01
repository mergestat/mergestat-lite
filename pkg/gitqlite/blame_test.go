package gitqlite

import (
	"context"
	"io"
	"strconv"
	"strings"
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
	rows, err := fixtureDB.Query("SELECT line_no,commit_id from blame limit 100")
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
		results, err := blame.Exec(context.Background(), cont.File, &blame.Options{Directory: fixtureRepoDir})
		if err != nil {
			t.Fatal(err)
		}
		lineNo, err := strconv.Atoi(line[0])
		if err != nil {
			t.Fatal(err)
		}
		if strings.Compare(line[1], results[lineNo].SHA) != 0 {
			t.Fatalf("expected %s SHA in blame at line %d, got %s", results[i+1].SHA, i+1, line[0])
		}
	}
}

// TODO implement this with a join on commits
func TestBlameAuthorEmail(t *testing.T) {

}
