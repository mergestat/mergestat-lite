package funcs

import (
	"encoding/json"

	"github.com/BurntSushi/toml"
	"go.riyazali.net/sqlite"
)

// TomlToJson implements toml_to_json sql function.
// The function signature of the equivalent sql function is:
//     toml_to_json(string) string
type TomlToJson struct{}

func (y *TomlToJson) Args() int           { return 1 }
func (y *TomlToJson) Deterministic() bool { return true }

func (y *TomlToJson) Apply(context *sqlite.Context, value ...sqlite.Value) {
	var x interface{}
	if _, err := toml.Decode(value[0].Text(), &x); err != nil {
		context.ResultError(err)
	} else {
		if j, err := json.Marshal(x); err != nil {
			context.ResultError(err)
		} else {
			context.ResultText(string(j))
		}
	}
}
