// Package extensions provide implementation of the various underlying sqlite3 virtual tables [https://www.sqlite.org/vtab.html] and user defined functions
// that askgit uses under-the-hood. This module can be side-effect-imported in other modules to include the functionality
// of the sqlite3 extensions there.
package extensions

import (
	"context"
	"time"

	"github.com/askgitdev/askgit/extensions/internal/enry"
	"github.com/askgitdev/askgit/extensions/internal/git"
	"github.com/askgitdev/askgit/extensions/internal/git/native"
	"github.com/askgitdev/askgit/extensions/internal/github"
	"github.com/askgitdev/askgit/extensions/internal/helpers"
	"github.com/pkg/errors"
	"github.com/shurcooL/githubv4"
	"go.riyazali.net/sqlite"
	"golang.org/x/oauth2"
	"golang.org/x/time/rate"
)

func RegisterFn(fns ...OptionFn) func(ext *sqlite.ExtensionApi) (_ sqlite.ErrorCode, err error) {
	var opt = &Options{}
	for _, fn := range fns {
		fn(opt)
	}

	// return an extension function that register modules with sqlite when this package is loaded
	return func(ext *sqlite.ExtensionApi) (_ sqlite.ErrorCode, err error) {
		// register virtual table modules
		var modules = map[string]sqlite.Module{
			"commits": &git.LogModule{Locator: opt.Locator, Context: opt.Context},
			"refs":    &git.RefModule{Locator: opt.Locator, Context: opt.Context},
			"stats":   native.NewStatsModule(opt.Locator, opt.Context),
			"files":   native.NewFilesModule(opt.Locator, opt.Context),
			"blame":   native.NewBlameModule(opt.Locator, opt.Context),
		}

		for name, mod := range modules {
			if err = ext.CreateModule(name, mod); err != nil {
				return sqlite.SQLITE_ERROR, errors.Wrapf(err, "failed to register %q module", name)
			}
		}

		var fns = map[string]sqlite.Function{
			"commit_from_tag": &git.CommitFromTagFn{},
		}

		for name, fn := range fns {
			if err = ext.CreateFunction(name, fn); err != nil {
				return sqlite.SQLITE_ERROR, errors.Wrapf(err, "failed to register %q function", name)
			}
		}

		// only conditionally register the utility functions
		if opt.ExtraFunctions {
			if sqliteErr, err := helpers.Register(ext); err != nil {
				return sqliteErr, err
			}

			if sqliteErr, err := enry.Register(ext); err != nil {
				return sqliteErr, err
			}
		}

		// conditionally register the GitHub functionality
		if opt.GitHub {
			rateLimiter := github.GetGitHubRateLimitFromCtx(opt.Context)
			if rateLimiter == nil {
				rateLimiter = rate.NewLimiter(rate.Every(1*time.Second), 2)
			}

			githubOpts := &github.Options{
				RateLimiter: rateLimiter,
				Client: func() *githubv4.Client {
					httpClient := oauth2.NewClient(context.Background(), oauth2.StaticTokenSource(
						&oauth2.Token{AccessToken: github.GetGitHubTokenFromCtx(opt.Context)},
					))
					client := githubv4.NewClient(httpClient)
					return client
				},
				PerPage: github.GetGitHubPerPageFromCtx(opt.Context),
			}

			if opt.GitHubClientGetter != nil {
				githubOpts.Client = opt.GitHubClientGetter
			}

			var modules = map[string]sqlite.Module{
				"github_stargazers":         github.NewStargazersModule(githubOpts),
				"github_starred_repos":      github.NewStarredReposModule(githubOpts),
				"github_user_repos":         github.NewUserReposModule(githubOpts),
				"github_org_repos":          github.NewOrgReposModule(githubOpts),
				"github_repo_issues":        github.NewIssuesModule(githubOpts),
				"github_repo_pull_requests": github.NewPRModule(githubOpts),
			}

			modules["github_issues"] = modules["github_repo_issues"]
			modules["github_pull_requests"] = modules["github_repo_pull_requests"]
			modules["github_prs"] = modules["github_repo_pull_requests"]
			modules["github_repo_prs"] = modules["github_repo_pull_requests"]

			// register GitHub tables
			for name, mod := range modules {
				if err = ext.CreateModule(name, mod); err != nil {
					return sqlite.SQLITE_ERROR, errors.Wrapf(err, "failed to register GitHub %q module", name)
				}
			}

			var fns = map[string]sqlite.Function{
				"github_stargazer_count": github.NewStarredReposFunc(githubOpts),
			}

			// register GitHub funcs
			for name, fn := range fns {
				if err = ext.CreateFunction(name, fn); err != nil {
					return sqlite.SQLITE_ERROR, errors.Wrapf(err, "failed to register GitHub %q function", name)
				}
			}
		}

		return sqlite.SQLITE_OK, nil
	}
}
