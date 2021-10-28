package github

import (
	"context"
	"io"

	"github.com/augmentable-dev/vtab"
	"github.com/rs/zerolog"
	"github.com/shurcooL/githubv4"
	"go.riyazali.net/sqlite"
)

type issueForComments struct {
	Id       githubv4.String
	Number   int
	Comments struct {
		Nodes    []*comment
		PageInfo struct {
			EndCursor   githubv4.String
			HasNextPage bool
		}
	} `graphql:"comments(first: $perpage, after: $commentcursor)"`
}
type comment struct {
	Body   string
	Author struct {
		Login string
		Url   string
	}
	CreatedAt  githubv4.DateTime
	DatabaseId int
	Id         githubv4.GitObjectID
	UpdatedAt  githubv4.DateTime
	Url        githubv4.URI
}
type fetchIssuesCommentsResults struct {
	Edges       []*issueForComments
	HasNextPage bool
	EndCursor   *githubv4.String
	StartCursor *githubv4.String
}

func (i *iterIssuesComments) fetchIssueComments(ctx context.Context, startCursor *githubv4.String, endCursor *githubv4.String) (*fetchIssuesCommentsResults, error) {
	var IssueQuery struct {
		Repository struct {
			Owner struct {
				Login string
			}
			Name   string
			Issues struct {
				Nodes    []*issueForComments
				PageInfo struct {
					StartCursor githubv4.String
					EndCursor   githubv4.String
					HasNextPage bool
				}
			} `graphql:"issues(first: $perpage, after: $issuecursor, orderBy: $issueorder)"`
		} `graphql:"repository(owner: $owner, name: $name)"`
	}
	variables := map[string]interface{}{
		"owner":         githubv4.String(i.owner),
		"name":          githubv4.String(i.name),
		"perpage":       githubv4.Int(i.PerPage),
		"commentcursor": endCursor,
		"issuecursor":   startCursor,
		"issueorder":    i.issueOrder,
	}

	err := i.Client().Query(ctx, &IssueQuery, variables)
	if err != nil {
		return nil, err
	}

	return &fetchIssuesCommentsResults{
		IssueQuery.Repository.Issues.Nodes,
		IssueQuery.Repository.Issues.PageInfo.HasNextPage,
		&IssueQuery.Repository.Issues.PageInfo.EndCursor,
		&IssueQuery.Repository.Issues.PageInfo.StartCursor,
	}, nil
}

type iterIssuesComments struct {
	*Options
	owner          string
	name           string
	currentIssue   int
	currentComment int
	results        *fetchIssuesCommentsResults
	issueOrder     *githubv4.IssueOrder
}

func (i *iterIssuesComments) logger() *zerolog.Logger {
	logger := i.Logger.With().Int("per-page", i.PerPage).Str("owner", i.owner).Str("name", i.name).Logger()
	if i.issueOrder != nil {
		logger = logger.With().Str("order_by", string(i.issueOrder.Field)).Str("order_dir", string(i.issueOrder.Direction)).Logger()
	}
	return &logger
}

func (i *iterIssuesComments) Column(ctx vtab.Context, c int) error {
	current := i.results.Edges[i.currentIssue].Comments.Nodes[i.currentComment]
	col := issuesCommentCols[c]

	switch col.Name {
	case "c_author_login":
		ctx.ResultText(current.Author.Login)
	case "c_author_url":
		ctx.ResultText(current.Author.Url)
	case "c_body":
		ctx.ResultText(current.Body)
	case "c_created_at":
		ctx.ResultText(current.CreatedAt.String())
	case "c_database_id":
		ctx.ResultInt(current.DatabaseId)
	case "c_id":
		ctx.ResultText(string(current.Id))
	case "c_updated_at":
		ctx.ResultText(current.UpdatedAt.String())
	case "c_url":
		ctx.ResultText(current.Url.String())
	case "pr_id":
		ctx.ResultText(string(i.results.Edges[i.currentIssue].Id))
	case "pr_number":
		ctx.ResultInt(i.results.Edges[i.currentIssue].Number)
	}
	return nil
}

