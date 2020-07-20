package gitqlite

import (
	"io"
	"testing"

	"github.com/go-git/go-git/v5/plumbing"
)

func TestRefCounts(t *testing.T) {
	instance, err := New(fixtureRepoDir, &Options{})
	if err != nil {
		t.Fatal(err)
	}
	refChecker, err := fixtureRepo.References()
	if err != nil {
		t.Fatal(err)
	}
	refCount := 0
	err = refChecker.ForEach(func(r *plumbing.Reference) error {
		refCount++
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	//check refs
	refRows, err := instance.DB.Query("SELECT * FROM refs")
	if err != nil {
		t.Fatal(err)
	}

	columns, err := refRows.Columns()
	if err != nil {
		t.Fatal(err)
	}
	expected := 3
	if len(columns) != expected {
		t.Fatalf("expected %d columns, got: %d", expected, len(columns))
	}
	numRows := GetRowsCount(refRows)
	if numRows != refCount {
		t.Fatalf("expected %d rows got : %d", refCount, numRows)
	}
	refRows, err = instance.DB.Query("SELECT * FROM refs")
	if err != nil {
		t.Fatal(err)
	}
	rowNum, contents, err := GetContents(refRows)
	if err != nil {
		t.Fatalf("err %d at row Number %d", err, rowNum)
	}
	for i, c := range contents {
		ref, err := refChecker.Next()
		if err != nil {
			if err == io.EOF {
				break
			} else {
				t.Fatal(err)
			}
		}
		if ref.Name().String() != c[0] {
			t.Fatalf("expected %s at row %d got %s", ref.Name().String(), i, c[0])
		}
		if ref.Type().String() != c[1] {
			t.Fatalf("expected %s at row %d got %s", ref.Type().String(), i, c[1])
		}
		if ref.Hash().String() != c[2] {
			t.Fatalf("expected %s at row %d got %s", ref.Hash().String(), i, c[2])
		}

	}
}
func BenchmarkRefCounts(b *testing.B) {
	for i := 0; i < b.N; i++ {
		instance, err := New(fixtureRepoDir, &Options{})
		if err != nil {
			b.Fatal(err)
		}
		rows, err := instance.DB.Query("SELECT * FROM refs")
		if err != nil {
			b.Fatal(err)
		}
		rowNum, _, err := GetContents(rows)
		if err != nil {
			b.Fatalf("err %d at row Number %d", err, rowNum)
		}
	}
}
