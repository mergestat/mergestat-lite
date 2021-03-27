// Package tables provide implementation of the various underlying sqlite3 virtual tables [https://www.sqlite.org/vtab.html]
// that askgit uses under-the-hood. This module can be side-effect-imported in other modules to include the functionality
// of the sqlite3 virtual tables there.
package tables

import (
	"github.com/augmentable-dev/askgit/tables/internal/funcs"
	"github.com/augmentable-dev/askgit/tables/internal/git"
	"github.com/pkg/errors"
	"go.riyazali.net/sqlite"
)

func init() {
	// register sqlite extension when this package is loaded
	sqlite.Register(func(ext *sqlite.ExtensionApi) (_ sqlite.ErrorCode, err error) {
		// register virtual table modules
		var modules = map[string]sqlite.Module{
			"git_blame":    &git.BlameModule{},
			"git_branches": &git.BranchesModule{},
			"git_files":    &git.FilesModule{},
			"git_log":      &git.LogModule{},
			"git_stats":    &git.StatsModule{},
			"git_tags":     &git.TagsModule{},
		}

		for name, mod := range modules {
			if err = ext.CreateModule(name, mod); err != nil {
				return sqlite.SQLITE_ERROR, errors.Wrapf(err, "failed to register %q module", name)
			}
		}

		// register sql functions
		var fns = map[string]sqlite.Function{
			"str_split":    &funcs.StringSplit{},
			"toml_to_json": &funcs.TomlToJson{},
			"yaml_to_json": &funcs.YamlToJson{},
			"xml_to_json":  &funcs.XmlToJson{},
		}

		// alias yaml_to_json => yml_to_json
		fns["yml_to_json"] = fns["yaml_to_json"]

		for name, fn := range fns {
			if err = ext.CreateFunction(name, fn); err != nil {
				return sqlite.SQLITE_ERROR, errors.Wrapf(err, "failed to register %q function", name)
			}
		}

		return sqlite.SQLITE_OK, nil
	})
}
