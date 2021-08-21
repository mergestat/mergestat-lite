package sourcegraph

import (
	"github.com/askgitdev/askgit/extensions/services"
	"github.com/shurcooL/graphql"
	"golang.org/x/time/rate"
)

type Options struct {
	Client      func() *graphql.Client
	RateLimiter *rate.Limiter
	// PerPage is the default number of items per page to use when making a paginated GitHub API request
	PerPage int
}

// GetSourcegraphTokenFromCtx looks up the sourcegraphToken key in the supplied context and returns it if set
func GetSourcegraphTokenFromCtx(ctx services.Context) string {
	return ctx["sourcegraphToken"]
}
