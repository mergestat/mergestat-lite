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
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/mergestat/mergestat-lite/extensions/options"
	"github.com/mergestat/mergestat-lite/extensions/services"
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
	cache := sync.Map{}

	return options.RepoLocatorFn(func(ctx context.Context, path string) (*git.Repository, error) {
		if cached, ok := cache.Load(path); ok {
			return cached.(*git.Repository), nil
		}

		repo, err := rl.Open(ctx, path)
		if err != nil {
			return nil, err
		}

		cache.Store(path, repo)
		return repo, nil
	})
}

// determineCloneDir returns the path to a directory on disk where a repository will be cloned to
// given a baseCloneDir. If baseCloneDir == "", a tmp dir will be created, otherwise a directory
// path will be determined based on the URL (HTTP(s) or SSH) of the provided repository.
// The bool returned (2nd return val) indicates whether the output dir is in a tmp directory or not.
func determineCloneDir(path, baseCloneDir string) (string, bool, error) {
	var err error
	var parsed *url.URL
	if parsed, err = url.ParseRequestURI(path); err != nil {
		return "", false, errors.Wrap(err, "invalid remote url")
	}

	// if no clone directory is specified, use a tmp dir
	if baseCloneDir == "" {
		if baseCloneDir, err = os.MkdirTemp("", "mergestat"); err != nil {
			return "", false, errors.Wrap(err, "failed to create a temporary directory")
		}

		return baseCloneDir, true, nil
	}

	// if clone directory is set, get the abs path
	if baseCloneDir, err = filepath.Abs(baseCloneDir); err != nil {
		return "", false, errors.Wrap(err, "failed to retrieve absolute path for clone directory")
	}

	// then use the parsed path to determine where repos should end up
	if strings.HasPrefix(path, "http") {
		baseCloneDir = filepath.Join(baseCloneDir, parsed.Hostname(), parsed.EscapedPath()[1:])
	} else { // assume it's an ssh repo
		baseCloneDir = filepath.Join(baseCloneDir, strings.Replace(parsed.String(), ":", "/", 1))
	}

	if _, err = os.Stat(baseCloneDir); os.IsNotExist(err) {
		if err = os.MkdirAll(baseCloneDir, 0755); err != nil {
			return "", false, errors.Wrap(err, "failed to create clone directory")
		}
	}

	return baseCloneDir, false, nil
}

// HttpLocator returns a repo locator capable of cloning remote
// http repositories on-demand into temporary storage. It is recommended
// that you club it with something like CachedLocator to improve performance
// and remove the need to clone a single repository multiple times.
func HttpLocator(o *MultiLocatorOptions) func() services.RepoLocator {
	return func() services.RepoLocator {
		return options.RepoLocatorFn(func(ctx context.Context, path string) (*git.Repository, error) {
			var err error
			if _, err = url.ParseRequestURI(path); err != nil {
				return nil, errors.Wrap(err, "invalid remote url")
			}

			var cd string
			var isTmp bool
			if cd, isTmp, err = determineCloneDir(path, o.CloneDir); err != nil {
				return nil, errors.Wrap(err, "could not determine clone directory")
			}

			return git.PlainCloneContext(ctx, cd, isTmp, &git.CloneOptions{URL: path, InsecureSkipTLS: o.InsecureSkipTLS})
		})
	}
}

// httpLocatorWithAuth returns a func that returns a repo locator 🤯
// its primary intended use is below in the MultiLocator, which receives options.
// If HTTP auth options are supplied, they will be used when cloning an https (only https) repo.
func httpLocatorWithAuth(user, pass string, rl services.RepoLocator) func() services.RepoLocator {
	return func() services.RepoLocator {
		return options.RepoLocatorFn(func(ctx context.Context, path string) (*git.Repository, error) {
			if !strings.HasPrefix(path, "https") {
				return rl.Open(ctx, path)
			}

			if parsed, err := url.Parse(path); err != nil {
				return nil, err
			} else {
				pass, passSet := parsed.User.Password()
				if parsed.User.Username() == "" && !passSet {
					parsed.User = url.UserPassword(user, pass)
					path = parsed.String()
				}
			}

			return rl.Open(ctx, path)
		})
	}
}

// SSHLocator returns a repo locator capable of cloning remote
// ssh repositories on-demand into temporary storage. It is recommended
// that you club it with something like CachedLocator to improve performance
// and remove the need to clone a single repository multiple times.
func SSHLocator(o *MultiLocatorOptions) func() services.RepoLocator {
	return func() services.RepoLocator {
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

			var cd string
			var isTmp bool
			var err error
			if cd, isTmp, err = determineCloneDir(path, o.CloneDir); err != nil {
				return nil, errors.Wrap(err, "could not determine clone directory")
			}

			var auth ssh.AuthMethod
			if auth, err = ssh.DefaultAuthBuilder(user); err != nil {
				return nil, errors.Wrap(err, "failed to create an SSH authentication method")
			}

			return git.PlainCloneContext(ctx, cd, isTmp, &git.CloneOptions{URL: path, Auth: auth, InsecureSkipTLS: o.InsecureSkipTLS})
		})
	}
}

type MultiLocatorOptions struct {
	HTTPAuth        *http.BasicAuth
	CloneDir        string
	InsecureSkipTLS bool
}

// MultiLocator returns a locator service that work with multiple git protocols
// and is able to pick the correct underlying locator based on path provided.
func MultiLocator(o *MultiLocatorOptions) services.RepoLocator {
	if o == nil {
		o = &MultiLocatorOptions{}
	}
	var locators = map[string]func() services.RepoLocator{
		"http": HttpLocator(o),
		"ssh":  SSHLocator(o),
		"file": DiskLocator,
	}

	return options.RepoLocatorFn(func(ctx context.Context, path string) (*git.Repository, error) {
		var fn = locators["file"] // file is the default locator
		if strings.HasPrefix(path, "http") || strings.HasPrefix(path, "https") {
			fn = locators["http"]
			if o.HTTPAuth != nil {
				fn = httpLocatorWithAuth(o.HTTPAuth.Username, o.HTTPAuth.Password, fn())
			}
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
