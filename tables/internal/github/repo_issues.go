package github

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/augmentable-dev/vtab"
	"github.com/shurcooL/githubv4"
	"go.riyazali.net/sqlite"
	"golang.org/x/oauth2"
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
}

// repoOwnerAndName returns the "owner" and "name" (respective return values) or an error
// given the inputs to the iterator. This allows for both `SELECT * FROM github_stargazers('askgitdev/starq')`
// and `SELECT * FROM github_stargazers('askgitdev', 'starq')
func (i *iterIssues) repoOwnerAndName() (string, string, error) {
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
		ctx.ResultText(fmt.Sprint((i.results.Edges[i.current].Node.Closed)))
	case 7:
		t := i.results.Edges[i.current].Node.ClosedAt
		if t.IsZero() {
			ctx.ResultText(" ")
		} else {
			ctx.ResultText(t.Format(time.RFC3339Nano))
		}
	case 8:
		ctx.ResultInt(i.results.Edges[i.current].Node.Comments.TotalCount)
	case 9:
		t := i.results.Edges[i.current].Node.CreatedAt
		if t.IsZero() {
			ctx.ResultText(" ")
		} else {
			ctx.ResultText(t.Format(time.RFC3339Nano))
		}
	case 10:
		ctx.ResultText(fmt.Sprint(i.results.Edges[i.current].Node.CreatedViaEmail))
	case 11:
		ctx.ResultInt(i.results.Edges[i.current].Node.DatabaseId)
	case 12:
		ctx.ResultText(i.results.Edges[i.current].Node.Editor.Login)
	case 13:
		ctx.ResultText(i.results.Edges[i.current].Node.Editor.URL)
	case 14:
		ctx.ResultText(fmt.Sprint(i.results.Edges[i.current].Node.IncludesCreatedEdit))
	case 15:
		ctx.ResultText(fmt.Sprint(i.results.Edges[i.current].Node.IsReadByViewer))
	case 16:
		ctx.ResultInt(i.results.Edges[i.current].Node.Labels.TotalCount)
	case 17:
		t := i.results.Edges[i.current].Node.LastEditedAt
		if t.IsZero() {
			ctx.ResultText("")
		} else {
			ctx.ResultText(t.Format(time.RFC3339Nano))
		}
	case 18:
		ctx.ResultText(fmt.Sprint(i.results.Edges[i.current].Node.Locked))
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
			ctx.ResultText("")
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
			ctx.ResultText("")
		} else {
			ctx.ResultText(t.Format(time.RFC3339Nano))
		}
	case 28:
		ctx.ResultText(i.results.Edges[i.current].Node.Url.String())
	case 29:
		ctx.ResultInt(i.results.Edges[i.current].Node.UserContentEdits.TotalCount)
	case 30:
		ctx.ResultText(fmt.Sprint(i.results.Edges[i.current].Node.ViewerCanReact))
	case 31:
		ctx.ResultText(fmt.Sprint(i.results.Edges[i.current].Node.ViewerCanSubscribe))
	case 32:
		ctx.ResultText(fmt.Sprint(i.results.Edges[i.current].Node.ViewerCanUpdate))
	case 33:
		ctx.ResultText(fmt.Sprint(i.results.Edges[i.current].Node.ViewerDidAuthor))
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

			owner, name, err := i.repoOwnerAndName()
			if err != nil {
				return nil, err
			}

			var cursor *githubv4.String
			if i.results != nil {
				cursor = i.results.EndCursor
			}

			results, err := fetchIssues(context.Background(), &fetchIssuesOptions{i.client, owner, name, 100, cursor, i.issueOrder})
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
	{Name: "owner", Type: sqlite.SQLITE_TEXT, NotNull: true, Hidden: true, Filters: []*vtab.ColumnFilter{{Op: sqlite.INDEX_CONSTRAINT_EQ, Required: true, OmitCheck: true}}, OrderBy: vtab.ASC | vtab.DESC},
	{Name: "reponame", Type: sqlite.SQLITE_TEXT, NotNull: true, Hidden: true, Filters: []*vtab.ColumnFilter{{Op: sqlite.INDEX_CONSTRAINT_EQ}}, OrderBy: vtab.NONE},
	{Name: "author_login", Type: sqlite.SQLITE_TEXT, NotNull: false, Hidden: false, Filters: nil, OrderBy: vtab.ASC | vtab.DESC},
	{Name: "author_url", Type: sqlite.SQLITE_TEXT, NotNull: false, Hidden: false, Filters: nil, OrderBy: vtab.ASC | vtab.DESC},
	{Name: "body", Type: sqlite.SQLITE_TEXT, NotNull: false, Hidden: false, Filters: nil, OrderBy: vtab.NONE},
	{Name: "body_text", Type: sqlite.SQLITE_TEXT, NotNull: false, Hidden: false, Filters: nil, OrderBy: vtab.NONE},
	{Name: "closed", Type: sqlite.SQLITE_TEXT, NotNull: false, Hidden: false, Filters: nil, OrderBy: vtab.ASC | vtab.DESC},
	{Name: "closed_at", Type: sqlite.SQLITE_TEXT, NotNull: false, Hidden: false, Filters: nil, OrderBy: vtab.ASC | vtab.DESC},
	{Name: "comment_count", Type: sqlite.SQLITE_INTEGER, NotNull: false, Hidden: false, Filters: nil, OrderBy: vtab.ASC | vtab.DESC},
	{Name: "created_at", Type: sqlite.SQLITE_TEXT, NotNull: false, Hidden: false, Filters: nil, OrderBy: vtab.ASC | vtab.DESC},
	{Name: "created_via_email", Type: sqlite.SQLITE_TEXT, NotNull: false, Hidden: false, Filters: nil, OrderBy: vtab.NONE},
	{Name: "database_id", Type: sqlite.SQLITE_TEXT, NotNull: false, Hidden: false, Filters: nil, OrderBy: vtab.NONE},
	{Name: "editor_login", Type: sqlite.SQLITE_TEXT, NotNull: false, Hidden: false, Filters: nil, OrderBy: vtab.ASC | vtab.DESC},
	{Name: "editor_url", Type: sqlite.SQLITE_TEXT, NotNull: false, Hidden: false, Filters: nil, OrderBy: vtab.NONE},
	{Name: "includes_created_edit", Type: sqlite.SQLITE_TEXT, NotNull: false, Hidden: false, Filters: nil, OrderBy: vtab.NONE},
	{Name: "is_read_by_viewer", Type: sqlite.SQLITE_TEXT, NotNull: false, Hidden: false, Filters: nil, OrderBy: vtab.ASC | vtab.DESC},
	{Name: "label_count", Type: sqlite.SQLITE_INTEGER, NotNull: false, Hidden: false, Filters: nil, OrderBy: vtab.ASC | vtab.DESC},
	{Name: "last_edited_at", Type: sqlite.SQLITE_TEXT, NotNull: false, Hidden: false, Filters: nil, OrderBy: vtab.NONE},
	{Name: "locked", Type: sqlite.SQLITE_TEXT, NotNull: false, Hidden: false, Filters: nil, OrderBy: vtab.ASC | vtab.DESC},
	{Name: "milestone_count", Type: sqlite.SQLITE_INTEGER, NotNull: false, Hidden: false, Filters: nil, OrderBy: vtab.ASC | vtab.DESC},
	{Name: "milestone_progress", Type: sqlite.SQLITE_FLOAT, NotNull: false, Hidden: false, Filters: nil, OrderBy: vtab.ASC | vtab.DESC},
	{Name: "issue_number", Type: sqlite.SQLITE_INTEGER, NotNull: false, Hidden: false, Filters: nil, OrderBy: vtab.ASC | vtab.DESC},
	{Name: "participant_count", Type: sqlite.SQLITE_INTEGER, NotNull: false, Hidden: false, Filters: nil, OrderBy: vtab.ASC | vtab.DESC},
	{Name: "published_at", Type: sqlite.SQLITE_TEXT, NotNull: false, Hidden: false, Filters: nil, OrderBy: vtab.ASC | vtab.DESC},
	{Name: "reaction_count", Type: sqlite.SQLITE_INTEGER, NotNull: false, Hidden: false, Filters: nil, OrderBy: vtab.ASC | vtab.DESC},
	{Name: "state", Type: sqlite.SQLITE_TEXT, NotNull: false, Hidden: false, Filters: nil, OrderBy: vtab.NONE},
	{Name: "title", Type: sqlite.SQLITE_TEXT, NotNull: false, Hidden: false, Filters: nil, OrderBy: vtab.ASC | vtab.DESC},
	{Name: "updated_at", Type: sqlite.SQLITE_TEXT, NotNull: false, Hidden: false, Filters: nil, OrderBy: vtab.ASC | vtab.DESC},
	{Name: "url", Type: sqlite.SQLITE_TEXT, NotNull: false, Hidden: false, Filters: nil, OrderBy: vtab.ASC | vtab.DESC},
	{Name: "user_edits_count", Type: sqlite.SQLITE_INTEGER, NotNull: false, Hidden: false, Filters: nil, OrderBy: vtab.NONE},
	{Name: "viewer_can_react", Type: sqlite.SQLITE_TEXT, NotNull: false, Hidden: false, Filters: nil, OrderBy: vtab.NONE},
	{Name: "viewer_can_subscripe", Type: sqlite.SQLITE_TEXT, NotNull: false, Hidden: false, Filters: nil, OrderBy: vtab.NONE},
	{Name: "viewer_can_update", Type: sqlite.SQLITE_TEXT, NotNull: false, Hidden: false, Filters: nil, OrderBy: vtab.NONE},
	{Name: "viewer_did_author", Type: sqlite.SQLITE_TEXT, NotNull: false, Hidden: false, Filters: nil, OrderBy: vtab.NONE},
	{Name: "viewerSubscription", Type: sqlite.SQLITE_TEXT, NotNull: false, Hidden: false, Filters: nil, OrderBy: vtab.NONE},
}

func NewIssuesModule(githubToken string, rateLimiter *rate.Limiter) sqlite.Module {
	httpClient := oauth2.NewClient(context.Background(), oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: githubToken},
	))
	client := githubv4.NewClient(httpClient)

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
		issueOrder := &githubv4.IssueOrder{
			Field:     githubv4.IssueOrderFieldCreatedAt,
			Direction: githubv4.OrderDirectionDesc,
		}
		for _, order := range orders { //adding this for loop for scalability. might need to order the data by more columns in the future.
			switch order.ColumnIndex {
			case 21:
				if !order.Desc {
					issueOrder.Direction = githubv4.OrderDirectionAsc
				}
			}
		}

		return &iterIssues{fullNameOrOwner, name, client, -1, nil, rateLimiter, issueOrder}, nil
	})
}
