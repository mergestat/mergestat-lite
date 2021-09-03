package github

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/askgitdev/askgit/extensions/services"
	"github.com/shurcooL/githubv4"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

type Options struct {
	Client      func() *githubv4.Client
	RateLimiter *rate.Limiter
	// PerPage is the default number of items per page to use when making a paginated GitHub API request
	PerPage int
	Logger  *zap.Logger
}

// GetGitHubTokenFromCtx looks up the githubToken key in the supplied context and returns it if set
func GetGitHubTokenFromCtx(ctx services.Context) string {
	return ctx["githubToken"]
}

// GetGitHubRateLimitFromCtx looks up the githubRateLimit key in the supplied context and parses it to return a client
// side rate limit in the form "(number of reqs)/(number of seconds)". For instance a string "2/3" would yield a rate limiter
// that permis 2 requests every 3 seconds. A single integer is also permitted, which assumes the "denominator" is 1 second.
// So a value of "5" would simple mean 5 requests per second.
// If the string cannot be parsed, nil is returned.
func GetGitHubRateLimitFromCtx(ctx services.Context) *rate.Limiter {
	if val, ok := ctx["githubRateLimit"]; ok {
		if strings.Contains(val, "/") {
			parts := strings.SplitN(val, "/", 2)
			if len(parts) != 2 {
				return nil
			}

			var first, second int
			var err error
			if first, err = strconv.Atoi(parts[0]); err != nil {
				return nil
			}
			if second, err = strconv.Atoi(parts[1]); err != nil {
				return nil
			}

			return rate.NewLimiter(
				rate.Every(time.Second*time.Duration(second)),
				first,
			)
		} else {
			if perSec, ok := ctx.GetInt("githubRateLimit"); ok {
				return rate.NewLimiter(rate.Every(time.Second), perSec)
			} else {
				return nil
			}
		}
	} else {
		return nil
	}
}

// GetGitHubPerPageFromCtx looks up the githubPerPage key in the supplied context and returns it if set,
// otherwise it returns a default of 100
func GetGitHubPerPageFromCtx(ctx services.Context) int {
	if val, ok := ctx.GetInt("githubPerPage"); ok && val != 0 {
		return val
	} else {
		return 100
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
