package askgit

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	git "github.com/libgit2/git2go/v31"
)

var (
	fixtureRepoCloneURL = "https://github.com/augmentable-dev/tickgit"
	fixtureRepo         *git.Repository
	fixtureRepoDir      string
)

func TestMain(m *testing.M) {
	close, err := initFixtureRepo()
	if err != nil {
		panic(err)
	}
	code := m.Run()

	err = close()
	if err != nil {
		panic(err)
	}

	os.Exit(code)
}

func initFixtureRepo() (func() error, error) {
	dir, err := ioutil.TempDir("", "repo")
	if err != nil {
		return nil, err
	}
	fixtureRepo, err = git.Clone(fixtureRepoCloneURL, dir, &git.CloneOptions{})

	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	fixtureRepoDir = dir

	return func() error {
		err := os.RemoveAll(dir)
		if err != nil {
			return err
		}
		return nil
	}, nil
}

func GetContents(rows *sql.Rows) (int, [][]string, error) {
	count := 0
	columns, err := rows.Columns()
	if err != nil {
		return count, nil, err
	}

	pointers := make([]interface{}, len(columns))
	container := make([]sql.NullString, len(columns))
	var ret [][]string

	for i := range pointers {
		pointers[i] = &container[i]
	}

	for rows.Next() {
		err = rows.Scan(pointers...)
		if err != nil {
			return count, nil, err
		}

		r := make([]string, len(columns))
		for i, c := range container {
			if c.Valid {
				r[i] = c.String
			} else {
				r[i] = "NULL"
			}
		}
		ret = append(ret, r)
	}
	return count, ret, err

}
func TestDB(t *testing.T) {
	ag, err := New(&Options{
		RepoPath:    fixtureRepoDir,
		UseGitCLI:   false,
		GitHubToken: os.Getenv("GITHUB_TOKEN"),
		QueryOnly:   true,
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = ag.db.Exec("CREATE TABLE test (one TEXT, two TEXT)")
	if err.Error() != "attempt to write a readonly database" {
		t.Fatal(err)
	}
	ag, err = New(&Options{
		RepoPath:    fixtureRepoDir,
		UseGitCLI:   false,
		GitHubToken: os.Getenv("GITHUB_TOKEN"),
		QueryOnly:   false,
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = ag.db.Exec("CREATE TABLE test (one TEXT, two TEXT)")
	if err != nil {
		t.Fatal(err)
	}

}
