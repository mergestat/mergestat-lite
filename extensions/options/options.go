package options

import (
	"context"

	"github.com/askgitdev/askgit/extensions/services"
	"github.com/go-git/go-git/v5"
	"github.com/shurcooL/githubv4"
)

// Options is the container for various different options
// and configurations that can be passed to tables.RegisterFn
// to conditionally include or tweak the extension module's behaviour
type Options struct {
	Locator services.RepoLocator

	// ExtraFunctions is used to determine whether or not to register the extra utility functions
	// bundled with this extension
	ExtraFunctions bool

	// GitHub set to true to register the GitHub tables/funcs
	GitHub bool

	// GitHubClientGetter overrides the default GitHub v4 client
	GitHubClientGetter func() *githubv4.Client

	// Context is a key-value store to pass along values to the underlying extensions
	Context services.Context
}

// OptionFn represents any function capable of customising or providing options
type OptionFn func(*Options)

// WithExtraFunctions configures the extension to also register the bundled
// utility sql routines.
func WithExtraFunctions() OptionFn {
	return func(o *Options) { o.ExtraFunctions = true }
}

// WithGitHub configures the extension to also register the GitHub related tables and funcs
func WithGitHub() OptionFn {
	return func(o *Options) { o.GitHub = true }
}

// WithGitHubClientGetter configures a way to use a custom GitHubv4 client
func WithGitHubClientGetter(getter func() *githubv4.Client) OptionFn {
	return func(o *Options) { o.GitHubClientGetter = getter }
}

// RepoLocatorFn is an adapter type that adapts any function with compatible
// signature to a RepoLocator instance.
type RepoLocatorFn func(ctx context.Context, path string) (*git.Repository, error)

func (fn RepoLocatorFn) Open(ctx context.Context, path string) (*git.Repository, error) {
	return fn(ctx, path)
}

// WithRepoLocator uses the provided locator implementation
// for locating and opening git repositories.
func WithRepoLocator(loc services.RepoLocator) OptionFn {
	return func(o *Options) { o.Locator = loc }
}

// WithContextValue sets a value on the options context.
// It will override any existing value set with the same key
func WithContextValue(key, value string) OptionFn {
	return func(o *Options) {
		if o.Context == nil {
			o.Context = make(map[string]string)
		}
		o.Context[key] = value
	}
}
