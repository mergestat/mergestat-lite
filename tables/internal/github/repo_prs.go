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

type pullRequest struct {
	ActiveLockReason  githubv4.LockReason
	Additions         int
	Author            user
	AuthorAssociation githubv4.CommentAuthorAssociation
	BaseRefOid        githubv4.GitObjectID
	BaseRefName       string
	BaseRepository    struct {
		NameWithOwner string
	}
	Body         string
	BodyText     string
	ChangedFiles int
	Closed       bool
	ClosedAt     githubv4.DateTime
	Comments     struct {
		TotalCount int
	}
	Commits struct {
		TotalCount int
	}
	CreatedAt       githubv4.DateTime
	CreatedViaEmail bool
	DatabaseID      int
	Deletions       int
	Editor          struct {
		Login string
	}
	HeadRefName    string
	HeadRefOid     githubv4.GitObjectID
	HeadRepository struct {
		NameWithOwner string
	}
	IsCrossRepository bool
	IsDraft           bool
	Labels            struct {
		TotalCount int
	}
	LastEditedAt        githubv4.DateTime
	Locked              bool
	MaintainerCanModify bool
	Mergeable           githubv4.MergeableState
	Merged              bool
	MergedAt            githubv4.DateTime
	MergedBy            user
	Number              int
	Participants        struct {
		TotalCount int
	}
	PublishedAt    githubv4.DateTime
	ReviewDecision githubv4.PullRequestReviewDecision
	ReviewRequests struct {
		TotalCount int
	}
	ReviewThreads struct {
		TotalCount int
	}
	Reviews struct {
		TotalCount int
	}
	State            githubv4.PullRequestState
	Title            string
	UpdatedAt        githubv4.DateTime
	Url              githubv4.URI
	UserContentEdits struct {
		TotalCount int
	}
}

type fetchPROptions struct {
	Client      *githubv4.Client
	Owner       string
	Name        string
	PerPage     int
	StartCursor *githubv4.String
	PROrder     *githubv4.IssueOrder
}

type fetchPRResults struct {
	Edges       []*pullRequest
	HasNextPage bool
	EndCursor   *githubv4.String
}

func fetchPR(ctx context.Context, input *fetchPROptions) (*fetchPRResults, error) {
	var PRQuery struct {
		Repository struct {
			Owner struct {
				Login string
			}
			Name         string
			PullRequests struct {
				Nodes    []*pullRequest
				PageInfo struct {
					EndCursor   githubv4.String
					HasNextPage bool
				}
			} `graphql:"pullRequests(first: $perpage, after: $prcursor, orderBy: $prorder)"`
		} `graphql:"repository(owner: $owner, name: $name)"`
	}
	variables := map[string]interface{}{
		"owner":    githubv4.String(input.Owner),
		"name":     githubv4.String(input.Name),
		"perpage":  githubv4.Int(input.PerPage),
		"prcursor": (*githubv4.String)(input.StartCursor),
		"prorder":  input.PROrder,
	}

	err := input.Client.Query(ctx, &PRQuery, variables)

	if err != nil {
		return nil, err
	}

	return &fetchPRResults{
		PRQuery.Repository.PullRequests.Nodes,
		PRQuery.Repository.PullRequests.PageInfo.HasNextPage,
		&PRQuery.Repository.PullRequests.PageInfo.EndCursor,
	}, nil
}

type iterPR struct {
	fullNameOrOwner string
	name            string
	client          *githubv4.Client
	current         int
	results         *fetchPRResults
	rateLimiter     *rate.Limiter
	prOrder         *githubv4.IssueOrder
}

