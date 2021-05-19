package ghqlite

import (
	"database/sql"
	"os"
	"testing"
	_ "github.com/augmentable-dev/askgit/pkg/sqlite"
	"github.com/mattn/go-sqlite3"
	"go.riyazali.net/sqlite"
)

var (
	DB *sql.DB
)

func init() {
	sqlite.Register(func(_ *sqlite.ExtensionApi) (sqlite.ErrorCode, error) { return sqlite.SQLITE_OK, nil })
	sql.Register("ghqlite", &sqlite3.SQLiteDriver{
		ConnectHook: func(conn *sqlite3.SQLiteConn) error {
			err := conn.CreateModule("github_org_repos", NewReposModule(OwnerTypeOrganization, ReposModuleOptions{
				Token: os.Getenv("GITHUB_TOKEN"),
			}))
			if err != nil {
				return err
			}

			err = conn.CreateModule("github_user_repos", NewReposModule(OwnerTypeUser, ReposModuleOptions{
				Token: os.Getenv("GITHUB_TOKEN"),
			}))
			if err != nil {
				return err
			}

			err = conn.CreateModule("github_pull_requests", NewPullRequestsModule(PullRequestsModuleOptions{
				Token: os.Getenv("GITHUB_TOKEN"),
			}))
			if err != nil {
				return err
			}
			err = conn.CreateModule("github_issues", NewIssuesModule(IssuesModuleOptions{
				Token: os.Getenv("GITHUB_TOKEN"),
			}))
			if err != nil {
				return err
			}

			return nil
		},
	})
}

func TestMain(m *testing.M) {
	err := initFixtureDB()
	if err != nil {
		panic(err)
	}
	code := m.Run()
	err = DB.Close()
	if err != nil {
		panic(err)
	}
	os.Exit(code)
}

func initFixtureDB() error {
	db, err := sql.Open("ghqlite", ":memory:")
	if err != nil {
		return err
	}

	DB = db
	return nil
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
