package gitqlite

import (
	"context"
	"database/sql"
	"io/ioutil"
	"os"
	"testing"

	git "github.com/libgit2/git2go/v31"
	"github.com/mattn/go-sqlite3"
)

var (
	fixtureRepoCloneURL = "https://github.com/augmentable-dev/tickgit"
	fixtureRepo         *git.Repository
	fixtureRepoDir      string
	fixtureDB           *sql.DB
)

func TestMain(m *testing.M) {
	sql.Register("gitqlite", &sqlite3.SQLiteDriver{})

	close, err := initFixtureRepo()
	if err != nil {
		panic(err)
	}
	err = initFixtureDB(fixtureRepoDir)
	if err != nil {
		panic(err)
	}
	code := m.Run()
	
	err = close()
	if err != nil {
		panic(err)
	}

	err = fixtureDB.Close()
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

func initFixtureDB(repoPath string) error {
	db, err := sql.Open("gitqlite", ":memory:")
	if err != nil {
		return err
	}

	conn, err := db.Conn(context.Background())
	if err != nil {
		return err
	}
	defer conn.Close()

	var sqliteConn *sqlite3.SQLiteConn
	err = conn.Raw(func(driverConn interface{}) error {
		sqliteConn = driverConn.(*sqlite3.SQLiteConn)
		return nil
	})
	if err != nil {
		return err
	}

	err = sqliteConn.CreateModule("commits", NewGitLogModule(&GitLogModuleOptions{RepoPath: repoPath}))
	if err != nil {
		return err
	}

	err = sqliteConn.CreateModule("commits_cli", NewGitLogCLIModule(&GitLogCLIModuleOptions{RepoPath: repoPath}))
	if err != nil {
		return err
	}

	err = sqliteConn.CreateModule("stats", NewGitStatsModule(&GitStatsModuleOptions{RepoPath: repoPath}))
	if err != nil {
		return err
	}

	err = sqliteConn.CreateModule("files", NewGitFilesModule(&GitFilesModuleOptions{RepoPath: repoPath}))
	if err != nil {
		return err
	}

	err = sqliteConn.CreateModule("tags", NewGitTagsModule(&GitTagsModuleOptions{RepoPath: repoPath}))
	if err != nil {
		return err
	}

	err = sqliteConn.CreateModule("branches", NewGitBranchesModule(&GitBranchesModuleOptions{RepoPath: repoPath}))
	if err != nil {
		return err
	}

	err = sqliteConn.CreateModule("blame", NewGitBlameModule(&GitBlameModuleOptions{RepoPath: repoPath}))
	if err != nil {
		return err
	}

	fixtureDB = db
	return nil
}

func GetRowsCount(rows *sql.Rows) int {
	count := 0
	for rows.Next() {
		count++
	}

	return count
}

func GetRowContents(rows *sql.Rows) (colCount int, contents [][]string, err error) {
	columns, err := rows.Columns()
	if err != nil {
		return colCount, nil, err
	}

	pointers := make([]interface{}, len(columns))
	container := make([]sql.NullString, len(columns))

	for i := range pointers {
		pointers[i] = &container[i]
	}

	for rows.Next() {
		err = rows.Scan(pointers...)
		if err != nil {
			return colCount, nil, err
		}

		r := make([]string, len(columns))
		for i, c := range container {
			if c.Valid {
				r[i] = c.String
			} else {
				r[i] = "NULL"
			}
		}
		contents = append(contents, r)
	}
	return colCount, contents, err

}
