package github

import (
	"context"
	"time"

	"github.com/askgitdev/askgit/extensions/options"
	"github.com/pkg/errors"
	"github.com/shurcooL/githubv4"
	"go.riyazali.net/sqlite"
	"golang.org/x/oauth2"
	"golang.org/x/time/rate"
)

// Register registers GitHub related functionality as a SQLite extension
func Register(ext *sqlite.ExtensionApi, opt *options.Options) (_ sqlite.ErrorCode, err error) {
	rateLimiter := GetGitHubRateLimitFromCtx(opt.Context)
	if rateLimiter == nil {
		rateLimiter = rate.NewLimiter(rate.Every(1*time.Second), 2)
	}

	githubOpts := &Options{
		RateLimiter: rateLimiter,
		Client: func() *githubv4.Client {
			httpClient := oauth2.NewClient(context.Background(), oauth2.StaticTokenSource(
				&oauth2.Token{AccessToken: GetGitHubTokenFromCtx(opt.Context)},
			))
			client := githubv4.NewClient(httpClient)
			return client
		},
		PerPage: GetGitHubPerPageFromCtx(opt.Context),
	}

	if opt.GitHubClientGetter != nil {
		githubOpts.Client = opt.GitHubClientGetter
	}

	var modules = map[string]sqlite.Module{
		"github_stargazers":         NewStargazersModule(githubOpts),
		"github_starred_repos":      NewStarredReposModule(githubOpts),
		"github_user_repos":         NewUserReposModule(githubOpts),
		"github_org_repos":          NewOrgReposModule(githubOpts),
		"github_repo_issues":        NewIssuesModule(githubOpts),
		"github_repo_pull_requests": NewPRModule(githubOpts),
		"github_repo_check_suites":  NewCheckSuiteModule(githubOpts),
	}

	modules["github_issues"] = modules["github_repo_issues"]
	modules["github_pull_requests"] = modules["github_repo_pull_requests"]
	modules["github_prs"] = modules["github_repo_pull_requests"]
	modules["github_repo_prs"] = modules["github_repo_pull_requests"]
	modules["github_check_suites"] = modules["github_repo_check_suites"]

	// register GitHub tables
	for name, mod := range modules {
		if err = ext.CreateModule(name, mod); err != nil {
			println(err.Error())
			return sqlite.SQLITE_ERROR, errors.Wrapf(err, "failed to register GitHub %q module", name)
		}
	}

	var fns = map[string]sqlite.Function{
		"github_stargazer_count":   NewStarredReposFunc(githubOpts),
		"github_repo_file_content": NewRepoFileContentFunc(githubOpts),
	}

	// register GitHub funcs
	for name, fn := range fns {
		if err = ext.CreateFunction(name, fn); err != nil {
			return sqlite.SQLITE_ERROR, errors.Wrapf(err, "failed to register GitHub %q function", name)
		}
	}
	return sqlite.SQLITE_OK, nil
}
