package github

import (
	"errors"
	"strconv"
	"strings"

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

// orderByToGitHubOrder is a helper that takes a boolean indicating whether DESC or ASC and returns
// a corresponding OrderDirection from the githubv4 library
func orderByToGitHubOrder(desc bool) githubv4.OrderDirection {
	if desc {
		return githubv4.OrderDirectionDesc
	} else {
		return githubv4.OrderDirectionAsc
	}
}

// repoOwnerAndName returns the "owner" and "name" (respective return values) or an error
// given the inputs to the iterator. This allows for both `SELECT * FROM github_table('askgitdev/askgit')`
// and `SELECT * FROM github_table('askgitdev', 'askgit')
func repoOwnerAndName(name, fullNameOrOwner string) (string, string, error) {
	if name == "" {
		split_string := strings.Split(fullNameOrOwner, "/")
		if len(split_string) != 2 {
			return "", "", errors.New("invalid repo name, must be of format owner/name")
		}
		return split_string[0], split_string[1], nil
	} else {
		return fullNameOrOwner, name, nil
	}
}
