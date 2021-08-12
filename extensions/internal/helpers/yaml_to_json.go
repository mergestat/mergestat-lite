package helpers

import (
	"github.com/ghodss/yaml"
	"go.riyazali.net/sqlite"
)

// YamlToJson implements yaml_to_json sql function.
// The function signature of the equivalent sql function is:
//     yaml_to_json(string) string
type YamlToJson struct{}

func (y *YamlToJson) Args() int           { return 1 }
func (y *YamlToJson) Deterministic() bool { return true }

func (y *YamlToJson) Apply(context *sqlite.Context, value ...sqlite.Value) {
	if json, err := yaml.YAMLToJSON(value[0].Blob()); err != nil {
		context.ResultError(err)
	} else {
		context.ResultText(string(json))
	}
}
