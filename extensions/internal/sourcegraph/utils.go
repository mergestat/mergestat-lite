package sourcegraph

import (
	"github.com/askgitdev/askgit/extensions/services"
	"github.com/shurcooL/graphql"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

type Options struct {
	Client      func() *graphql.Client
	RateLimiter *rate.Limiter
	PerPage     int
	Logger      *zap.Logger
}

// GetSourcegraphTokenFromCtx looks up the sourcegraphToken key in the supplied context and returns it if set
func GetSourcegraphTokenFromCtx(ctx services.Context) string {
	return ctx["sourcegraphToken"]
}
