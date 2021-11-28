package enry

import (
	"github.com/mergestat/mergestat/extensions/options"
	"github.com/pkg/errors"
	"go.riyazali.net/sqlite"
)

// Register registers enry related functionality as a SQLite extension
func Register(ext *sqlite.ExtensionApi, _ *options.Options) (_ sqlite.ErrorCode, err error) {
	var fns = map[string]sqlite.Function{
		"enry_detect_language":  &EnryDetectLanguage{},
		"enry_is_binary":        &EnryIsBinary{},
		"enry_is_configuration": &EnryIsConfiguration{},
		"enry_is_documentation": &EnryIsDocumentation{},
		"enry_is_dot_file":      &EnryIsDotFile{},
		"enry_is_generated":     &EnryIsGenerated{},
		"enry_is_image":         &EnryIsImage{},
		"enry_is_test":          &EnryIsTest{},
		"enry_is_vendor":        &EnryIsVendor{},
	}

	for name, fn := range fns {
		if err = ext.CreateFunction(name, fn); err != nil {
			return sqlite.SQLITE_ERROR, errors.Wrapf(err, "failed to register %q function", name)
		}
	}

	return sqlite.SQLITE_OK, nil
}
