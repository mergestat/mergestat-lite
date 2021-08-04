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

type pullRequest struct {
	ActiveLockReason githubv4.LockReason
	Assignees        struct {
		Nodes []user
	} `graphql:"assignees(first:100)"`
	Additions         githubv4.Int
	Author            user
	AuthorAssociation githubv4.CommentAuthorAssociation

	BodyText    githubv4.String
	BaseRefOid  githubv4.GitObjectID
	BaseRefName string
	BaseRef     struct {
		Name string
	}
	BaseRepository struct {
		Name  string
		Owner struct {
			Login string
		}
	}
	Body string
	//BodyHTML           string
	//CanBeRebased       bool
	ChecksResourcePath githubv4.URI
	ChecksURL          githubv4.URI

	Comments struct {
		TotalCount int
	} `graphql:"comments(first:100)"`
	Commits struct {
		TotalCount int
	} `graphql:"commits(first:100)"`
	ChangedFiles    int
	Closed          bool
	ClosedAt        githubv4.DateTime
	CreatedAt       githubv4.DateTime
	CreatedViaEmail bool
	Deletions       int
	DatabaseID      githubv4.Int
	Editor          struct {
		Login string
	}
	Files struct {
		TotalCount int
	}
	HeadRepository struct {
		Name string
	}
	HeadRepositoryOwner struct {
		Login string
	}
	HeadRefOid  githubv4.GitObjectID
	HeadRefName string
	HeadRef     struct {
		Name string
	}
	IncludesCreatedEdit bool
	IsCrossRepository   bool
	IsDraft             bool
	Labels              struct {
		// Nodes []struct {
		// 	Name string
		// }
		TotalCount int
	} `graphql:"labels(first:100)"`
	LastEditedAt githubv4.DateTime
	Locked       githubv4.Boolean
	MergedAt     githubv4.DateTime
	MergeCommit  struct {
		Oid githubv4.GitObjectID
	}
	Merged              githubv4.Boolean
	Mergeable           githubv4.MergeableState
	MergedBy            user
	MaintainerCanModify githubv4.Boolean
	//MergeStateStatuses  string
	Milestone struct {
		Number int
	}
	Number       githubv4.Int
	Participants struct {
		TotalCount int
	}
	Permalink      githubv4.URI
	PublishedAt    githubv4.DateTime
	ReviewDecision githubv4.PullRequestReviewDecision
	ReviewRequests struct {
		Nodes []struct {
			RequestedReviewer []interface{}
		}
	} `graphql:"reviewRequests(first:100)"`
	ReviewThreads struct {
		TotalCount int
	}
	Reviews struct {
		TotalCount int
	}
	State            githubv4.PullRequestState
	Title            githubv4.String
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
	PROrder     *githubv4.PullRequestOrder
}

type fetchPRResults struct {
	Edges       []*prEdge
	HasNextPage bool
	EndCursor   *githubv4.String
}

type prEdge struct {
	Cursor string
	Node   pullRequest
}

