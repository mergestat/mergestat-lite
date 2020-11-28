package ghqlite

import (
	"database/sql"
	"fmt"
	"os"
	"testing"

	"github.com/mattn/go-sqlite3"
)

var (
	DB *sql.DB
)

func init() {
	sql.Register("ghqlite", &sqlite3.SQLiteDriver{
		ConnectHook: func(conn *sqlite3.SQLiteConn) error {
			err := conn.CreateModule("github_repos", NewReposModule())
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

	_, err = db.Exec(fmt.Sprintf("CREATE VIRTUAL TABLE IF NOT EXISTS repos USING github_repos(%s, '%s');", "augmentable-dev", os.Getenv("GITHUB_TOKEN")))
	if err != nil {
		return err
	}

	DB = db
	return nil
}
