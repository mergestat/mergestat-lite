package ghqlite

import (
	"context"

	"github.com/google/go-github/github"
)

// RepoIterator iterates over GitHub repositories belonging to a single owner
type RepoIterator struct {
	githubIter *GitHubIterator
	owner      string
	ownerType  OwnerType
}

type OwnerType string

// TODO this behavior might need to be split out into two separate iterators
// one for orgs, one for users
const (
	OwnerTypeOrganization OwnerType = "Organization"
	OwnerTypeUser         OwnerType = "User"
)

// NewRepoIterator creates a *RepoIterator from an owner (GitHub organization or user)
// oauth token and options. If the token is an empty string, no authentication is used
// note that unauthenticated requests are subject to a more stringent rate limit from the API
func NewRepoIterator(owner string, ownerType OwnerType, options GitHubIteratorOptions) *RepoIterator {
	repoIter := &RepoIterator{
		owner:     owner,
		ownerType: ownerType,
	}
	githubIter := NewGitHubIterator(repoIter.fetchRepoPage, options)
	repoIter.githubIter = githubIter

	return repoIter
}

func (repoIter *RepoIterator) fetchRepoPage(githubIter *GitHubIterator, p int) ([]interface{}, *github.Response, error) {
	listOpt := github.ListOptions{Page: p}

	// use the user provided per page value, if it's greater than 0
	// otherwise don't set it and use the GitHub API default
	if githubIter.options.PerPage > 0 {
		listOpt.PerPage = githubIter.options.PerPage
	}

	switch repoIter.ownerType {
	case OwnerTypeOrganization:
		opt := &github.RepositoryListByOrgOptions{
			ListOptions: listOpt,
		}
		repos, res, err := githubIter.options.Client.Repositories.ListByOrg(context.Background(), repoIter.owner, opt)
		items := make([]interface{}, len(repos))
		for i, r := range repos {
			items[i] = r
		}
		return items, res, err
	case OwnerTypeUser:
		opt := &github.RepositoryListOptions{
			ListOptions: listOpt,
		}
		repos, res, err := githubIter.options.Client.Repositories.List(context.Background(), repoIter.owner, opt)
		items := make([]interface{}, len(repos))
		for i, r := range repos {
			items[i] = r
		}
		return items, res, err
	}

	// should never reach this point
	return nil, nil, nil
}

func (repoIter *RepoIterator) Next() (*github.Repository, error) {
	repo, err := repoIter.githubIter.Next()
	if err != nil {
		return nil, err
	}

	if repo == nil {
		return nil, nil
	}

	return repo.(*github.Repository), nil
}
