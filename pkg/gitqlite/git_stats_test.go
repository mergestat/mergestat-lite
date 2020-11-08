package gitqlite

import (
	"io"
	"testing"

	"github.com/augmentable-dev/askgit/pkg/gitlog"
)

func TestStatsTable(t *testing.T) {
	instance, err := New(fixtureRepoDir, &Options{})
	if err != nil {
		t.Fatal(err)
	}

	iter, err := gitlog.Execute(fixtureRepoDir)
	if err != nil {
		t.Fatal(err)
	}

	wantCommits := make([]*gitlog.Commit, 0)

	for {
		commit, err := iter.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			t.Fatal(err)
		}
		wantCommits = append(wantCommits, commit)
	}

	if err != nil {
		t.Fatal(err)
	}

	rows, err := instance.DB.Query("SELECT count(*) FROM stats GROUP BY commit_id")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	_, contents, err := GetContents(rows)
	if err != nil {
		t.Fatal(err)
	}

	if len(contents) != len(wantCommits) {
		t.Fatalf("want: %d commits, got: %d", len(wantCommits), len(contents))
	}

	// columns, err := rows.Columns()
	// if err != nil {
	// 	t.Fatal(err)
	// }

	// expected := 4
	// if len(columns) != expected {
	// 	t.Fatalf("expected %d columns, got: %d", expected, len(columns))
	// }

	// TODO: find a good way to do feature checking. Stats doesn't include commits that are merges so it is hard to do a pure count of commits for the table.
	// instance, err = New(fixtureRepoDir, &Options{})
	// if err != nil {
	// 	t.Fatal(err)
	// }
	// rows, err = instance.DB.Query("SELECT DISTINCT commit_id FROM stats")
	// if err != nil {
	// 	t.Fatal(err)
	// }
	// defer rows.Close()
	// numRows := GetRowsCount(rows)

	// expected = commitCount
	// if numRows != expected {
	// 	t.Fatalf("expected %d rows got: %d", expected, numRows)
	// }
}

func BenchmarkStats(b *testing.B) {
	for i := 0; i < b.N; i++ {
		instance, err := New(fixtureRepoDir, &Options{})
		if err != nil {
			b.Fatal(err)
		}
		rows, err := instance.DB.Query("SELECT * FROM stats")
		if err != nil {
			b.Fatal(err)
		}
		rowNum, _, err := GetContents(rows)
		if err != nil {
			b.Fatalf("err %d at row Number %d", err, rowNum)
		}
	}
}
