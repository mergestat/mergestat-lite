package github

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/augmentable-dev/vtab"
	"github.com/mergestat/mergestat/extensions/options"
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
	Additions int
	Author    struct {
		Email string
		Name  string
		Date  githubv4.DateTime
	}
	ChangedFiles int
	Committer    struct {
		Email string
		Name  string
		Date  githubv4.DateTime
	}
	Deletions int
	Oid       githubv4.GitObjectID
	Message   string
	Url       githubv4.URI
}

type fetchPRCommitsResults struct {
	RateLimit   *options.GitHubRateLimitResponse
	PR          *pullRequestForCommits
	HasNextPage bool
	EndCursor   *githubv4.String
}

func (i *iterPRCommits) fetchPRCommits(ctx context.Context, endCursor *githubv4.String) (*fetchPRCommitsResults, error) {
	var PRQuery struct {
		RateLimit  *options.GitHubRateLimitResponse
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
		RateLimit:   PRQuery.RateLimit,
		PR:          &PRQuery.Repository.PullRequest,
		HasNextPage: PRQuery.Repository.PullRequest.Commits.PageInfo.HasNextPage,
		EndCursor:   &PRQuery.Repository.PullRequest.Commits.PageInfo.EndCursor,
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
	current := i.results.PR.Commits.Nodes[i.currentCommit]
	col := prCommitsCols[c]

	switch col.Name {
	case "pr_number":
		ctx.ResultInt(i.prNumber)
	case "hash":
		ctx.ResultText(string(current.Commit.Oid))
	case "message":
		ctx.ResultText(current.Commit.Message)
	case "author_name":
		ctx.ResultText(current.Commit.Author.Name)
	case "author_email":
		ctx.ResultText(current.Commit.Author.Email)
	case "author_when":
		t := current.Commit.Author.Date
		if t.IsZero() {
			ctx.ResultNull()
		} else {
			ctx.ResultText(t.Format(time.RFC3339Nano))
		}
	case "committer_name":
		ctx.ResultText(current.Commit.Committer.Name)
	case "committer_email":
		ctx.ResultText(current.Commit.Committer.Email)
	case "committer_when":
		t := current.Commit.Committer.Date
		if t.IsZero() {
			ctx.ResultNull()
		} else {
			ctx.ResultText(t.Format(time.RFC3339Nano))
		}
	case "additions":
		ctx.ResultInt(current.Commit.Additions)
	case "deletions":
		ctx.ResultInt(current.Commit.Deletions)
	case "changed_files":
		ctx.ResultInt(current.Commit.ChangedFiles)
	case "name":
		ctx.ResultText(current.Commit.Committer.Name)
	case "url":
		ctx.ResultText(current.Commit.Url.String())
	}
	return nil
}

func (i *iterPRCommits) Next() (vtab.Row, error) {
	i.currentCommit += 1

	if i.results == nil || i.currentCommit >= len(i.results.PR.Commits.Nodes) {
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
			l.Info().Msgf("fetching page of pr_comments for %s/%s", i.owner, i.name)
			results, err := i.fetchPRCommits(context.Background(), cursor)
			if err != nil {
				return nil, err
			}

			i.Options.RateLimitHandler(results.RateLimit)

			i.results = results
			i.currentCommit = 0

			if len(results.PR.Commits.Nodes) == 0 {
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

	{Name: "hash", Type: "TEXT"},
	{Name: "message", Type: "TEXT"},
	{Name: "author_name", Type: "TEXT"},
	{Name: "author_email", Type: "TEXT"},
	{Name: "author_when", Type: "DATETIME"},
	{Name: "committer_name", Type: "TEXT"},
	{Name: "committer_email", Type: "TEXT"},
	{Name: "committer_when", Type: "DATETIME"},
	{Name: "additions", Type: "INT"},
	{Name: "deletions", Type: "INT"},
	{Name: "changed_files", Type: "INT"},
	{Name: "url", Type: "TEXT"},
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
