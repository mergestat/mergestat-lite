package helpers

import (
	"github.com/pkg/errors"
	"go.riyazali.net/sqlite"
)

func Register(ext *sqlite.ExtensionApi) (_ sqlite.ErrorCode, err error) {
	var fns = map[string]sqlite.Function{
		"str_split":    &StringSplit{},
		"toml_to_json": &TomlToJson{},
		"yaml_to_json": &YamlToJson{},
		"xml_to_json":  &XmlToJson{},
	}

	// alias yaml_to_json => yml_to_json
	fns["yml_to_json"] = fns["yaml_to_json"]

	for name, fn := range fns {
		if err = ext.CreateFunction(name, fn); err != nil {
			return sqlite.SQLITE_ERROR, errors.Wrapf(err, "failed to register %q function", name)
		}
	}

	return sqlite.SQLITE_OK, nil
}
