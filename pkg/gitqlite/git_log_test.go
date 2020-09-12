package gitqlite

import (
	"fmt"
	"testing"

	git "github.com/libgit2/git2go/v30"
)

func TestCommits(t *testing.T) {
	instance, err := New(fixtureRepoDir, &Options{})
	if err != nil {
		t.Fatal(err)
	}

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

func TestCommitByID(t *testing.T) {
	instance, err := New(fixtureRepoDir, &Options{})
	if err != nil {
		t.Fatal(err)
	}

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

	rows, err := instance.DB.Query(fmt.Sprintf("SELECT * FROM commits WHERE id = '%s'", commit.Id().String()))
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	_, contents, err := GetContents(rows)
	if err != nil {
		t.Fatal(err)
	}

	count := len(contents)

	if count != 1 {
		t.Fatalf("expected 1 commit, got %d", count)
	}

	if contents[0][0] != commit.Id().String() {
		t.Fatalf("expected commit ID: %s, got %s", commit.Id().String(), contents[0][0])
	}
}

func BenchmarkCommitCounts(b *testing.B) {
	for i := 0; i < b.N; i++ {
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
}
