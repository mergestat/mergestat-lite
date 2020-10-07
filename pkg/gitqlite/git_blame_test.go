package gitqlite

import (
	"io"
	"testing"
)

func TestBlame(t *testing.T) {
	instance, err := New(fixtureRepoDir, &Options{})
	if err != nil {
		t.Fatal(err)
	}
	opt := &commitFileIterOptions{}
	fileIter, err := NewCommitFileIter(fixtureRepo, opt)
	if err != nil {
		t.Fatal(err)
	}
	if err != nil {
		t.Fatal(err)
	}

	branchRows, err := instance.DB.Query("SELECT DISTINCT name FROM blame")
	if err != nil {
		t.Fatal(err)
	}

	rowNum, contents, err := GetContents(branchRows)
	if err != nil {
		t.Fatalf("err %d at row Number %d", err, rowNum)
	}
	for _, i := range contents {
		for _, x := range i {
			file, err := fileIter.Next()
			if err != nil && err != io.EOF {
				t.Fatal(err)
			}
			if file.path+file.Name != x {
				t.Fatalf("error %s missing from output in location near %s", file.path+file.Name, x)
			}
		}
	}

}

func BenchmarkBlameCount(b *testing.B) {
	for i := 0; i < b.N; i++ {
		instance, err := New(fixtureRepoDir, &Options{})
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
