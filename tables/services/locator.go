package services

import (
	"context"
	"github.com/go-git/go-git/v5"
)

// RepoLocator is a service that the virtual modules rely upon
// to create or open an existing git repository.
type RepoLocator interface {
	// Open opens the git repository specified by path.
	// The path need not be a file system path only and can also represent
	// a networked resource. The implementation should return a handle
	// to the initialized git repository instance, or throw an error.
	Open(ctx context.Context, path string) (*git.Repository, error)
}