func (i *iterPR) Column(ctx *sqlite.Context, c int) error {
	current := i.results.Edges[i.current]
	switch c {
	case 0:
		ctx.ResultText(i.fullNameOrOwner)
	case 1:
		ctx.ResultText(i.name)
	case 2:
		ctx.ResultInt(int(current.Additions))
	case 3:
		ctx.ResultText(current.Author.Login)
	case 4:
		ctx.ResultText(string(current.AuthorAssociation))
	case 5:
		ctx.ResultText(string(current.BaseRefOid))
	case 6:
		ctx.ResultText(current.BaseRefName)
	case 7:
		ctx.ResultText(current.BaseRepository.NameWithOwner)
	case 8:
		ctx.ResultText(current.Body)
	case 9:
		ctx.ResultText(current.BodyText)
	case 10:
		ctx.ResultInt(current.ChangedFiles)
	case 11:
		ctx.ResultInt(t1f0(current.Closed))
	case 12:
		t := current.ClosedAt
		if t.IsZero() {
			ctx.ResultNull()
		} else {
			ctx.ResultText(t.Format(time.RFC3339Nano))
		}
	case 13:
		ctx.ResultInt(current.Comments.TotalCount)
	case 14:
		ctx.ResultInt(current.Commits.TotalCount)
	case 15:
		t := current.CreatedAt
		if t.IsZero() {
			ctx.ResultNull()
		} else {
			ctx.ResultText(t.Format(time.RFC3339Nano))
		}
	case 16:
		ctx.ResultInt(t1f0(current.CreatedViaEmail))
	case 17:
		ctx.ResultInt(current.DatabaseID)
	case 18:
		ctx.ResultInt(current.Deletions)
	case 19:
		ctx.ResultText(current.Editor.Login)
	case 20:
		ctx.ResultText(current.HeadRefName)
	case 21:
		ctx.ResultText(string(current.HeadRefOid))
	case 22:
		ctx.ResultText(string(current.HeadRepository.NameWithOwner))
	case 23:
		ctx.ResultInt(t1f0(current.IsDraft))
	case 24:
		ctx.ResultInt(current.Labels.TotalCount)
	case 25:
		t := current.LastEditedAt
		if t.IsZero() {
			ctx.ResultNull()
		} else {
			ctx.ResultText(t.Format(time.RFC3339Nano))
		}
	case 26:
		ctx.ResultInt(t1f0(current.Locked))
	case 27:
		ctx.ResultInt(t1f0(current.MaintainerCanModify))
	case 28:
		ctx.ResultText(string(current.Mergeable))
	case 29:
		ctx.ResultInt(t1f0(current.Merged))
	case 30:
		t := current.MergedAt
		if t.IsZero() {
			ctx.ResultNull()
		} else {
			ctx.ResultText(t.Format(time.RFC3339Nano))
		}
	case 31:
		ctx.ResultText(current.MergedBy.Login)
	case 32:
		ctx.ResultInt(current.Number)
	case 33:
		ctx.ResultInt(current.Participants.TotalCount)
	case 34:
		t := current.PublishedAt
		if t.IsZero() {
			ctx.ResultNull()
		} else {
			ctx.ResultText(t.Format(time.RFC3339Nano))
		}
	case 35:
		ctx.ResultText(string(current.ReviewDecision))
	case 36:
		ctx.ResultInt(current.ReviewRequests.TotalCount)
	case 37:
		ctx.ResultInt(current.ReviewThreads.TotalCount)
	case 38:
		ctx.ResultInt(current.Reviews.TotalCount)
	case 39:
		ctx.ResultText(string(current.State))
	case 40:
		ctx.ResultText(current.Title)
	case 41:
		t := current.UpdatedAt
		if t.IsZero() {
			ctx.ResultNull()
		} else {
			ctx.ResultText(t.Format(time.RFC3339Nano))
		}
	case 42:
		ctx.ResultText(current.Url.String())
	case 43:
		ctx.ResultInt(current.UserContentEdits.TotalCount)
	}
	return nil
}

