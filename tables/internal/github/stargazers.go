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

type stargazer struct {
	Login           string
	Email           string
	Name            string
	Bio             string
	Company         string
	IsHireable      bool
	AvatarUrl       string
	CreatedAt       time.Time
	UpdatedAt       time.Time
	TwitterUsername string
	WebsiteUrl      string
	Location        string
}

type fetchStarsOptions struct {
	Client      *githubv4.Client
	Owner       string
	Name        string
	PerPage     int
	StartCursor *githubv4.String
	StarOrder   *githubv4.StarOrder
}

type fetchStarsResults struct {
	Edges       []*stargazerEdge
	HasNextPage bool
	EndCursor   *githubv4.String
}

type stargazerEdge struct {
	StarredAt string
	Node      stargazer
}

func fetchStars(ctx context.Context, input *fetchStarsOptions) (*fetchStarsResults, error) {
	var starsQuery struct {
		Repository struct {
			Owner struct {
				Login string
			}
			Name       string
			Stargazers struct {
				Edges    []*stargazerEdge
				PageInfo struct {
					EndCursor   githubv4.String
					HasNextPage bool
				}
			} `graphql:"stargazers(first: $perpage, after: $stargazersCursor, orderBy: $starorder)"`
		} `graphql:"repository(owner: $owner, name: $name)"`
	}
	variables := map[string]interface{}{
		"owner":            githubv4.String(input.Owner),
		"name":             githubv4.String(input.Name),
		"perpage":          githubv4.Int(input.PerPage),
		"stargazersCursor": (*githubv4.String)(input.StartCursor),
		"starorder":        input.StarOrder,
	}

	err := input.Client.Query(ctx, &starsQuery, variables)

	if err != nil {
		return nil, err
	}

	return &fetchStarsResults{
		starsQuery.Repository.Stargazers.Edges,
		starsQuery.Repository.Stargazers.PageInfo.HasNextPage,
		&starsQuery.Repository.Stargazers.PageInfo.EndCursor,
	}, nil
}

type iterStargazers struct {
	fullNameOrOwner string
	name            string
	client          *githubv4.Client
	current         int
	results         *fetchStarsResults
	rateLimiter     *rate.Limiter
	starOrder       *githubv4.StarOrder
	perPage         int
}

func (i *iterStargazers) Column(ctx *sqlite.Context, c int) error {
	switch c {
	case 0:
		ctx.ResultText(i.fullNameOrOwner)
	case 1:
		ctx.ResultText(i.name)
	case 2:
		ctx.ResultText(i.results.Edges[i.current].Node.Login)
	case 3:
		ctx.ResultText(i.results.Edges[i.current].Node.Email)
	case 4:
		ctx.ResultText(i.results.Edges[i.current].Node.Name)
	case 5:
		ctx.ResultText(i.results.Edges[i.current].Node.Bio)
	case 6:
		ctx.ResultText(i.results.Edges[i.current].Node.Company)
	case 7:
		ctx.ResultText(i.results.Edges[i.current].Node.AvatarUrl)
	case 8:
		t := i.results.Edges[i.current].Node.CreatedAt
		if t.IsZero() {
			ctx.ResultNull()
		} else {
			ctx.ResultText(t.Format(time.RFC3339Nano))
		}
	case 9:
		t := i.results.Edges[i.current].Node.UpdatedAt
		if t.IsZero() {
			ctx.ResultNull()
		} else {
			ctx.ResultText(t.Format(time.RFC3339Nano))
		}
	case 10:
		ctx.ResultText(i.results.Edges[i.current].Node.TwitterUsername)
	case 11:
		ctx.ResultText(i.results.Edges[i.current].Node.WebsiteUrl)
	case 12:
		ctx.ResultText(i.results.Edges[i.current].Node.Location)
	case 13:
		ctx.ResultText(i.results.Edges[i.current].StarredAt)
	}
	return nil
}

