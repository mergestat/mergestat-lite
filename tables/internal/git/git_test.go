package git_test

import (
	"database/sql"
	"github.com/augmentable-dev/askgit/pkg/locator"
	_ "github.com/augmentable-dev/askgit/pkg/sqlite"
	"github.com/augmentable-dev/askgit/tables"
	_ "github.com/mattn/go-sqlite3"
	"go.riyazali.net/sqlite"
	"os"
	"testing"
)

func init() {
	// register sqlite extension when this package is loaded
	sqlite.Register(tables.RegisterFn(
		tables.WithExtraFunctions(), tables.WithRepoLocator(locator.CachedLocator(locator.MultiLocator())),
	))
}

// tests' entrypoint that registers the extension
// automatically with all loaded database connections
func TestMain(m *testing.M) { os.Exit(m.Run()) }

// Memory represents a uri to an in-memory database
const Memory = "file:testing.db?mode=memory"

// Connect opens a connection with the sqlite3 database using
// the given data source address and pings it to check liveliness.
func Connect(t *testing.T, dataSourceName string) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", dataSourceName)
	if err != nil {
		t.Fatalf("failed to open connection: %v", err.Error())
	}

	if err = db.Ping(); err != nil {
		t.Fatalf("failed to open connection: %v", err.Error())
	}

	return db
}
