package gitqlite

import (
	"fmt"
	"path"
	"strconv"
	"testing"

	git "github.com/libgit2/git2go/v30"
)

func TestFileCounts(t *testing.T) {
	instance, err := New(fixtureRepoDir, &Options{})
	if err != nil {
		t.Fatal(err)
	}

	commitChecker, err := fixtureRepo.Walk()
	if err != nil {
		t.Fatal(err)
	}

	err = commitChecker.PushHead()
	if err != nil {
		t.Fatal(err)
	}

	defer commitChecker.Free()

	numFiles := 0
	err = commitChecker.Iterate(func(commit *git.Commit) bool {
		numFiles++
		return true
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
}

func TestFileColumns(t *testing.T) {
	instance, err := New(fixtureRepoDir, &Options{})
	if err != nil {
		t.Fatal(err)
	}

	columnQuery, err := instance.DB.Query("SELECT * FROM files LIMIT 1")
	if err != nil {
		t.Fatal(err)
	}
	defer columnQuery.Close()

	columns, err := columnQuery.Columns()
	if err != nil {
		t.Fatal(err)
	}

	if len(columns) != 6 {
		t.Fatalf("expected %d columns got : %d", 6, len(columns))
	}

	_, contents, err := GetContents(columnQuery)
	if err != nil {
		t.Fatal(err)
	}

	commitID, err := git.NewOid(contents[0][0])
	if err != nil {
		t.Fatal(err)
	}

	commit, err := fixtureRepo.LookupCommit(commitID)
	if err != nil {
		t.Fatal(err)
	}
	defer commit.Free()

	tree, err := commit.Tree()
	if err != nil {
		t.Fatal(err)
	}
	defer tree.Free()

	if contents[0][1] != tree.Id().String() {
		t.Fatalf("expected tree_id %s, got: %s", tree.Id().String(), contents[0][1])
	}

	entry, err := tree.EntryByPath(contents[0][3])
	if err != nil {
		t.Fatal(err)
	}

	if entry.Name != path.Base(contents[0][3]) {
		t.Fatalf("expected file_name to be %s got %s", entry.Name, path.Base(contents[0][3]))
	}
}

func TestFileByID(t *testing.T) {
	instance, err := New(fixtureRepoDir, &Options{})
	if err != nil {
		t.Fatal(err)
	}

	o, err := fixtureRepo.RevparseSingle("HEAD~3")
	if err != nil {
		t.Fatal(err)
	}
	defer o.Free()

	commit, err := o.AsCommit()
	if err != nil {
		t.Fatal(err)
	}
	defer commit.Free()

	rows, err := instance.DB.Query(fmt.Sprintf("SELECT count(*) FROM files WHERE commit_id = '%s'", commit.Id().String()))
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	tree, err := commit.Tree()
	if err != nil {
		t.Fatal(err)
	}
	defer tree.Free()

	count := 0
	err = tree.Walk(func(path string, entry *git.TreeEntry) int {
		if entry.Type == git.ObjectBlob {
			count++
		}
		return 0
	})
	if err != nil {
		t.Fatal(err)
	}

	_, contents, err := GetContents(rows)
	if err != nil {
		t.Fatal(err)
	}

	gotCount, err := strconv.Atoi(contents[0][0])
	if err != nil {
		t.Fatal(err)
	}

	if gotCount != count {
		t.Fatalf("expected %d, got %d", count, gotCount)
	}
}