func (i *iterPR) Next() (vtab.Row, error) {
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

			results, err := fetchPR(context.Background(), &fetchPROptions{i.client, owner, name, 100, cursor, i.prOrder})
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

var PRCols = []vtab.Column{
	{Name: "owner", Type: sqlite.SQLITE_TEXT, NotNull: true, Hidden: true, Filters: []*vtab.ColumnFilter{{Op: sqlite.INDEX_CONSTRAINT_EQ, Required: true, OmitCheck: true}}},
	{Name: "reponame", Type: sqlite.SQLITE_TEXT, NotNull: true, Hidden: true, Filters: []*vtab.ColumnFilter{{Op: sqlite.INDEX_CONSTRAINT_EQ}}},
	{Name: "additions", Type: sqlite.SQLITE_INTEGER},
	{Name: "author_login", Type: sqlite.SQLITE_TEXT},
	{Name: "author_association", Type: sqlite.SQLITE_TEXT},
	{Name: "base_ref_oid", Type: sqlite.SQLITE_TEXT},
	{Name: "base_ref_name", Type: sqlite.SQLITE_TEXT},
	{Name: "base_repository_name", Type: sqlite.SQLITE_TEXT},
	{Name: "body", Type: sqlite.SQLITE_TEXT},
	{Name: "body_text", Type: sqlite.SQLITE_TEXT},
	{Name: "changed_files", Type: sqlite.SQLITE_INTEGER},
	{Name: "closed", Type: sqlite.SQLITE_INTEGER},
	{Name: "closed_at", Type: sqlite.SQLITE_TEXT},
	{Name: "comment_count", Type: sqlite.SQLITE_INTEGER, OrderBy: vtab.ASC | vtab.DESC},
	{Name: "commit_count", Type: sqlite.SQLITE_INTEGER},
	{Name: "created_at", Type: sqlite.SQLITE_TEXT, OrderBy: vtab.ASC | vtab.DESC},
	{Name: "created_via_email", Type: sqlite.SQLITE_INTEGER},
	{Name: "database_id", Type: sqlite.SQLITE_INTEGER},
	{Name: "deletions", Type: sqlite.SQLITE_INTEGER},
	{Name: "editor_login", Type: sqlite.SQLITE_TEXT},
	{Name: "head_ref_name", Type: sqlite.SQLITE_TEXT},
	{Name: "head_ref_oid", Type: sqlite.SQLITE_TEXT},
	{Name: "head_repository_name", Type: sqlite.SQLITE_TEXT},
	{Name: "is_draft", Type: sqlite.SQLITE_INTEGER},
	{Name: "label_count", Type: sqlite.SQLITE_INTEGER},
	{Name: "last_edited_at", Type: sqlite.SQLITE_TEXT},
	{Name: "locked", Type: sqlite.SQLITE_INTEGER},
	{Name: "maintainer_can_modify", Type: sqlite.SQLITE_TEXT},
	{Name: "mergeable", Type: sqlite.SQLITE_TEXT},
	{Name: "merged", Type: sqlite.SQLITE_INTEGER},
	{Name: "merged_at", Type: sqlite.SQLITE_TEXT},
	{Name: "merged_by", Type: sqlite.SQLITE_TEXT},
	{Name: "number", Type: sqlite.SQLITE_INTEGER},
	{Name: "participant_count", Type: sqlite.SQLITE_INTEGER},
	{Name: "published_at", Type: sqlite.SQLITE_TEXT},
	{Name: "review_decision", Type: sqlite.SQLITE_TEXT},
	{Name: "review_request_count", Type: sqlite.SQLITE_TEXT},
	{Name: "review_thread_count", Type: sqlite.SQLITE_INTEGER},
	{Name: "review_count", Type: sqlite.SQLITE_INTEGER},
	{Name: "state", Type: sqlite.SQLITE_TEXT},
	{Name: "title", Type: sqlite.SQLITE_TEXT},
	{Name: "updated_at", Type: sqlite.SQLITE_TEXT, OrderBy: vtab.ASC | vtab.DESC},
	{Name: "url", Type: sqlite.SQLITE_TEXT},
	{Name: "user_content_edits_count", Type: sqlite.SQLITE_INTEGER},
}

func NewPRModule(opts *Options) sqlite.Module {
	return vtab.NewTableFunc("github_repo_prs", PRCols, func(constraints []*vtab.Constraint, orders []*sqlite.OrderBy) (vtab.Iterator, error) {
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
		var prOrder *githubv4.IssueOrder
		if len(orders) == 1 {
			order := orders[0]
			prOrder = &githubv4.IssueOrder{}
			switch order.ColumnIndex {
			case 13:
				prOrder.Field = githubv4.IssueOrderFieldComments
			case 15:
				prOrder.Field = githubv4.IssueOrderFieldCreatedAt
			case 41:
				prOrder.Field = githubv4.IssueOrderFieldUpdatedAt
			}
			prOrder.Direction = orderByToGitHubOrder(order.Desc)
		}

		return &iterPR{fullNameOrOwner, name, opts.Client(), -1, nil, opts.RateLimiter, prOrder}, nil
	})
}
