package gitqlite

import (
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

	// columnQuery, err := instance.DB.Query("SELECT * FROM files LIMIT 1")
	// if err != nil {
	// 	t.Fatal(err)
	// }
	// columns, err := columnQuery.Columns()
	// if err != nil {
	// 	t.Fatal(err)
	// }
	// if len(columns) != 6 {
	// 	t.Fatalf("expected %d columns got : %d", 6, len(columns))
	// }
	// commitChecker, err = fixtureRepo.Log(&git.LogOptions{
	// 	From:  headRef.Hash(),
	// 	Order: git.LogOrderCommitterTime,
	// })
	// if err != nil {
	// 	t.Fatal(err)
	// }
	// commit, err := commitChecker.Next()
	// if err != nil {
	// 	t.Fatal(err)
	// }
	// tree, err := commit.Tree()
	// if err != nil {
	// 	t.Fatal(err)
	// }
	// files := tree.Files()
	// file, err := files.Next()
	// if err != nil {
	// 	t.Fatal(err)
	// }
	// index, contents, err := GetContents(columnQuery)
	// if err != nil {
	// 	t.Fatalf("%s in GetContents at row %d", err, index)
	// }

	// if file.Name != contents[0][3] {
	// 	t.Fatalf("Expected fileName to be %s got %s", file.Name, contents[0][2])
	// }
	// if file.Hash.String() != contents[0][2] {
	// 	t.Fatalf("Expected Hash %s got Hash %s", file.Hash.String(), contents[0][0])
	// }

}
