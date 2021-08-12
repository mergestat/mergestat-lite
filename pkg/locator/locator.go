// Package locator provides various different implementations of
// git.RepoLocator service. The locator service is used by the implementations
// of Git virtual modules to query for / locate a given repository.
//
// The various different implementations of this interface provided in this package
// provide different ways to locate the service, while some provides additional services
// such as caching or switching between implementations.
package locator

import (
	"context"
	"io/ioutil"
	"net/url"
	"os"
	"strings"

	"github.com/askgitdev/askgit/extensions/options"
	"github.com/askgitdev/askgit/extensions/services"
	"github.com/go-git/go-billy/v5/osfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/cache"
	"github.com/go-git/go-git/v5/storage/filesystem"
	"github.com/pkg/errors"
)

// DiskLocator is a repo locator implementation that opens on-disk repository at the specified path.
func DiskLocator() services.RepoLocator {
	return options.RepoLocatorFn(func(_ context.Context, path string) (*git.Repository, error) {
		return git.PlainOpen(path)
	})
}

// CachedLocator is decorator function that takes a RepoLocator instance
// and returns another one that caches output from the underlying locator
// using path as the key.
func CachedLocator(rl services.RepoLocator) services.RepoLocator {
	var cache = make(map[string]*git.Repository)

	return options.RepoLocatorFn(func(ctx context.Context, path string) (*git.Repository, error) {
		if cached, ok := cache[path]; ok {
			return cached, nil
		}

		repo, err := rl.Open(ctx, path)
		if err != nil {
			return nil, err
		}

		cache[path] = repo
		return repo, nil
	})
}

// HttpLocator returns a repo locator capable of cloning remote
// http repositories on-demand into temporary storage. It is recommended
// that you club it with something like CachedLocator to improve performance
// and remove the need to clone a single repository multiple times.
func HttpLocator() services.RepoLocator {
	return options.RepoLocatorFn(func(ctx context.Context, path string) (*git.Repository, error) {
		var err error
		if _, err = url.ParseRequestURI(path); err != nil {
			return nil, errors.Wrap(err, "invalid remote url")
		}

		var dir string
		if dir, err = ioutil.TempDir(os.TempDir(), "askgit"); err != nil {
			return nil, errors.Wrap(err, "failed to create a temporary directory")
		}

		var storer = filesystem.NewStorage(osfs.New(dir), cache.NewObjectLRUDefault())
		return git.CloneContext(ctx, storer, storer.Filesystem(), &git.CloneOptions{URL: path, NoCheckout: true})
	})
}

// MultiLocator returns a locator service that work with multiple git protocols
// and is able to pick the correct underlying locator based on path provided.
func MultiLocator() services.RepoLocator {
	var locators = map[string]func() services.RepoLocator{
		"http": HttpLocator,
		"file": DiskLocator,
	}

	return options.RepoLocatorFn(func(ctx context.Context, path string) (*git.Repository, error) {
		var fn = locators["file"] // file is the default locator
		if strings.HasPrefix(path, "http") || strings.HasPrefix(path, "https") {
			fn = locators["http"]
		}
		return fn().Open(ctx, path)
	})
}
