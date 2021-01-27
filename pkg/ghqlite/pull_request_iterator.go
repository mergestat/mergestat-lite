package ghqlite

import (
	"context"
	"log"
	"os"

	"github.com/google/go-github/github"
	"github.com/shurcooL/githubv4"
	"golang.org/x/oauth2"
)

// RepoPullRequestIterator iterates over GitHub pull requests belonging to a single repository
type RepoPullRequestIterator struct {
	//options    RepoPullRequestIteratorOptions
	repo  *Repo
	pr    []pullRequest
	index int
}
type user struct {
	Login string
	URL   string
}
type Repo struct {
	Repository struct {
		DatabaseID  githubv4.Int
		URL         githubv4.URI
		Description string
		Owner       struct {
			Login string
			Url   string
		}
		Name string

		PullRequests struct {
			Nodes    []pullRequest
			PageInfo struct {
				EndCursor   githubv4.String
				HasNextPage bool
			}
		} `graphql:"pullRequests(last:100, after : $pullRequestCursor)"`
	} `graphql:"repository(owner: $owner, name: $name)"`
}
type pullRequest struct {
	ActiveLockReason githubv4.LockReason
	Assignees        struct {
		Nodes []user
	} `graphql:"assignees(first:100)"`
	Additions         githubv4.Int
	Author            user
	AuthorAssociation githubv4.CommentAuthorAssociation

	BodyText    githubv4.String
	BaseRefOid  githubv4.GitObjectID
	BaseRefName string
	BaseRef     struct {
		Name string
	}
	BaseRepository struct {
		Name  string
		Owner struct {
			Login string
		}
	}
	Body string
	//BodyHTML           string
	//CanBeRebased       bool
	ChecksResourcePath githubv4.URI
	ChecksURL          githubv4.URI

	Comments struct {
		TotalCount int
	} `graphql:"comments(first:100)"`
	Commits struct {
		TotalCount int
	} `graphql:"commits(first:100)"`
	ChangedFiles    int
	Closed          bool
	ClosedAt        githubv4.DateTime
	CreatedAt       githubv4.DateTime
	CreatedViaEmail bool
	Deletions       int
	DatabaseID      githubv4.Int
	Editor          struct {
		Login string
	}
	Files struct {
		TotalCount int
	}
	HeadRepository struct {
		Name string
	}
	HeadRepositoryOwner struct {
		Login string
	}
	HeadRefOid  githubv4.GitObjectID
	HeadRefName string
	HeadRef     struct {
		Name string
	}
	IncludesCreatedEdit bool
	IsCrossRepository   bool
	IsDraft             bool
	Labels              struct {
		// Nodes []struct {
		// 	Name string
		// }
		TotalCount int
	} `graphql:"labels(first:100)"`
	LastEditedAt githubv4.DateTime
	Locked       githubv4.Boolean
	MergedAt     githubv4.DateTime
	MergeCommit  struct {
		Oid githubv4.GitObjectID
	}
	Merged              githubv4.Boolean
	Mergeable           githubv4.MergeableState
	MergedBy            user
	MaintainerCanModify githubv4.Boolean
	//MergeStateStatuses  string
	Milestone struct {
		Number int
	}
	Number       githubv4.Int
	Participants struct {
		TotalCount int
	}
	Permalink      githubv4.URI
	PublishedAt    githubv4.DateTime
	ReviewDecision githubv4.PullRequestReviewDecision
	ReviewRequests struct {
		Nodes []struct {
			RequestedReviewer []interface{}
		}
	} `graphql:"reviewRequests(first:100)"`
	ReviewThreads struct {
		TotalCount int
	}
	Reviews struct {
		TotalCount int
	}
	State            githubv4.PullRequestState
	Title            githubv4.String
	UpdatedAt        githubv4.DateTime
	Url              githubv4.URI
	UserContentEdits struct {
		TotalCount int
	}
}

type RepoPullRequestIteratorOptions struct {
	GitHubIteratorOptions
	github.PullRequestListOptions
}

func NewRepoPullRequestIterator(repoOwner, repoName string, options RepoPullRequestIteratorOptions) *RepoPullRequestIterator {
	prIter := &RepoPullRequestIterator{nil, nil, 0}
	//githubIter := NewGitHubIterator(prIter.fetchRepoPullRequestsPage, options.GitHubIteratorOptions)
	src := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: os.Getenv("GITHUB_AUTH_TOKEN")},
	)
	httpClient := oauth2.NewClient(context.Background(), src)

	client := githubv4.NewClient(httpClient)
	variables := map[string]interface{}{
		"owner":             githubv4.String(repoOwner),
		"name":              githubv4.String(repoName),
		"pullRequestCursor": (*githubv4.String)(nil),
	}
	var q Repo
	var pr []pullRequest
	for {
		err := client.Query(context.Background(), &q, variables)
		if err != nil {
			log.Fatalln(err)
		}
		pr = append(pr, q.Repository.PullRequests.Nodes...)
		if !q.Repository.PullRequests.PageInfo.HasNextPage {
			break
		}
		variables["pullRequestCursor"] = githubv4.String(q.Repository.PullRequests.PageInfo.EndCursor)
	}

	prIter.pr = pr
	prIter.repo = &q
	return prIter
}

func (prIter *RepoPullRequestIterator) Next() (*pullRequest, error) {
	if prIter.index < len(prIter.pr) {
		pr := prIter.pr[prIter.index]
		prIter.index++
		return &pr, nil
	} else {
		return nil, nil
	}

}
