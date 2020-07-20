package gitqlite

import (
	"io"
	"testing"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
)

func TestCommitCounts(t *testing.T) {
	instance, err := New(fixtureRepoDir, &Options{})
	if err != nil {
		t.Fatal(err)
	}

	headRef, err := fixtureRepo.Head()
	if err != nil {
		t.Fatal(err)
	}
	commitChecker, err := fixtureRepo.Log(&git.LogOptions{
		From:  headRef.Hash(),
		Order: git.LogOrderCommitterTime,
	})
	if err != nil {
		t.Fatal(err)
	}

	commitCount := 0
	err = commitChecker.ForEach(func(c *object.Commit) error {
		commitCount++
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	//checks commits
	rows, err := instance.DB.Query("SELECT * FROM commits")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	columns, err := rows.Columns()
	if err != nil {
		t.Fatal(err)
	}
	expected := 14
	if len(columns) != expected {
		t.Fatalf("expected %d columns, got: %d", expected, len(columns))
	}
	numRows := GetRowsCount(rows)

	expected = commitCount
	if numRows != expected {
		t.Fatalf("expected %d rows got: %d", expected, numRows)
	}

	rows, err = instance.DB.Query("SELECT id, author_name FROM commits")
	if err != nil {
		t.Fatal(err)
	}
	rowNum, contents, err := GetContents(rows)
	if err != nil {
		t.Fatalf("err %d at row Number %d", err, rowNum)
	}
	for i, c := range contents {
		commit, err := commitChecker.Next()
		if err != nil {
			if err == io.EOF {
				break
			} else {
				t.Fatal(err)
			}
		}
		if commit.ID().String() != c[0] {
			t.Fatalf("expected %s at row %d got %s", commit.ID().String(), i, c[0])
		}
		if commit.Author.Name != c[1] {
			t.Fatalf("expected %s at row %d got %s", commit.Author.Name, i, c[1])
		}

	}

}
func BenchmarkCommitCounts(b *testing.B) {
	instance, err := New(fixtureRepoDir, &Options{})
	if err != nil {
		b.Fatal(err)
	}
	rows, err := instance.DB.Query("SELECT * FROM commits")
	if err != nil {
		b.Fatal(err)
	}
	rowNum, _, err := GetContents(rows)
	if err != nil {
		b.Fatalf("err %d at row Number %d", err, rowNum)
	}
}
