package gitqlite

import (
	"fmt"
	"testing"

	git "github.com/libgit2/git2go/v31"
)

func TestBlameCounts(t *testing.T) {
	testCases := []test{
		{"checkFileNums", "SELECT count(distinct path) from blame", []string{fmt.Sprint(getFilesCount(t))}},
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
}

func getFilesCount(t *testing.T) int {
	head, err := fixtureRepo.Head()
	if err != nil {
		t.Fatal(err)
	}
	defer head.Free()

	commit, err := fixtureRepo.LookupCommit(head.Target())
	if err != nil {
		t.Fatal(err)
	}
	tree, err := commit.Tree()
	if err != nil {
		t.Fatal(err)
	}

	var entries []string
	var ids []*git.Oid
	err = tree.Walk(func(s string, entry *git.TreeEntry) int {
		if entry.Type.String() == "Blob" {
			entries = append(entries, s+entry.Name)
			ids = append(ids, entry.Id)
		}
		return 0
	})
	if err != nil {
		t.Fatal(err)
	}
	return len(entries)
}
