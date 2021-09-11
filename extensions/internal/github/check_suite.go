package github

import (
	"context"
	"io"
	"time"

	"github.com/augmentable-dev/vtab"
	"github.com/shurcooL/githubv4"
	"go.riyazali.net/sqlite"
	"go.uber.org/zap"
)

// fetch checkRun from the checkSuiteConnection
type checkSuite struct {
	App struct {
		Name    githubv4.String
		LogoUrl githubv4.URI
	}
	Branch struct {
		Name githubv4.String
	}
	Commit struct {
		Oid githubv4.GitObjectID
	}
	Creator struct {
		Login githubv4.String
	}
	Conclusion githubv4.CheckConclusionState
	CreatedAt  githubv4.DateTime
	Repository struct {
		NameWithOwner string
	}
	ResourcePath githubv4.URI
	UpdatedAt    githubv4.DateTime
	Url          githubv4.URI
	WorkflowRun  struct {
		Workflow struct {
			Name githubv4.String
		}
	}
	CheckRuns struct {
		Nodes []*checkRun
	} `graphql:"checkRuns(first: 50)"`
}

type checkRunRow struct {
	checkRun
	AppName            string
	AppLogoUrl         string
	BranchName         string
	CommitId           string
	User               string
	WorkflowName       string
	WorkflowConclusion string
	Repository         string
}
type checkRun struct {
	Name        githubv4.String
	Title       githubv4.String
	Conclusion  githubv4.CheckConclusionState
	Summary     githubv4.String
	Status      githubv4.String
	StartedAt   githubv4.DateTime
	CompletedAt githubv4.DateTime
	Url         githubv4.URI
}

type ref struct {
	Name   string
	Target struct {
		Commit struct {
			CheckSuites struct {
				Nodes    []*checkSuite
				PageInfo struct {
					EndCursor   githubv4.String
					HasNextPage bool
				}
			} `graphql:"checkSuites(first: 50)"`
		} `graphql:"... on Commit"`
	}
}

type fetchCheckSuiteResults struct {
	Edges       []*checkRunRow
	HasNextPage bool
	EndCursor   *githubv4.String
}

type iterCheckSuites struct {
	*Options
	owner   string
	name    string
	current int
	results *fetchCheckSuiteResults
}

func (i *iterCheckSuites) fetchCheckSuiteResults(ctx context.Context, startCursor *githubv4.String) (*fetchCheckSuiteResults, error) {
	var CheckSuiteQuery struct {
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
			} `graphql:"refs(first: $perpage, after: $checkCursor, refPrefix: \"refs/heads/\")"`
		} `graphql:"repository(owner: $owner, name: $name)"`
	}

	variables := map[string]interface{}{
		"owner":       githubv4.String(i.owner),
		"name":        githubv4.String(i.name),
		"perpage":     githubv4.Int(i.PerPage),
		"checkCursor": startCursor,
	}
	err := i.Client().Query(ctx, &CheckSuiteQuery, variables)
	if err != nil {
		return nil, err
	}
	rows := getCheckRowsFromRefs(CheckSuiteQuery.Repository.Refs.Nodes)
	return &fetchCheckSuiteResults{rows, CheckSuiteQuery.Repository.Refs.PageInfo.HasNextPage, &CheckSuiteQuery.Repository.Refs.PageInfo.EndCursor}, nil
}

func (i *iterCheckSuites) logger() *zap.SugaredLogger {
	logger := i.Logger.Sugar().With("per-page", i.PerPage, "owner", i.owner, "name", i.name)
	return logger
}

