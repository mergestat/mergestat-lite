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

type fetchReposOptions struct {
	Client      *githubv4.Client
	Login       string
	PerPage     int
	StartCursor *githubv4.String
}

type fetchReposResults struct {
	StarredRepos []*starredRepo
	HasNextPage  bool
	EndCursor    *githubv4.String
}

//TODO: Add more fields
//TODO: Resolve error from StargazerCount and ProjectsUrl

type starredRepo struct {
	Name           string
	Url            string
	Description    string
	CreatedAt      time.Time
	PushedAt       time.Time
	UpdatedAt      time.Time
	StargazerCount int
	NameWithOwner  string
	// ProjectsUrl    *githubv4.URI
}

func fetchRepos(ctx context.Context, input *fetchReposOptions) (*fetchReposResults, error) {
	var reposQuery struct {
		User struct {
			Login               string
			StarredRepositories struct {
				Nodes    []*starredRepo
				PageInfo struct {
					EndCursor   githubv4.String
					HasNextPage bool
				}
			} `graphql:"starredRepositories(first: $perpage, after: $startcursor)"`
		} `graphql:"user(login: $login)"`
	}

	variables := map[string]interface{}{
		"perpage":     githubv4.Int(input.PerPage),
		"startcursor": input.StartCursor,
		"login":       githubv4.String(input.Login),
	}

	err := input.Client.Query(ctx, &reposQuery, variables)

	if err != nil {
		return nil, err
	}

	return &fetchReposResults{
		reposQuery.User.StarredRepositories.Nodes,
		reposQuery.User.StarredRepositories.PageInfo.HasNextPage,
		&reposQuery.User.StarredRepositories.PageInfo.EndCursor,
	}, nil

}

type iterStarredRepos struct {
	login       string
	client      *githubv4.Client
	current     int
	results     *fetchReposResults
	rateLimiter *rate.Limiter
}

func (i *iterStarredRepos) Column(ctx *sqlite.Context, c int) error {

	switch c {
	case 0:
		ctx.ResultText(i.login)
	case 1:
		ctx.ResultText(i.results.StarredRepos[i.current].Name)
	case 2:
		ctx.ResultText(i.results.StarredRepos[i.current].Url)
	case 3:
		ctx.ResultText(i.results.StarredRepos[i.current].Description)
	case 4:
		t := i.results.StarredRepos[i.current].CreatedAt
		if t.IsZero() {
			ctx.ResultText(" ")
		} else {
			ctx.ResultText(t.Format(time.RFC3339Nano))
		}
	case 5:
		t := i.results.StarredRepos[i.current].PushedAt
		if t.IsZero() {
			ctx.ResultText(" ")
		} else {
			ctx.ResultText(t.Format(time.RFC3339Nano))
		}
	case 6:
		t := i.results.StarredRepos[i.current].UpdatedAt
		if t.IsZero() {
			ctx.ResultText(" ")
		} else {
			ctx.ResultText(t.Format(time.RFC3339Nano))
		}
	case 7:
		ctx.ResultInt(i.results.StarredRepos[i.current].StargazerCount)
	case 8:
		ctx.ResultText(i.results.StarredRepos[i.current].NameWithOwner)
		// 	// case 7:
		// 	return i.results.StarredRepos[i.current].ProjectsUrl, nil
	}
	return nil
}

func (i *iterStarredRepos) Next() (vtab.Row, error) {
	i.current += 1

	if i.results == nil || i.current >= len(i.results.StarredRepos) {
		if i.results == nil || i.results.HasNextPage {
			err := i.rateLimiter.Wait(context.Background())
			if err != nil {
				return nil, err
			}

			var cursor *githubv4.String
			if i.results != nil {
				cursor = i.results.EndCursor
			}
			results, err := fetchRepos(context.Background(), &fetchReposOptions{i.client, i.login, 100, cursor})
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
	{Name: "login", Type: sqlite.SQLITE_TEXT, NotNull: false, Hidden: true, Filters: []*vtab.ColumnFilter{{Op: sqlite.INDEX_CONSTRAINT_EQ, Required: true, OmitCheck: true}}, OrderBy: vtab.NONE},
	{Name: "name", Type: sqlite.SQLITE_TEXT, NotNull: false, Hidden: false, Filters: nil, OrderBy: vtab.NONE},
	{Name: "url", Type: sqlite.SQLITE_TEXT, NotNull: false, Hidden: false, Filters: nil, OrderBy: vtab.NONE},
	{Name: "description", Type: sqlite.SQLITE_TEXT, NotNull: false, Hidden: false, Filters: nil, OrderBy: vtab.NONE},
	{Name: "created_at", Type: sqlite.SQLITE_TEXT, NotNull: false, Hidden: false, Filters: nil, OrderBy: vtab.NONE},
	{Name: "pushed_at", Type: sqlite.SQLITE_TEXT, NotNull: false, Hidden: false, Filters: nil, OrderBy: vtab.NONE},
	{Name: "updated_at", Type: sqlite.SQLITE_TEXT, NotNull: false, Hidden: false, Filters: nil, OrderBy: vtab.NONE},
	{Name: "stargazer_count", Type: sqlite.SQLITE_INTEGER, NotNull: true, Hidden: false, Filters: nil, OrderBy: vtab.NONE},
	{Name: "name_with_owner", Type: sqlite.SQLITE_TEXT, NotNull: false, Hidden: false, Filters: nil, OrderBy: vtab.NONE},
	//{Name: "projects_url", Type: sqlite.SQLITE_TEXT, NotNull: false, Hidden: false, Filters: nil, OrderBy: vtab.NONE},
}

func NewStarredReposModule(opts *Options) sqlite.Module {
	return vtab.NewTableFunc("github_starred_repos", starredReposCols, func(constraints []*vtab.Constraint, order []*sqlite.OrderBy) (vtab.Iterator, error) {
		var login string
		for _, constraint := range constraints {
			if constraint.Op == sqlite.INDEX_CONSTRAINT_EQ {
				switch constraint.ColIndex {
				case 0:
					login = constraint.Value.Text()
				}
			}
		}

		return &iterStarredRepos{login, opts.Client(), -1, nil, opts.RateLimiter}, nil
	})
}
