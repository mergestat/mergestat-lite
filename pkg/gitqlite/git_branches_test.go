package gitqlite

import (
	"testing"

	git "github.com/libgit2/git2go/v30"
)

func TestBranches(t *testing.T) {
	branchIter, err := fixtureRepo.NewBranchIterator(git.BranchAll)
	if err != nil {
		t.Fatal(err)
	}
	defer branchIter.Free()

	branchRows, err := fixtureDB.Query("SELECT * FROM branches")
	if err != nil {
		t.Fatal(err)
	}

	rowNum, contents, err := GetRowContents(branchRows)
	if err != nil {
		t.Fatalf("err %d at row Number %d", err, rowNum)
	}

	for i, c := range contents {
		branch, _, err := branchIter.Next()
		if err != nil {
			if branch != nil {
				t.Fatal(err)
			}
		}

		name, err := branch.Name()
		if err != nil {
			t.Fatal(err)
		}

		if name != c[0] {
			t.Fatalf("exepcted %s at row %d got %s", name, i, c[0])
		}

		switch branch.Type() {
		case git.ReferenceSymbolic:
			if branch.SymbolicTarget() != c[2] {
				t.Fatalf("expected %s at row %d got %s", branch.SymbolicTarget(), i, c[2])
			}
		case git.ReferenceOid:
			if branch.Target().String() != c[2] {
				t.Fatalf("expected %s at row %d got %s", branch.Target().String(), i, c[2])
			}
		}

	}
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
