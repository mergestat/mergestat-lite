package gitqlite

import (
	"testing"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
)

func TestFileCounts(t *testing.T) {
	instance, err := New(fixtureRepoDir, &Options{})
	if err != nil {
		t.Fatal(err)
	}

	headRef, err := fixtureRepo.Head()
	if err != nil {
		t.Fatal(err)
	}
	commitChecker, err := fixtureRepo.Log(&git.LogOptions{
		From:  headRef.Hash(),
		Order: git.LogOrderCommitterTime,
	})
	if err != nil {
		t.Fatal(err)
	}
	numFiles := 0
	err = commitChecker.ForEach(func(c *object.Commit) error {
		numFiles++
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	fileRows, err := instance.DB.Query("SELECT DISTINCT commit_id FROM files")
	if err != nil {
		t.Fatal(err)
	}
	defer fileRows.Close()
	numFileRows := GetRowsCount(fileRows)
	if numFileRows != numFiles {
		t.Fatalf("expected %d rows got : %d", numFiles, numFileRows)
	}
	columnQuery, err := instance.DB.Query("SELECT * FROM files LIMIT 1")
	if err != nil {
		t.Fatal(err)
	}
	columns, err := columnQuery.Columns()
	if err != nil {
		t.Fatal(err)
	}
	if len(columns) != 6 {
		t.Fatalf("expected %d columns got : %d", 6, len(columns))
	}
}
