package ghqlite

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/google/go-github/github"
	"github.com/shurcooL/githubv4"
	"golang.org/x/oauth2"
	"golang.org/x/time/rate"
)

// GitHubIterator iterates over resources from the GitHub API
type GitHubIterator struct {
	options GitHubIteratorOptions
}

type githubIteratorPage struct {
	items []interface{}
	res   *github.Response
}

type GitHubIteratorOptions struct {
	Client       *githubv4.Client // GitHub API client to use when making requests
	Token        string
	PerPage      int           // number of items per page, GitHub API caps it at 100
	PreloadPages int           // number of pages to "preload" - i.e. download concurrently
	RateLimiter  *rate.Limiter // rate limiter to use (tune to avoid hitting the API rate limits)
}

// we define a custom http.Transport here that removes the Accept header
// see this issue for why it needs to be done this way: https://github.com/google/go-github/issues/999
// the header is removed as the defaults used by go-github sometimes cause 502s from the GitHub API
type noAcceptTransport struct {
	originalTransport http.RoundTripper
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

func (t *noAcceptTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	r.Header.Del("Accept")
	return t.originalTransport.RoundTrip(r)
}

// NewGitHubIterator creates a *GitHubIterator
// oauth token and options. If the token is an empty string, no authentication is used
// note that unauthenticated requests are subject to a more stringent rate limit from the API
func NewGitHubIterator(options GitHubIteratorOptions) *GitHubIterator {
	fmt.Println(options)
	fmt.Println(options.Token)
	if options.Client == nil {
		if options.Token != "" { // if token is specified setup an oauth http client
			ts := oauth2.StaticTokenSource(
				&oauth2.Token{AccessToken: options.Token},
			)
			tc := oauth2.NewClient(context.Background(), ts)

			//tc.Transport = &noAcceptTransport{tc.Transport}
			options.Client = githubv4.NewClient(tc)
		} else {
			options.Client = githubv4.NewClient(nil)
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
		options.RateLimiter = rate.NewLimiter(rate.Every(10*time.Second), 8)
	}
	return &GitHubIterator{options}
}
