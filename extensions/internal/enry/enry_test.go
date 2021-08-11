package enry

import (
	"database/sql"
	"log"
	"os"
	"testing"

	_ "github.com/askgitdev/askgit/pkg/sqlite"
	"go.riyazali.net/sqlite"
)

// FixtureDatabase represents the database connection to run the test against
var FixtureDatabase *sql.DB

func init() {
	// register sqlite extension when this package is loaded
	sqlite.Register(func(ext *sqlite.ExtensionApi) (_ sqlite.ErrorCode, err error) {
		return Register(ext, nil)
	})
}

func TestMain(m *testing.M) {
	var err error
	if FixtureDatabase, err = sql.Open("sqlite3", "file:testing.db?mode=memory"); err != nil {
		log.Fatalf("failed to open database connection: %v", err)
	}

	os.Exit(m.Run())
}
