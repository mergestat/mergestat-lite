// +build shared

// This file provides a build target while building the dynamically loadable shared object library.
// It imports github.com/askgitdev/askgit/extensions which provides the actual extension implementation.
package main

import (
	"os"

	"github.com/askgitdev/askgit/extensions"
	"github.com/askgitdev/askgit/extensions/options"
	"github.com/askgitdev/askgit/pkg/locator"
	"go.riyazali.net/sqlite"
)

func init() {
	sqlite.Register(extensions.RegisterFn(
		options.WithExtraFunctions(),
		options.WithRepoLocator(locator.CachedLocator(locator.MultiLocator())),
		options.WithGitHub(),
		options.WithContextValue("githubToken", os.Getenv("GITHUB_TOKEN")),
		options.WithContextValue("githubPerPage", os.Getenv("GITHUB_PER_PAGE")),
		options.WithContextValue("githubRateLimit", os.Getenv("GITHUB_RATE_LIMIT")),
	))
}

func main() { /* noting here fellas */ }
