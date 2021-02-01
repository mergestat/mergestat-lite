package ghqlite

import (
	"context"
	"fmt"

	"github.com/google/go-github/github"
	"github.com/shurcooL/githubv4"
)

// RepoPullRequestIterator iterates over GitHub pull requests belonging to a single repository
type RepoPullRequestIterator struct {
	//options    RepoPullRequestIteratorOptions
	repo        *Repo
	pr          []pullRequest
	index       int
	githubIter  GitHubIterator
	prVariables map[string]interface{}
}

type RepoPullRequestIteratorOptions struct {
	GitHubIteratorOptions
	github.PullRequestListOptions
}

func NewRepoPullRequestIterator(repoOwner, repoName string, options RepoPullRequestIteratorOptions) *RepoPullRequestIterator {
	githubIter := NewGitHubIterator(options.GitHubIteratorOptions)
	prIter := &RepoPullRequestIterator{nil, nil, 0, *githubIter, nil}

	variables := map[string]interface{}{
		"owner":             githubv4.String(repoOwner),
		"name":              githubv4.String(repoName),
		"pullRequestCursor": (*githubv4.String)(nil),
	}
	prIter.prVariables = variables
	err := getPullRequests(githubIter, prIter)
	if err != nil {
		fmt.Println(err.Error())
		panic(err)
	}
	return prIter
}

func (prIter *RepoPullRequestIterator) Next() (*pullRequest, error) {
	if prIter.index < len(prIter.pr) {
		pr := prIter.pr[prIter.index]
		prIter.index++
		return &pr, nil
	} else {
		var q Repo
		err := prIter.githubIter.options.Client.Query(context.Background(), &q, prIter.prVariables)
		if err != nil {
			return nil, err
		}
		if !q.Repository.PullRequests.PageInfo.HasNextPage {
			return nil, nil
		}
		err = getPullRequests(&prIter.githubIter, prIter)
		if err != nil {
			return nil, err
		}
		prIter.index = 0
		return prIter.Next()

	}

}
func getPullRequests(githubIter *GitHubIterator, prIter *RepoPullRequestIterator) error {
	var repo Repo
	var pr []pullRequest

	for i := 0; i < githubIter.options.PreloadPages; i++ {
		err := githubIter.options.Client.Query(context.Background(), &repo, prIter.prVariables)
		if err != nil {
			return err
		}
		pr = append(pr, repo.Repository.PullRequests.Nodes...)
		if !repo.Repository.PullRequests.PageInfo.HasNextPage {
			break
		}
		prIter.prVariables["pullRequestCursor"] = githubv4.String(repo.Repository.PullRequests.PageInfo.EndCursor)
	}
	// no need to be constantly reassigning the repo
	if prIter.repo == nil {
		prIter.repo = &repo
	}
	prIter.pr = pr
	return nil
}
