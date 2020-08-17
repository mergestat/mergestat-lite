package gitqlite

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gitsight/go-vcsurl"
	"github.com/go-git/go-git/v5"
	"github.com/google/go-github/github"
	"github.com/mattn/go/src/strconv"
)

func TestPr(t *testing.T) {

	if remote, err := vcsurl.Parse(fixtureRepoCloneURL); err == nil { // if it can be parsed
		if r, err := remote.Remote(vcsurl.HTTPS); err == nil { // if it can be resolved into an HTTPS remote
			var dir string
			fixtureRepoCloneURL = strings.ReplaceAll(fixtureRepoCloneURL, ".git", "")
			if strings.Contains(fixtureRepoCloneURL, "http") {
				stuff := strings.Split(fixtureRepoCloneURL, "/")
				owner, rep := stuff[3], stuff[4]
				dir, err = ioutil.TempDir("", ":"+owner+":"+rep+":")
				if err != nil {
					t.Fatal(err)
				}
			} else {
				stuff := strings.Split(fixtureRepoCloneURL, ":")
				x := stuff[1]
				y := strings.Split(x, "/")
				owner, rep := y[0], y[1]
				dir, err = ioutil.TempDir("", ":"+owner+":"+rep+":")
				if err != nil {
					t.Fatal(err)
				}
			}

			_, err = git.PlainClone(dir, false, &git.CloneOptions{
				URL: r,
			})
			if err != nil {
				t.Fatal(err)
			}

			defer func() {
				err := os.RemoveAll(dir)
				if err != nil {
					t.Fatal(err)
				}
			}()

			fixtureRepoCloneURL = dir
		}
	}
	fixtureRepoCloneURL, err := filepath.Abs(fixtureRepoCloneURL)
	if err != nil {
		t.Fatal(err)
	}
	instance, err := New(fixtureRepoCloneURL, &Options{})
	if err != nil {
		t.Fatal(err)
	}

	client := github.NewClient(nil)

	pr, _, err := client.PullRequests.List(context.Background(), "augmentable-dev", "tickgit", &github.PullRequestListOptions{State: "all", ListOptions: github.ListOptions{PerPage: 50}})
	if err != nil {
		t.Fatal(err)
	}
	//prCount := len(pr)

	//checks pr's
	rows, err := instance.DB.Query("SELECT * FROM pr")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	columns, err := rows.Columns()
	if err != nil {
		t.Fatal(err)
	}
	expected := 7
	if len(columns) != expected {
		t.Fatalf("expected %d columns, got: %d", expected, len(columns))
	}
	_, _, err = GetContents(rows)
	if err != nil {
		t.Fatalf("error in retrieving rows")
	}
	// numRows := GetRowsCount(rows)

	// expected = prCount
	// if numRows != expected {
	// 	t.Fatalf("expected %d rows got: %d", expected, numRows)
	// }
	instance, err = New(fixtureRepoCloneURL, &Options{})
	if err != nil {
		t.Fatal(err)
	}
	rows, err = instance.DB.Query("SELECT number, author FROM pr")
	if err != nil {
		t.Fatal(err)
	}
	rowNum, contents, err := GetContents(rows)
	if err != nil {
		t.Fatalf("err %d at row Number %d", err, rowNum)
	}
	for i, c := range contents {
		tempr := pr[i]
		expected, err := strconv.ParseInt(c[0], 10, 32)
		if err != nil {
			t.Fatal(err)
		}
		if *tempr.Number != int(expected) {
			t.Fatalf("expected %d at row %d got %s", *tempr.Number, i, c[0])
		}
		if tempr.User.GetLogin() != c[1] {
			t.Fatalf("expected %s at row %d got %s", tempr.User.GetLogin(), i, c[1])
		}

	}
}