func (i *iterStargazers) Next() (vtab.Row, error) {
	i.current += 1

	if i.results == nil || i.current >= len(i.results.Edges) {
		if i.results == nil || i.results.HasNextPage {
			err := i.rateLimiter.Wait(context.Background())
			if err != nil {
				return nil, err
			}

			owner, name, err := repoOwnerAndName(i.name, i.fullNameOrOwner)
			if err != nil {
				return nil, err
			}

			var cursor *githubv4.String
			if i.results != nil {
				cursor = i.results.EndCursor
			}

			results, err := fetchStars(context.Background(), &fetchStarsOptions{i.client, owner, name, i.perPage, cursor, i.starOrder})
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

var stargazersCols = []vtab.Column{
	{Name: "owner", Type: sqlite.SQLITE_TEXT, NotNull: true, Hidden: true, Filters: []*vtab.ColumnFilter{{Op: sqlite.INDEX_CONSTRAINT_EQ, Required: true, OmitCheck: true}}},
	{Name: "reponame", Type: sqlite.SQLITE_TEXT, NotNull: true, Hidden: true, Filters: []*vtab.ColumnFilter{{Op: sqlite.INDEX_CONSTRAINT_EQ}}},
	{Name: "login", Type: sqlite.SQLITE_TEXT, NotNull: false, Hidden: false, Filters: nil},
	{Name: "email", Type: sqlite.SQLITE_TEXT, NotNull: false, Hidden: false, Filters: nil},
	{Name: "name", Type: sqlite.SQLITE_TEXT, NotNull: false, Hidden: false, Filters: nil},
	{Name: "bio", Type: sqlite.SQLITE_TEXT, NotNull: false, Hidden: false, Filters: nil},
	{Name: "company", Type: sqlite.SQLITE_TEXT, NotNull: false, Hidden: false, Filters: nil},
	{Name: "avatar_url", Type: sqlite.SQLITE_TEXT, NotNull: false, Hidden: false, Filters: nil},
	{Name: "created_at", Type: sqlite.SQLITE_TEXT, NotNull: false, Hidden: false, Filters: nil},
	{Name: "updated_at", Type: sqlite.SQLITE_TEXT, NotNull: false, Hidden: false, Filters: nil},
	{Name: "twitter", Type: sqlite.SQLITE_TEXT, NotNull: false, Hidden: false, Filters: nil},
	{Name: "website", Type: sqlite.SQLITE_TEXT, NotNull: false, Hidden: false, Filters: nil},
	{Name: "location", Type: sqlite.SQLITE_TEXT, NotNull: false, Hidden: false, Filters: nil},
	{Name: "starred_at", Type: sqlite.SQLITE_TEXT, NotNull: false, Hidden: false, Filters: nil, OrderBy: vtab.ASC | vtab.DESC},
}

func NewStargazersModule(opts *Options) sqlite.Module {
	return vtab.NewTableFunc("github_stargazers", stargazersCols, func(constraints []*vtab.Constraint, orders []*sqlite.OrderBy) (vtab.Iterator, error) {
		var fullNameOrOwner, name string
		for _, constraint := range constraints {
			if constraint.Op == sqlite.INDEX_CONSTRAINT_EQ {
				switch constraint.ColIndex {
				case 0:
					fullNameOrOwner = constraint.Value.Text()
				case 1:
					name = constraint.Value.Text()
				}
			}
		}

		var starOrder *githubv4.StarOrder
		// for now we can only support single field order bys
		if len(orders) == 1 {
			order := orders[0]
			starOrder = &githubv4.StarOrder{}
			switch order.ColumnIndex {
			case 13:
				starOrder.Field = githubv4.StarOrderFieldStarredAt
			}
			starOrder.Direction = orderByToGitHubOrder(order.Desc)
		}

		return &iterStargazers{fullNameOrOwner, name, opts.Client(), -1, nil, opts.RateLimiter, starOrder, opts.PerPage}, nil
	})
}
