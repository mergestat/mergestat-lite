package git

import (
	"github.com/askgitdev/askgit/extensions/internal/git/native"
	"github.com/askgitdev/askgit/extensions/internal/git/utils"
	"github.com/askgitdev/askgit/extensions/options"
	"github.com/pkg/errors"
	"go.riyazali.net/sqlite"
)

// Register registers git related functionality as a SQLite extension
func Register(ext *sqlite.ExtensionApi, opt *options.Options) (_ sqlite.ErrorCode, err error) {

	moduleOpts := &utils.ModuleOptions{
		Locator: opt.Locator,
		Context: opt.Context,
		Logger:  opt.Logger,
	}

	// register virtual table modules
	var modules = map[string]sqlite.Module{
		"commits": NewLogModule(moduleOpts),
		"refs":    NewRefModule(moduleOpts),
		"stats":   native.NewStatsModule(moduleOpts),
		"files":   native.NewFilesModule(moduleOpts),
		"blame":   native.NewBlameModule(moduleOpts),
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
