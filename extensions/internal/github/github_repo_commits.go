package github

import (
	"context"
	"io"
	"time"

	"github.com/augmentable-dev/vtab"
	"github.com/rs/zerolog"
	"github.com/shurcooL/githubv4"
	"go.riyazali.net/sqlite"
)

type repositoryForCommits struct {
	Commits struct {
		History struct {
			Nodes    []*objectCommit
			PageInfo struct {
				EndCursor   githubv4.String
				HasNextPage bool
			}
		} `graphql:"history(first:100,after: $commitObjectCursor)"`
	} `graphql:"... on Commit"`
}

type objectCommit struct {
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

type fetchRepositoryCommitsResults struct {
	Comments    *repositoryForCommits
	HasNextPage bool
	EndCursor   *githubv4.String
}

func (i *iterRepositoryCommits) fetchRepositoryCommits(ctx context.Context, endCursor *githubv4.String) (*fetchRepositoryCommitsResults, error) {
	var repoQuery struct {
		Repository struct {
			Owner struct {
				Login string
			}
			Name   string
			Object repositoryForCommits `graphql:"object(expression: $branch)"`
		} `graphql:"repository(owner: $owner, name: $name)"`
	}
	variables := map[string]interface{}{
		"owner":  githubv4.String(i.owner),
		"name":   githubv4.String(i.name),
		"branch": githubv4.String(i.branch),
		//"perPage":      githubv4.Int(i.PerPage),
		"commitObjectCursor": endCursor,
	}

	err := i.Client().Query(ctx, &repoQuery, variables)
	if err != nil {
		return nil, err
	}

	return &fetchRepositoryCommitsResults{
		&repoQuery.Repository.Object,
		repoQuery.Repository.Object.Commits.History.PageInfo.HasNextPage,
		&repoQuery.Repository.Object.Commits.History.PageInfo.EndCursor,
	}, nil
}

type iterRepositoryCommits struct {
	*Options
	owner         string
	name          string
	branch        string
	currentCommit int
	results       *fetchRepositoryCommitsResults
}

func (i *iterRepositoryCommits) logger() *zerolog.Logger {
	logger := i.Logger.With().Int("per-page", i.PerPage).Str("owner", i.owner).Str("name", i.name).Str("branch", i.branch).Logger()
	return &logger
}

func (i *iterRepositoryCommits) Column(ctx vtab.Context, c int) error {
	current := i.results.Comments.Commits.History.Nodes[i.currentCommit]
	col := repoCommitCols[c]

	switch col.Name {
	case "hash":
		ctx.ResultText(string(current.Oid))
	case "message":
		ctx.ResultText(current.Message)
	case "author_name":
		ctx.ResultText(current.Author.Name)
	case "author_email":
		ctx.ResultText(current.Author.Email)
	case "author_when":
		t := current.Author.Date
		if t.IsZero() {
			ctx.ResultNull()
		} else {
			ctx.ResultText(t.Format(time.RFC3339Nano))
		}
	case "committer_name":
		ctx.ResultText(current.Committer.Name)
	case "committer_email":
		ctx.ResultText(current.Committer.Email)
	case "committer_when":
		t := current.Committer.Date
		if t.IsZero() {
			ctx.ResultNull()
		} else {
			ctx.ResultText(t.Format(time.RFC3339Nano))
		}
	case "additions":
		ctx.ResultInt(current.Additions)
	case "deletions":
		ctx.ResultInt(current.Deletions)
	case "changed_files":
		ctx.ResultInt(current.ChangedFiles)
	case "name":
		ctx.ResultText(current.Committer.Name)
	case "url":
		ctx.ResultText(current.Url.String())
	}
	return nil
}

func (i *iterRepositoryCommits) Next() (vtab.Row, error) {
	i.currentCommit += 1

	if i.results == nil || i.currentCommit >= len(i.results.Comments.Commits.History.Nodes) {
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
			l.Info().Msgf("fetching page of repository_commits for %s/%s", i.owner, i.name)
			results, err := i.fetchRepositoryCommits(context.Background(), cursor)
			if err != nil {
				return nil, err
			}

			i.results = results
			i.currentCommit = 0

			if len(results.Comments.Commits.History.Nodes) == 0 {
				return nil, io.EOF
			}
		} else {
			return nil, io.EOF
		}
	}

	return i, nil

}

var repoCommitCols = []vtab.Column{
	{Name: "owner", Type: "TEXT", NotNull: true, Hidden: true, Filters: []*vtab.ColumnFilter{{Op: sqlite.INDEX_CONSTRAINT_EQ, OmitCheck: true}}},
	{Name: "reponame", Type: "TEXT", NotNull: true, Hidden: true, Filters: []*vtab.ColumnFilter{{Op: sqlite.INDEX_CONSTRAINT_EQ, OmitCheck: true}}},
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

func NewRepoCommitsModule(opts *Options) sqlite.Module {
	return vtab.NewTableFunc("github_repo_commits", repoCommitCols, func(constraints []*vtab.Constraint, orders []*sqlite.OrderBy) (vtab.Iterator, error) {
		var fullNameOrOwner, name, owner string
		var nameOrNumber *sqlite.Value
		branch := "main"
		var err error
		numArgs := 0 // if 1 only fullNameOrOwner is set and default to main if 2 fullNameOrOwner
		for _, constraint := range constraints {
			if constraint.Op == sqlite.INDEX_CONSTRAINT_EQ {
				switch constraint.ColIndex {
				case 0:
					fullNameOrOwner = constraint.Value.Text()
				case 1:
					nameOrNumber = constraint.Value
				case 2:
					if constraint.Value.Text() == "" {
						branch = "main"
					} else {
						branch = constraint.Value.Text()
						numArgs = 3
					}
				}

			}
		}
		if numArgs != 3 {
			owner, name, err = repoOwnerAndName("", fullNameOrOwner)
			if err != nil {
				return nil, err
			}
		} else {
			owner = fullNameOrOwner
			name = nameOrNumber.Text()
		}

		iter := &iterRepositoryCommits{opts, owner, name, branch, -1, nil}
		iter.logger().Info().Msgf("starting GitHub repo_commits iterator for %s/%s pr : %s", owner, name, branch)
		return iter, nil
	}, vtab.EarlyOrderByConstraintExit(true))
}
