// Package tables provide implementation of the various underlying sqlite3 virtual tables [https://www.sqlite.org/vtab.html]
// that askgit uses under-the-hood. This module can be side-effect-imported in other modules to include the functionality
// of the sqlite3 virtual tables there.
package tables

import (
	"github.com/askgitdev/askgit/tables/internal/funcs"
	"github.com/askgitdev/askgit/tables/internal/funcs/enry"
	"github.com/askgitdev/askgit/tables/internal/git"
	"github.com/askgitdev/askgit/tables/internal/git/native"
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
			"commits": &git.LogModule{Locator: opt.Locator, Context: opt.Context},
			"refs":    &git.RefModule{Locator: opt.Locator, Context: opt.Context},
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
				"str_split":             &funcs.StringSplit{},
				"toml_to_json":          &funcs.TomlToJson{},
				"yaml_to_json":          &funcs.YamlToJson{},
				"xml_to_json":           &funcs.XmlToJson{},
				"enry_detect_language":  &enry.EnryDetectLanguage{},
				"enry_is_binary":        &enry.EnryIsBinary{},
				"enry_is_configuration": &enry.EnryIsConfiguration{},
				"enry_is_documentation": &enry.EnryIsDocumentation{},
				"enry_is_dot_file":      &enry.EnryIsDotFile{},
				"enry_is_generated":     &enry.EnryIsGenerated{},
				"enry_is_image":         &enry.EnryIsImage{},
				//"enry_is_test":          &enry.EnryIsTest{},
				"enry_is_vendor": &enry.EnryIsVendor{},
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
