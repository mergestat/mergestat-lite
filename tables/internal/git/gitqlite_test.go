package git

import (
	"context"
	"database/sql"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"testing"

	"github.com/pkg/errors"
	"go.riyazali.net/sqlite"

	_ "github.com/askgitdev/askgit/pkg/sqlite"
	git "github.com/libgit2/git2go/v31"
	_ "github.com/mattn/go-sqlite3"
)

var (
	fixtureRepoCloneURL = "https://github.com/augmentable-dev/tickgit"
	fixtureRepo         *git.Repository
	fixtureRepoDir      string
	fixtureDB           *sql.DB
)

func init() {
	// register sqlite extension when this package is loaded
	sqlite.Register(func(ext *sqlite.ExtensionApi) (_ sqlite.ErrorCode, err error) {
		var modules = map[string]sqlite.Module{
			"git_blame":    &BlameModule{},
			"git_branches": &BranchesModule{},
			"git_files":    &FilesModule{},
			"git_log":      &LogModule{},
			"git_stats":    &StatsModule{},
			"git_tags":     &TagsModule{},
		}

		for name, mod := range modules {
			if err = ext.CreateModule(name, mod); err != nil {
				return sqlite.SQLITE_ERROR, errors.Wrap(err, "failed to register 'git_blame' module")
			}
		}

		return sqlite.SQLITE_OK, nil
	})
}

func TestMain(m *testing.M) {
	var err error

	var done func() error
	if done, err = initFixtureRepo(); err != nil {
		log.Fatalf("failed to initialize fixture repository: %v", err)
	}
	defer func() {
		err := done()
		log.Fatal(err)
	}()

	if err = initFixtureDB(fixtureRepoDir); err != nil {
		log.Fatalf("failed to initialize fixture database: %v", err)
	}

	os.Exit(m.Run())
}

func initFixtureRepo() (_ func() error, err error) {
	if fixtureRepoDir, err = ioutil.TempDir("", "repo"); err != nil {
		return nil, err
	}
	if fixtureRepo, err = git.Clone(fixtureRepoCloneURL, fixtureRepoDir, &git.CloneOptions{}); err != nil {
		return nil, err
	}
	return func() error { return os.RemoveAll(fixtureRepoDir) }, nil
}

func initFixtureDB(repoPath string) (err error) {
	if fixtureDB, err = sql.Open("sqlite3", ":memory:"); err != nil {
		return err
	}

	conn, err := fixtureDB.Conn(context.Background())
	if err != nil {
		return err
	}
	defer conn.Close()

	var sp = fmt.Sprintf
	var stmts = []string{
		sp("CREATE VIRTUAL TABLE commits  USING git_log(path=%q)", repoPath),
		sp("CREATE VIRTUAL TABLE branches USING git_branches(path=%q)", repoPath),
		sp("CREATE VIRTUAL TABLE blame    USING git_blame(path=%q)", repoPath),
		sp("CREATE VIRTUAL TABLE files    USING git_files(path=%q)", repoPath),
		sp("CREATE VIRTUAL TABLE stats    USING git_stats(path=%q)", repoPath),
		sp("CREATE VIRTUAL TABLE tags     USING git_tags(path=%q)", repoPath),
	}

	for _, stmt := range stmts {
		if _, err := conn.ExecContext(context.Background(), stmt, nil); err != nil {
			return errors.Wrap(err, "failed to create virtual table")
		}
	}

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
	return colCount, contents, rows.Err()

}
