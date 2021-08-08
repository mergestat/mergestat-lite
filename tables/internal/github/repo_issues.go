package github

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/augmentable-dev/vtab"
	"github.com/shurcooL/githubv4"
	"go.riyazali.net/sqlite"
	"golang.org/x/time/rate"
)

type issue struct {
	ActiveLockReason githubv4.LockReason
	Author           struct {
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

type fetchIssuesOptions struct {
	Client      *githubv4.Client
	Owner       string
	Name        string
	PerPage     int
	StartCursor *githubv4.String
	IssueOrder  *githubv4.IssueOrder
}

type fetchIssuesResults struct {
	Edges       []*issueEdge
	HasNextPage bool
	EndCursor   *githubv4.String
}

type issueEdge struct {
	Cursor string
	Node   issue
}

func fetchIssues(ctx context.Context, input *fetchIssuesOptions) (*fetchIssuesResults, error) {
	var issuesQuery struct {
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
		"owner":       githubv4.String(input.Owner),
		"name":        githubv4.String(input.Name),
		"perpage":     githubv4.Int(input.PerPage),
		"issuecursor": (*githubv4.String)(input.StartCursor),
		"issueorder":  input.IssueOrder,
	}

	err := input.Client.Query(ctx, &issuesQuery, variables)

	if err != nil {
		return nil, err
	}

	return &fetchIssuesResults{
		issuesQuery.Repository.Issues.Edges,
		issuesQuery.Repository.Issues.PageInfo.HasNextPage,
		&issuesQuery.Repository.Issues.PageInfo.EndCursor,
	}, nil
}

type iterIssues struct {
	fullNameOrOwner string
	name            string
	client          *githubv4.Client
	current         int
	results         *fetchIssuesResults
	rateLimiter     *rate.Limiter
	issueOrder      *githubv4.IssueOrder
	perPage         int
}

func (i *iterIssues) Column(ctx *sqlite.Context, c int) error {
	current := i.results.Edges[i.current]
	col := issuesCols[c]

	switch col.Name {
	case "owner":
		ctx.ResultText(i.fullNameOrOwner)
	case "reponame":
		ctx.ResultText(i.name)
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

			results, err := fetchIssues(context.Background(), &fetchIssuesOptions{i.client, owner, name, i.perPage, cursor, i.issueOrder})
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

var issuesCols = []vtab.Column{
	{Name: "owner", Type: sqlite.SQLITE_TEXT, NotNull: true, Hidden: true, Filters: []*vtab.ColumnFilter{{Op: sqlite.INDEX_CONSTRAINT_EQ, Required: true, OmitCheck: true}}},
	{Name: "reponame", Type: sqlite.SQLITE_TEXT, NotNull: true, Hidden: true, Filters: []*vtab.ColumnFilter{{Op: sqlite.INDEX_CONSTRAINT_EQ}}},
	{Name: "author_login", Type: sqlite.SQLITE_TEXT},
	{Name: "body", Type: sqlite.SQLITE_TEXT},
	{Name: "closed", Type: sqlite.SQLITE_INTEGER},
	{Name: "closed_at", Type: sqlite.SQLITE_TEXT},
	{Name: "comment_count", Type: sqlite.SQLITE_INTEGER, OrderBy: vtab.ASC | vtab.DESC},
	{Name: "created_at", Type: sqlite.SQLITE_TEXT, OrderBy: vtab.ASC | vtab.DESC},
	{Name: "created_via_email", Type: sqlite.SQLITE_INTEGER},
	{Name: "database_id", Type: sqlite.SQLITE_TEXT},
	{Name: "editor_login", Type: sqlite.SQLITE_TEXT},
	{Name: "includes_created_edit", Type: sqlite.SQLITE_INTEGER},
	{Name: "label_count", Type: sqlite.SQLITE_INTEGER},
	{Name: "last_edited_at", Type: sqlite.SQLITE_TEXT},
	{Name: "locked", Type: sqlite.SQLITE_INTEGER},
	{Name: "milestone_count", Type: sqlite.SQLITE_INTEGER},
	{Name: "number", Type: sqlite.SQLITE_INTEGER},
	{Name: "participant_count", Type: sqlite.SQLITE_INTEGER},
	{Name: "published_at", Type: sqlite.SQLITE_TEXT},
	{Name: "reaction_count", Type: sqlite.SQLITE_INTEGER},
	{Name: "state", Type: sqlite.SQLITE_TEXT},
	{Name: "title", Type: sqlite.SQLITE_TEXT},
	{Name: "updated_at", Type: sqlite.SQLITE_TEXT, OrderBy: vtab.ASC | vtab.DESC},
	{Name: "url", Type: sqlite.SQLITE_TEXT},
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

		return &iterIssues{fullNameOrOwner, name, opts.Client(), -1, nil, opts.RateLimiter, issueOrder, opts.PerPage}, nil
	})
}
