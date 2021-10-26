package github

import (
	"context"
	"io"

	"github.com/augmentable-dev/vtab"
	"github.com/rs/zerolog"
	"github.com/shurcooL/githubv4"
	"go.riyazali.net/sqlite"
)

type pullRequestForComments struct {
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
type fetchPRCommentsResults struct {
	Edges       []*pullRequestForComments
	HasNextPage bool
	EndCursor   *githubv4.String
	StartCursor *githubv4.String
}

func (i *iterPRComments) fetchPRComments(ctx context.Context, startCursor *githubv4.String, endCursor *githubv4.String) (*fetchPRCommentsResults, error) {
	var PRQuery struct {
		Repository struct {
			Owner struct {
				Login string
			}
			Name         string
			PullRequests struct {
				Nodes    []*pullRequestForComments
				PageInfo struct {
					StartCursor githubv4.String
					EndCursor   githubv4.String
					HasNextPage bool
				}
			} `graphql:"pullRequests(first: $perpage, after: $prcursor, orderBy: $prorder)"`
		} `graphql:"repository(owner: $owner, name: $name)"`
	}
	variables := map[string]interface{}{
		"owner":         githubv4.String(i.owner),
		"name":          githubv4.String(i.name),
		"perpage":       githubv4.Int(i.PerPage),
		"commentcursor": endCursor,
		"prcursor":      startCursor,
		"prorder":       i.prOrder,
	}

	err := i.Client().Query(ctx, &PRQuery, variables)
	if err != nil {
		return nil, err
	}

	return &fetchPRCommentsResults{
		PRQuery.Repository.PullRequests.Nodes,
		PRQuery.Repository.PullRequests.PageInfo.HasNextPage,
		&PRQuery.Repository.PullRequests.PageInfo.EndCursor,
		&PRQuery.Repository.PullRequests.PageInfo.StartCursor,
	}, nil
}

type iterPRComments struct {
	*Options
	owner          string
	name           string
	currentPR      int
	currentComment int
	results        *fetchPRCommentsResults
	prOrder        *githubv4.IssueOrder
}

func (i *iterPRComments) logger() *zerolog.Logger {
	logger := i.Logger.With().Int("per-page", i.PerPage).Str("owner", i.owner).Str("name", i.name).Logger()
	if i.prOrder != nil {
		logger = logger.With().Str("order_by", string(i.prOrder.Field)).Str("order_dir", string(i.prOrder.Direction)).Logger()
	}
	return &logger
}

func (i *iterPRComments) Column(ctx vtab.Context, c int) error {
	current := i.results.Edges[i.currentPR].Comments.Nodes[i.currentComment]
	col := prCommentCols[c]

	switch col.Name {
	case "c_author_login":
		ctx.ResultText(current.Author.Login)
	case "c_author_url":
		ctx.ResultText(current.Author.Url)
	case "c_created_at":
		ctx.ResultText(current.CreatedAt.String())
	case "c_database_id":
		ctx.ResultText(string(current.DatabaseId))
	case "c_id":
		ctx.ResultText(string(current.Id))
	case "c_updated_at":
		ctx.ResultText(current.UpdatedAt.String())
	case "c_url":
		ctx.ResultText(current.Url.String())
	case "pr_id":
		ctx.ResultText(string(i.results.Edges[i.currentPR].Id))
	case "pr_number":
		ctx.ResultInt(i.results.Edges[i.currentPR].Number)
	}
	return nil
}

func (i *iterPRComments) Next() (vtab.Row, error) {
	// if no results have been pulled... pull them
	if i.results == nil {
		err := i.RateLimiter.Wait(context.Background())
		if err != nil {
			return nil, err
		}

		var cursor, commentCursor *githubv4.String

		l := i.logger().With().Interface("cursor", cursor).Logger()
		l.Info().Msgf("fetching page of repo_pull_requests for %s/%s", i.owner, i.name)
		results, err := i.fetchPRComments(context.Background(), cursor, commentCursor)
		if err != nil {
			return nil, err
		}

		i.results = results
		i.currentPR = 0
		i.currentComment = -1

		if len(results.Edges) == 0 {
			l.Info().Msgf("no pull requests found", i.owner, i.name)
			return nil, io.EOF
		}
	}

	// if there are more comments to be had on a pull request then iterate on them
	if len(i.results.Edges[i.currentPR].Comments.Nodes)-1 > i.currentComment {
		i.currentComment++
		return i, nil
	} else if i.results.Edges[i.currentPR].Comments.PageInfo.HasNextPage {
		currentComments := i.results.Edges[i.currentPR].Comments
		commentCursor := &currentComments.PageInfo.EndCursor
		prCursor := i.results.StartCursor

		l := i.logger().With().Interface("commentCursor", commentCursor).Interface("prCursor", prCursor).Logger()
		l.Info().Msgf("fetching page of comments for %s/%s", i.owner, i.name)

		results, err := i.fetchPRComments(context.Background(), prCursor, commentCursor)
		if err != nil {
			return nil, err
		}
		i.results = results
		i.currentComment = 0
		return i, nil
	}
	// while there are no more comments on a pull request go to the next pull request
	for len(i.results.Edges)-1 > i.currentPR {
		i.currentPR++
		if len(i.results.Edges[i.currentPR].Comments.Nodes) != 0 {
			i.currentComment = 0
			return i, nil
		}
	}
	// if there are no more pull requests then pull the next set of pull requests
	for i.results.HasNextPage {
		err := i.RateLimiter.Wait(context.Background())
		if err != nil {
			return nil, err
		}

		var cursor, commentCursor *githubv4.String
		if i.results != nil {
			cursor = i.results.EndCursor
		}

		l := i.logger().With().Interface("commentCursor", commentCursor).Interface("prCursor", cursor).Logger()
		l.Info().Msgf("fetching page of repo_pull_requests for %s/%s", i.owner, i.name)
		results, err := i.fetchPRComments(context.Background(), cursor, commentCursor)
		if err != nil {
			return nil, err
		}

		i.results = results
		i.currentPR = -1
		i.currentComment = 0

		if len(results.Edges) == 0 {
			return nil, io.EOF
		}
		for len(i.results.Edges)-1 > i.currentPR {
			i.currentPR++
			if len(i.results.Edges[i.currentPR].Comments.Nodes) != 0 {
				i.currentComment = 0
				return i, nil
			}
		}
	}
	return nil, io.EOF

}

var prCommentCols = []vtab.Column{
	{Name: "owner", Type: "TEXT", NotNull: true, Hidden: true, Filters: []*vtab.ColumnFilter{{Op: sqlite.INDEX_CONSTRAINT_EQ, OmitCheck: true}}},
	{Name: "reponame", Type: "TEXT", NotNull: true, Hidden: true, Filters: []*vtab.ColumnFilter{{Op: sqlite.INDEX_CONSTRAINT_EQ, OmitCheck: true}}},
	{Name: "c_author_login", Type: "TEXT"},
	{Name: "c_author_url", Type: "TEXT"},
	{Name: "c_created_at", Type: "TEXT"},
	{Name: "c_database_id", Type: "TEXT"},
	{Name: "c_id", Type: "TEXT"},
	{Name: "c_updated_at", Type: "TEXT"},
	{Name: "c_url", Type: "TEXT"},
	{Name: "pr_id", Type: "TEXT"},
	{Name: "pr_number", Type: "INT"},
}

func NewPRCommentsModule(opts *Options) sqlite.Module {
	return vtab.NewTableFunc("github_repo_prs", prCommentCols, func(constraints []*vtab.Constraint, orders []*sqlite.OrderBy) (vtab.Iterator, error) {
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
			switch prCommentCols[order.ColumnIndex].Name {
			case "comment_count":
				prOrder.Field = githubv4.IssueOrderFieldComments
			case "created_at":
				prOrder.Field = githubv4.IssueOrderFieldCreatedAt
			case "updated_at":
				prOrder.Field = githubv4.IssueOrderFieldUpdatedAt
			}
			prOrder.Direction = orderByToGitHubOrder(order.Desc)
		}

		iter := &iterPRComments{opts, owner, name, -1, -1, nil, prOrder}
		iter.logger().Info().Msgf("starting GitHub repo_pull_requests iterator for %s/%s", owner, name)
		return iter, nil
	}, vtab.EarlyOrderByConstraintExit(true))
}
