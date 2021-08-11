package github_test

import (
	"context"
	"database/sql"
	"os"
	"path"
	"testing"

	"github.com/askgitdev/askgit/extensions"
	_ "github.com/askgitdev/askgit/pkg/sqlite"
	"github.com/dnaeon/go-vcr/v2/cassette"
	"github.com/dnaeon/go-vcr/v2/recorder"
	_ "github.com/mattn/go-sqlite3"
	"github.com/shurcooL/githubv4"
	"go.riyazali.net/sqlite"
	"golang.org/x/oauth2"
)

var httpClient = oauth2.NewClient(context.Background(), oauth2.StaticTokenSource(
	&oauth2.Token{AccessToken: os.Getenv("GITHUB_TOKEN")},
))

func newRecorder(t *testing.T) func() {
	r, err := recorder.New(path.Join("fixtures", t.Name()))
	if err != nil {
		t.Fatal(err)
	}
	r.SkipRequestLatency = true
	r.SetTransport(httpClient.Transport)
	httpClient.Transport = r

	r.AddSaveFilter(func(i *cassette.Interaction) error {
		delete(i.Request.Headers, "Authorization")
		return nil
	})

	return func() {
		err := r.Stop()
		if err != nil {
			t.Fatal(err)
		}
	}
}

// tests' entrypoint that registers the extension
// automatically with all loaded database connections
func TestMain(m *testing.M) {
	// register sqlite extension when this package is loaded
	sqlite.Register(extensions.RegisterFn(
		extensions.WithExtraFunctions(),
		extensions.WithGitHub(),
		extensions.WithGitHubClientGetter(func() *githubv4.Client {
			return githubv4.NewClient(httpClient)
		}),
	))
	os.Exit(m.Run())
}

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
