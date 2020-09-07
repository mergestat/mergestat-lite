package gitqlite

import (
	"testing"
)

// TODO: fix this up
// func TestBranches(t *testing.T) {
// 	instance, err := New(fixtureRepoDir, &Options{})
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	localBranchIterator, err := fixtureRepo.Branches()
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	branchRows, err := instance.DB.Query("SELECT * FROM branches")
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	rowNum, contents, err := GetContents(branchRows)
// 	if err != nil {
// 		t.Fatalf("err %d at row Number %d", err, rowNum)
// 	}
// 	for i, c := range contents {
// 		branch, err := localBranchIterator.Next()
// 		if err != nil {
// 			if err == io.EOF {
// 				branch, err = remoteBranchIterator.Next()
// 				if err != nil {
// 					if err == io.EOF {
// 						break
// 					} else {
// 						t.Fatal(err)
// 					}
// 				}
// 			} else {
// 				t.Fatal(err)
// 			}
// 		}
// 		if branch.Name().Short() != c[0] || branch.Hash().String() != c[4] {
// 			t.Fatalf("expected %s at row %d got %s", branch.Name().String(), i, c[0])
// 		}
// 		if branch.Hash().String() != c[4] {
// 			t.Fatalf("expected %s at row %d got %s", branch.Hash().String(), i, c[4])
// 		}

// 	}
// }

func BenchmarkBranchCount(b *testing.B) {
	for i := 0; i < b.N; i++ {
		instance, err := New(fixtureRepoDir, &Options{SkipGitCLI: true})
		if err != nil {
			b.Fatal(err)
		}
		rows, err := instance.DB.Query("SELECT * FROM branches")
		if err != nil {
			b.Fatal(err)
		}
		rowNum, _, err := GetContents(rows)
		if err != nil {
			b.Fatalf("err %d at row Number %d", err, rowNum)
		}
	}
}
