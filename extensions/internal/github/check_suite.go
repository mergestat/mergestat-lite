package github

import (
	"context"
	"encoding/json"
	"io"

	"github.com/augmentable-dev/vtab"
	"github.com/shurcooL/githubv4"
	"go.riyazali.net/sqlite"
	"golang.org/x/time/rate"
)

// type checkSuite struct {
// 	App struct {
// 		Name string
// 	}
// 	Branch    struct{
// 		Name string
// 	}
// 	CheckRuns  struct{
// 		TotalCount int
// 	}
// 	Commit struct{
// 		Oid githubv4.GitObjectID
// 		Message string
// 	}
// 	conclusion githubv4.CheckConclusionState
// 	CreatedAt       githubv4.DateTime
// 	Creator struct {
// 		Login string
// 	}
// 	// need to consider the cost of the below options
// 	// databaseId
// 	// matchingPullRequests
// 	Push struct{
// 		URI githubv4.URI
// 		NextSha githubv4.GitObjectID
// 		PreviousSha githubv4.GitObjectID
// 	}
// 	// repository
// 	ResourcePath githubv4.URI
// 	Status githubv4.CheckStatusState
// 	UpdatedAt githubv4.DateTime
// 	Url       githubv4.URI
// 	WorkflowRun struct {
// 		ResourcePath githubv4.URI
// 		RunNumber int
// 		Url githubv4.URI
// 	}
// }

type checkSuites struct {
	PageInfo struct {
		EndCursor   githubv4.String
		HasNextPage bool
	}
	Edges []struct { //checkSuitesEdge
		Node struct { //CheckSuite
			Conclusion githubv4.CheckConclusionState
			CreatedAt  githubv4.DateTime
			Status     githubv4.CheckStatusState
			App        struct {
				Name string
			}
			Branch struct {
				Name string
			}
			CheckRuns struct { //CheckRunConnection
				TotalCount int
				Nodes      []struct { //[CheckRun]
					Conclusion  string
					CompletedAt githubv4.DateTime
					Name        string
					Summary     string
					Title       string
					Status      string
				}
			} `graphql:"checkRuns(last: 10)"`
		}
	}
}
type csPullRequest struct {
	Commits struct {
		TotalCount int
		Edges      []struct {
			Node struct {
				Commit struct {
					CheckSuites checkSuites `graphql:"checkSuites(last: 1)"`
				}
			}
		}
	} `graphql:"commits(last: 10)"`
}
type fetchCheckSuiteOptions struct {
	Client      *githubv4.Client
	Owner       string
	Name        string
	PerPage     int
	StartCursor *githubv4.String
	PROrder     *githubv4.IssueOrder
}

type fetchCheckSuiteResults struct {
	Edges       []*csPullRequest
	HasNextPage bool
	EndCursor   *githubv4.String
}

type CheckSuiteEdge struct {
	Cursor string
	Node   *[]csPullRequest
}

func fetchCheckSuite(ctx context.Context, input *fetchCheckSuiteOptions) (*fetchCheckSuiteResults, error) {
	var issuesQuery struct {
		Repository struct {
			Owner struct {
				Login string
			}
			Name         string
			PullRequests struct {
				Nodes    []*csPullRequest
				PageInfo struct {
					EndCursor   githubv4.String
					HasNextPage bool
				}
			} `graphql:"pullRequests(first: $perpage, after: $prcursor, orderBy: $prorder)"`
		} `graphql:"repository(owner: $owner, name: $name)"`
	}
	variables := map[string]interface{}{
		"owner":    githubv4.String(input.Owner),
		"name":     githubv4.String(input.Name),
		"perpage":  githubv4.Int(input.PerPage),
		"prcursor": (*githubv4.String)(input.StartCursor),
		"prorder":  input.PROrder,
	}

	err := input.Client.Query(ctx, &issuesQuery, variables)

	if err != nil {
		return nil, err
	}

	return &fetchCheckSuiteResults{
		issuesQuery.Repository.PullRequests.Nodes,
		issuesQuery.Repository.PullRequests.PageInfo.HasNextPage,
		&issuesQuery.Repository.PullRequests.PageInfo.EndCursor,
	}, nil
}

type iterCheckSuites struct {
	fullNameOrOwner              string
	name                         string
	client                       *githubv4.Client
	currentCsPr                  int
	currentCsPrCommitEdge        int
	currentCsPrEdgeCommitCsEdges int
	results                      *fetchCheckSuiteResults
	rateLimiter                  *rate.Limiter
	issueOrder                   *githubv4.IssueOrder
	perPage                      int
}

