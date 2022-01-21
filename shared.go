//go:build shared
// +build shared

// This file provides a build target while building the dynamically loadable shared object library.
// It imports github.com/mergestat/mergestat/extensions which provides the actual extension implementation.
package main

import (
	"os"

	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/mergestat/mergestat/extensions"
	"github.com/mergestat/mergestat/extensions/options"
	"github.com/mergestat/mergestat/pkg/locator"
	"go.riyazali.net/sqlite"
)

func init() {
	githubToken := os.Getenv("GITHUB_TOKEN")
	var multiLocOpt *locator.MultiLocatorOptions
	if githubToken != "" {
		multiLocOpt = &locator.MultiLocatorOptions{
			HTTPAuth: &http.BasicAuth{Username: githubToken},
		}
	}

	sqlite.Register(extensions.RegisterFn(
		options.WithExtraFunctions(),
		options.WithRepoLocator(locator.CachedLocator(locator.MultiLocator(multiLocOpt))),
		options.WithGitHub(),
		options.WithContextValue("githubToken", githubToken),
		options.WithContextValue("githubPerPage", os.Getenv("GITHUB_PER_PAGE")),
		options.WithContextValue("githubRateLimit", os.Getenv("GITHUB_RATE_LIMIT")),
		options.WithSourcegraph(),
		options.WithContextValue("sourcegraphToken", os.Getenv("SOURCEGRAPH_TOKEN")),
		options.WithNPM(),
	))
}

func main() { /* noting here fellas */ }
