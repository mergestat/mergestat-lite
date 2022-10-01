package github

import (
	"context"
	"encoding/json"
	"io"
	"time"

	"github.com/augmentable-dev/vtab"
	"github.com/mergestat/mergestat-lite/extensions/options"
	"github.com/rs/zerolog"
	"github.com/shurcooL/githubv4"
	"go.riyazali.net/sqlite"
)

type pullRequest struct {
	ActiveLockReason githubv4.LockReason
	Additions        int
	Author           struct {
		Login     string
		AvatarUrl *githubv4.URI
		User      struct {
			Name string
		} `graphql:"... on User"`
	}
	AuthorAssociation githubv4.CommentAuthorAssociation
	BaseRefOid        githubv4.GitObjectID
	BaseRefName       string
	BaseRepository    struct {
		NameWithOwner string
	}
	Body         string
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
		Nodes      []struct {
			Name string
		}
	} `graphql:"labels(first: 15)"`
	LastEditedAt        githubv4.DateTime
	Locked              bool
	MaintainerCanModify bool
	Mergeable           githubv4.MergeableState
	Merged              bool
	MergedAt            githubv4.DateTime
	MergedBy            struct {
		Login string
	}
	Number       int
	Participants struct {
		TotalCount int
	}
	PublishedAt    githubv4.DateTime
	ReviewDecision githubv4.PullRequestReviewDecision
	State          githubv4.PullRequestState
	Title          string
	UpdatedAt      githubv4.DateTime
	Url            githubv4.URI
}

type fetchPRResults struct {
	RateLimit   *options.GitHubRateLimitResponse
	Edges       []*pullRequest
	HasNextPage bool
	EndCursor   *githubv4.String
}

func (i *iterPRs) fetchPRs(ctx context.Context, startCursor *githubv4.String) (*fetchPRResults, error) {
	var PRQuery struct {
		RateLimit  *options.GitHubRateLimitResponse
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
		"owner":    githubv4.String(i.owner),
		"name":     githubv4.String(i.name),
		"perpage":  githubv4.Int(i.PerPage),
		"prcursor": startCursor,
		"prorder":  i.prOrder,
	}

	err := i.Client().Query(ctx, &PRQuery, variables)
	if err != nil {
		return nil, err
	}

	return &fetchPRResults{
		RateLimit:   PRQuery.RateLimit,
		Edges:       PRQuery.Repository.PullRequests.Nodes,
		HasNextPage: PRQuery.Repository.PullRequests.PageInfo.HasNextPage,
		EndCursor:   &PRQuery.Repository.PullRequests.PageInfo.EndCursor,
	}, nil
}

type iterPRs struct {
	*Options
	owner   string
	name    string
	current int
	results *fetchPRResults
	prOrder *githubv4.IssueOrder
}

func (i *iterPRs) logger() *zerolog.Logger {
	logger := i.Logger.With().Int("per-page", i.PerPage).Str("owner", i.owner).Str("name", i.name).Logger()
	if i.prOrder != nil {
		logger = logger.With().Str("order_by", string(i.prOrder.Field)).Str("order_dir", string(i.prOrder.Direction)).Logger()
	}
	return &logger
}

