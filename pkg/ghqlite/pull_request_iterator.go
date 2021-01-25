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
	Additions   githubv4.Int
	Author      user
	BodyText    githubv4.String
	ClosedAt    githubv4.DateTime
	CreatedAt   githubv4.DateTime
	DatabaseID  githubv4.Int
	Locked      githubv4.Boolean
	MergedAt    githubv4.DateTime
	MergeCommit struct {
		Oid githubv4.GitObjectID
	}
	Number    githubv4.Int
	State     githubv4.PullRequestState
	Title     githubv4.String
	UpdatedAt githubv4.DateTime
	// ReviewRequests struct {
	// 	Nodes []struct {

	// 	}
	// } `graphql:"reviewRequests(first:100)"`
	HeadRefOid  githubv4.GitObjectID
	HeadRefName string
	HeadRef     struct {
		Name string
	}
	BaseRefOid  githubv4.GitObjectID
	BaseRefName string
	BaseRef     struct {
		Name string
	}
	AuthorAssociation githubv4.CommentAuthorAssociation
	Merged            githubv4.Boolean
	Mergeable         githubv4.MergeableState
	MergedBy          user
	Comments          struct {
		TotalCount int
	} `graphql:"comments(first:1)"`
	MaintainerCanModify githubv4.Boolean
	Commits             struct {
		TotalCount int
	} `graphql:"commits(first:1)"`
	Deletions    int
	ChangedFiles int
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
