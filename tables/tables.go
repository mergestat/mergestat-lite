// Package tables provide implementation of the various underlying sqlite3 virtual tables [https://www.sqlite.org/vtab.html]
// that askgit uses under-the-hood. This module can be side-effect-imported in other modules to include the functionality
// of the sqlite3 virtual tables there.
package tables

import (
	"github.com/augmentable-dev/askgit/tables/internal/git"
	"github.com/pkg/errors"
	"go.riyazali.net/sqlite"
)

func init() {
	// register sqlite extension when this package is loaded
	sqlite.Register(func(ext *sqlite.ExtensionApi) (_ sqlite.ErrorCode, err error) {
		if err = ext.CreateModule("git_blame", &git.BlameModule{}); err != nil {
			return sqlite.SQLITE_ERROR, errors.Wrap(err, "failed to register 'git_blame' module")
		}

		if err = ext.CreateModule("git_branches", &git.BranchesModule{}); err != nil {
			return sqlite.SQLITE_ERROR, errors.Wrap(err, "failed to register 'git_branches' module")
		}

		if err = ext.CreateModule("git_files", &git.FilesModule{}); err != nil {
			return sqlite.SQLITE_ERROR, errors.Wrap(err, "failed to register 'git_files' module")
		}

		if err = ext.CreateModule("git_log", &git.LogModule{}); err != nil {
			return sqlite.SQLITE_ERROR, errors.Wrap(err, "failed to register 'git_log' module")
		}

		if err = ext.CreateModule("git_stats", &git.StatsModule{}); err != nil {
			return sqlite.SQLITE_ERROR, errors.Wrap(err, "failed to register 'git_stats' module")
		}

		if err = ext.CreateModule("git_tags", &git.TagsModule{}); err != nil {
			return sqlite.SQLITE_ERROR, errors.Wrap(err, "failed to register 'git_tags' module")
		}

		return sqlite.SQLITE_OK, nil
	})
}
