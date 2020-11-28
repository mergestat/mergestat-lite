package ghqlite

import (
	"context"
	"net/http"
	"time"

	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
	"golang.org/x/sync/errgroup"
	"golang.org/x/time/rate"
)

// RepoIterator iterates over GitHub repositories belonging to a single owner
type RepoIterator struct {
	options      *RepoIteratorOptions
	owner        string
	ownerType    OwnerType
	currentPages []*page
	totalPages   *int
	pageIndex    int
	repoIndex    int
}

type page struct {
	repos []*github.Repository
	res   *github.Response
}

type OwnerType string

// TODO this behavior might need to be split out into two separate iterators
// one for orgs, one for users
const (
	OwnerTypeOrganization OwnerType = "Organization"
	OwnerTypeUser         OwnerType = "User"
)

// RepoIteratorOptions determines how the iterator should behave
type RepoIteratorOptions struct {
	Client       *github.Client // GitHub API client to use when making requests
	PerPage      int            // number of repos per page, GitHub API caps it at 100
	PreloadPages int            // number of pages to "preload" - i.e. download concurrently
	RateLimiter  *rate.Limiter  // rate limiter to use (tune to avoid hitting the API rate limits)
}

// we define a custom http.Transport here that removes the Accept header
// see this issue for why it needs to be done this way: https://github.com/google/go-github/issues/999
// the header is removed as the defaults used by go-github sometimes cause 502s from the GitHub API
type noAcceptTransport struct {
	originalTransport http.RoundTripper
}

func (t *noAcceptTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	r.Header.Del("Accept")
	return t.originalTransport.RoundTrip(r)
}

// NewRepoIterator creates a *RepoIterator from an owner (GitHub organization or user)
// oauth token and options. If the token is an empty string, no authentication is used
// note that unauthenticated requests are subject to a more stringent rate limit from the API
func NewRepoIterator(owner string, ownerType OwnerType, token string, options *RepoIteratorOptions) *RepoIterator {
	if options.Client == nil {
		if token != "" { // if token is specified setup an oauth http client
			ts := oauth2.StaticTokenSource(
				&oauth2.Token{AccessToken: token},
			)
			tc := oauth2.NewClient(context.Background(), ts)

			tc.Transport = &noAcceptTransport{tc.Transport}
			options.Client = github.NewClient(tc)
		} else {
			options.Client = github.NewClient(nil)
		}
	}
	if options.PreloadPages <= 0 {
		// we want to make sure this value is always at least 1 - it's the number of pages
		// the iterator will fetch concurrently
		options.PreloadPages = 1
	}
	if options.RateLimiter == nil {
		// if the rate limiter is not provided, supply a default one
		// https://docs.github.com/en/free-pro-team@latest/developers/apps/rate-limits-for-github-apps
		options.RateLimiter = rate.NewLimiter(rate.Every(10*time.Second), 15)
	}
	return &RepoIterator{options, owner, ownerType, nil, nil, 0, 0}
}

// fetchPage retrieves a single page of repos
func (iter *RepoIterator) fetchPage(p int) ([]*github.Repository, *github.Response, error) {
	listOpt := github.ListOptions{Page: p}

	// use the user provided per page value, if it's greater than 0
	// otherwise don't set it and use the GitHub API default
	if iter.options.PerPage > 0 {
		listOpt.PerPage = iter.options.PerPage
	}

	switch iter.ownerType {
	case OwnerTypeOrganization:
		opt := &github.RepositoryListByOrgOptions{
			ListOptions: listOpt,
		}
		return iter.options.Client.Repositories.ListByOrg(context.Background(), iter.owner, opt)
	case OwnerTypeUser:
		opt := &github.RepositoryListOptions{
			ListOptions: listOpt,
		}
		return iter.options.Client.Repositories.List(context.Background(), iter.owner, opt)
	}

	// should never reach this point
	return nil, nil, nil
}

// fetchPages retries a *set* of pages given a nextPage
// if X is the nextPage and N is the preload pages value
// this will call fetchPage N times retrieving the X+N page
func (iter *RepoIterator) fetchPages(nextPage int) error {

	// retrieve the N pages concurrently
	g := new(errgroup.Group)
	for p := 0; p < iter.options.PreloadPages; p++ {

		// if we already know the total number of expected pages, and we're requesting a page outside of that
		// break the loop, since we've reached the end
		// if a current page is nil, it indicates we're over the last page
		if iter.totalPages != nil && nextPage+p > *iter.totalPages {
			iter.currentPages[p] = nil
			break
		}

		func(p int) {
			g.Go(func() error {
				// apply the rate limiter here
				err := iter.options.RateLimiter.Wait(context.Background())
				if err != nil {
					return err
				}

				// fetch the page
				repos, res, err := iter.fetchPage(nextPage + p)
				if err != nil {
					return err
				}

				// TODO remove this commented line at some point, it can be useful for debugging rate limit issues
				// fmt.Println(res.Rate.Limit, res.Rate.Remaining, res.Rate.Reset.Format(time.RFC3339))

				// if there are repos returned
				// if we've preloaded pages beyond the end of list, responses won't have repos
				if len(repos) > 0 {
					// store the new page we just retrieved
					// in currentPages in the right place
					newPage := page{repos, res}
					iter.currentPages[p] = &newPage
				}

				// if the response tells us what the last page is, set it
				// this is used above to check whether additional pages should be fetched
				if res.LastPage != 0 {
					iter.totalPages = &res.LastPage
				}

				return nil
			})
		}(p)
	}

	return g.Wait()
}

// Next yields the next repository in the iterator
// it should return nil, nil if the iteration is complete and there are no more repositories to retrieve
func (iter *RepoIterator) Next() (*github.Repository, error) {

	// if we are at the very beginning of the iteration, there will be no (nil) currentPages
	if iter.currentPages == nil {
		// initialize the currentPages the size of the number of pages to preload
		iter.currentPages = make([]*page, iter.options.PreloadPages)
		// fetch the first pages (starting at 1, but fetching N pages where N = number to preload)
		err := iter.fetchPages(1)
		if err != nil {
			return nil, err
		}
	}

	// if the repoIndex has exceeded the number of repos held in the current page by 1
	// increment to the next page and reset the repo index
	if iter.repoIndex == len(iter.currentPages[iter.pageIndex].repos) {
		iter.pageIndex++
		iter.repoIndex = 0
	}

	// if we've gone over the last page, however
	if iter.pageIndex == len(iter.currentPages) {
		// retrieve the last page we were on (but exhausted already)
		lastPage := iter.currentPages[iter.pageIndex-1]
		// if the API response for this previous page indicates there's no next page
		// we're at the end of the iteration, return nil
		next := lastPage.res.NextPage
		if next == 0 {
			return nil, nil
		}

		// otherwise, reset the page index and fetch the next batch of pages
		iter.pageIndex = 0
		err := iter.fetchPages(next)
		if err != nil {
			return nil, err
		}
	}

	// if we've reached a nil page
	// which is possible, as part of the batch may have exceeded the total number of pages
	// we're at the end of iteration
	if iter.currentPages[iter.pageIndex] == nil {
		return nil, nil
	}

	currentPage := iter.currentPages[iter.pageIndex]
	// fmt.Println(len(currentPage.repos))

	// finally, pull out the current repo the indices point to to be returned
	currentRepo := currentPage.repos[iter.repoIndex]
	iter.repoIndex++

	return currentRepo, nil
}
