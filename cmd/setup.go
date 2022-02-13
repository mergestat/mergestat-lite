package cmd

import (
	"os"

	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/mergestat/mergestat/extensions"
	"github.com/mergestat/mergestat/extensions/options"
	"github.com/mergestat/mergestat/pkg/locator"
	"go.riyazali.net/sqlite"

	// bring in sqlite 🙌
	_ "github.com/mattn/go-sqlite3"
	_ "github.com/mergestat/mergestat/pkg/sqlite"
)

func registerExt() {
	multiLocOpt := &locator.MultiLocatorOptions{
		CloneDir:        cloneDir,
		InsecureSkipTLS: gitSSLNoVerify != "",
	}
	if githubToken != "" {
		multiLocOpt.HTTPAuth = &http.BasicAuth{Username: githubToken}
	}

	sqlite.Register(
		extensions.RegisterFn(
			options.WithExtraFunctions(),
			options.WithRepoLocator(locator.CachedLocator(locator.LoggingLocator(
				&logger,
				locator.MultiLocator(multiLocOpt),
			))),
			options.WithContextValue("defaultRepoPath", repo),
			options.WithGitHub(),
			options.WithContextValue("githubToken", githubToken),
			options.WithContextValue("githubPerPage", os.Getenv("GITHUB_PER_PAGE")),
			options.WithContextValue("githubRateLimit", os.Getenv("GITHUB_RATE_LIMIT")),
			options.WithSourcegraph(),
			options.WithContextValue("sourcegraphToken", sourcegraphToken),
			options.WithNPM(),
			options.WithLogger(&logger),
		),
	)
}