func (i *iterCheckSuites) Column(ctx *sqlite.Context, c int) error {
	// x, err := json.Marshal(i.results.Edges)
	// if err != nil {
	// 	println(err.Error())
	// 	return err
	// }
	// println(x)
	println("top: ", i.currentCsPr, " middle: ", i.currentCsPrCommitEdge, " bottom: ", i.currentCsPrEdgeCommitCsEdges)
	current := i.results.Edges[i.currentCsPr].Commits.Edges[i.currentCsPrCommitEdge].Node.Commit.CheckSuites.Edges[i.currentCsPrEdgeCommitCsEdges]
	col := checkSuiteCols[c]

	switch col.Name {
	case "results":
		result, err := json.Marshal(current)
		if err != nil {
			ctx.ResultText(string(result))
		} else {
			ctx.ResultText(err.Error())
		}
		// case "reponame":
		// 	ctx.ResultText(i.name)
		// case "author_login":
		// 	ctx.ResultText(current.Node.Author.Login)
		// case "body":
		// 	ctx.ResultText(current.Node.Body)
		// case "closed":
		// 	ctx.ResultInt(t1f0(current.Node.Closed))
		// case "closed_at":
		// 	t := current.Node.ClosedAt
		// 	if t.IsZero() {
		// 		ctx.ResultNull()
		// 	} else {
		// 		ctx.ResultText(t.Format(time.RFC3339Nano))
		// 	}
		// case "comment_count":
		// 	ctx.ResultInt(current.Node.Comments.TotalCount)
		// case "created_at":
		// 	t := current.Node.CreatedAt
		// 	if t.IsZero() {
		// 		ctx.ResultNull()
		// 	} else {
		// 		ctx.ResultText(t.Format(time.RFC3339Nano))
		// 	}
		// case "created_via_email":
		// 	ctx.ResultInt(t1f0(current.Node.CreatedViaEmail))
		// case "database_id":
		// 	ctx.ResultInt(current.Node.DatabaseId)
		// case "editor_login":
		// 	ctx.ResultText(current.Node.Editor.Login)
		// case "includes_created_edit":
		// 	ctx.ResultInt(t1f0(current.Node.IncludesCreatedEdit))
		// case "label_count":
		// 	ctx.ResultInt(current.Node.Labels.TotalCount)
		// case "last_edited_at":
		// 	t := current.Node.LastEditedAt
		// 	if t.IsZero() {
		// 		ctx.ResultNull()
		// 	} else {
		// 		ctx.ResultText(t.Format(time.RFC3339Nano))
		// 	}
		// case "locked":
		// 	ctx.ResultInt(t1f0(current.Node.Locked))
		// case "milestone_number":
		// 	ctx.ResultInt(current.Node.Milestone.Number)
		// case "number":
		// 	ctx.ResultInt(current.Node.Number)
		// case "participant_count":
		// 	ctx.ResultInt(current.Node.Participants.TotalCount)
		// case "published_at":
		// 	t := current.Node.PublishedAt
		// 	if t.IsZero() {
		// 		ctx.ResultNull()
		// 	} else {
		// 		ctx.ResultText(t.Format(time.RFC3339Nano))
		// 	}
		// case "reaction_count":
		// 	ctx.ResultInt(current.Node.Reactions.TotalCount)
		// case "state":
		// 	ctx.ResultText(fmt.Sprint(current.Node.State))
		// case "title":
		// 	ctx.ResultText(current.Node.Title)
		// case "updated_at":
		// 	t := current.Node.UpdatedAt
		// 	if t.IsZero() {
		// 		ctx.ResultNull()
		// 	} else {
		// 		ctx.ResultText(t.Format(time.RFC3339Nano))
		// 	}
		// case "url":
		// 	ctx.ResultText(current.Node.Url.String())}
	}
	return nil
}

