package cmd

import (
	"github.com/augmentable-dev/askgit/pkg/locator"
	"github.com/augmentable-dev/askgit/tables"
	"go.riyazali.net/sqlite"
)

// bring in sqlite 🙌
import _ "github.com/augmentable-dev/askgit/pkg/sqlite"
import _ "github.com/mattn/go-sqlite3"

func init() {
	sqlite.Register(
		tables.RegisterFn(
			tables.WithExtraFunctions(),
			tables.WithRepoLocator(locator.CachedLocator(locator.MultiLocator())),
		),
	)
}
