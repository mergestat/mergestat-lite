package funcs

import (
	"database/sql"
	"log"
	"os"
	"testing"

	_ "github.com/askgitdev/askgit/pkg/sqlite"
	"github.com/pkg/errors"
	"go.riyazali.net/sqlite"
)

// FixtureDatabase represents the database connection to run the test against
var FixtureDatabase *sql.DB

func init() {
	// register sqlite extension when this package is loaded
	sqlite.Register(func(ext *sqlite.ExtensionApi) (_ sqlite.ErrorCode, err error) {
		var fns = map[string]sqlite.Function{
			"str_split":             &StringSplit{},
			"toml_to_json":          &TomlToJson{},
			"yaml_to_json":          &YamlToJson{},
			"xml_to_json":           &XmlToJson{},
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

func TestMain(m *testing.M) {
	var err error
	if FixtureDatabase, err = sql.Open("sqlite3", "file:testing.db?mode=memory"); err != nil {
		log.Fatalf("failed to open database connection: %v", err)
	}

	os.Exit(m.Run())
}
