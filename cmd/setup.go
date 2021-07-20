package cmd

import (
	"github.com/askgitdev/askgit/pkg/locator"
	"github.com/askgitdev/askgit/tables"
	"go.riyazali.net/sqlite"

	// bring in sqlite 🙌
	_ "github.com/askgitdev/askgit/pkg/sqlite"
	_ "github.com/mattn/go-sqlite3"
)

func registerExt() {
	sqlite.Register(
		tables.RegisterFn(
			tables.WithExtraFunctions(),
			tables.WithRepoLocator(locator.CachedLocator(locator.MultiLocator())),
			tables.WithContextValue("defaultRepoPath", repo),
		),
	)
}