func (i *iterCheckSuites) Next() (vtab.Row, error) {
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

			i.logger().With("cursor", cursor).Infof("fetching page of repo_check_runs for %s/%s", i.owner, i.name)
			results, err := i.fetchCheckSuiteResults(context.Background(), cursor)
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

var checkCols = []vtab.Column{
	{Name: "owner", Type: "TEXT", NotNull: true, Hidden: true, Filters: []*vtab.ColumnFilter{{Op: sqlite.INDEX_CONSTRAINT_EQ, Required: true, OmitCheck: true}}},
	{Name: "reponame", Type: "TEXT", NotNull: true, Hidden: true, Filters: []*vtab.ColumnFilter{{Op: sqlite.INDEX_CONSTRAINT_EQ, OmitCheck: true}}},
	{Name: "name", Type: "TEXT"},
	{Name: "workflow_name", Type: "TEXT"},
	{Name: "repository", Type: "TEXT"},
	{Name: "branch", Type: "TEXT"},
	{Name: "commitId", Type: "TEXT"},
	{Name: "conclusion", Type: "TEXT"},
	{Name: "workflow_conclusion", Type: "TEXT"},
	{Name: "status", Type: "TEXT"},
	{Name: "summary", Type: "TEXT"},
	{Name: "user", Type: "TEXT"},
	{Name: "started_at", Type: "DATETIME"},
	{Name: "completed_at", Type: "DATETIME"},
	{Name: "url", Type: "TEXT"},
	{Name: "app_name", Type: "TEXT"},
	{Name: "app_logo_url", Type: "TEXT"},
}

func (i *iterCheckSuites) Column(ctx *sqlite.Context, c int) error {
	current := i.results.Edges[i.current]
	col := checkCols[c]

	switch col.Name {
	case "branch":
		ctx.ResultText(current.BranchName)
	case "commitId":
		ctx.ResultText(current.CommitId)
	case "name":
		ctx.ResultText(string(current.Name))
	case "repository":
		ctx.ResultText(current.Repository)
	case "workflow_name":
		ctx.ResultText(current.WorkflowName)
	case "conclusion":
		ctx.ResultText(string(current.Conclusion))
	case "summary":
		ctx.ResultText(string(current.Summary))
	case "status":
		ctx.ResultText(string(current.Status))
	case "user":
		ctx.ResultText(current.User)
	case "url":
		ctx.ResultText(current.Url.String())
	case "app_name":
		ctx.ResultText(current.AppName)
	case "app_logo_url":
		ctx.ResultText(current.AppLogoUrl)
	case "workflow_conclusion":
		ctx.ResultText(current.WorkflowConclusion)
	case "started_at":
		t := current.StartedAt
		if t.IsZero() {
			ctx.ResultNull()
		} else {
			ctx.ResultText(t.Format(time.RFC3339Nano))
		}
	case "completed_at":
		t := current.CompletedAt
		if t.IsZero() {
			ctx.ResultNull()
		} else {
			ctx.ResultText(t.Format(time.RFC3339Nano))
		}
	}
	return nil
}

func NewCheckModule(opts *Options) sqlite.Module {
	return vtab.NewTableFunc("github_repo_checks", checkCols, func(constraints []*vtab.Constraint, orders []*sqlite.OrderBy) (vtab.Iterator, error) {
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

		iter := &iterCheckSuites{opts, owner, name, -1, nil}
		iter.logger().Infof("starting GitHub check iterator for %s/%s", owner, name)
		return iter, nil
	})
}

func getCheckRowsFromRefs(refs []*ref) []*checkRunRow {
	var rows []*checkRunRow
	for _, r := range refs {
		for _, suite := range r.Target.Commit.CheckSuites.Nodes {
			for _, check := range suite.CheckRuns.Nodes {
				if check.Name != "" {
					var row = checkRunRow{
						checkRun:           *check,
						AppName:            string(suite.App.Name),
						AppLogoUrl:         suite.App.LogoUrl.String(),
						BranchName:         string(suite.Branch.Name),
						CommitId:           string(suite.Commit.Oid),
						User:               string(suite.Creator.Login),
						WorkflowName:       string(suite.WorkflowRun.Workflow.Name),
						WorkflowConclusion: string(suite.Conclusion),
						Repository:         suite.Repository.NameWithOwner,
					}
					rows = append(rows, &row)
				}
			}
		}
	}
	return rows
}
