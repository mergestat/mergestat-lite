package gitqlite

import (
	"fmt"
	"testing"
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
	iter, err := NewCommitFileIter(fixtureRepo, &commitFileIterOptions{head.Target().String()})
	if err != nil {
		t.Fatal(err)
	}
	return len(iter.treeEntries)
}
