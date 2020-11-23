package gitqlite

import (
	"fmt"
	"strings"
	"testing"

	git "github.com/libgit2/git2go/v30"
)

type test struct {
	name  string
	query string
	want  []string
}

func TestCommits(t *testing.T) {

	//this only works if you test one thing at a time.
	testCases := []test{
		{"checkCommits", "SELECT COUNT(*) FROM commits", getCommitCount(t)},
		{"getAuthors", "SELECT author_name FROM commits", getAuthors(t)},
		{"getID's", "SELECT id FROM commits", getIDs(t)},
	}
	for _, tc := range testCases {
		expected := tc.want
		results := runQuery(t, tc.query)
		if len(expected) != len(results) {
			t.Fatalf("expected %d entries got %d, test: %s, %s, %s", len(expected), len(results), tc.name, expected, results)
		}
		for x := 0; x < len(expected); x++ {
			if results[x] != expected[x] {
				t.Fatalf("expected %s, got %s, test %s", expected[x], results[x], tc.name)
			}
		}
	}

}
func runQuery(t *testing.T, query string) []string {
	instance, err := New(fixtureRepoDir, &Options{})
	if err != nil {
		t.Fatal(err)
	}
	//checks commits
	rows, err := instance.DB.Query(query)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	rowNum, contents, err := GetContents(rows)
	if err != nil {
		t.Fatalf("err %d at row Number %d", err, rowNum)
	}
	var ret []string
	for _, entry := range contents {
		ret = append(ret, entry...)
	}

	return ret
}
func getAuthors(t *testing.T) []string {
	//revWalk := createRevWalk(t)
	revWalk, err := fixtureRepo.Walk()
	if err != nil {
		t.Fatal(err)
	}

	err = revWalk.PushHead()
	if err != nil {
		t.Fatal(err)
	}
	var authors []string

	err = revWalk.Iterate(func(commit *git.Commit) bool {
		authors = append(authors, commit.Author().Name)
		return true
	})
	if err != nil {
		t.Fatal(err)
	}

	return authors
}
func getIDs(t *testing.T) []string {
	revWalk, err := fixtureRepo.Walk()
	if err != nil {
		t.Fatal(err)
	}

	err = revWalk.PushHead()
	if err != nil {
		t.Fatal(err)
	}
	var IDs []string
	err = revWalk.Iterate(func(commit *git.Commit) bool {
		IDs = append(IDs, commit.Id().String())
		return true
	})
	if err != nil {
		t.Fatal(err)
	}

	return IDs
}
func getCommitCount(t *testing.T) []string {
	//revWalk := createRevWalk(t)
	revWalk, err := fixtureRepo.Walk()
	if err != nil {
		t.Fatal(err)
	}

	err = revWalk.PushHead()
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	err = revWalk.Iterate(func(commit *git.Commit) bool {
		count++
		return true
	})
	if err != nil {
		t.Fatal(err)
	}
	r := fmt.Sprintf("%d", count)
	ret := strings.Split(r, " ")

	//revWalk.Free()
	return ret
}

// func createRevWalk(t *testing.T) *git.RevWalk {
// 	revWalk, err := fixtureRepo.Walk()
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	defer revWalk.Free()

// 	err = revWalk.PushHead()
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	return revWalk
// }
func TestCommitByID(t *testing.T) {
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

	rows, err := fixtureDB.Query(fmt.Sprintf("SELECT * FROM commits WHERE id = '%s'", commit.Id().String()))
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
		rows, err := fixtureDB.Query("SELECT * FROM commits")
		if err != nil {
			b.Fatal(err)
		}
		rowNum, _, err := GetContents(rows)
		if err != nil {
			b.Fatalf("err %d at row Number %d", err, rowNum)
		}
	}
}
