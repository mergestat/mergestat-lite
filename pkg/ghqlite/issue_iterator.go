package ghqlite

import (
	"context"

	"github.com/google/go-github/github"
	"github.com/shurcooL/githubv4"
)

// RepoIssueIterator iterates over GitHub pull requests belonging to a single repository
type RepoIssueIterator struct {
	//options    RepoPullRequestIteratorOptions
	repo           *Repo
	issues         []issue
	index          int
	githubIter     GitHubIterator
	issueVariables map[string]interface{}
}

type RepoIssueIteratorOptions struct {
	GitHubIteratorOptions
	github.IssueListByRepoOptions
}

func NewRepoIssueIterator(repoOwner, repoName string, options RepoIssueIteratorOptions) *RepoIssueIterator {
	githubIter := NewGitHubIterator(options.GitHubIteratorOptions)
	issueIter := &RepoIssueIterator{nil, nil, 0, *githubIter, nil}
	variables := map[string]interface{}{
		"owner":             githubv4.String(repoOwner),
		"name":              githubv4.String(repoName),
		"pullRequestCursor": (*githubv4.String)(nil),
		"issueCursor":       (*githubv4.String)(nil),
	}
	issueIter.issueVariables = variables
	err := getIssues(githubIter, issueIter)
	if err != nil {
		panic(err)
	}

	return issueIter
}
func getIssues(githubIter *GitHubIterator, issueIter *RepoIssueIterator) error {
	var repo Repo
	var issues []issue

	for i := 0; i < githubIter.options.PreloadPages; i++ {
		err := githubIter.options.Client.Query(context.Background(), &repo, issueIter.issueVariables)
		if err != nil {
			return err
		}
		issues = append(issues, repo.Repository.Issues.Nodes...)
		if !repo.Repository.Issues.PageInfo.HasNextPage {
			break
		}
		issueIter.issueVariables["issueCursor"] = githubv4.String(repo.Repository.Issues.PageInfo.EndCursor)
	}
	// no need to be constantly reassigning the repo
	if issueIter.repo == nil {
		issueIter.repo = &repo
	}
	issueIter.issues = issues
	return nil
}

func (issueIter *RepoIssueIterator) Next() (*issue, error) {
	if issueIter.index < len(issueIter.issues) {
		pr := issueIter.issues[issueIter.index]
		issueIter.index++
		return &pr, nil
	} else {
		var q Repo
		err := issueIter.githubIter.options.Client.Query(context.Background(), &q, issueIter.issueVariables)
		if err != nil {
			return nil, err
		}
		if !q.Repository.Issues.PageInfo.HasNextPage {
			return nil, nil
		}
		err = getIssues(&issueIter.githubIter, issueIter)
		if err != nil {
			return nil, err
		}
		issueIter.index = 0
		return issueIter.Next()

	}

}
