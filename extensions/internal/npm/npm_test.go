package npm_test

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"path"
	"testing"

	"github.com/dnaeon/go-vcr/v2/recorder"
	_ "github.com/mattn/go-sqlite3"
	"github.com/mergestat/mergestat/extensions"
	"github.com/mergestat/mergestat/extensions/options"
	_ "github.com/mergestat/mergestat/pkg/sqlite"
	"go.riyazali.net/sqlite"
)

// FixtureDatabase represents the database connection to run the test against
var FixtureDatabase *sql.DB
var httpClient = http.DefaultClient

func TestMain(m *testing.M) {
	// register sqlite extension when this package is loaded
	sqlite.Register(extensions.RegisterFn(
		options.WithNPM(),
		options.WithNPMHttpClient(httpClient),
	))

	var err error
	if FixtureDatabase, err = sql.Open("sqlite3", "file:testing.db?mode=memory"); err != nil {
		log.Fatalf("failed to open database connection: %v", err)
	}

	os.Exit(m.Run())
}

func newRecorder(t *testing.T) func() {
	r, err := recorder.New(path.Join("fixtures", t.Name()))
	if err != nil {
		t.Fatal(err)
	}

	r.SkipRequestLatency = true
	r.SetTransport(http.DefaultTransport)
	httpClient.Transport = r

	return func() {
		err := r.Stop()
		if err != nil {
			t.Fatal(err)
		}
	}
}
