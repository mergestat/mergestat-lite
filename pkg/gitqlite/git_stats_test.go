package gitqlite

import (
	"io"
	"strconv"
	"testing"

	"github.com/augmentable-dev/askgit/pkg/gitlog"
)

func TestStatsIterator(t *testing.T) {
	iter, err := NewCommitStatsIter(fixtureRepo, &commitStatsIterOptions{})
	if err != nil {
		t.Fatal(err)
	}

	for {
		stat, err := iter.Next()
		if err == io.EOF {
			break
		}
		if stat.commitID == "" {
			t.Fatal("invalid commit")
		}
	}
}

func TestStatsTable(t *testing.T) {
	rows, err := fixtureDB.Query("SELECT * FROM stats")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	_, contents, err := GetRowContents(rows)
	if err != nil {
		t.Fatal(err)
	}

	if len(contents[0]) != 4 {
		t.Fatalf("expected 4 columns, got %d", len(contents[0]))
	}

}

func TestStatsTableCommitIDIndex(t *testing.T) {
	rows, err := fixtureDB.Query("SELECT * FROM stats WHERE commit_id = (SELECT id FROM commits('" + fixtureRepoCloneURL + "') LIMIT 1)")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	_, contents, err := GetRowContents(rows)
	if err != nil {
		t.Fatal(err)
	}

	if len(contents[0]) != 4 {
		t.Fatalf("expected 4 columns, got %d", len(contents[0]))
	}

	// TODO actually test the results here?
	// this test case added to activate the code path that looks up commit stats by commit id
	// (avoiding a full table scan)
}

func TestStatsTotals(t *testing.T) {
	iter, err := gitlog.Execute(fixtureRepoDir)
	if err != nil {
		t.Fatal(err)
	}

	totalAdditions := 0
	totalDeletions := 0
	for {
		commit, err := iter.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			t.Fatal(err)
		}
		totalAdditions += commit.Additions
		totalDeletions += commit.Deletions
	}

	if err != nil {
		t.Fatal(err)
	}

	rows, err := fixtureDB.Query("SELECT sum(additions) AS a, sum(deletions) AS d FROM stats")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	_, contents, err := GetRowContents(rows)
	if err != nil {
		t.Fatal(err)
	}

	gotAdditions, err := strconv.Atoi(contents[0][0])
	if err != nil {
		t.Fatal(err)
	}

	gotDeletions, err := strconv.Atoi(contents[0][1])
	if err != nil {
		t.Fatal(err)
	}

	if totalAdditions != gotAdditions {
		t.Fatalf("expected: %d, got: %d total additions", totalAdditions, gotAdditions)
	}

	if totalDeletions != gotDeletions {
		t.Fatalf("expected: %d, got: %d total deletions", totalDeletions, gotDeletions)
	}
}

func BenchmarkStats(b *testing.B) {
	for i := 0; i < b.N; i++ {
		rows, err := fixtureDB.Query("SELECT * FROM stats")
		if err != nil {
			b.Fatal(err)
		}
		rowNum, _, err := GetRowContents(rows)
		if err != nil {
			b.Fatalf("err %d at row Number %d", err, rowNum)
		}
	}
}