func (i *iterPRs) Column(ctx vtab.Context, c int) error {
	current := i.results.Edges[i.current]
	col := prCols[c]

	switch col.Name {
	case "additions":
		ctx.ResultInt(int(current.Additions))
	case "author_login":
		ctx.ResultText(current.Author.Login)
	case "author_avatar_url":
		if current.Author.AvatarUrl != nil {
			ctx.ResultText(current.Author.AvatarUrl.String())
		} else {
			ctx.ResultNull()
		}
	case "author_name":
		ctx.ResultText(current.Author.User.Name)
	case "author_association":
		ctx.ResultText(string(current.AuthorAssociation))
	case "base_ref_oid":
		ctx.ResultText(string(current.BaseRefOid))
	case "base_ref_name":
		ctx.ResultText(current.BaseRefName)
	case "base_repository_name":
		ctx.ResultText(current.BaseRepository.NameWithOwner)
	case "body":
		ctx.ResultText(current.Body)
	case "changed_files":
		ctx.ResultInt(current.ChangedFiles)
	case "closed":
		ctx.ResultInt(t1f0(current.Closed))
	case "closed_at":
		t := current.ClosedAt
		if t.IsZero() {
			ctx.ResultNull()
		} else {
			ctx.ResultText(t.Format(time.RFC3339Nano))
		}
	case "comment_count":
		ctx.ResultInt(current.Comments.TotalCount)
	case "commit_count":
		ctx.ResultInt(current.Commits.TotalCount)
	case "created_at":
		t := current.CreatedAt
		if t.IsZero() {
			ctx.ResultNull()
		} else {
			ctx.ResultText(t.Format(time.RFC3339Nano))
		}
	case "created_via_email":
		ctx.ResultInt(t1f0(current.CreatedViaEmail))
	case "database_id":
		ctx.ResultInt(current.DatabaseID)
	case "deletions":
		ctx.ResultInt(current.Deletions)
	case "editor_login":
		ctx.ResultText(current.Editor.Login)
	case "head_ref_name":
		ctx.ResultText(current.HeadRefName)
	case "head_ref_oid":
		ctx.ResultText(string(current.HeadRefOid))
	case "head_repository_name":
		ctx.ResultText(string(current.HeadRepository.NameWithOwner))
	case "is_draft":
		ctx.ResultInt(t1f0(current.IsDraft))
	case "label_count":
		ctx.ResultInt(current.Labels.TotalCount)
	case "labels":
		labels := make([]string, len(current.Labels.Nodes))
		for l, label := range current.Labels.Nodes {
			labels[l] = label.Name
		}
		js, err := json.Marshal(labels)
		if err != nil {
			i.logger().Err(err).Msgf("could not marshal PR labels")
			ctx.ResultNull()
		} else {
			ctx.ResultText(string(js))
		}
	case "last_edited_at":
		t := current.LastEditedAt
		if t.IsZero() {
			ctx.ResultNull()
		} else {
			ctx.ResultText(t.Format(time.RFC3339Nano))
		}
	case "locked":
		ctx.ResultInt(t1f0(current.Locked))
	case "maintainer_can_modify":
		ctx.ResultInt(t1f0(current.MaintainerCanModify))
	case "mergeable":
		ctx.ResultText(string(current.Mergeable))
	case "merged":
		ctx.ResultInt(t1f0(current.Merged))
	case "merged_at":
		t := current.MergedAt
		if t.IsZero() {
			ctx.ResultNull()
		} else {
			ctx.ResultText(t.Format(time.RFC3339Nano))
		}
	case "merged_by":
		ctx.ResultText(current.MergedBy.Login)
	case "number":
		ctx.ResultInt(current.Number)
	case "participant_count":
		ctx.ResultInt(current.Participants.TotalCount)
	case "published_at":
		t := current.PublishedAt
		if t.IsZero() {
			ctx.ResultNull()
		} else {
			ctx.ResultText(t.Format(time.RFC3339Nano))
		}
	case "review_decision":
		ctx.ResultText(string(current.ReviewDecision))
	case "state":
		ctx.ResultText(string(current.State))
	case "title":
		ctx.ResultText(current.Title)
	case "updated_at":
		t := current.UpdatedAt
		if t.IsZero() {
			ctx.ResultNull()
		} else {
			ctx.ResultText(t.Format(time.RFC3339Nano))
		}
	case "url":
		ctx.ResultText(current.Url.String())
	}
	return nil
}

