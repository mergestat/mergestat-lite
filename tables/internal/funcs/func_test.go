package funcs

import (
	"database/sql"
	_ "github.com/askgitdev/askgit/pkg/sqlite"
	"github.com/pkg/errors"
	"go.riyazali.net/sqlite"
	"log"
	"os"
	"testing"
)

// FixtureDatabase represents the database connection to run the test against
var FixtureDatabase *sql.DB

func init() {
	// register sqlite extension when this package is loaded
	sqlite.Register(func(ext *sqlite.ExtensionApi) (_ sqlite.ErrorCode, err error) {
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
	})
}

func TestMain(m *testing.M) {
	var err error
	if FixtureDatabase, err = sql.Open("sqlite3", "file:testing.db?mode=memory"); err != nil {
		log.Fatalf("failed to open database connection: %v", err)
	}

	os.Exit(m.Run())
}
