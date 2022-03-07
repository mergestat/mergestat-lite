package github

import (
	"context"
	"encoding/json"

	"github.com/shurcooL/githubv4"
	"go.riyazali.net/sqlite"
)

// github_user_info
// This function takes in a user's login and returns user information as JSON
type userInfo struct {
	opts *Options
}

func (s *userInfo) Args() int           { return 1 }
func (s *userInfo) Deterministic() bool { return false }

func (s *userInfo) Apply(ctx *sqlite.Context, value ...sqlite.Value) {
	err := s.opts.RateLimiter.Wait(context.Background())
	if err != nil {
		ctx.ResultError(err)
		return
	}
	login := value[0].Text()
	var query struct {
		User struct {
			Bio             string            `json:"bio"`
			AvatarUrl       githubv4.URI      `json:"avatarUrl"`
			Company         string            `json:"company"`
			CreatedAt       githubv4.DateTime `json:"createdAt"`
			Email           string            `json:"email"`
			IsHireable      bool              `json:"isHireable"`
			IsEmployee      bool              `json:"isEmployee"`
			Name            string            `json:"name"`
			TwitterUsername string            `json:"twitterUsername"`
		} `graphql:"user(login: $login)" json:"user"`
	}

	variables := map[string]interface{}{
		"login": githubv4.String(login),
	}

	l := s.opts.Logger.With().Str("login", login).Logger()
	l.Info().Msgf("fetching user information for: %s", login)

	err = s.opts.Client().Query(context.TODO(), &query, variables)
	if err != nil {
		ctx.ResultError(err)
		return
	}

	resultString, err := json.Marshal(query.User)
	if err != nil {
		ctx.ResultError(err)
		return
	}

	ctx.ResultText(string(resultString))
}

func NewGitHubUserFunc(opts *Options) sqlite.Function {
	return &userInfo{opts}
}