func (i *iterCheckSuites) Next() (vtab.Row, error) {
	i.currentCsPrEdgeCommitCsEdges++
	// check to see if need to iterate to next commitEdge in current PR
	if i.results == nil || i.currentCsPrEdgeCommitCsEdges >= len(i.results.Edges[i.currentCsPr].Commits.Edges[i.currentCsPrCommitEdge].Node.Commit.CheckSuites.Edges) {
		if i.results != nil && i.currentCsPrCommitEdge+1 < len(i.results.Edges[i.currentCsPr].Commits.Edges) {
			//iterate to next commit and reset checksuiteEdges counter
			i.currentCsPrCommitEdge++
			i.currentCsPrEdgeCommitCsEdges = 0
			//return i, nil
		}
		// if the above does not trigger check the topmost pull request layer and increment counter if appropriate
		if i.results != nil && i.currentCsPr+1 < len(i.results.Edges) {
			i.currentCsPr++
			i.currentCsPrCommitEdge = 0
			i.currentCsPrEdgeCommitCsEdges = 0
			//return i, nil
		}
		// if both the above are false then check if there is a next page to pull
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

			results, err := fetchCheckSuite(context.Background(), &fetchCheckSuiteOptions{i.client, owner, name, i.perPage, cursor, i.issueOrder})
			if err != nil {
				return nil, err
			}

			i.results = results
			println("got results")
			x, err := json.Marshal(i.results)
			if err != nil {
				println(err.Error())
			}
			println(string(x))
			println("num topmost edges", len(results.Edges))
			println("num next edges: ", len(results.Edges[0].Commits.Edges))
			println("and one deeper: ", len(results.Edges[0].Commits.Edges[0].Node.Commit.CheckSuites.Edges))
			i.currentCsPr = 0
			i.currentCsPrCommitEdge = 0
			i.currentCsPrEdgeCommitCsEdges = 0

		} else {
			return nil, io.EOF
		}
	}
	if i.currentCsPr >= len(i.results.Edges) || i.currentCsPrCommitEdge >= len(i.results.Edges[i.currentCsPr].Commits.Edges) || i.currentCsPrEdgeCommitCsEdges >= len(i.results.Edges[i.currentCsPr].Commits.Edges[i.currentCsPrCommitEdge].Node.Commit.CheckSuites.Edges) {
		return i.Next()
	}
	return i, nil
}

var checkSuiteCols = []vtab.Column{
	{Name: "owner", Type: "TEXT", NotNull: true, Hidden: true, Filters: []*vtab.ColumnFilter{{Op: sqlite.INDEX_CONSTRAINT_EQ, Required: true, OmitCheck: true}}},
	{Name: "reponame", Type: "TEXT", NotNull: true, Hidden: true, Filters: []*vtab.ColumnFilter{{Op: sqlite.INDEX_CONSTRAINT_EQ}}},
	{Name: "results", Type: "TEXT"},
	// {Name: "body", Type: "TEXT"},
	// {Name: "closed", Type: "BOOLEAN"},
	// {Name: "closed_at", Type: "DATETIME"},
	// {Name: "comment_count", Type: "INT", OrderBy: vtab.ASC | vtab.DESC},
	// {Name: "created_at", Type: "DATETIME", OrderBy: vtab.ASC | vtab.DESC},
	// {Name: "created_via_email", Type: "BOOLEAN"},
	// {Name: "database_id", Type: "TEXT"},
	// {Name: "editor_login", Type: "TEXT"},
	// {Name: "includes_created_edit", Type: "BOOLEAN"},
	// {Name: "label_count", Type: "INT"},
	// {Name: "last_edited_at", Type: "DATETIME"},
	// {Name: "locked", Type: "BOOLEAN"},
	// {Name: "milestone_count", Type: "INT"},
	// {Name: "number", Type: "INT"},
	// {Name: "participant_count", Type: "INT"},
	// {Name: "published_at", Type: "DATETIME"},
	// {Name: "reaction_count", Type: "INT"},
	// {Name: "state", Type: "TEXT"},
	// {Name: "title", Type: "TEXT"},
	// {Name: "updated_at", Type: "DATETIME", OrderBy: vtab.ASC | vtab.DESC},
	// {Name: "url", Type: "TEXT"},
}

func NewCheckSuiteModule(opts *Options) sqlite.Module {
	return vtab.NewTableFunc("github_repo_issues", checkSuiteCols, func(constraints []*vtab.Constraint, orders []*sqlite.OrderBy) (vtab.Iterator, error) {
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

		// var issueOrder *githubv4.IssueOrder
		// if len(orders) == 1 {
		// 	order := orders[0]

		// 	issueOrder = &githubv4.IssueOrder{}
		// 	switch issuesCols[order.ColumnIndex].Name {
		// 	case "comment_count":
		// 		issueOrder.Field = githubv4.IssueOrderFieldComments
		// 	case "created_at":
		// 		issueOrder.Field = githubv4.IssueOrderFieldCreatedAt
		// 	case "updated_at":
		// 		issueOrder.Field = githubv4.IssueOrderFieldUpdatedAt
		// 	}
		// 	issueOrder.Direction = orderByToGitHubOrder(order.Desc)
		// }

		return &iterCheckSuites{fullNameOrOwner, name, opts.Client(), 0, 0, 0, nil, opts.RateLimiter, &githubv4.IssueOrder{Field: githubv4.IssueOrderFieldCreatedAt, Direction: githubv4.OrderDirectionAsc}, opts.PerPage}, nil
	})
}
