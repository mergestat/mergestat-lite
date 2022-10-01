// Package extensions provide implementation of the various underlying sqlite3 virtual tables [https://www.sqlite.org/vtab.html] and user defined functions
// that mergestat uses under-the-hood. This module can be side-effect-imported in other modules to include the functionality
// of the sqlite3 extensions there.
package extensions

import (
	"github.com/mergestat/mergestat-lite/extensions/internal/enry"
	"github.com/mergestat/mergestat-lite/extensions/internal/git"
	"github.com/mergestat/mergestat-lite/extensions/internal/github"
	"github.com/mergestat/mergestat-lite/extensions/internal/golang"
	"github.com/mergestat/mergestat-lite/extensions/internal/helpers"
	"github.com/mergestat/mergestat-lite/extensions/internal/npm"
	"github.com/mergestat/mergestat-lite/extensions/internal/sourcegraph"
	"github.com/mergestat/mergestat-lite/extensions/options"
	"go.riyazali.net/sqlite"
)

func RegisterFn(fns ...options.OptionFn) func(ext *sqlite.ExtensionApi) (_ sqlite.ErrorCode, err error) {
	var opt = &options.Options{}
	for _, fn := range fns {
		fn(opt)
	}

	// return an extension function that register modules with sqlite when this package is loaded
	return func(ext *sqlite.ExtensionApi) (_ sqlite.ErrorCode, err error) {
		if !opt.ExcludeGit {
			// register the git tables
			if sqliteErr, err := git.Register(ext, opt); err != nil {
				return sqliteErr, err
			}
		}

		// only conditionally register the utility functions
		if opt.ExtraFunctions {
			if sqliteErr, err := helpers.Register(ext, opt); err != nil {
				return sqliteErr, err
			}

			if sqliteErr, err := enry.Register(ext, opt); err != nil {
				return sqliteErr, err
			}

			if sqliteErr, err := golang.Register(ext, opt); err != nil {
				return sqliteErr, err
			}
		}

		// conditionally register the GitHub functionality
		if opt.GitHub {
			if sqliteErr, err := github.Register(ext, opt); err != nil {
				return sqliteErr, err
			}
		}

		if opt.Sourcegraph {
			if sqliteErr, err := sourcegraph.Register(ext, opt); err != nil {
				return sqliteErr, err
			}
		}

		if opt.NPM {
			if sqliteErr, err := npm.Register(ext, opt); err != nil {
				return sqliteErr, err
			}
		}

		return sqlite.SQLITE_OK, nil
	}
}
