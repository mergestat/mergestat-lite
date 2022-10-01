package sourcegraph

import (
	"github.com/mergestat/mergestat-lite/extensions/services"
	"github.com/rs/zerolog"
	"github.com/shurcooL/graphql"
	"golang.org/x/time/rate"
)

type Options struct {
	Client      func() *graphql.Client
	RateLimiter *rate.Limiter
	PerPage     int
	Logger      *zerolog.Logger
}

// GetSourcegraphTokenFromCtx looks up the sourcegraphToken key in the supplied context and returns it if set
func GetSourcegraphTokenFromCtx(ctx services.Context) string {
	return ctx["sourcegraphToken"]
}
