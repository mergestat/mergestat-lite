package github

import (
	"context"
	"io"

	"github.com/augmentable-dev/vtab"
	"github.com/mergestat/mergestat-lite/extensions/options"
	"github.com/rs/zerolog"
	"github.com/shurcooL/githubv4"
	"go.riyazali.net/sqlite"
)

type ref struct {
	Name   githubv4.String
	Prefix githubv4.String
	Target struct {
		Commit struct {
			Oid    githubv4.GitObjectID
			Author struct {
				Name  githubv4.String
				Email githubv4.String
			}
		} `graphql:"... on Commit"`
	}
}

type fetchBranchResults struct {
	RateLimit   *options.GitHubRateLimitResponse
	Edges       []*ref
	HasNextPage bool
	EndCursor   *githubv4.String
}

func (i *iterBranches) fetchBranches(ctx context.Context, startCursor *githubv4.String) (*fetchBranchResults, error) {
	var BranchQuery struct {
		RateLimit  *options.GitHubRateLimitResponse
		Repository struct {
			Owner struct {
				Login string
			}
			Name string
			Refs struct {
				Nodes    []*ref
				PageInfo struct {
					EndCursor   githubv4.String
					HasNextPage bool
				}
			} `graphql:"refs(refPrefix: $refs, after: $refcursor, first: $perpage)"`
		} `graphql:"repository(owner: $owner, name: $name)"`
	}
	variables := map[string]interface{}{
		"owner":     githubv4.String(i.owner),
		"name":      githubv4.String(i.name),
		"perpage":   githubv4.Int(i.PerPage),
		"refs":      githubv4.String("refs/heads/"),
		"refcursor": startCursor,
	}

	err := i.Client().Query(ctx, &BranchQuery, variables)
	if err != nil {
		return nil, err
	}

	return &fetchBranchResults{
		RateLimit:   BranchQuery.RateLimit,
		Edges:       BranchQuery.Repository.Refs.Nodes,
		HasNextPage: BranchQuery.Repository.Refs.PageInfo.HasNextPage,
		EndCursor:   &BranchQuery.Repository.Refs.PageInfo.EndCursor,
	}, nil
}

type iterBranches struct {
	*Options
	owner   string
	name    string
	current int
	results *fetchBranchResults
}

func (i *iterBranches) logger() *zerolog.Logger {
	logger := i.Logger.With().Int("per-page", i.PerPage).Str("owner", i.owner).Str("name", i.name).Logger()
	return &logger
}

func (i *iterBranches) Column(ctx vtab.Context, c int) error {
	current := i.results.Edges[i.current]
	col := branchCols[c]

	switch col.Name {
	case "name":
		ctx.ResultText(string(current.Name))
	case "author_name":
		ctx.ResultText(string(current.Target.Commit.Author.Name))
	case "author_email":
		ctx.ResultText(string(current.Target.Commit.Author.Email))
	case "commit_hash":
		ctx.ResultText(string(current.Target.Commit.Oid))
	}
	return nil
}

func (i *iterBranches) Next() (vtab.Row, error) {
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
			l.Info().Msgf("fetching page of repo_branches for %s/%s", i.owner, i.name)
			results, err := i.fetchBranches(context.Background(), cursor)

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

var branchCols = []vtab.Column{
	{Name: "owner", Type: "TEXT", NotNull: true, Hidden: true, Filters: []*vtab.ColumnFilter{{Op: sqlite.INDEX_CONSTRAINT_EQ, OmitCheck: true}}},
	{Name: "reponame", Type: "TEXT", NotNull: true, Hidden: true, Filters: []*vtab.ColumnFilter{{Op: sqlite.INDEX_CONSTRAINT_EQ, OmitCheck: true}}},
	{Name: "name", Type: "TEXT"},
	{Name: "author_name", Type: "TEXT"},
	{Name: "author_email", Type: "TEXT"},
	{Name: "commit_hash", Type: "TEXT"},
}

func NewBranchModule(opts *Options) sqlite.Module {
	return vtab.NewTableFunc("github_repo_branches", branchCols, func(constraints []*vtab.Constraint, orders []*sqlite.OrderBy) (vtab.Iterator, error) {
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

		iter := &iterBranches{opts, owner, name, -1, nil}
		iter.logger().Info().Msgf("starting GitHub repo_branches iterator for %s/%s", owner, name)
		return iter, nil
	}, vtab.EarlyOrderByConstraintExit(true))
}
