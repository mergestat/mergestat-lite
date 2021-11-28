package git

import (
	"github.com/mergestat/mergestat/extensions/internal/git/native"
	"github.com/mergestat/mergestat/extensions/internal/git/utils"
	"github.com/mergestat/mergestat/extensions/options"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"go.riyazali.net/sqlite"
)

// Register registers git related functionality as a SQLite extension
func Register(ext *sqlite.ExtensionApi, opt *options.Options) (_ sqlite.ErrorCode, err error) {
	moduleOpts := &utils.ModuleOptions{
		Locator: opt.Locator,
		Context: opt.Context,
		Logger:  opt.Logger,
	}

	// by default use a NOOP logger so we don't need nil checks within the modules
	if moduleOpts.Logger == nil {
		l := zerolog.Nop()
		moduleOpts.Logger = &l
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
