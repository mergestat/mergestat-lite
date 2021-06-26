package ghqlite

import (
	"fmt"
	"os"
	"testing"
	"time"

	"golang.org/x/time/rate"
)

func TestRepoIterator(t *testing.T) {
	testCases := []GitHubIteratorOptions{
		{PerPage: 1, PreloadPages: 1},
		{PerPage: 5, PreloadPages: 2},
		{PerPage: 100, PreloadPages: 2},
	}

	for _, options := range testCases {
		options.Token = os.Getenv("GITHUB_TOKEN")
		options.RateLimiter = rate.NewLimiter(rate.Every(2*time.Second), 1)
	}

	minRepos := 10
	for i, options := range testCases {
		iter := NewRepoIterator("askgitdev", OwnerTypeOrganization, options)

		t.Run(fmt.Sprintf("Case#%d", i), func(t *testing.T) {
			for k := 0; k < minRepos; k++ {
				repo, err := iter.Next()
				if err != nil {
					t.Fatal(err)
				}
				if repo == nil {
					t.Fatalf("expected at least %d repos", minRepos)
				}
				if repo.GetName() == "" {
					t.Fatalf("expected a repo name")
				}
			}
		})

	}
}
