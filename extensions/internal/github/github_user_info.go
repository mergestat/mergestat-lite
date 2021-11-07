package github

import (
	"context"
	"encoding/json"

	"github.com/shurcooL/githubv4"
	"go.riyazali.net/sqlite"
)

// github_user_info
// This function takes in a user's login and returns user information in a single block of text
type userInfo struct {
	opts *Options
}

func (s *userInfo) Args() int           { return 1 }
func (s *userInfo) Deterministic() bool { return true }

func (s *userInfo) Apply(ctx *sqlite.Context, value ...sqlite.Value) {
	err := s.opts.RateLimiter.Wait(context.Background())
	if err != nil {
		ctx.ResultError(err)
		return
	}
	var login = value[0].Text()
	var query struct {
		User struct {
			Bio             string
			AvatarUrl       githubv4.URI
			Company         string
			CreatedAt       githubv4.DateTime
			Email           string
			IsHireable      bool
			IsEmployee      bool
			Name            string
			TwitterUsername string
		} `graphql:"user(login: $login)"`
	}
	variables := map[string]interface{}{
		"login": githubv4.String(login),
	}

	l := s.opts.Logger.With().Str("login", login).Logger()
	l.Info().Msgf("fetching user information for: %s", login)

	err = s.opts.Client().Query(context.Background(), &query, variables)
	if err != nil {
		ctx.ResultError(err)
		return
	}
	resultString, err := json.Marshal(query)
	if err != nil {
		ctx.ResultError(err)
		return
	}
	ctx.ResultText(string(resultString))
}
func NewGithubUserFunc(opts *Options) sqlite.Function {
	return &userInfo{opts}
}
