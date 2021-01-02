package gitqlite

import (
	"io"
	"testing"

	git "github.com/libgit2/git2go/v31"
)

func TestBranches(t *testing.T) {
	testCases := []test{
		{"checkCommits", "SELECT name, target FROM branches", getAllBranchInfo(t)},
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

func getAllBranchInfo(t *testing.T) []string {
	var branchInfo []string
	branchIter, err := fixtureRepo.NewBranchIterator(git.BranchAll)
	if err != nil {
		t.Fatal(err)
	}
	defer branchIter.Free()
	for {
		branch, _, err := branchIter.Next()
		if err != nil {
			if branch != nil {
				t.Fatal(err)
			}
			if err == io.EOF {
				break
			}
		}
		if branch != nil {
			name, err := branch.Name()
			if err != nil {
				break
			}
			branchInfo = append(branchInfo, name)

			switch branch.Type() {
			case git.ReferenceSymbolic:
				branchInfo = append(branchInfo, branch.SymbolicTarget())
			case git.ReferenceOid:
				branchInfo = append(branchInfo, branch.Target().String())
			}
		} else {
			break
		}
	}
	return branchInfo
}

func BenchmarkBranchCount(b *testing.B) {
	for i := 0; i < b.N; i++ {
		rows, err := fixtureDB.Query("SELECT * FROM branches")
		if err != nil {
			b.Fatal(err)
		}
		rowNum, _, err := GetRowContents(rows)
		if err != nil {
			b.Fatalf("err %d at row Number %d", err, rowNum)
		}
	}
}
