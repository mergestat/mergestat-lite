package cmd

import (
	"os"

	"github.com/askgitdev/askgit/extensions"
	"github.com/askgitdev/askgit/extensions/options"
	"github.com/askgitdev/askgit/pkg/locator"
	"go.riyazali.net/sqlite"

	// bring in sqlite 🙌
	_ "github.com/askgitdev/askgit/pkg/sqlite"
	_ "github.com/mattn/go-sqlite3"
)

func registerExt() {
	sqlite.Register(
		extensions.RegisterFn(
			options.WithExtraFunctions(),
			options.WithRepoLocator(locator.CachedLocator(locator.MultiLocator())),
			options.WithContextValue("defaultRepoPath", repo),
			options.WithGitHub(),
			options.WithContextValue("githubToken", githubToken),
			options.WithContextValue("sourcegraphToken", sourcegraphToken),
			options.WithContextValue("githubPerPage", os.Getenv("GITHUB_PER_PAGE")),
			options.WithContextValue("githubRateLimit", os.Getenv("GITHUB_RATE_LIMIT")),
		),
	)
}
