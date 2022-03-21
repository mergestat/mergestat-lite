package github

import (
	"context"
	"errors"
	"strings"

	"github.com/shurcooL/githubv4"
	"go.riyazali.net/sqlite"
)

type starCount struct {
	opts *Options
}

func (s *starCount) Args() int           { return -1 }
func (s *starCount) Deterministic() bool { return false }
func (s *starCount) Apply(ctx *sqlite.Context, values ...sqlite.Value) {
	err := s.opts.RateLimiter.Wait(context.Background())
	if err != nil {
		ctx.ResultError(err)
		return
	}

	var starsCountQuery struct {
		RateLimit  *RateLimitResponse
		Repository struct {
			StargazerCount int
		} `graphql:"repository(owner: $owner, name: $name)"`
	}

	var owner, name string
	switch len(values) {
	case 0:
		ctx.ResultError(errors.New("need to supply a repo"))
		return
	case 1:
		split_string := strings.Split(values[0].Text(), "/")
		if len(split_string) != 2 {
			ctx.ResultError(errors.New("invalid repo name, must be of format owner/name"))
			return
		}
		owner = split_string[0]
		name = split_string[1]
	default:
		owner = values[0].Text()
		name = values[1].Text()
	}

	l := s.opts.Logger.With().Str("owner", owner).Str("name", name).Logger()
	l.Info().Msgf("fetching number of GitHub stargazers for: %s/%s", owner, name)

	variables := map[string]interface{}{
		"owner": githubv4.String(owner),
		"name":  githubv4.String(name),
	}
	err = s.opts.Client().Query(context.Background(), &starsCountQuery, variables)
	if err != nil {
		ctx.ResultError(err)
		return
	}

	s.opts.RateLimitHandler(starsCountQuery.RateLimit)

	ctx.ResultInt(starsCountQuery.Repository.StargazerCount)
}

func NewStarredReposFunc(opts *Options) sqlite.Function {
	return &starCount{opts}
}