func fetchPR(ctx context.Context, input *fetchPROptions) (*fetchPRResults, error) {
	var PRQuery struct {
		Repository struct {
			Owner struct {
				Login string
			}
			Name        string
			pullRequest struct {
				Edges    []*prEdge
				PageInfo struct {
					EndCursor   githubv4.String
					HasNextPage bool
				}
			} `graphql:"PR(first: $perpage, after: $prcursor, orderBy: $prorder)"`
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
		PRQuery.Repository.pullRequest.Edges,
		PRQuery.Repository.pullRequest.PageInfo.HasNextPage,
		&PRQuery.Repository.pullRequest.PageInfo.EndCursor,
	}, nil
}

type iterPR struct {
	fullNameOrOwner string
	name            string
	client          *githubv4.Client
	current         int
	results         *fetchPRResults
	rateLimiter     *rate.Limiter
	issueOrder      *githubv4.IssueOrder
}

func (i *iterPR) Column(ctx *sqlite.Context, c int) error {
	switch c {
	case 0:
		ctx.ResultText(i.fullNameOrOwner)
	case 1:
		ctx.ResultText(i.name)
	case 2:
		ctx.ResultText(i.results.Edges[i.current].Node.Author.Login)
	case 3:
		assigned := ""
		for _, assign := range i.results.Edges[i.current].Node.Assignees.Nodes {
			assigned += assign.Login + " "
		}
		ctx.ResultText(assigned)
	case 4:
		ctx.ResultText(i.results.Edges[i.current].Node.Author.URL)
	case 5:
		ctx.ResultText(string(i.results.Edges[i.current].Node.AuthorAssociation))
	case 7:
		ctx.ResultText(i.results.Edges[i.current].Node.Body)
	case 8:
		ctx.ResultText(string(i.results.Edges[i.current].Node.BodyText))
	case 9:
		ctx.ResultText(string(i.results.Edges[i.current].Node.BaseRefOid))
	case 10:
		ctx.ResultText(i.results.Edges[i.current].Node.BaseRepository.Name)
	case 11:
		ctx.ResultText(i.results.Edges[i.current].Node.BaseRepository.Owner.Login)
	case 12:
		ctx.ResultText(i.results.Edges[i.current].Node.ChecksResourcePath.String())
	case 13:
		ctx.ResultText(i.results.Edges[i.current].Node.ChecksURL.Opaque)
	case 14:
		ctx.ResultInt(i.results.Edges[i.current].Node.Comments.TotalCount)
	case 15:
		ctx.ResultInt(i.results.Edges[i.current].Node.Commits.TotalCount)
	case 16:
		ctx.ResultInt(t1f0(i.results.Edges[i.current].Node.Closed))

	case 17:
		t := i.results.Edges[i.current].Node.ClosedAt
		if t.IsZero() {
			ctx.ResultNull()
		} else {
			ctx.ResultText(t.Format(time.RFC3339Nano))
		}
	case 18:
		t := i.results.Edges[i.current].Node.CreatedAt
		if t.IsZero() {
			ctx.ResultNull()
		} else {
			ctx.ResultText(t.Format(time.RFC3339Nano))
		}
	case 19:
		ctx.ResultInt(t1f0(i.results.Edges[i.current].Node.CreatedViaEmail))
	case 20:
		ctx.ResultInt(i.results.Edges[i.current].Node.Deletions)
	case 21:
		ctx.ResultInt(int(i.results.Edges[i.current].Node.DatabaseID))
	case 22:
		ctx.ResultText(i.results.Edges[i.current].Node.Editor.Login)
	case 23:
		ctx.ResultInt(int(i.results.Edges[i.current].Node.Files.TotalCount))
	case 24:
		ctx.ResultText(i.results.Edges[i.current].Node.HeadRepository.Name)
	case 25:
		ctx.ResultText(i.results.Edges[i.current].Node.HeadRepositoryOwner.Login)
	case 26:
		ctx.ResultText(string(i.results.Edges[i.current].Node.HeadRefOid))
	case 27:
		ctx.ResultText(i.results.Edges[i.current].Node.HeadRefName)
	case 28:
		ctx.ResultText(i.results.Edges[i.current].Node.HeadRef.Name)
	case 29:
		ctx.ResultInt(t1f0(i.results.Edges[i.current].Node.IncludesCreatedEdit))
	case 30:
		ctx.ResultInt(t1f0(i.results.Edges[i.current].Node.IsCrossRepository))
	case 31:
		ctx.ResultInt(t1f0(i.results.Edges[i.current].Node.IsDraft))
	case 32:
		ctx.ResultInt(i.results.Edges[i.current].Node.Labels.TotalCount)
	case 33:
		t := i.results.Edges[i.current].Node.LastEditedAt
		if t.IsZero() {
			ctx.ResultNull()
		} else {
			ctx.ResultText(t.Format(time.RFC3339Nano))
		}
	case 34:
		ctx.ResultInt(t1f0(i.results.Edges[i.current].Node.Locked))
	case 35:
		t := i.results.Edges[i.current].Node.MergedAt
		if t.IsZero() {
			ctx.ResultNull()
		} else {
			ctx.ResultText(t.Format(time.RFC3339Nano))
		}
	case 36:
		ctx.ResultText(i.results.Edges[i.current].Node.MergedBy.Login)
	case 37:
		ctx.ResultInt(t1f0(bool(i.results.Edges[i.current].Node.Merged)))
	case 38:
		ctx.ResultInt(t1f0(bool(i.results.Edges[i.current].Node.MaintainerCanModify)))
	case 39:
		ctx.ResultInt(i.results.Edges[i.current].Node.Milestone.Number)
	case 40:
		ctx.ResultInt(int(i.results.Edges[i.current].Node.Number))
	case 41:
		ctx.ResultInt(i.results.Edges[i.current].Node.Participants.TotalCount)
	case 42:
		ctx.ResultText(i.results.Edges[i.current].Node.Permalink.Scheme)
	case 43:
		t := i.results.Edges[i.current].Node.PublishedAt
		if t.IsZero() {
			ctx.ResultNull()
		} else {
			ctx.ResultText(t.Format(time.RFC3339Nano))
		}
	case 44:
		ctx.ResultText(string(i.results.Edges[i.current].Node.ReviewDecision))
	case 45:
		reviewRequests := ""
		for _, request := range i.results.Edges[i.current].Node.ReviewRequests.Nodes {
			reviewRequests += fmt.Sprint(request.RequestedReviewer) + " "
		}
		ctx.ResultText(reviewRequests)
	case 46:
		ctx.ResultInt(i.results.Edges[i.current].Node.ReviewThreads.TotalCount)
	case 47:
		ctx.ResultInt(i.results.Edges[i.current].Node.Reviews.TotalCount)
	case 48:
		ctx.ResultText(string(i.results.Edges[i.current].Node.State))
	case 49:
		ctx.ResultText(string(i.results.Edges[i.current].Node.Title))
	case 50:
		t := i.results.Edges[i.current].Node.UpdatedAt
		if t.IsZero() {
			ctx.ResultNull()
		} else {
			ctx.ResultText(t.Format(time.RFC3339Nano))
		}
	case 51:
		ctx.ResultText(i.results.Edges[i.current].Node.Url.String())
	case 52:
		ctx.ResultInt(i.results.Edges[i.current].Node.UserContentEdits.TotalCount)
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

			results, err := fetchPR(context.Background(), &fetchPROptions{i.client, owner, name, 100, cursor, i.issueOrder})
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
	{Name: "author_login", Type: sqlite.SQLITE_TEXT, NotNull: false, Hidden: false, Filters: nil},
	{Name: "assignees", Type: sqlite.SQLITE_TEXT, NotNull: false, Hidden: false, Filters: nil, OrderBy: vtab.ASC | vtab.DESC},
	{Name: "author_url", Type: sqlite.SQLITE_TEXT, NotNull: false, Hidden: false, Filters: nil},
	{Name: "author_association", Type: sqlite.SQLITE_TEXT, NotNull: false, Hidden: false, Filters: nil},
	{Name: "body", Type: sqlite.SQLITE_TEXT, NotNull: false, Hidden: false, Filters: nil},
	{Name: "body_text", Type: sqlite.SQLITE_TEXT, NotNull: false, Hidden: false, Filters: nil},
	{Name: "base_ref_oid", Type: sqlite.SQLITE_TEXT, NotNull: false, Hidden: false, Filters: nil},
	{Name: "base_repository_name", Type: sqlite.SQLITE_TEXT, NotNull: false, Hidden: false, Filters: nil},
	{Name: "base_repository_owner_login", Type: sqlite.SQLITE_TEXT, NotNull: false, Hidden: false, Filters: nil},
	{Name: "checks_resource_path", Type: sqlite.SQLITE_TEXT, NotNull: false, Hidden: false, Filters: nil},
	{Name: "checks_url", Type: sqlite.SQLITE_INTEGER, NotNull: false, Hidden: false, Filters: nil},
	{Name: "comment_count", Type: sqlite.SQLITE_INTEGER, NotNull: false, Hidden: false, Filters: nil, OrderBy: vtab.ASC | vtab.DESC},
	{Name: "commit_count", Type: sqlite.SQLITE_INTEGER, NotNull: false, Hidden: false, Filters: nil, OrderBy: vtab.ASC | vtab.DESC},
	{Name: "closed", Type: sqlite.SQLITE_INTEGER, NotNull: false, Hidden: false, Filters: nil},
	{Name: "closed_at", Type: sqlite.SQLITE_TEXT, NotNull: false, Hidden: false, Filters: nil},
	{Name: "created_at", Type: sqlite.SQLITE_TEXT, NotNull: false, Hidden: false, Filters: nil, OrderBy: vtab.ASC | vtab.DESC},
	{Name: "created_via_email", Type: sqlite.SQLITE_INTEGER, NotNull: false, Hidden: false, Filters: nil},
	{Name: "deletions", Type: sqlite.SQLITE_INTEGER, NotNull: false, Hidden: false, Filters: nil, OrderBy: vtab.ASC | vtab.DESC},
	{Name: "database_id", Type: sqlite.SQLITE_INTEGER, NotNull: false, Hidden: false, Filters: nil},
	{Name: "editor_login", Type: sqlite.SQLITE_TEXT, NotNull: false, Hidden: false, Filters: nil},
	{Name: "file_count", Type: sqlite.SQLITE_INTEGER, NotNull: false, Hidden: false, Filters: nil, OrderBy: vtab.ASC | vtab.DESC},
	{Name: "head_repository", Type: sqlite.SQLITE_TEXT, NotNull: false, Hidden: false, Filters: nil},
	{Name: "head_repository_owner", Type: sqlite.SQLITE_TEXT, NotNull: false, Hidden: false, Filters: nil},
	{Name: "head_ref_oid", Type: sqlite.SQLITE_TEXT, NotNull: false, Hidden: false, Filters: nil},
	{Name: "head_ref_name", Type: sqlite.SQLITE_TEXT, NotNull: false, Hidden: false, Filters: nil},
	{Name: "head_ref", Type: sqlite.SQLITE_TEXT, NotNull: false, Hidden: false, Filters: nil},
	{Name: "includes_created_edit", Type: sqlite.SQLITE_INTEGER, NotNull: false, Hidden: false, Filters: nil},
	{Name: "is_cross_repository", Type: sqlite.SQLITE_INTEGER, NotNull: false, Hidden: false, Filters: nil},
	{Name: "is_draft", Type: sqlite.SQLITE_INTEGER, NotNull: false, Hidden: false, Filters: nil},
	{Name: "label_count", Type: sqlite.SQLITE_INTEGER, NotNull: false, Hidden: false, Filters: nil},
	{Name: "last_edited_at", Type: sqlite.SQLITE_TEXT, NotNull: false, Hidden: false, Filters: nil},
	{Name: "locked", Type: sqlite.SQLITE_INTEGER, NotNull: false, Hidden: false, Filters: nil},
	{Name: "merged_at", Type: sqlite.SQLITE_TEXT, NotNull: false, Hidden: false, Filters: nil},
	{Name: "merge_commit_sha", Type: sqlite.SQLITE_TEXT, NotNull: false, Hidden: false, Filters: nil},
	{Name: "merged", Type: sqlite.SQLITE_INTEGER, NotNull: false, Hidden: false, Filters: nil},
	{Name: "mergeable", Type: sqlite.SQLITE_INTEGER, NotNull: false, Hidden: false, Filters: nil},
	{Name: "merged_by", Type: sqlite.SQLITE_TEXT, NotNull: false, Hidden: false, Filters: nil},
	{Name: "maintainer_can_modify", Type: sqlite.SQLITE_TEXT, NotNull: false, Hidden: false, Filters: nil},
	{Name: "milestone_number", Type: sqlite.SQLITE_INTEGER, NotNull: false, Hidden: false, Filters: nil},
	{Name: "pr_number", Type: sqlite.SQLITE_INTEGER, NotNull: false, Hidden: false, Filters: nil},
	{Name: "participant_count", Type: sqlite.SQLITE_INTEGER, NotNull: false, Hidden: false, Filters: nil},
	{Name: "permalink", Type: sqlite.SQLITE_TEXT, NotNull: false, Hidden: false, Filters: nil},
	{Name: "published_at", Type: sqlite.SQLITE_TEXT, NotNull: false, Hidden: false, Filters: nil},
	{Name: "review_decision", Type: sqlite.SQLITE_TEXT, NotNull: false, Hidden: false, Filters: nil},
	{Name: "review_requests", Type: sqlite.SQLITE_TEXT, NotNull: false, Hidden: false, Filters: nil},
	{Name: "review_threads", Type: sqlite.SQLITE_INTEGER, NotNull: false, Hidden: false, Filters: nil},
	{Name: "review_count", Type: sqlite.SQLITE_INTEGER, NotNull: false, Hidden: false, Filters: nil},
	{Name: "state", Type: sqlite.SQLITE_TEXT, NotNull: false, Hidden: false, Filters: nil},
	{Name: "title", Type: sqlite.SQLITE_TEXT, NotNull: false, Hidden: false, Filters: nil},
	{Name: "updated_at", Type: sqlite.SQLITE_TEXT, NotNull: false, Hidden: false, Filters: nil, OrderBy: vtab.ASC | vtab.DESC},
	{Name: "url", Type: sqlite.SQLITE_TEXT, NotNull: false, Hidden: false, Filters: nil},
	{Name: "user_edits_count", Type: sqlite.SQLITE_INTEGER, NotNull: false, Hidden: false, Filters: nil},
}

func NewPRModule(opts *Options) sqlite.Module {
	return vtab.NewTableFunc("github_repo_PR", PRCols, func(constraints []*vtab.Constraint, orders []*sqlite.OrderBy) (vtab.Iterator, error) {
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

		return &iterPR{fullNameOrOwner, name, opts.Client(), -1, nil, opts.RateLimiter, issueOrder}, nil
	})
}
