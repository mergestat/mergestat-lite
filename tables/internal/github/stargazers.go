package github

import (
	"context"
	"errors"
	"io"
	"strings"
	"time"

	"github.com/augmentable-dev/vtab"
	"github.com/shurcooL/githubv4"
	"go.riyazali.net/sqlite"
	"golang.org/x/oauth2"
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
	Edges       []edge //why not []*Edge
	HasNextPage bool
	EndCursor   *githubv4.String
}

type edge struct { //change to a more informative name.
	StarredAt string    //StarredAt insists on being a string whereas createdAt and updatedAt are time.Time.
	Node      stargazer //change to a more informative name
}

func fetchStars(ctx context.Context, input *fetchStarsOptions) (*fetchStarsResults, error) {
	var starsQuery struct {
		Repository struct {
			Owner struct {
				Login string
			}
			Name       string
			Stargazers struct {
				Edges    []edge
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
}

// repoOwnerAndName returns the "owner" and "name" (respective return values) or an error
// given the inputs to the iterator. This allows for both `SELECT * FROM github_stargazers('askgitdev/starq')`
// and `SELECT * FROM github_stargazers('askgitdev', 'starq')
func (i *iterStargazers) repoOwnerAndName() (string, string, error) {
	if i.name == "" {

		split_string := strings.Split(i.fullNameOrOwner, "/")
		if len(split_string) != 2 {
			return "", "", errors.New("invalid repo name, must be of format owner/name")
		}
		return split_string[0], split_string[1], nil
	} else {
		return i.fullNameOrOwner, i.name, nil
	}
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
			ctx.ResultText(" ")
		} else {
			ctx.ResultText(t.Format(time.RFC3339Nano))
		}
	case 9:
		t := i.results.Edges[i.current].Node.UpdatedAt
		if t.IsZero() {
			ctx.ResultText(" ")
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

			owner, name, err := i.repoOwnerAndName()
			if err != nil {
				return nil, err
			}

			var cursor *githubv4.String
			if i.results != nil {
				cursor = i.results.EndCursor
			}

			results, err := fetchStars(context.Background(), &fetchStarsOptions{i.client, owner, name, 100, cursor, i.starOrder})
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

var cols_stargazers = []vtab.Column{
	{Name: "owner", Type: sqlite.SQLITE_TEXT, NotNull: true, Hidden: true, Filters: []*vtab.ColumnFilter{{Op: sqlite.INDEX_CONSTRAINT_EQ, Required: true, OmitCheck: true}}, OrderBy: vtab.NONE},
	{Name: "reponame", Type: sqlite.SQLITE_TEXT, NotNull: true, Hidden: true, Filters: []*vtab.ColumnFilter{{Op: sqlite.INDEX_CONSTRAINT_EQ}}, OrderBy: vtab.NONE},
	{Name: "login", Type: sqlite.SQLITE_TEXT, NotNull: false, Hidden: false, Filters: nil, OrderBy: vtab.NONE},
	{Name: "email", Type: sqlite.SQLITE_TEXT, NotNull: false, Hidden: false, Filters: nil, OrderBy: vtab.NONE},
	{Name: "name", Type: sqlite.SQLITE_TEXT, NotNull: false, Hidden: false, Filters: nil, OrderBy: vtab.NONE},
	{Name: "bio", Type: sqlite.SQLITE_TEXT, NotNull: false, Hidden: false, Filters: nil, OrderBy: vtab.NONE},
	{Name: "company", Type: sqlite.SQLITE_TEXT, NotNull: false, Hidden: false, Filters: nil, OrderBy: vtab.NONE},
	{Name: "avatar_url", Type: sqlite.SQLITE_TEXT, NotNull: false, Hidden: false, Filters: nil, OrderBy: vtab.NONE},
	{Name: "created_at", Type: sqlite.SQLITE_TEXT, NotNull: false, Hidden: false, Filters: nil, OrderBy: vtab.NONE},
	{Name: "updated_at", Type: sqlite.SQLITE_TEXT, NotNull: false, Hidden: false, Filters: nil, OrderBy: vtab.NONE},
	{Name: "twitter", Type: sqlite.SQLITE_TEXT, NotNull: false, Hidden: false, Filters: nil, OrderBy: vtab.NONE},
	{Name: "website", Type: sqlite.SQLITE_TEXT, NotNull: false, Hidden: false, Filters: nil, OrderBy: vtab.NONE},
	{Name: "location", Type: sqlite.SQLITE_TEXT, NotNull: false, Hidden: false, Filters: nil, OrderBy: vtab.NONE},
	{Name: "starred_at", Type: sqlite.SQLITE_TEXT, NotNull: false, Hidden: false, Filters: nil, OrderBy: vtab.ASC | vtab.DESC},
}

func NewStargazersModule(githubToken string, rateLimiter *rate.Limiter) sqlite.Module {
	httpClient := oauth2.NewClient(context.Background(), oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: githubToken},
	))
	client := githubv4.NewClient(httpClient)

	return vtab.NewTableFunc("github_stargazers", cols_stargazers, func(constraints []*vtab.Constraint, orders []*sqlite.OrderBy) (vtab.Iterator, error) {
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

		starOrder := &githubv4.StarOrder{
			Field:     githubv4.StarOrderFieldStarredAt,
			Direction: githubv4.OrderDirectionDesc,
		}

		for _, order := range orders { //adding this for loop for scalability. might need to order the data by more columns in the future.
			switch order.ColumnIndex {
			case 13:
				if !order.Desc {
					starOrder.Direction = githubv4.OrderDirectionAsc
				}
			}
		}

		return &iterStargazers{fullNameOrOwner, name, client, -1, nil, rateLimiter, starOrder}, nil
	})
}
