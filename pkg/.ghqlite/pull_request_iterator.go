package ghqlite

import (
	"context"

	"github.com/google/go-github/github"
)

// RepoPullRequestIterator iterates over GitHub pull requests belonging to a single repository
type RepoPullRequestIterator struct {
	options    RepoPullRequestIteratorOptions
	githubIter *GitHubIterator
	repoOwner  string
	repoName   string
}

type RepoPullRequestIteratorOptions struct {
	GitHubIteratorOptions
	github.PullRequestListOptions
}

func NewRepoPullRequestIterator(repoOwner, repoName string, options RepoPullRequestIteratorOptions) *RepoPullRequestIterator {
	prIter := &RepoPullRequestIterator{options, nil, repoOwner, repoName}
	githubIter := NewGitHubIterator(prIter.fetchRepoPullRequestsPage, options.GitHubIteratorOptions)
	prIter.githubIter = githubIter

	return prIter
}

func (prIter *RepoPullRequestIterator) fetchRepoPullRequestsPage(githubIter *GitHubIterator, p int) ([]interface{}, *github.Response, error) {
	options := prIter.options
	options.PullRequestListOptions.Page = p

	// use the user provided per page value, if it's greater than 0
	// otherwise don't set it and use the GitHub API default
	if options.PerPage > 0 {
		options.PullRequestListOptions.PerPage = githubIter.options.PerPage
	}

	prs, res, err := githubIter.options.Client.PullRequests.List(context.Background(), prIter.repoOwner, prIter.repoName, &options.PullRequestListOptions)
	items := make([]interface{}, len(prs))
	for i, r := range prs {
		items[i] = r
	}
	return items, res, err
}

func (prIter *RepoPullRequestIterator) Next() (*github.PullRequest, error) {
	pr, err := prIter.githubIter.Next()
	if err != nil {
		return nil, err
	}

	if pr == nil {
		return nil, nil
	}

	return pr.(*github.PullRequest), nil
}
