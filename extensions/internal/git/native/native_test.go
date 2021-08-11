package native_test

import (
	"database/sql"
	"os"
	"testing"

	"github.com/askgitdev/askgit/extensions"
	"github.com/askgitdev/askgit/extensions/options"
	"github.com/askgitdev/askgit/pkg/locator"
	_ "github.com/askgitdev/askgit/pkg/sqlite"
	_ "github.com/mattn/go-sqlite3"
	"go.riyazali.net/sqlite"
)

func init() {
	// register sqlite extension when this package is loaded
	sqlite.Register(extensions.RegisterFn(
		options.WithExtraFunctions(), options.WithRepoLocator(locator.CachedLocator(locator.MultiLocator())),
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
