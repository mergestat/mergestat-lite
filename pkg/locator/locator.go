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

	"github.com/go-git/go-billy/v5/osfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/cache"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/go-git/go-git/v5/storage/filesystem"
	"github.com/mergestat/mergestat/extensions/options"
	"github.com/mergestat/mergestat/extensions/services"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
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
		if dir, err = ioutil.TempDir(os.TempDir(), "mergestat"); err != nil {
			return nil, errors.Wrap(err, "failed to create a temporary directory")
		}

		var storer = filesystem.NewStorage(osfs.New(dir), cache.NewObjectLRUDefault())
		return git.CloneContext(ctx, storer, storer.Filesystem(), &git.CloneOptions{URL: path, NoCheckout: true})
	})
}

// SSHLocator returns a repo locator capable of cloning remote
// ssh repositories on-demand into temporary storage. It is recommended
// that you club it with something like CachedLocator to improve performance
// and remove the need to clone a single repository multiple times.
func SSHLocator() services.RepoLocator {
	return options.RepoLocatorFn(func(ctx context.Context, path string) (*git.Repository, error) {
		path = strings.TrimPrefix(path, "ssh://")

		// TODO(patrickdevivo) maybe a little hacky instead of properly parsing the url, strip out the username first
		// if it's set, otherwise default to "git"
		var user string
		split := strings.SplitN(path, "@", 2)
		switch len(split) {
		case 1:
			user = "git"
			path = split[0]
		case 2:
			user = split[0]
			path = split[1]
		}

		var dir string
		var err error
		if dir, err = ioutil.TempDir(os.TempDir(), "mergestat"); err != nil {
			return nil, errors.Wrap(err, "failed to create a temporary directory")
		}

		var storer = filesystem.NewStorage(osfs.New(dir), cache.NewObjectLRUDefault())
		var auth ssh.AuthMethod
		if auth, err = ssh.DefaultAuthBuilder(user); err != nil {
			return nil, errors.Wrap(err, "failed to create an SSH authentication method")
		}
		return git.CloneContext(ctx, storer, storer.Filesystem(), &git.CloneOptions{URL: path, NoCheckout: true, Auth: auth})
	})
}

// MultiLocator returns a locator service that work with multiple git protocols
// and is able to pick the correct underlying locator based on path provided.
func MultiLocator() services.RepoLocator {
	var locators = map[string]func() services.RepoLocator{
		"http": HttpLocator,
		"ssh":  SSHLocator,
		"file": DiskLocator,
	}

	return options.RepoLocatorFn(func(ctx context.Context, path string) (*git.Repository, error) {
		var fn = locators["file"] // file is the default locator
		if strings.HasPrefix(path, "http") || strings.HasPrefix(path, "https") {
			fn = locators["http"]
		}
		if strings.HasPrefix(path, "ssh") {
			fn = locators["ssh"]
		}
		return fn().Open(ctx, path)
	})
}

// LoggingLocator returns a locator that logs
func LoggingLocator(logger *zerolog.Logger, rl services.RepoLocator) services.RepoLocator {
	return options.RepoLocatorFn(func(ctx context.Context, path string) (*git.Repository, error) {
		logger.Info().Str("path", path).Msgf("opening repo")
		return rl.Open(ctx, path)
	})
}
