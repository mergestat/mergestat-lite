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

type stargazerEdge struct {
	StarredAt string
	Node      stargazer
}

type fetchStarsResults struct {
	RateLimit   *options.GitHubRateLimitResponse
	Edges       []*stargazerEdge
	HasNextPage bool
	EndCursor   *githubv4.String
}

func (i *iterStargazers) fetchStars(ctx context.Context, startCursor *githubv4.String) (*fetchStarsResults, error) {
	var starsQuery struct {
		RateLimit  *options.GitHubRateLimitResponse
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
		"owner":            githubv4.String(i.owner),
		"name":             githubv4.String(i.name),
		"perpage":          githubv4.Int(i.PerPage),
		"stargazersCursor": startCursor,
		"starorder":        i.starOrder,
	}

	err := i.Client().Query(ctx, &starsQuery, variables)
	if err != nil {
		return nil, err
	}

	return &fetchStarsResults{
		RateLimit:   starsQuery.RateLimit,
		Edges:       starsQuery.Repository.Stargazers.Edges,
		HasNextPage: starsQuery.Repository.Stargazers.PageInfo.HasNextPage,
		EndCursor:   &starsQuery.Repository.Stargazers.PageInfo.EndCursor,
	}, nil
}

type iterStargazers struct {
	*Options
	owner     string
	name      string
	current   int
	results   *fetchStarsResults
	starOrder *githubv4.StarOrder
}

func (i *iterStargazers) logger() *zerolog.Logger {
	logger := i.Logger.With().Int("per-page", i.PerPage).Str("owner", i.owner).Str("name", i.name).Logger()
	if i.starOrder != nil {
		logger = logger.With().Str("order_by", string(i.starOrder.Field)).Str("order_dir", string(i.starOrder.Direction)).Logger()
	}
	return &logger
}

func (i *iterStargazers) Column(ctx vtab.Context, c int) error {
	current := i.results.Edges[i.current]
	switch stargazersCols[c].Name {
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
			l.Info().Msgf("fetching page of stargazers for %s/%s", i.owner, i.name)
			results, err := i.fetchStars(context.Background(), cursor)

			i.Options.GitHubPostRequestHook()

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

var stargazersCols = []vtab.Column{
	{Name: "owner", Type: "TEXT", Hidden: true, Filters: []*vtab.ColumnFilter{{Op: sqlite.INDEX_CONSTRAINT_EQ, OmitCheck: true}}},
	{Name: "reponame", Type: "TEXT", Hidden: true, Filters: []*vtab.ColumnFilter{{Op: sqlite.INDEX_CONSTRAINT_EQ, OmitCheck: true}}},
	{Name: "login", Type: "TEXT"},
	{Name: "email", Type: "TEXT"},
	{Name: "name", Type: "TEXT"},
	{Name: "bio", Type: "TEXT"},
	{Name: "company", Type: "TEXT"},
	{Name: "avatar_url", Type: "TEXT"},
	{Name: "created_at", Type: "DATETIME"},
	{Name: "updated_at", Type: "DATETIME"},
	{Name: "twitter", Type: "TEXT"},
	{Name: "website", Type: "TEXT"},
	{Name: "location", Type: "TEXT"},
	{Name: "starred_at", Type: "DATETIME", OrderBy: vtab.ASC | vtab.DESC, Filters: []*vtab.ColumnFilter{
		{Op: sqlite.INDEX_CONSTRAINT_GT}, {Op: sqlite.INDEX_CONSTRAINT_GE},
		{Op: sqlite.INDEX_CONSTRAINT_LT}, {Op: sqlite.INDEX_CONSTRAINT_LE},
	}},
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

		owner, name, err := repoOwnerAndName(name, fullNameOrOwner)
		if err != nil {
			return nil, err
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

		iter := &iterStargazers{opts, owner, name, -1, nil, starOrder}
		iter.logger().Info().Msgf("starting GitHub stargazers iterator for %s/%s", owner, name)
		return iter, nil
	}, vtab.EarlyOrderByConstraintExit(true))
}
