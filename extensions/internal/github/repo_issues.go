package github

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/augmentable-dev/vtab"
	"github.com/mergestat/mergestat/extensions/options"
	"github.com/rs/zerolog"
	"github.com/shurcooL/githubv4"
	"go.riyazali.net/sqlite"
)

type issue struct {
	Author struct {
		Login string
	}
	Body     string
	Closed   bool
	ClosedAt githubv4.DateTime
	Comments struct {
		TotalCount int
	}
	CreatedAt       githubv4.DateTime
	CreatedViaEmail bool
	DatabaseId      int
	Editor          struct {
		Login string
	}
	IncludesCreatedEdit bool
	IsReadByViewer      bool
	Labels              struct {
		TotalCount int
	}
	LastEditedAt githubv4.DateTime
	Locked       bool
	Milestone    struct {
		Number int
	}
	Number       int
	Participants struct {
		TotalCount int
	}
	PublishedAt githubv4.DateTime
	Reactions   struct {
		TotalCount int
	}
	State     githubv4.IssueState
	Title     string
	UpdatedAt githubv4.DateTime
	Url       githubv4.URI
}

type fetchIssuesResults struct {
	RateLimit   *options.GitHubRateLimitResponse
	Edges       []*issueEdge
	HasNextPage bool
	EndCursor   *githubv4.String
}

type issueEdge struct {
	Cursor string
	Node   issue
}

func (i *iterIssues) fetchIssues(ctx context.Context, startCursor *githubv4.String) (*fetchIssuesResults, error) {
	var issuesQuery struct {
		RateLimit  *options.GitHubRateLimitResponse
		Repository struct {
			Owner struct {
				Login string
			}
			Name   string
			Issues struct {
				Edges    []*issueEdge
				PageInfo struct {
					EndCursor   githubv4.String
					HasNextPage bool
				}
			} `graphql:"issues(first: $perpage, after: $issuecursor, orderBy: $issueorder)"`
		} `graphql:"repository(owner: $owner, name: $name)"`
	}
	variables := map[string]interface{}{
		"owner":       githubv4.String(i.owner),
		"name":        githubv4.String(i.name),
		"perpage":     githubv4.Int(i.PerPage),
		"issuecursor": startCursor,
		"issueorder":  i.issueOrder,
	}

	err := i.Client().Query(ctx, &issuesQuery, variables)
	if err != nil {
		return nil, err
	}

	return &fetchIssuesResults{
		RateLimit:   issuesQuery.RateLimit,
		Edges:       issuesQuery.Repository.Issues.Edges,
		HasNextPage: issuesQuery.Repository.Issues.PageInfo.HasNextPage,
		EndCursor:   &issuesQuery.Repository.Issues.PageInfo.EndCursor,
	}, nil
}

type iterIssues struct {
	*Options
	owner      string
	name       string
	current    int
	results    *fetchIssuesResults
	issueOrder *githubv4.IssueOrder
}

func (i *iterIssues) logger() *zerolog.Logger {
	logger := i.Logger.With().Int("per-page", i.PerPage).Str("owner", i.owner).Str("name", i.name).Logger()
	if i.issueOrder != nil {
		logger = logger.With().Str("order_by", string(i.issueOrder.Field)).Str("order_dir", string(i.issueOrder.Direction)).Logger()
	}
	return &logger
}

func (i *iterIssues) Column(ctx vtab.Context, c int) error {
	current := i.results.Edges[i.current]
	col := issuesCols[c]

	switch col.Name {
	case "author_login":
		ctx.ResultText(current.Node.Author.Login)
	case "body":
		ctx.ResultText(current.Node.Body)
	case "closed":
		ctx.ResultInt(t1f0(current.Node.Closed))
	case "closed_at":
		t := current.Node.ClosedAt
		if t.IsZero() {
			ctx.ResultNull()
		} else {
			ctx.ResultText(t.Format(time.RFC3339Nano))
		}
	case "comment_count":
		ctx.ResultInt(current.Node.Comments.TotalCount)
	case "created_at":
		t := current.Node.CreatedAt
		if t.IsZero() {
			ctx.ResultNull()
		} else {
			ctx.ResultText(t.Format(time.RFC3339Nano))
		}
	case "created_via_email":
		ctx.ResultInt(t1f0(current.Node.CreatedViaEmail))
	case "database_id":
		ctx.ResultInt(current.Node.DatabaseId)
	case "editor_login":
		ctx.ResultText(current.Node.Editor.Login)
	case "includes_created_edit":
		ctx.ResultInt(t1f0(current.Node.IncludesCreatedEdit))
	case "label_count":
		ctx.ResultInt(current.Node.Labels.TotalCount)
	case "last_edited_at":
		t := current.Node.LastEditedAt
		if t.IsZero() {
			ctx.ResultNull()
		} else {
			ctx.ResultText(t.Format(time.RFC3339Nano))
		}
	case "locked":
		ctx.ResultInt(t1f0(current.Node.Locked))
	case "milestone_number":
		ctx.ResultInt(current.Node.Milestone.Number)
	case "number":
		ctx.ResultInt(current.Node.Number)
	case "participant_count":
		ctx.ResultInt(current.Node.Participants.TotalCount)
	case "published_at":
		t := current.Node.PublishedAt
		if t.IsZero() {
			ctx.ResultNull()
		} else {
			ctx.ResultText(t.Format(time.RFC3339Nano))
		}
	case "reaction_count":
		ctx.ResultInt(current.Node.Reactions.TotalCount)
	case "state":
		ctx.ResultText(fmt.Sprint(current.Node.State))
	case "title":
		ctx.ResultText(current.Node.Title)
	case "updated_at":
		t := current.Node.UpdatedAt
		if t.IsZero() {
			ctx.ResultNull()
		} else {
			ctx.ResultText(t.Format(time.RFC3339Nano))
		}
	case "url":
		ctx.ResultText(current.Node.Url.String())
	}
	return nil
}

