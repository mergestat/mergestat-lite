package helpers

import (
	"github.com/mergestat/mergestat-lite/extensions/options"
	"github.com/pkg/errors"
	"go.riyazali.net/sqlite"
)

// Register registers helpers as a SQLite extension
func Register(ext *sqlite.ExtensionApi, _ *options.Options) (_ sqlite.ErrorCode, err error) {
	var fns = map[string]sqlite.Function{
		"str_split":    &StringSplit{},
		"toml_to_json": &TomlToJson{},
		"yaml_to_json": &YamlToJson{},
		"xml_to_json":  &XmlToJson{},
		"time_diff":    &TimeDiff{},
		"approx_dur":   &ApproxDuration{},
	}

	// alias yaml_to_json => yml_to_json
	fns["yml_to_json"] = fns["yaml_to_json"]

	for name, fn := range fns {
		if err = ext.CreateFunction(name, fn); err != nil {
			return sqlite.SQLITE_ERROR, errors.Wrapf(err, "failed to register %q function", name)
		}
	}

	var modules = map[string]sqlite.Module{
		"grep":      NewGrepModule(),
		"str_split": NewStrSplitModule(),
	}

	for name, mod := range modules {
		if err = ext.CreateModule(name, mod); err != nil {
			return sqlite.SQLITE_ERROR, errors.Wrapf(err, "failed to register %q module", name)
		}
	}

	return sqlite.SQLITE_OK, nil
}
