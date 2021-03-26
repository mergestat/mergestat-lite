// +build ignore

package git_test

import (
	"testing"

	git "github.com/libgit2/git2go/v31"
)

func TestCommitCounts(t *testing.T) {
	revWalk, err := fixtureRepo.Walk()
	if err != nil {
		t.Fatal(err)
	}
	defer revWalk.Free()

	err = revWalk.PushHead()
	if err != nil {
		t.Fatal(err)
	}

	commitCount := 0
	err = revWalk.Iterate(func(c *git.Commit) bool {
		commitCount++
		return true
	})
	if err != nil {
		t.Fatal(err)
	}

	//checks commits
	rows, err := fixtureDB.Query("SELECT * FROM commits_cli")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	columns, err := rows.Columns()
	if err != nil {
		t.Fatal(err)
	}
	expected := 11
	if len(columns) != expected {
		t.Fatalf("expected %d columns, got: %d", expected, len(columns))
	}
	numRows := GetRowsCount(rows)

	expected = commitCount
	if numRows != expected {
		t.Fatalf("expected %d rows got: %d", expected, numRows)
	}

	rows, err = fixtureDB.Query("SELECT id, author_name FROM commits_cli")
	if err != nil {
		t.Fatal(err)
	}
	rowNum, contents, err := GetRowContents(rows)
	if err != nil {
		t.Fatalf("err %d at row Number %d", err, rowNum)
	}

	i := 0
	err = revWalk.Iterate(func(commit *git.Commit) bool {
		c := contents[i]
		if commit.Id().String() != c[0] {
			t.Fatalf("expected %s at row %d got %s", commit.Id().String(), i, c[0])
		}

		if commit.Author().Name != c[1] {
			t.Fatalf("expected %s at row %d got %s", commit.Author().Name, i, c[1])
		}

		i++
		return true
	})
	if err != nil {
		t.Fatal(err)
	}

}
func BenchmarkCLICommitCounts(b *testing.B) {
	for i := 0; i < b.N; i++ {
		rows, err := fixtureDB.Query("SELECT * FROM commits")
		if err != nil {
			b.Fatal(err)
		}
		rowNum, _, err := GetRowContents(rows)
		if err != nil {
			b.Fatalf("err %d at row Number %d", err, rowNum)
		}
	}
}