func (i *iterIssues) Next() (vtab.Row, error) {
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
			l.Info().Msgf("fetching page of repo_issues for %s/%s", i.owner, i.name)
			results, err := i.fetchIssues(context.Background(), cursor)

			i.Options.GitHubPostRequestHook()

			if err != nil {
				return nil, err
			}

			i.Options.RateLimitHandler(results.RateLimit)

			i.results = results
			i.current = 0

			if len(results.Edges) == 0 {
				return nil, io.EOF
			}
		} else {
			return nil, io.EOF
		}
	}

	return i, nil
}

var issuesCols = []vtab.Column{
	{Name: "owner", Type: "TEXT", NotNull: true, Hidden: true, Filters: []*vtab.ColumnFilter{{Op: sqlite.INDEX_CONSTRAINT_EQ, OmitCheck: true}}},
	{Name: "reponame", Type: "TEXT", NotNull: true, Hidden: true, Filters: []*vtab.ColumnFilter{{Op: sqlite.INDEX_CONSTRAINT_EQ, OmitCheck: true}}},
	{Name: "author_login", Type: "TEXT"},
	{Name: "body", Type: "TEXT"},
	{Name: "closed", Type: "BOOLEAN"},
	{Name: "closed_at", Type: "DATETIME"},
	{Name: "comment_count", Type: "INT", OrderBy: vtab.ASC | vtab.DESC},
	{Name: "created_at", Type: "DATETIME", OrderBy: vtab.ASC | vtab.DESC},
	{Name: "created_via_email", Type: "BOOLEAN"},
	{Name: "database_id", Type: "TEXT"},
	{Name: "editor_login", Type: "TEXT"},
	{Name: "includes_created_edit", Type: "BOOLEAN"},
	{Name: "label_count", Type: "INT"},
	{Name: "last_edited_at", Type: "DATETIME"},
	{Name: "locked", Type: "BOOLEAN"},
	{Name: "milestone_count", Type: "INT"},
	{Name: "number", Type: "INT"},
	{Name: "participant_count", Type: "INT"},
	{Name: "published_at", Type: "DATETIME"},
	{Name: "reaction_count", Type: "INT"},
	{Name: "state", Type: "TEXT"},
	{Name: "title", Type: "TEXT"},
	{Name: "updated_at", Type: "DATETIME", OrderBy: vtab.ASC | vtab.DESC},
	{Name: "url", Type: "TEXT"},
}

func NewIssuesModule(opts *Options) sqlite.Module {
	return vtab.NewTableFunc("github_repo_issues", issuesCols, func(constraints []*vtab.Constraint, orders []*sqlite.OrderBy) (vtab.Iterator, error) {
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

		var issueOrder *githubv4.IssueOrder
		if len(orders) == 1 {
			order := orders[0]
			issueOrder = &githubv4.IssueOrder{}
			switch issuesCols[order.ColumnIndex].Name {
			case "comment_count":
				issueOrder.Field = githubv4.IssueOrderFieldComments
			case "created_at":
				issueOrder.Field = githubv4.IssueOrderFieldCreatedAt
			case "updated_at":
				issueOrder.Field = githubv4.IssueOrderFieldUpdatedAt
			}
			issueOrder.Direction = orderByToGitHubOrder(order.Desc)
		}

		iter := &iterIssues{opts, owner, name, -1, nil, issueOrder}
		iter.logger().Info().Msgf("starting GitHub repo_issues iterator for %s/%s", owner, name)
		return iter, nil
	}, vtab.EarlyOrderByConstraintExit(true))
}
