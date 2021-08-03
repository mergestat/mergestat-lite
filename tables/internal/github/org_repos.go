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

type fetchOrgReposOptions struct {
	Client          *githubv4.Client
	Login           string
	PerPage         int
	OrgReposCursor  *githubv4.String
	RepositoryOrder *githubv4.RepositoryOrder
}

type fetchOrgReposResults struct {
	OrgRepos    []*orgRepo
	HasNextPage bool
	EndCursor   *githubv4.String
}

type orgRepo struct {
	Name           string
	Description    string
	CreatedAt      time.Time
	UpdatedAt      time.Time
	PushedAt       time.Time
	StargazerCount int
	//Privacy        githubv4.RepositoryPrivacy
	//PrimaryLanguage string
}

func fetchOrgRepos(ctx context.Context, input *fetchOrgReposOptions) (*fetchOrgReposResults, error) {
	var reposQuery struct {
		Organization struct {
			Login        string
			Repositories struct {
				Nodes    []*orgRepo
				PageInfo struct {
					EndCursor   githubv4.String
					HasNextPage bool
				}
			} `graphql:"repositories(first: $perpage, after: $orgReposCursor, orderBy: $repositoryOrder)"`
		} `graphql:"organization(login: $login)"`
	}

	variables := map[string]interface{}{
		"login":           githubv4.String(input.Login),
		"perpage":         githubv4.Int(input.PerPage),
		"orgReposCursor":  (*githubv4.String)(input.OrgReposCursor),
		"repositoryOrder": input.RepositoryOrder,
	}

	err := input.Client.Query(ctx, &reposQuery, variables)

	if err != nil {
		return nil, err
	}

	return &fetchOrgReposResults{
		reposQuery.Organization.Repositories.Nodes,
		reposQuery.Organization.Repositories.PageInfo.HasNextPage,
		&reposQuery.Organization.Repositories.PageInfo.EndCursor,
	}, nil
}

type iterOrgRepos struct {
	login       string
	client      *githubv4.Client
	current     int
	results     *fetchOrgReposResults
	rateLimiter *rate.Limiter
	repoOrder   *githubv4.RepositoryOrder
}

func (i *iterOrgRepos) Column(ctx *sqlite.Context, c int) error {

	switch c {
	case 0:
		ctx.ResultText(i.login)
	case 1:
		ctx.ResultText(i.results.OrgRepos[i.current].Name)
	case 2:
		ctx.ResultText(i.results.OrgRepos[i.current].Description)
	case 3:
		t := i.results.OrgRepos[i.current].CreatedAt
		if t.IsZero() {
			ctx.ResultText(" ")
		} else {
			ctx.ResultText(t.Format(time.RFC3339Nano))
		}
	case 4:
		t := i.results.OrgRepos[i.current].UpdatedAt
		if t.IsZero() {
			ctx.ResultText(" ")
		} else {
			ctx.ResultText(t.Format(time.RFC3339Nano))
		}
	case 5:
		t := i.results.OrgRepos[i.current].PushedAt
		if t.IsZero() {
			ctx.ResultText(" ")
		} else {
			ctx.ResultText(t.Format(time.RFC3339Nano))
		}
	case 6:
		ctx.ResultInt(i.results.OrgRepos[i.current].StargazerCount)

	}
	return nil
}

func (i *iterOrgRepos) Next() (vtab.Row, error) {
	i.current += 1

	if i.results == nil || i.current >= len(i.results.OrgRepos) {
		if i.results == nil || i.results.HasNextPage {
			err := i.rateLimiter.Wait(context.Background())
			if err != nil {
				return nil, err
			}

			var cursor *githubv4.String
			if i.results != nil {
				cursor = i.results.EndCursor
			}
			results, err := fetchOrgRepos(context.Background(), &fetchOrgReposOptions{i.client, i.login, 100, cursor, i.repoOrder})
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

var orgReposCols = []vtab.Column{
	{Name: "login", Type: sqlite.SQLITE_TEXT, NotNull: false, Hidden: true, Filters: []*vtab.ColumnFilter{{Op: sqlite.INDEX_CONSTRAINT_EQ, Required: true, OmitCheck: true}}, OrderBy: vtab.NONE},
	{Name: "name", Type: sqlite.SQLITE_TEXT, NotNull: false, Hidden: false, Filters: nil, OrderBy: vtab.ASC | vtab.DESC},
	{Name: "description", Type: sqlite.SQLITE_TEXT, NotNull: false, Hidden: false, Filters: nil, OrderBy: vtab.NONE},
	{Name: "created_at", Type: sqlite.SQLITE_TEXT, NotNull: false, Hidden: false, Filters: nil, OrderBy: vtab.ASC | vtab.DESC},
	{Name: "updated_at", Type: sqlite.SQLITE_TEXT, NotNull: false, Hidden: false, Filters: nil, OrderBy: vtab.ASC | vtab.DESC},
	{Name: "pushed_at", Type: sqlite.SQLITE_TEXT, NotNull: false, Hidden: false, Filters: nil, OrderBy: vtab.ASC | vtab.DESC},
	{Name: "stargazers", Type: sqlite.SQLITE_TEXT, NotNull: false, Hidden: false, Filters: nil, OrderBy: vtab.ASC | vtab.DESC},
}

func NewOrgReposModule(opts *Options) sqlite.Module {
	return vtab.NewTableFunc("github_org_repos", orgReposCols, func(constraints []*vtab.Constraint, orders []*sqlite.OrderBy) (vtab.Iterator, error) {
		var login string
		for _, constraint := range constraints {
			if constraint.Op == sqlite.INDEX_CONSTRAINT_EQ {
				switch constraint.ColIndex {
				case 0:
					login = constraint.Value.Text()
				}
			}
		}

		repoOrder := &githubv4.RepositoryOrder{
			Field:     githubv4.RepositoryOrderFieldCreatedAt, //Turns out leaving this struct blank does not assign default values to its fields. So I assigned the default values that are followed in graphQL
			Direction: githubv4.OrderDirectionAsc,
		}

		if len(orders) == 1 {
			for _, order := range orders {
				if order.Desc {
					repoOrder.Direction = githubv4.OrderDirectionDesc
				}
				switch order.ColumnIndex {
				case 1:
					repoOrder.Field = githubv4.RepositoryOrderFieldName
				case 2:
					repoOrder.Field = githubv4.RepositoryOrderFieldCreatedAt
				case 3:
					repoOrder.Field = githubv4.RepositoryOrderFieldUpdatedAt
				case 4:
					repoOrder.Field = githubv4.RepositoryOrderFieldPushedAt
				case 5:
					repoOrder.Field = githubv4.RepositoryOrderFieldStargazers

				}
			}
		}

		return &iterOrgRepos{login, opts.Client(), -1, nil, opts.RateLimiter, repoOrder}, nil
	})
}
