package github

import (
	"context"
	"fmt"
	"io"
	"strconv"

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
	} `graphql:"comments(first: $perPage, after: $commentcursor,orderBy: $orderBy)"`
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
	Comments    *issueForComments
	OrderBy     *githubv4.IssueCommentOrder
	HasNextPage bool
	EndCursor   *githubv4.String
}

func (i *iterIssuesComments) fetchIssueComments(ctx context.Context, endCursor *githubv4.String) (*fetchIssuesCommentsResults, error) {
	var IssueQuery struct {
		Repository struct {
			Owner struct {
				Login string
			}
			Name  string
			Issue issueForComments `graphql:"issue(number: $issueNumber)"`
		} `graphql:"repository(owner: $owner, name: $name)"`
	}
	variables := map[string]interface{}{
		"owner":         githubv4.String(i.owner),
		"name":          githubv4.String(i.name),
		"issueNumber":   githubv4.Int(i.issueNumber),
		"perPage":       githubv4.Int(i.PerPage),
		"orderBy":       i.orderBy,
		"commentcursor": endCursor,
	}

	err := i.Client().Query(ctx, &IssueQuery, variables)
	if err != nil {
		return nil, err
	}

	return &fetchIssuesCommentsResults{
		&IssueQuery.Repository.Issue,
		i.orderBy,
		IssueQuery.Repository.Issue.Comments.PageInfo.HasNextPage,
		&IssueQuery.Repository.Issue.Comments.PageInfo.EndCursor,
	}, nil
}

type iterIssuesComments struct {
	*Options
	owner          string
	name           string
	issueNumber    int
	currentComment int
	orderBy        *githubv4.IssueCommentOrder
	results        *fetchIssuesCommentsResults
}

func (i *iterIssuesComments) logger() *zerolog.Logger {
	logger := i.Logger.With().Int("per-page", i.PerPage).Str("owner", i.owner).Str("name", i.name).Int("issue-number", i.issueNumber).Int("current-comment", i.currentComment).Logger()
	if i.orderBy != nil {
		logger = logger.With().Str("order_by", string(i.orderBy.Field)).Str("order_dir", string(i.orderBy.Direction)).Logger()
	}
	return &logger
}

func (i *iterIssuesComments) Column(ctx vtab.Context, c int) error {
	current := i.results.Comments.Comments.Nodes[i.currentComment]
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
	case "issue_id":
		ctx.ResultText(string(i.results.Comments.Id))
	case "issue_number":
		ctx.ResultInt(i.results.Comments.Number)
	}
	return nil
}

func (i *iterIssuesComments) Next() (vtab.Row, error) {
	i.currentComment += 1

	if i.results == nil || i.currentComment >= len(i.results.Comments.Comments.Nodes) {
		if i.results == nil || i.results.HasNextPage {
			err := i.RateLimiter.Wait(context.Background())
			if err != nil {
				return nil, err
			}

			var cursor *githubv4.String
			if i.results != nil {
				cursor = i.results.EndCursor
			}

			l := i.logger().With().Interface("cursor", cursor).Logger()
			l.Info().Msgf("fetching page of issue_comments for %s/%s", i.owner, i.name)
			results, err := i.fetchIssueComments(context.Background(), cursor)
			if err != nil {
				return nil, err
			}

			i.results = results
			i.currentComment = 0

			if len(results.Comments.Comments.Nodes) == 0 {
				return nil, io.EOF
			}
		} else {
			return nil, io.EOF
		}
	}

	return i, nil

}

var issuesCommentCols = []vtab.Column{
	{Name: "owner", Type: "TEXT", NotNull: true, Hidden: true, Filters: []*vtab.ColumnFilter{{Op: sqlite.INDEX_CONSTRAINT_EQ, OmitCheck: true}}},
	{Name: "reponame", Type: "TEXT", NotNull: true, Hidden: true, Filters: []*vtab.ColumnFilter{{Op: sqlite.INDEX_CONSTRAINT_EQ, OmitCheck: true}}},
	{Name: "issue_number", Type: "INT", NotNull: true, Hidden: true, Filters: []*vtab.ColumnFilter{{Op: sqlite.INDEX_CONSTRAINT_EQ, OmitCheck: true}}},
	{Name: "c_author_login", Type: "TEXT"},
	{Name: "c_author_url", Type: "TEXT"},
	{Name: "c_body", Type: "TEXT"},
	{Name: "c_created_at", Type: "TEXT"},
	{Name: "c_database_id", Type: "INT"},
	{Name: "c_id", Type: "TEXT"},
	{Name: "c_updated_at", Type: "TEXT", OrderBy: vtab.ASC | vtab.DESC},
	{Name: "c_url", Type: "TEXT"},
	{Name: "issue_id", Type: "TEXT"},
}

func NewIssueCommentsModule(opts *Options) sqlite.Module {
	return vtab.NewTableFunc("github_repo_issue_comments", issuesCommentCols, func(constraints []*vtab.Constraint, orders []*sqlite.OrderBy) (vtab.Iterator, error) {
		var fullNameOrOwner, name, owner string
		var number int
		flag := false
		for _, constraint := range constraints {
			if constraint.Op == sqlite.INDEX_CONSTRAINT_EQ {
				switch constraint.ColIndex {
				case 0:
					fullNameOrOwner = constraint.Value.Text()
				case 1:
					name = constraint.Value.Text()
				case 2:
					if constraint.Value.Int() <= 0 {
						return nil, fmt.Errorf("please pass an issue number greater than 0")
					}
					number = constraint.Value.Int()
					flag = true
				}

			}
		}
		if !flag {
			var err error
			number, err = strconv.Atoi(name)
			if err != nil {
				return nil, err
			}
			owner, name, err = repoOwnerAndName("", fullNameOrOwner)
			if err != nil {
				return nil, err
			}

			if number <= 0 {
				return nil, fmt.Errorf("please input a valid issue number greater than 0")
			}
		} else {
			owner = fullNameOrOwner
		}
		var commentOrder githubv4.IssueCommentOrder
		if len(orders) == 1 {
			order := orders[0]
			commentOrder.Field = githubv4.IssueCommentOrderFieldUpdatedAt
			commentOrder.Direction = orderByToGitHubOrder(order.Desc)
		} else {
			commentOrder.Field = githubv4.IssueCommentOrderFieldUpdatedAt
			commentOrder.Direction = githubv4.OrderDirectionAsc
		}

		iter := &iterIssuesComments{opts, owner, name, number, -1, &commentOrder, nil}
		iter.logger().Info().Msgf("starting GitHub repo_issues_comment iterator for %s/%s issue : %d", owner, name, number)
		return iter, nil
	}, vtab.EarlyOrderByConstraintExit(true))
}
