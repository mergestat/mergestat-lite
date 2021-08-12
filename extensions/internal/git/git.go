package git

import (
	"github.com/askgitdev/askgit/extensions/internal/git/native"
	"github.com/askgitdev/askgit/extensions/options"
	"github.com/pkg/errors"
	"go.riyazali.net/sqlite"
)

// Register registers git related functionality as a SQLite extension
func Register(ext *sqlite.ExtensionApi, opt *options.Options) (_ sqlite.ErrorCode, err error) {
	// register virtual table modules
	var modules = map[string]sqlite.Module{
		"commits": &LogModule{Locator: opt.Locator, Context: opt.Context},
		"refs":    &RefModule{Locator: opt.Locator, Context: opt.Context},
		"stats":   native.NewStatsModule(opt.Locator, opt.Context),
		"files":   native.NewFilesModule(opt.Locator, opt.Context),
		"blame":   native.NewBlameModule(opt.Locator, opt.Context),
	}

	for name, mod := range modules {
		if err = ext.CreateModule(name, mod); err != nil {
			return sqlite.SQLITE_ERROR, errors.Wrapf(err, "failed to register %q module", name)
		}
	}

	var fns = map[string]sqlite.Function{
		"commit_from_tag": &CommitFromTagFn{},
	}

	for name, fn := range fns {
		if err = ext.CreateFunction(name, fn); err != nil {
			return sqlite.SQLITE_ERROR, errors.Wrapf(err, "failed to register %q function", name)
		}
	}

	return sqlite.SQLITE_OK, nil
}
