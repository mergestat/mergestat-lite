package ghqlite

import (
	"os"
	"testing"

	"github.com/google/go-github/github"
)

func TestRepoPullRequestIterator(t *testing.T) {
	iter := NewRepoPullRequestIterator("askgitdev", "askgit", RepoPullRequestIteratorOptions{
		GitHubIteratorOptions: GitHubIteratorOptions{Token: os.Getenv("GITHUB_TOKEN"), PerPage: 100, PreloadPages: 5},
		PullRequestListOptions: github.PullRequestListOptions{
			State:     "all",
			Sort:      "created",
			Direction: "desc",
		},
	})

	atLeastAsManyPRs := 10
	count := 0
	for {
		pr, err := iter.Next()
		if err != nil {
			t.Fatal(err)
		}
		if pr == nil {
			break
		}
		count++
	}

	if count < atLeastAsManyPRs {
		t.Fatalf("expected at least %d PRs, got %d", atLeastAsManyPRs, count)
	}
}
