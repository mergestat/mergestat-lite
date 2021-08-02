package github

import (
	"strconv"

	"github.com/askgitdev/askgit/tables/services"
	"github.com/shurcooL/githubv4"
	"golang.org/x/time/rate"
)

type Options struct {
	Client      func() *githubv4.Client
	RateLimiter *rate.Limiter
}

// GetGitHubTokenFromCtx looks up the githubToken key in the supplied context and returns it if set
func GetGitHubTokenFromCtx(ctx services.Context) string {
	return ctx["githubToken"]
}

// GetGithubReqPerSecondFromCtx looks up the githubReqPerSec key in the supplied context and returns it if set,
// otherwise it returns a default of 1
func GetGithubReqPerSecondFromCtx(ctx services.Context) int {
	defaultValue := 1
	if githubReqPerSec, ok := ctx["githubReqPerSec"]; ok && githubReqPerSec != "" {
		if i, err := strconv.Atoi(githubReqPerSec); err != nil {
			return i
		} else {
			return defaultValue
		}
	} else {
		return defaultValue
	}
}

// t1f0 converts a bool to an int
func t1f0(b bool) int {
	if b {
		return 1
	}
	return 0
}
