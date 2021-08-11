package cmd

import (
	"os"

	"github.com/askgitdev/askgit/extensions"
	"github.com/askgitdev/askgit/pkg/locator"
	"go.riyazali.net/sqlite"

	// bring in sqlite 🙌
	_ "github.com/askgitdev/askgit/pkg/sqlite"
	_ "github.com/mattn/go-sqlite3"
)

func registerExt() {
	sqlite.Register(
		extensions.RegisterFn(
			extensions.WithExtraFunctions(),
			extensions.WithRepoLocator(locator.CachedLocator(locator.MultiLocator())),
			extensions.WithContextValue("defaultRepoPath", repo),
			extensions.WithGitHub(),
			extensions.WithContextValue("githubToken", githubToken),
			extensions.WithContextValue("githubPerPage", os.Getenv("GITHUB_PER_PAGE")),
			extensions.WithContextValue("githubRateLimit", os.Getenv("GITHUB_RATE_LIMIT")),
		),
	)
}
