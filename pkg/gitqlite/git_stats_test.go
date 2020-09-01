package gitqlite

import (
	"testing"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
)

func TestGoGitStats(t *testing.T) {
	instance, err := New(fixtureRepoDir, &Options{SkipGitCLI: true})
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

	statsCount := 0
	err = commitChecker.ForEach(func(c *object.Commit) error {
		x, err := c.Stats()
		if err != nil {
			return err
		}
		for _, i := range x {
			if i.Addition == 0 && i.Deletion == 0 {

			} else {
				statsCount++
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	//checks commits
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
	//for some reason rows close after above statement and ya gotta query again... and create the db again -_-
	instance, err = New(fixtureRepoDir, &Options{SkipGitCLI: true})
	if err != nil {
		t.Fatal(err)
	}
	rows, err = instance.DB.Query("SELECT * FROM stats")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	numRows := GetRowsCount(rows)

	expected = statsCount
	if numRows != expected {
		t.Fatalf("expected %d rows got: %d", expected, numRows)
	}

	//TODO: Value Checking instead of just making sure the number of rows and columns are correct
	/*rows, err = instance.DB.Query("SELECT commit_id, file, additions,deletions FROM stats")
	if err != nil {
		t.Fatal(err)
	}
	rowNum, contents, err := GetContents(rows)
	if err != nil {
		t.Fatalf("err %d at row Number %d", err, rowNum)
	}
	statsIndex := 0
	// reset commitchecker
	commitChecker, err = fixtureRepo.Log(&git.LogOptions{
		From:  headRef.Hash(),
		Order: git.LogOrderCommitterTime,
	})
	if err != nil {
		t.Fatal(err)
	}

	commit, err := commitChecker.Next()
	if err != nil {
		t.Fatal(err)
	}
	//for the range of contents check against each stat to see if there are discrepancies
	for i, c := range contents {
		stats, err := commit.Stats()
		if err != nil {
			t.Fatal(err)
		}
		if len(stats) > 0 {
			if commit.ID().String() != c[0] {
				t.Fatalf("expected %s %d %d at row %d got %s %s %s", commit.ID().String(), stats[statsIndex].Addition, stats[statsIndex].Deletion, i, c[0], c[2], c[3])
			}
			// if stats[statsIndex].Name != c[1] && c[1] != "NULL" {
			// 	t.Fatalf("expected %s, %d, %d at row %d got %s", stats[statsIndex].Name, stats[statsIndex].Addition, stats[statsIndex].Deletion, i, c[1])
			// }
			if statsIndex < len(stats) {
				statsIndex++
			}
			for statsIndex < len(stats) && stats[statsIndex].Addition == 0 && stats[statsIndex].Deletion == 0 {
				statsIndex++
			}
		}
		if statsIndex > len(stats)-1 {
			commit, err = commitChecker.Next()
			if err != nil {
				if err == io.EOF {
					break
				} else {
					t.Fatal(err)
				}
			}
			statsIndex = 0
		}

	}*/

}
func BenchmarkGoGitstatsCounts(b *testing.B) {
	for i := 0; i < b.N; i++ {
		instance, err := New(fixtureRepoDir, &Options{SkipGitCLI: true})
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
