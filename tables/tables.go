// Package tables provide implementation of the various underlying sqlite3 virtual tables [https://www.sqlite.org/vtab.html]
// that askgit uses under-the-hood. This module can be side-effect-imported in other modules to include the functionality
// of the sqlite3 virtual tables there.
package tables

import (
	"github.com/askgitdev/askgit/tables/internal/funcs"
	"github.com/askgitdev/askgit/tables/internal/git"
	"github.com/pkg/errors"
	"go.riyazali.net/sqlite"
)

func RegisterFn(fns ...OptionFn) func(ext *sqlite.ExtensionApi) (_ sqlite.ErrorCode, err error) {
	var opt = &Options{}
	for _, fn := range fns {
		fn(opt)
	}

	// return an extension function that register modules with sqlite when this package is loaded
	return func(ext *sqlite.ExtensionApi) (_ sqlite.ErrorCode, err error) {
		// register virtual table modules
		var modules = map[string]sqlite.Module{
			"commits": &git.LogModule{Locator: opt.Locator},
			"refs":    &git.RefModule{Locator: opt.Locator},
			"stats":   git.NewStatsModule(opt.Locator),
		}

		for name, mod := range modules {
			if err = ext.CreateModule(name, mod); err != nil {
				return sqlite.SQLITE_ERROR, errors.Wrapf(err, "failed to register %q module", name)
			}
		}

		var fns = map[string]sqlite.Function{
			"commit_from_tag": &git.CommitFromTagFn{},
		}

		for name, fn := range fns {
			if err = ext.CreateFunction(name, fn); err != nil {
				return sqlite.SQLITE_ERROR, errors.Wrapf(err, "failed to register %q function", name)
			}
		}

		// only conditionally register the utility functions
		if opt.ExtraFunctions {
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
		}

		return sqlite.SQLITE_OK, nil
	}
}
