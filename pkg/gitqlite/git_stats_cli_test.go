package gitqlite

import (
	"strings"
	"testing"

	"github.com/augmentable-dev/askgit/pkg/gitlog"
)

func TestStats(t *testing.T) {
	instance, err := New(fixtureRepoDir, &Options{UseGitCLI: true})
	if err != nil {
		t.Fatal(err)
	}

	iter, err := gitlog.Execute(fixtureRepoDir)
	if err != nil {
		t.Fatal(err)
	}
	commit, err := iter.Next()
	if err != nil {
		t.Fatal(err)
	}
	vc := StatsCLICursor{repoPath: fixtureRepoDir, iter: iter, current: commit, statIndex: 0}
	rows, err := instance.DB.Query("SELECT * FROM stats")
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
	//instance closes after read and needs to be reopened
	instance, err = New(fixtureRepoDir, &Options{})
	if err != nil {
		t.Fatal(err)
	}

	rows, err = instance.DB.Query("SELECT commit_id, file FROM stats")
	if err != nil {
		t.Fatal(err)
	}
	rowNum, contents, err := GetContents(rows)
	if err != nil {
		t.Fatalf("err %d at row Number %d", err, rowNum)
	}
	for i, c := range contents {
		if strings.Compare(vc.current.SHA, c[0]) != 0 {
			t.Fatalf("expected %s at row %d got %s", vc.current.SHA, i, c[0])
		}
		if len(vc.current.Stats) > vc.statIndex {
			if vc.current.Stats[vc.statIndex].File != c[1] && c[1] != "NULL" {
				t.Fatalf("expected %s at row %d got %s", vc.current.Stats[vc.statIndex].File, i, c[1])
			}
		}
		err = vc.Next()
		if err != nil {
			t.Fatal(err)
		}

	}

}

func BenchmarkStatsCounts(b *testing.B) {
	for i := 0; i < b.N; i++ {
		instance, err := New(fixtureRepoDir, &Options{UseGitCLI: true})
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
