package github

import (
	"context"
	"fmt"
	"io"

	"github.com/augmentable-dev/vtab"
	"github.com/rs/zerolog"
	"github.com/shurcooL/githubv4"
	"go.riyazali.net/sqlite"
)

type pullRequestForCommits struct {
	Id      githubv4.String
	Number  int
	Commits struct {
		Nodes []struct {
			Commit *prCommit
		}
		PageInfo struct {
			EndCursor   githubv4.String
			HasNextPage bool
		}
	} `graphql:"commits(first: $perPage, after: $commitcursor)"`
}

type prCommit struct {
	Additions    int
	ChangedFiles int
	Committer    struct {
		Email string
		Name  string
	}
	Deletions int
	Id        githubv4.GitObjectID
	Oid       githubv4.GitObjectID
	Message   string
	Url       githubv4.URI
}

type fetchPRCommitsResults struct {
	Comments    *pullRequestForCommits
	HasNextPage bool
	EndCursor   *githubv4.String
}

func (i *iterPRCommits) fetchPRCommits(ctx context.Context, endCursor *githubv4.String) (*fetchPRCommitsResults, error) {
	var PRQuery struct {
		Repository struct {
			Owner struct {
				Login string
			}
			Name        string
			PullRequest pullRequestForCommits `graphql:"pullRequest(number: $prNumber)"`
		} `graphql:"repository(owner: $owner, name: $name)"`
	}
	variables := map[string]interface{}{
		"owner":        githubv4.String(i.owner),
		"name":         githubv4.String(i.name),
		"prNumber":     githubv4.Int(i.prNumber),
		"perPage":      githubv4.Int(i.PerPage),
		"commitcursor": endCursor,
	}

	err := i.Client().Query(ctx, &PRQuery, variables)
	if err != nil {
		return nil, err
	}

	return &fetchPRCommitsResults{
		&PRQuery.Repository.PullRequest,
		PRQuery.Repository.PullRequest.Commits.PageInfo.HasNextPage,
		&PRQuery.Repository.PullRequest.Commits.PageInfo.EndCursor,
	}, nil
}

type iterPRCommits struct {
	*Options
	owner         string
	name          string
	prNumber      int
	currentCommit int
	results       *fetchPRCommitsResults
}

func (i *iterPRCommits) logger() *zerolog.Logger {
	logger := i.Logger.With().Int("per-page", i.PerPage).Str("owner", i.owner).Str("name", i.name).Int("pr-number", i.prNumber).Logger()
	return &logger
}

func (i *iterPRCommits) Column(ctx vtab.Context, c int) error {
	current := i.results.Comments.Commits.Nodes[i.currentCommit]
	col := prCommitsCols[c]

	switch col.Name {
	case "commit_additions":
		ctx.ResultInt(current.Commit.Additions)
	case "commit_changed_files":
		ctx.ResultInt(current.Commit.ChangedFiles)
	case "commiter_email":
		ctx.ResultText(current.Commit.Committer.Email)
	case "commit_name":
		ctx.ResultText(current.Commit.Committer.Name)
	case "commit_message":
		ctx.ResultText(current.Commit.Message)
	case "commit_hash":
		ctx.ResultText(string(current.Commit.Id))
	case "commit_deletions":
		ctx.ResultInt(current.Commit.Deletions)
	case "commit_url":
		ctx.ResultText(current.Commit.Url.String())
	case "commit_oid":
		ctx.ResultText(string(current.Commit.Oid))
	case "commit_id":
		ctx.ResultText(string(current.Commit.Id))
	case "pr_number":
		ctx.ResultInt(i.prNumber)
	}
	return nil
}

func (i *iterPRCommits) Next() (vtab.Row, error) {
	i.currentCommit += 1

	if i.results == nil || i.currentCommit >= len(i.results.Comments.Commits.Nodes) {
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
			l.Info().Msgf("fetching page of pr_comments for %s/%s", i.owner, i.name)
			results, err := i.fetchPRCommits(context.Background(), cursor)
			if err != nil {
				return nil, err
			}

			i.results = results
			i.currentCommit = 0

			if len(results.Comments.Commits.Nodes) == 0 {
				return nil, io.EOF
			}
		} else {
			return nil, io.EOF
		}
	}

	return i, nil

}

var prCommitsCols = []vtab.Column{
	{Name: "owner", Type: "TEXT", NotNull: true, Hidden: true, Filters: []*vtab.ColumnFilter{{Op: sqlite.INDEX_CONSTRAINT_EQ, OmitCheck: true}}},
	{Name: "reponame", Type: "TEXT", NotNull: true, Hidden: true, Filters: []*vtab.ColumnFilter{{Op: sqlite.INDEX_CONSTRAINT_EQ, OmitCheck: true}}},
	{Name: "pr_number", Type: "INT", NotNull: true, Hidden: true, Filters: []*vtab.ColumnFilter{{Op: sqlite.INDEX_CONSTRAINT_EQ, OmitCheck: true}}},
	{Name: "commit_hash", Type: "INT", NotNull: true, Hidden: true, Filters: []*vtab.ColumnFilter{{Op: sqlite.INDEX_CONSTRAINT_EQ, OmitCheck: true}}},
	{Name: "commit_url", Type: "TEXT"},
	{Name: "commit_message", Type: "TEXT"},
	{Name: "commit_deletions", Type: "INT"},
	{Name: "commiter_email", Type: "TEXT"},
	{Name: "commiter_name", Type: "TEXT"},
	{Name: "commit_additions", Type: "INT"},
	{Name: "commit_changed_files", Type: "TEXT"},
}

func NewPRCommitsModule(opts *Options) sqlite.Module {
	return vtab.NewTableFunc("github_repo_pr_commits", prCommitsCols, func(constraints []*vtab.Constraint, orders []*sqlite.OrderBy) (vtab.Iterator, error) {
		var fullNameOrOwner, name, owner string
		var nameOrNumber *sqlite.Value
		var number int
		threeArgs := false // if true, user supplied 3 args, 1st is org name, 2nd is repo name, 3rd is pr number
		for _, constraint := range constraints {
			if constraint.Op == sqlite.INDEX_CONSTRAINT_EQ {
				switch constraint.ColIndex {
				case 0:
					fullNameOrOwner = constraint.Value.Text()
				case 1:
					nameOrNumber = constraint.Value
				case 2:
					if constraint.Value.Int() <= 0 {
						return nil, fmt.Errorf("please supply a pull request number")
					}
					number = constraint.Value.Int()
					threeArgs = true
				}

			}
		}
		if !threeArgs {
			if nameOrNumber == nil || nameOrNumber.Type() != sqlite.SQLITE_INTEGER {
				return nil, fmt.Errorf("please supply a valid pr number")
			}
			number = nameOrNumber.Int()
			var err error
			owner, name, err = repoOwnerAndName("", fullNameOrOwner)
			if err != nil {
				return nil, err
			}

			if number <= 0 {
				return nil, fmt.Errorf("please supply a valid pull request number")
			}
		} else {
			owner = fullNameOrOwner
			name = nameOrNumber.Text()
		}

		iter := &iterPRCommits{opts, owner, name, number, -1, nil}
		iter.logger().Info().Msgf("starting GitHub repo_pr_comment iterator for %s/%s pr : %d", owner, name, number)
		return iter, nil
	}, vtab.EarlyOrderByConstraintExit(true))
}
