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
	current := i.results.Edges[i.current]
	switch stargazersCols[c].Name {
	case "owner":
		ctx.ResultText(i.fullNameOrOwner)
	case "reponame":
		ctx.ResultText(i.name)
	case "login":
		ctx.ResultText(current.Node.Login)
	case "email":
		ctx.ResultText(current.Node.Email)
	case "name":
		ctx.ResultText(current.Node.Name)
	case "bio":
		ctx.ResultText(current.Node.Bio)
	case "company":
		ctx.ResultText(current.Node.Company)
	case "avatar_url":
		ctx.ResultText(current.Node.AvatarUrl)
	case "created_at":
		t := current.Node.CreatedAt
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
	case "twitter":
		ctx.ResultText(current.Node.TwitterUsername)
	case "website":
		ctx.ResultText(current.Node.WebsiteUrl)
	case "location":
		ctx.ResultText(current.Node.Location)
	case "starred_at":
		ctx.ResultText(current.StarredAt)
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
	{Name: "owner", Type: sqlite.SQLITE_TEXT, Hidden: true, Filters: []*vtab.ColumnFilter{{Op: sqlite.INDEX_CONSTRAINT_EQ, Required: true, OmitCheck: true}}},
	{Name: "reponame", Type: sqlite.SQLITE_TEXT, Hidden: true, Filters: []*vtab.ColumnFilter{{Op: sqlite.INDEX_CONSTRAINT_EQ}}},
	{Name: "login", Type: sqlite.SQLITE_TEXT},
	{Name: "email", Type: sqlite.SQLITE_TEXT},
	{Name: "name", Type: sqlite.SQLITE_TEXT},
	{Name: "bio", Type: sqlite.SQLITE_TEXT},
	{Name: "company", Type: sqlite.SQLITE_TEXT},
	{Name: "avatar_url", Type: sqlite.SQLITE_TEXT},
	{Name: "created_at", Type: sqlite.SQLITE_TEXT},
	{Name: "updated_at", Type: sqlite.SQLITE_TEXT},
	{Name: "twitter", Type: sqlite.SQLITE_TEXT},
	{Name: "website", Type: sqlite.SQLITE_TEXT},
	{Name: "location", Type: sqlite.SQLITE_TEXT},
	{Name: "starred_at", Type: sqlite.SQLITE_TEXT, OrderBy: vtab.ASC | vtab.DESC},
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
			switch stargazersCols[order.ColumnIndex].Name {
			case "starred_at":
				starOrder.Field = githubv4.StarOrderFieldStarredAt
			}
			starOrder.Direction = orderByToGitHubOrder(order.Desc)
		}

		return &iterStargazers{fullNameOrOwner, name, opts.Client(), -1, nil, opts.RateLimiter, starOrder, opts.PerPage}, nil
	})
}
