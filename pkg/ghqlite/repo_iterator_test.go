package ghqlite

import (
	"fmt"
	"testing"
)

func TestRepoIterator(t *testing.T) {
	testCases := []*RepoIteratorOptions{
		{PerPage: 1, PreloadPages: 1},
		{PerPage: 5, PreloadPages: 2},
		{PerPage: 100, PreloadPages: 2},
	}

	minRepos := 10
	for i, options := range testCases {
		iter := NewRepoIterator("augmentable-dev", OwnerTypeOrganization, "", options)

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
