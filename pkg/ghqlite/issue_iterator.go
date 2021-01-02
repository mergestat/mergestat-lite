package ghqlite

import (
	"context"

	"github.com/google/go-github/github"
)

// RepoIssueIterator iterates over GitHub pull requests belonging to a single repository
type RepoIssueIterator struct {
	options    RepoIssueIteratorOptions
	githubIter *GitHubIterator
	repoOwner  string
	repoName   string
}

type RepoIssueIteratorOptions struct {
	GitHubIteratorOptions
	github.IssueListByRepoOptions
}

func NewRepoIssueIterator(repoOwner, repoName string, options RepoIssueIteratorOptions) *RepoIssueIterator {
	issueIter := &RepoIssueIterator{options, nil, repoOwner, repoName}
	githubIter := NewGitHubIterator(issueIter.fetchRepoIssuePage, options.GitHubIteratorOptions)
	issueIter.githubIter = githubIter

	return issueIter
}

func (issueIter *RepoIssueIterator) fetchRepoIssuePage(githubIter *GitHubIterator, p int) ([]interface{}, *github.Response, error) {
	options := issueIter.options
	options.IssueListByRepoOptions.Page = p

	// use the user provided per page value, if it's greater than 0
	// otherwise don't set it and use the GitHub API default
	if options.PerPage > 0 {
		options.IssueListByRepoOptions.PerPage = githubIter.options.PerPage
	}

	issues, res, err := githubIter.options.Client.Issues.ListByRepo(context.Background(), issueIter.repoOwner, issueIter.repoName, &options.IssueListByRepoOptions)
	items := make([]interface{}, len(issues))
	for i, r := range issues {
		items[i] = r
	}

	return items, res, err
}

func (issueIter *RepoIssueIterator) Next() (*github.Issue, error) {
	issue, err := issueIter.githubIter.Next()
	if err != nil {
		print(err.Error())
		return nil, nil
	}

	if issue == nil {
		return nil, nil
	}

	return issue.(*github.Issue), nil
}