func (i *iterIssuesComments) Next() (vtab.Row, error) {
	// if no results have been pulled pull them
	if i.results == nil {
		err := i.RateLimiter.Wait(context.Background())
		if err != nil {
			return nil, err
		}

		var cursor, commentCursor *githubv4.String

		l := i.logger().With().Interface("cursor", cursor).Logger()
		l.Info().Msgf("fetching page of repo_issues for %s/%s", i.owner, i.name)
		results, err := i.fetchIssueComments(context.Background(), cursor, commentCursor)
		if err != nil {
			return nil, err
		}

		i.results = results
		i.currentIssue = 0
		i.currentComment = -1

		if len(results.Edges) == 0 {
			l.Info().Msgf("no issues found %s/%s", i.owner, i.name)
			return nil, io.EOF
		}
	}

	// if there are more comments to be had on a issue then iterate on them
	if len(i.results.Edges[i.currentIssue].Comments.Nodes)-1 > i.currentComment {
		i.currentComment++
		return i, nil
	} else if i.results.Edges[i.currentIssue].Comments.PageInfo.HasNextPage {
		currentComments := i.results.Edges[i.currentIssue].Comments
		commentCursor := &currentComments.PageInfo.EndCursor
		prCursor := i.results.StartCursor

		l := i.logger().With().Interface("commentCursor", commentCursor).Interface("prCursor", prCursor).Logger()
		l.Info().Msgf("fetching page of comments for %s/%s", i.owner, i.name)

		results, err := i.fetchIssueComments(context.Background(), prCursor, commentCursor)
		if err != nil {
			return nil, err
		}
		i.results = results
		i.currentComment = 0
		return i, nil
	}
	// while there are no more comments on a issue go to the next issue
	for len(i.results.Edges)-1 > i.currentIssue {
		i.currentIssue++
		if len(i.results.Edges[i.currentIssue].Comments.Nodes) != 0 {
			i.currentComment = 0
			return i, nil
		}
	}
	// if there are no more issues in current edges then pull the next set of issues
	for i.results.HasNextPage {
		err := i.RateLimiter.Wait(context.Background())
		if err != nil {
			return nil, err
		}

		var cursor, commentCursor *githubv4.String
		if i.results != nil {
			cursor = i.results.EndCursor
		}

		l := i.logger().With().Interface("commentCursor", commentCursor).Interface("issueCursor", cursor).Logger()
		l.Info().Msgf("fetching page of repo_issues for %s/%s", i.owner, i.name)
		results, err := i.fetchIssueComments(context.Background(), cursor, commentCursor)
		if err != nil {
			return nil, err
		}

		i.results = results
		i.currentIssue = -1
		i.currentComment = 0

		if len(results.Edges) == 0 {
			return nil, io.EOF
		}
		for len(i.results.Edges)-1 > i.currentIssue {
			i.currentIssue++
			if len(i.results.Edges[i.currentIssue].Comments.Nodes) != 0 {
				i.currentComment = 0
				return i, nil
			}
		}
	}
	return nil, io.EOF

}

var issuesCommentCols = []vtab.Column{
	{Name: "owner", Type: "TEXT", NotNull: true, Hidden: true, Filters: []*vtab.ColumnFilter{{Op: sqlite.INDEX_CONSTRAINT_EQ, OmitCheck: true}}},
	{Name: "reponame", Type: "TEXT", NotNull: true, Hidden: true, Filters: []*vtab.ColumnFilter{{Op: sqlite.INDEX_CONSTRAINT_EQ, OmitCheck: true}}},
	{Name: "c_author_login", Type: "TEXT"},
	{Name: "c_author_url", Type: "TEXT"},
	{Name: "c_body", Type: "TEXT"},
	{Name: "c_created_at", Type: "TEXT"},
	{Name: "c_database_id", Type: "INT"},
	{Name: "c_id", Type: "TEXT"},
	{Name: "c_updated_at", Type: "TEXT"},
	{Name: "c_url", Type: "TEXT"},
	{Name: "issue_id", Type: "TEXT"},
	{Name: "issue_number", Type: "INT"},
}

func NewIssueCommentsModule(opts *Options) sqlite.Module {
	return vtab.NewTableFunc("github_repo_issues", issuesCommentCols, func(constraints []*vtab.Constraint, orders []*sqlite.OrderBy) (vtab.Iterator, error) {
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
			switch issuesCommentCols[order.ColumnIndex].Name {
			case "comment_count":
				issueOrder.Field = githubv4.IssueOrderFieldComments
			case "created_at":
				issueOrder.Field = githubv4.IssueOrderFieldCreatedAt
			case "updated_at":
				issueOrder.Field = githubv4.IssueOrderFieldUpdatedAt
			}
			issueOrder.Direction = orderByToGitHubOrder(order.Desc)
		}

		iter := &iterIssuesComments{opts, owner, name, -1, -1, nil, issueOrder}
		iter.logger().Info().Msgf("starting GitHub repo_issues iterator for %s/%s", owner, name)
		return iter, nil
	}, vtab.EarlyOrderByConstraintExit(true))
}
