package github

import (
	"context"
	"io"
	"time"

	"github.com/augmentable-dev/vtab"
	"github.com/mergestat/mergestat/extensions/options"
	"github.com/rs/zerolog"
	"github.com/shurcooL/githubv4"
	"go.riyazali.net/sqlite"
)

type fetchStarredReposResults struct {
	RateLimit   *options.GitHubRateLimitResponse
	Edges       []*starredRepoEdge
	HasNextPage bool
	EndCursor   *githubv4.String
}

type starredRepoEdge struct {
	StarredAt string
	Node      *starredRepoNode
}

type starredRepoNode struct {
	Name           string
	Url            string
	Description    string
	CreatedAt      time.Time
	PushedAt       time.Time
	UpdatedAt      time.Time
	StargazerCount int
	NameWithOwner  string
}

func (i *iterStarredRepos) fetchStarredRepos(ctx context.Context, startCursor *githubv4.String) (*fetchStarredReposResults, error) {
	var reposQuery struct {
		RateLimit *options.GitHubRateLimitResponse
		User      struct {
			Login               string
			StarredRepositories struct {
				Edges    []*starredRepoEdge
				PageInfo struct {
					EndCursor   githubv4.String
					HasNextPage bool
				}
			} `graphql:"starredRepositories(first: $perpage, after: $startcursor, orderBy: $orderBy)"`
		} `graphql:"user(login: $login)"`
	}

	variables := map[string]interface{}{
		"perpage":     githubv4.Int(i.PerPage),
		"startcursor": startCursor,
		"login":       githubv4.String(i.login),
		"orderBy":     i.starOrder,
	}

	err := i.Client().Query(ctx, &reposQuery, variables)
	if err != nil {
		return nil, err
	}

	return &fetchStarredReposResults{
		RateLimit:   reposQuery.RateLimit,
		Edges:       reposQuery.User.StarredRepositories.Edges,
		HasNextPage: reposQuery.User.StarredRepositories.PageInfo.HasNextPage,
		EndCursor:   &reposQuery.User.StarredRepositories.PageInfo.EndCursor,
	}, nil

}

type iterStarredRepos struct {
	*Options
	login     string
	current   int
	results   *fetchStarredReposResults
	starOrder *githubv4.StarOrder
}

func (i *iterStarredRepos) logger() *zerolog.Logger {
	logger := i.Logger.With().Int("per-page", i.PerPage).Str("login", i.login).Logger()
	if i.starOrder != nil {
		logger = logger.With().Str("order_by", string(i.starOrder.Field)).Str("order_dir", string(i.starOrder.Direction)).Logger()
	}
	return &logger
}

func (i *iterStarredRepos) Column(ctx vtab.Context, c int) error {
	current := i.results.Edges[i.current]
	switch starredReposCols[c].Name {
	case "login":
		ctx.ResultText(i.login)
	case "name":
		ctx.ResultText(current.Node.Name)
	case "url":
		ctx.ResultText(current.Node.Url)
	case "description":
		ctx.ResultText(current.Node.Description)
	case "created_at":
		t := current.Node.CreatedAt
		if t.IsZero() {
			ctx.ResultNull()
		} else {
			ctx.ResultText(t.Format(time.RFC3339Nano))
		}
	case "pushed_at":
		t := current.Node.PushedAt
		if t.IsZero() {
			ctx.ResultNull()
		} else {
			ctx.ResultText(t.Format(time.RFC3339Nano))
		}
	case "updated_at":
		t := current.Node.UpdatedAt
		if t.IsZero() {
			ctx.ResultNull()
		} else {
			ctx.ResultText(t.Format(time.RFC3339Nano))
		}
	case "stargazer_count":
		ctx.ResultInt(current.Node.StargazerCount)
	case "name_with_owner":
		ctx.ResultText(current.Node.NameWithOwner)
	case "starred_at":
		ctx.ResultText(current.StarredAt)
	}
	return nil
}

func (i *iterStarredRepos) Next() (vtab.Row, error) {
	i.current += 1

	if i.results == nil || i.current >= len(i.results.Edges) {
		if i.results == nil || i.results.HasNextPage {
			err := i.RateLimiter.Wait(context.Background())
			if err != nil {
				return nil, err
			}

			var cursor *githubv4.String
			if i.results != nil {
				cursor = i.results.EndCursor
			}

			i.Options.GitHubPreRequestHook()

			l := i.logger().With().Interface("cursor", cursor).Logger()
			l.Info().Msgf("fetching page of starred_repos for %s", i.login)
			results, err := i.fetchStarredRepos(context.Background(), cursor)
			if err != nil {
				return nil, err
			}

			i.Options.RateLimitHandler(results.RateLimit)

			i.results = results
			i.current = 0

			if len(i.results.Edges) == 0 {
				return nil, io.EOF
			}
		} else {
			return nil, io.EOF
		}
	}

	return i, nil
}

var starredReposCols = []vtab.Column{
	{Name: "login", Type: "TEXT", NotNull: false, Hidden: true, Filters: []*vtab.ColumnFilter{{Op: sqlite.INDEX_CONSTRAINT_EQ, OmitCheck: true}}},
	{Name: "name", Type: "TEXT"},
	{Name: "url", Type: "TEXT"},
	{Name: "description", Type: "TEXT"},
	{Name: "created_at", Type: "DATETIME"},
	{Name: "pushed_at", Type: "DATETIME"},
	{Name: "updated_at", Type: "DATETIME"},
	{Name: "stargazer_count", Type: "INT"},
	{Name: "name_with_owner", Type: "TEXT"},
	{Name: "starred_at", Type: "DATETIME", OrderBy: vtab.ASC | vtab.DESC},
}

func NewStarredReposModule(opts *Options) sqlite.Module {
	return vtab.NewTableFunc("github_starred_repos", starredReposCols, func(constraints []*vtab.Constraint, orders []*sqlite.OrderBy) (vtab.Iterator, error) {
		var login string
		for _, constraint := range constraints {
			if constraint.Op == sqlite.INDEX_CONSTRAINT_EQ {
				switch constraint.ColIndex {
				case 0:
					login = constraint.Value.Text()
				}
			}
		}

		var starOrder *githubv4.StarOrder
		// for now we can only support single field order bys
		if len(orders) == 1 {
			starOrder = &githubv4.StarOrder{}
			order := orders[0]
			switch starredReposCols[order.ColumnIndex].Name {
			case "starred_at":
				starOrder.Field = githubv4.StarOrderFieldStarredAt
			}
			starOrder.Direction = orderByToGitHubOrder(order.Desc)
		}

		iter := &iterStarredRepos{opts, login, -1, nil, starOrder}
		iter.logger().Info().Msgf("starting GitHub starred_repos iterator for %s", login)
		return iter, nil
	}, vtab.EarlyOrderByConstraintExit(true))
}
