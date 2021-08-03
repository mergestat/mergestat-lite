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

type fetchUserReposOptions struct {
	Client          *githubv4.Client
	Login           string
	PerPage         int
	UserReposCursor *githubv4.String
	RepositoryOrder *githubv4.RepositoryOrder
}

type fetchUserReposResults struct {
	UserRepos   []*userRepo
	HasNextPage bool
	EndCursor   *githubv4.String
}

type userRepo struct {
	Name           string
	Description    string
	CreatedAt      time.Time
	UpdatedAt      time.Time
	PushedAt       time.Time
	StargazerCount int
	//Privacy        githubv4.RepositoryPrivacy
	//PrimaryLanguage string
}

func fetchUserRepos(ctx context.Context, input *fetchUserReposOptions) (*fetchUserReposResults, error) {
	var reposQuery struct {
		User struct {
			Login        string
			Repositories struct {
				Nodes    []*userRepo
				PageInfo struct {
					EndCursor   githubv4.String
					HasNextPage bool
				}
			} `graphql:"repositories(first: $perpage, after: $userReposCursor, orderBy: $repositoryOrder)"`
		} `graphql:"user(login: $login)"`
	}

	variables := map[string]interface{}{
		"login":           githubv4.String(input.Login),
		"perpage":         githubv4.Int(input.PerPage),
		"userReposCursor": (*githubv4.String)(input.UserReposCursor),
		"repositoryOrder": input.RepositoryOrder,
	}

	err := input.Client.Query(ctx, &reposQuery, variables)

	if err != nil {
		return nil, err
	}

	return &fetchUserReposResults{
		reposQuery.User.Repositories.Nodes,
		reposQuery.User.Repositories.PageInfo.HasNextPage,
		&reposQuery.User.Repositories.PageInfo.EndCursor,
	}, nil
}

type iterUserRepos struct {
	login       string
	client      *githubv4.Client
	current     int
	results     *fetchUserReposResults
	rateLimiter *rate.Limiter
	repoOrder   *githubv4.RepositoryOrder
}

func (i *iterUserRepos) Column(ctx *sqlite.Context, c int) error {
	switch c {
	case 0:
		ctx.ResultText(i.login)
	case 1:
		ctx.ResultText(i.results.UserRepos[i.current].Name)
	case 2:
		ctx.ResultText(i.results.UserRepos[i.current].Description)
	case 3:
		t := i.results.UserRepos[i.current].CreatedAt
		if t.IsZero() {
			ctx.ResultNull()
		} else {
			ctx.ResultText(t.Format(time.RFC3339Nano))
		}
	case 4:
		t := i.results.UserRepos[i.current].UpdatedAt
		if t.IsZero() {
			ctx.ResultNull()
		} else {
			ctx.ResultText(t.Format(time.RFC3339Nano))
		}
	case 5:
		t := i.results.UserRepos[i.current].PushedAt
		if t.IsZero() {
			ctx.ResultNull()
		} else {
			ctx.ResultText(t.Format(time.RFC3339Nano))
		}
	case 6:
		ctx.ResultInt(i.results.UserRepos[i.current].StargazerCount)
	}
	return nil
}

func (i *iterUserRepos) Next() (vtab.Row, error) {
	i.current += 1

	if i.results == nil || i.current >= len(i.results.UserRepos) {
		if i.results == nil || i.results.HasNextPage {
			err := i.rateLimiter.Wait(context.Background())
			if err != nil {
				return nil, err
			}

			var cursor *githubv4.String
			if i.results != nil {
				cursor = i.results.EndCursor
			}
			results, err := fetchUserRepos(context.Background(), &fetchUserReposOptions{i.client, i.login, 100, cursor, i.repoOrder})
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

var userReposCols = []vtab.Column{
	{Name: "login", Type: sqlite.SQLITE_TEXT, NotNull: false, Hidden: true, Filters: []*vtab.ColumnFilter{{Op: sqlite.INDEX_CONSTRAINT_EQ, Required: true, OmitCheck: true}}},
	{Name: "name", Type: sqlite.SQLITE_TEXT, NotNull: false, Hidden: false, Filters: nil, OrderBy: vtab.ASC | vtab.DESC},
	{Name: "description", Type: sqlite.SQLITE_TEXT, NotNull: false, Hidden: false, Filters: nil, OrderBy: vtab.NONE},
	{Name: "created_at", Type: sqlite.SQLITE_TEXT, NotNull: false, Hidden: false, Filters: nil, OrderBy: vtab.ASC | vtab.DESC},
	{Name: "updated_at", Type: sqlite.SQLITE_TEXT, NotNull: false, Hidden: false, Filters: nil, OrderBy: vtab.ASC | vtab.DESC},
	{Name: "pushed_at", Type: sqlite.SQLITE_TEXT, NotNull: false, Hidden: false, Filters: nil, OrderBy: vtab.ASC | vtab.DESC},
	{Name: "stargazers", Type: sqlite.SQLITE_TEXT, NotNull: false, Hidden: false, Filters: nil, OrderBy: vtab.ASC | vtab.DESC},
}

func NewUserReposModule(opts *Options) sqlite.Module {
	return vtab.NewTableFunc("github_user_repos", userReposCols, func(constraints []*vtab.Constraint, orders []*sqlite.OrderBy) (vtab.Iterator, error) {
		var login string
		for _, constraint := range constraints {
			if constraint.Op == sqlite.INDEX_CONSTRAINT_EQ {
				switch constraint.ColIndex {
				case 0:
					login = constraint.Value.Text()
				}
			}
		}

		var repoOrder *githubv4.RepositoryOrder
		// for now we can only support single field order bys
		if len(orders) == 1 {
			repoOrder = &githubv4.RepositoryOrder{}
			order := orders[0]
			switch order.ColumnIndex {
			case 1:
				repoOrder.Field = githubv4.RepositoryOrderFieldName
			case 3:
				repoOrder.Field = githubv4.RepositoryOrderFieldCreatedAt
			case 4:
				repoOrder.Field = githubv4.RepositoryOrderFieldUpdatedAt
			case 5:
				repoOrder.Field = githubv4.RepositoryOrderFieldPushedAt
			case 6:
				repoOrder.Field = githubv4.RepositoryOrderFieldStargazers
			}
			repoOrder.Direction = orderByToGitHubOrder(order.Desc)
		}

		return &iterUserRepos{login, opts.Client(), -1, nil, opts.RateLimiter, repoOrder}, nil
	})
}
