package gitqlite

import (
	"testing"

	git "github.com/libgit2/git2go/v30"
)

func TestStatsTable(t *testing.T) {
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
	rows, err := instance.DB.Query("SELECT DISTINCT commit_id FROM stats")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		t.Fatal(err)
	}

	expected := 4
	if len(columns) != expected {
		t.Fatalf("expected %d columns, got: %d", expected, len(columns))
	}
	numRows := GetRowsCount(rows)

	expected = commitCount
	if numRows != expected {
		t.Fatalf("expected %d rows got: %d", expected, numRows)
	}
}

func BenchmarkStats(b *testing.B) {
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
