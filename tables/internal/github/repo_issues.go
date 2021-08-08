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

type user struct {
	Login string
	URL   string
}

type issue struct {
	ActiveLockReason githubv4.LockReason
	//Assignees
	Author   user
	Body     string
	BodyText string
	Closed   bool
	ClosedAt githubv4.DateTime
	Comments struct {
		TotalCount int
	}
	CreatedAt           githubv4.DateTime
	CreatedViaEmail     bool
	DatabaseId          int
	Editor              user
	IncludesCreatedEdit bool
	IsReadByViewer      bool
	Labels              struct {
		TotalCount int
	}
	LastEditedAt githubv4.DateTime
	Locked       bool
	Milestone    struct {
		Number             int
		ProgressPercentage githubv4.Float
	}
	Number       int
	Participants struct {
		TotalCount int
	}
	PublishedAt githubv4.DateTime
	//reactionGroups
	Reactions struct {
		TotalCount int
	}
	State            githubv4.IssueState
	Title            string
	UpdatedAt        githubv4.DateTime
	Url              githubv4.URI
	UserContentEdits struct {
		TotalCount int
	}
	ViewerCanReact     bool
	ViewerCanSubscribe bool
	ViewerCanUpdate    bool
	ViewerDidAuthor    bool
	ViewerSubscription githubv4.SubscriptionState
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
	switch c {
	case 0:
		ctx.ResultText(i.fullNameOrOwner)
	case 1:
		ctx.ResultText(i.name)
	case 2:
		ctx.ResultText(i.results.Edges[i.current].Node.Author.Login)
	case 3:
		ctx.ResultText(i.results.Edges[i.current].Node.Author.URL)
	case 4:
		ctx.ResultText(i.results.Edges[i.current].Node.Body)
	case 5:
		ctx.ResultText(i.results.Edges[i.current].Node.BodyText)
	case 6:
		ctx.ResultInt(t1f0(i.results.Edges[i.current].Node.Closed))
	case 7:
		t := i.results.Edges[i.current].Node.ClosedAt
		if t.IsZero() {
			ctx.ResultNull()
		} else {
			ctx.ResultText(t.Format(time.RFC3339Nano))
		}
	case 8:
		ctx.ResultInt(i.results.Edges[i.current].Node.Comments.TotalCount)
	case 9:
		t := i.results.Edges[i.current].Node.CreatedAt
		if t.IsZero() {
			ctx.ResultNull()
		} else {
			ctx.ResultText(t.Format(time.RFC3339Nano))
		}
	case 10:
		ctx.ResultInt(t1f0(i.results.Edges[i.current].Node.CreatedViaEmail))
	case 11:
		ctx.ResultInt(i.results.Edges[i.current].Node.DatabaseId)
	case 12:
		ctx.ResultText(i.results.Edges[i.current].Node.Editor.Login)
	case 13:
		ctx.ResultText(i.results.Edges[i.current].Node.Editor.URL)
	case 14:
		ctx.ResultInt(t1f0(i.results.Edges[i.current].Node.IncludesCreatedEdit))
	case 15:
		ctx.ResultInt(t1f0(i.results.Edges[i.current].Node.IsReadByViewer))
	case 16:
		ctx.ResultInt(i.results.Edges[i.current].Node.Labels.TotalCount)
	case 17:
		t := i.results.Edges[i.current].Node.LastEditedAt
		if t.IsZero() {
			ctx.ResultNull()
		} else {
			ctx.ResultText(t.Format(time.RFC3339Nano))
		}
	case 18:
		ctx.ResultInt(t1f0(i.results.Edges[i.current].Node.Locked))
	case 19:
		ctx.ResultInt(i.results.Edges[i.current].Node.Milestone.Number)
	case 20:
		ctx.ResultFloat(float64(i.results.Edges[i.current].Node.Milestone.ProgressPercentage))
	case 21:
		ctx.ResultInt(i.results.Edges[i.current].Node.Number)
	case 22:
		ctx.ResultInt(i.results.Edges[i.current].Node.Participants.TotalCount)
	case 23:
		t := i.results.Edges[i.current].Node.PublishedAt
		if t.IsZero() {
			ctx.ResultNull()
		} else {
			ctx.ResultText(t.Format(time.RFC3339Nano))
		}
	case 24:
		ctx.ResultInt(i.results.Edges[i.current].Node.Reactions.TotalCount)
	case 25:
		ctx.ResultText(fmt.Sprint(i.results.Edges[i.current].Node.State))
	case 26:
		ctx.ResultText(i.results.Edges[i.current].Node.Title)
	case 27:
		t := i.results.Edges[i.current].Node.UpdatedAt
		if t.IsZero() {
			ctx.ResultNull()
		} else {
			ctx.ResultText(t.Format(time.RFC3339Nano))
		}
	case 28:
		ctx.ResultText(i.results.Edges[i.current].Node.Url.String())
	case 29:
		ctx.ResultInt(i.results.Edges[i.current].Node.UserContentEdits.TotalCount)
	case 30:
		ctx.ResultInt(t1f0(i.results.Edges[i.current].Node.ViewerCanReact))
	case 31:
		ctx.ResultInt(t1f0(i.results.Edges[i.current].Node.ViewerCanSubscribe))
	case 32:
		ctx.ResultInt(t1f0(i.results.Edges[i.current].Node.ViewerCanUpdate))

	case 33:
		ctx.ResultInt(t1f0(i.results.Edges[i.current].Node.ViewerDidAuthor))

	case 34:
		ctx.ResultText(fmt.Sprint(i.results.Edges[i.current].Node.ViewerSubscription))
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
	{Name: "author_login", Type: sqlite.SQLITE_TEXT, NotNull: false, Hidden: false, Filters: nil},
	{Name: "author_url", Type: sqlite.SQLITE_TEXT, NotNull: false, Hidden: false, Filters: nil},
	{Name: "body", Type: sqlite.SQLITE_TEXT, NotNull: false, Hidden: false, Filters: nil},
	{Name: "body_text", Type: sqlite.SQLITE_TEXT, NotNull: false, Hidden: false, Filters: nil},
	{Name: "closed", Type: sqlite.SQLITE_INTEGER, NotNull: false, Hidden: false, Filters: nil},
	{Name: "closed_at", Type: sqlite.SQLITE_TEXT, NotNull: false, Hidden: false, Filters: nil},
	{Name: "comment_count", Type: sqlite.SQLITE_INTEGER, NotNull: false, Hidden: false, Filters: nil, OrderBy: vtab.ASC | vtab.DESC},
	{Name: "created_at", Type: sqlite.SQLITE_TEXT, NotNull: false, Hidden: false, Filters: nil, OrderBy: vtab.ASC | vtab.DESC},
	{Name: "created_via_email", Type: sqlite.SQLITE_INTEGER, NotNull: false, Hidden: false, Filters: nil},
	{Name: "database_id", Type: sqlite.SQLITE_TEXT, NotNull: false, Hidden: false, Filters: nil},
	{Name: "editor_login", Type: sqlite.SQLITE_TEXT, NotNull: false, Hidden: false, Filters: nil},
	{Name: "editor_url", Type: sqlite.SQLITE_TEXT, NotNull: false, Hidden: false, Filters: nil},
	{Name: "includes_created_edit", Type: sqlite.SQLITE_INTEGER, NotNull: false, Hidden: false, Filters: nil},
	{Name: "is_read_by_viewer", Type: sqlite.SQLITE_INTEGER, NotNull: false, Hidden: false, Filters: nil},
	{Name: "label_count", Type: sqlite.SQLITE_INTEGER, NotNull: false, Hidden: false, Filters: nil},
	{Name: "last_edited_at", Type: sqlite.SQLITE_TEXT, NotNull: false, Hidden: false, Filters: nil},
	{Name: "locked", Type: sqlite.SQLITE_INTEGER, NotNull: false, Hidden: false, Filters: nil},
	{Name: "milestone_count", Type: sqlite.SQLITE_INTEGER, NotNull: false, Hidden: false, Filters: nil},
	{Name: "milestone_progress", Type: sqlite.SQLITE_FLOAT, NotNull: false, Hidden: false, Filters: nil},
	{Name: "issue_number", Type: sqlite.SQLITE_INTEGER, NotNull: false, Hidden: false, Filters: nil},
	{Name: "participant_count", Type: sqlite.SQLITE_INTEGER, NotNull: false, Hidden: false, Filters: nil},
	{Name: "published_at", Type: sqlite.SQLITE_TEXT, NotNull: false, Hidden: false, Filters: nil},
	{Name: "reaction_count", Type: sqlite.SQLITE_INTEGER, NotNull: false, Hidden: false, Filters: nil},
	{Name: "state", Type: sqlite.SQLITE_TEXT, NotNull: false, Hidden: false, Filters: nil},
	{Name: "title", Type: sqlite.SQLITE_TEXT, NotNull: false, Hidden: false, Filters: nil},
	{Name: "updated_at", Type: sqlite.SQLITE_TEXT, NotNull: false, Hidden: false, Filters: nil, OrderBy: vtab.ASC | vtab.DESC},
	{Name: "url", Type: sqlite.SQLITE_TEXT, NotNull: false, Hidden: false, Filters: nil},
	{Name: "user_edits_count", Type: sqlite.SQLITE_INTEGER, NotNull: false, Hidden: false, Filters: nil},
	{Name: "viewer_can_react", Type: sqlite.SQLITE_INTEGER, NotNull: false, Hidden: false, Filters: nil},
	{Name: "viewer_can_subscribe", Type: sqlite.SQLITE_INTEGER, NotNull: false, Hidden: false, Filters: nil},
	{Name: "viewer_can_update", Type: sqlite.SQLITE_INTEGER, NotNull: false, Hidden: false, Filters: nil},
	{Name: "viewer_did_author", Type: sqlite.SQLITE_INTEGER, NotNull: false, Hidden: false, Filters: nil},
	{Name: "viewer_subscription", Type: sqlite.SQLITE_TEXT, NotNull: false, Hidden: false, Filters: nil},
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
			switch order.ColumnIndex {
			case 8:
				issueOrder.Field = githubv4.IssueOrderFieldComments
			case 9:
				issueOrder.Field = githubv4.IssueOrderFieldCreatedAt
			case 27:
				issueOrder.Field = githubv4.IssueOrderFieldUpdatedAt
			}
			issueOrder.Direction = orderByToGitHubOrder(order.Desc)
		}

		return &iterIssues{fullNameOrOwner, name, opts.Client(), -1, nil, opts.RateLimiter, issueOrder, opts.PerPage}, nil
	})
}