func (i *iterPRs) Next() (vtab.Row, error) {
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
			l.Info().Msgf("fetching page of repo_pull_requests for %s/%s", i.owner, i.name)
			results, err := i.fetchPRs(context.Background(), cursor)

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

var prCols = []vtab.Column{
	{Name: "owner", Type: "TEXT", NotNull: true, Hidden: true, Filters: []*vtab.ColumnFilter{{Op: sqlite.INDEX_CONSTRAINT_EQ, OmitCheck: true}}},
	{Name: "reponame", Type: "TEXT", NotNull: true, Hidden: true, Filters: []*vtab.ColumnFilter{{Op: sqlite.INDEX_CONSTRAINT_EQ, OmitCheck: true}}},
	{Name: "additions", Type: "INT"},
	{Name: "author_login", Type: "TEXT"},
	{Name: "author_association", Type: "TEXT"},
	{Name: "author_avatar_url", Type: "TEXT"},
	{Name: "author_name", Type: "TEXT"},
	{Name: "base_ref_oid", Type: "TEXT"},
	{Name: "base_ref_name", Type: "TEXT"},
	{Name: "base_repository_name", Type: "TEXT"},
	{Name: "body", Type: "TEXT"},
	{Name: "changed_files", Type: "INT"},
	{Name: "closed", Type: "BOOLEAN"},
	{Name: "closed_at", Type: "DATETIME"},
	{Name: "comment_count", Type: "INT", OrderBy: vtab.ASC | vtab.DESC, Filters: []*vtab.ColumnFilter{
		{Op: sqlite.INDEX_CONSTRAINT_GT}, {Op: sqlite.INDEX_CONSTRAINT_GE},
		{Op: sqlite.INDEX_CONSTRAINT_LT}, {Op: sqlite.INDEX_CONSTRAINT_LE},
	}},
	{Name: "commit_count", Type: "INT"},
	{Name: "created_at", Type: "DATETIME", OrderBy: vtab.ASC | vtab.DESC, Filters: []*vtab.ColumnFilter{
		{Op: sqlite.INDEX_CONSTRAINT_GT}, {Op: sqlite.INDEX_CONSTRAINT_GE},
		{Op: sqlite.INDEX_CONSTRAINT_LT}, {Op: sqlite.INDEX_CONSTRAINT_LE},
	}},
	{Name: "created_via_email", Type: "BOOLEAN"},
	{Name: "database_id", Type: "INT"},
	{Name: "deletions", Type: "INT"},
	{Name: "editor_login", Type: "TEXT"},
	{Name: "head_ref_name", Type: "TEXT"},
	{Name: "head_ref_oid", Type: "TEXT"},
	{Name: "head_repository_name", Type: "TEXT"},
	{Name: "is_draft", Type: "BOOLEAN"},
	{Name: "label_count", Type: "INT"},
	{Name: "labels", Type: "JSON"},
	{Name: "last_edited_at", Type: "DATETIME"},
	{Name: "locked", Type: "BOOLEAN"},
	{Name: "maintainer_can_modify", Type: "BOOLEAN"},
	{Name: "mergeable", Type: "TEXT"},
	{Name: "merged", Type: "BOOLEAN"},
	{Name: "merged_at", Type: "DATETIME"},
	{Name: "merged_by", Type: "TEXT"},
	{Name: "number", Type: "INT"},
	{Name: "participant_count", Type: "INT"},
	{Name: "published_at", Type: "DATETIME"},
	{Name: "review_decision", Type: "TEXT"},
	{Name: "state", Type: "TEXT"},
	{Name: "title", Type: "TEXT"},
	{Name: "updated_at", Type: "DATETIME", OrderBy: vtab.ASC | vtab.DESC, Filters: []*vtab.ColumnFilter{
		{Op: sqlite.INDEX_CONSTRAINT_GT}, {Op: sqlite.INDEX_CONSTRAINT_GE},
		{Op: sqlite.INDEX_CONSTRAINT_LT}, {Op: sqlite.INDEX_CONSTRAINT_LE},
	}},
	{Name: "url", Type: "TEXT"},
}

func NewPRModule(opts *Options) sqlite.Module {
	return vtab.NewTableFunc("github_repo_prs", prCols, func(constraints []*vtab.Constraint, orders []*sqlite.OrderBy) (vtab.Iterator, error) {
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

		var prOrder *githubv4.IssueOrder
		if len(orders) == 1 {
			order := orders[0]
			prOrder = &githubv4.IssueOrder{}
			switch prCols[order.ColumnIndex].Name {
			case "comment_count":
				prOrder.Field = githubv4.IssueOrderFieldComments
			case "created_at":
				prOrder.Field = githubv4.IssueOrderFieldCreatedAt
			case "updated_at":
				prOrder.Field = githubv4.IssueOrderFieldUpdatedAt
			}
			prOrder.Direction = orderByToGitHubOrder(order.Desc)
		}

		iter := &iterPRs{opts, owner, name, -1, nil, prOrder}
		iter.logger().Info().Msgf("starting GitHub repo_pull_requests iterator for %s/%s", owner, name)
		return iter, nil
	}, vtab.EarlyOrderByConstraintExit(true))
}
