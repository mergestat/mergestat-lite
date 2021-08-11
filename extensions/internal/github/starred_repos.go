package github

import (
	"context"
	"io"
	"time"

	"github.com/augmentable-dev/vtab"
	"github.com/shurcooL/githubv4"
	"go.riyazali.net/sqlite"
	"golang.org/x/time/rate"
)

type fetchStarredReposOptions struct {
	Client      *githubv4.Client
	Login       string
	PerPage     int
	StartCursor *githubv4.String
	Order       *githubv4.StarOrder
}

type fetchStarredReposResults struct {
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

func fetchStarredRepos(ctx context.Context, input *fetchStarredReposOptions) (*fetchStarredReposResults, error) {
	var reposQuery struct {
		User struct {
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
		"perpage":     githubv4.Int(input.PerPage),
		"startcursor": input.StartCursor,
		"login":       githubv4.String(input.Login),
		"orderBy":     input.Order,
	}

	err := input.Client.Query(ctx, &reposQuery, variables)
	if err != nil {
		return nil, err
	}

	return &fetchStarredReposResults{
		reposQuery.User.StarredRepositories.Edges,
		reposQuery.User.StarredRepositories.PageInfo.HasNextPage,
		&reposQuery.User.StarredRepositories.PageInfo.EndCursor,
	}, nil

}

type iterStarredRepos struct {
	login       string
	client      *githubv4.Client
	current     int
	results     *fetchStarredReposResults
	rateLimiter *rate.Limiter
	starOrder   *githubv4.StarOrder
	perPage     int
}

func (i *iterStarredRepos) Column(ctx *sqlite.Context, c int) error {
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
			err := i.rateLimiter.Wait(context.Background())
			if err != nil {
				return nil, err
			}

			var cursor *githubv4.String
			if i.results != nil {
				cursor = i.results.EndCursor
			}
			results, err := fetchStarredRepos(context.Background(), &fetchStarredReposOptions{i.client, i.login, i.perPage, cursor, i.starOrder})
			if err != nil {
				return nil, err
			}

			i.results = results
			i.current = 0

		} else {
			return nil, io.EOF
		}
	}

	return i, nil
}

var starredReposCols = []vtab.Column{
	{Name: "login", Type: sqlite.SQLITE_TEXT, NotNull: false, Hidden: true, Filters: []*vtab.ColumnFilter{{Op: sqlite.INDEX_CONSTRAINT_EQ, Required: true, OmitCheck: true}}},
	{Name: "name", Type: sqlite.SQLITE_TEXT},
	{Name: "url", Type: sqlite.SQLITE_TEXT},
	{Name: "description", Type: sqlite.SQLITE_TEXT},
	{Name: "created_at", Type: sqlite.SQLITE_TEXT},
	{Name: "pushed_at", Type: sqlite.SQLITE_TEXT},
	{Name: "updated_at", Type: sqlite.SQLITE_TEXT},
	{Name: "stargazer_count", Type: sqlite.SQLITE_INTEGER},
	{Name: "name_with_owner", Type: sqlite.SQLITE_TEXT},
	{Name: "starred_at", Type: sqlite.SQLITE_TEXT, OrderBy: vtab.ASC | vtab.DESC},
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

		return &iterStarredRepos{login, opts.Client(), -1, nil, opts.RateLimiter, starOrder, opts.PerPage}, nil
	})
}
