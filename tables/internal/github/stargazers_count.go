package github

import (
	"context"
	"errors"
	"strings"

	"github.com/shurcooL/githubv4"
	"go.riyazali.net/sqlite"
	"golang.org/x/oauth2"
	"golang.org/x/time/rate"
)

type starCount struct { //change to starCount
	rateLimiter *rate.Limiter
	client      *githubv4.Client
}

func (s *starCount) Args() int           { return -1 }
func (s *starCount) Deterministic() bool { return false }
func (s *starCount) Apply(ctx *sqlite.Context, values ...sqlite.Value) {
	err := s.rateLimiter.Wait(context.Background())
	if err != nil {
		ctx.ResultError(err)
		return
	}

	var starsCountQuery struct {
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

	variables := map[string]interface{}{
		"owner": githubv4.String(owner),
		"name":  githubv4.String(name),
	}
	err = s.client.Query(context.Background(), &starsCountQuery, variables)
	if err != nil {
		ctx.ResultError(err)
		return
	}
	ctx.ResultInt(starsCountQuery.Repository.StargazerCount)
}

func NewStarredReposFunc(githubToken string, rateLimiter *rate.Limiter) sqlite.Function {
	httpClient := oauth2.NewClient(context.Background(), oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: githubToken},
	))
	client := githubv4.NewClient(httpClient)
	return &starCount{rateLimiter: rateLimiter, client: client}
}
