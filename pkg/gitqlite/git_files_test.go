package gitqlite

import (
	"fmt"
	"path"
	"strconv"
	"testing"

	git "github.com/libgit2/git2go/v31"
)

func TestFileCounts(t *testing.T) {
	testCases := []test{
		{"checkCommits", "SELECT count(distinct commit_id) from files", getCommitCount(t)},
	}
	for _, tc := range testCases {
		expected := tc.want
		results := runQuery(t, tc.query)
		if len(expected) != len(results) {
			t.Fatalf("expected %d entries got %d, test: %s, %s, %s", len(expected), len(results), tc.name, expected, results)
		}
		for x := 0; x < len(expected); x++ {
			if results[x] != expected[x] {
				t.Fatalf("expected %s, got %s, test %s", expected[x], results[x], tc.name)
			}
		}
	}
	// instance, err := New(fixtureRepoDir, &Options{})
	// if err != nil {
	// 	t.Fatal(err)
	// }

	// commitChecker, err := fixtureRepo.Walk()
	// if err != nil {
	// 	t.Fatal(err)
	// }

	// err = commitChecker.PushHead()
	// if err != nil {
	// 	t.Fatal(err)
	// }

	// defer commitChecker.Free()

	// numFiles := 0
	// err = commitChecker.Iterate(func(commit *git.Commit) bool {
	// 	numFiles++
	// 	return true
	// })
	// if err != nil {
	// 	t.Fatal(err)
	// }

	// fileRows, err := instance.DB.Query("SELECT DISTINCT commit_id FROM files")
	// if err != nil {
	// 	t.Fatal(err)
	// }
	// defer fileRows.Close()

	// numFileRows := GetRowsCount(fileRows)
	// if numFileRows != numFiles {
	// 	t.Fatalf("expected %d rows got : %d", numFiles, numFileRows)
	// }
}

func TestFileColumns(t *testing.T) {

	columnQuery, err := fixtureDB.Query("SELECT * FROM files LIMIT 1")
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

	_, contents, err := GetRowContents(columnQuery)
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

	rows, err := fixtureDB.Query(fmt.Sprintf("SELECT count(*) FROM files WHERE commit_id = '%s'", commit.Id().String()))
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

	_, contents, err := GetRowContents(rows)
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
