package askgit

import (
	"encoding/json"
	"strings"

	"github.com/BurntSushi/toml"
	xml_parser "github.com/clbanning/mxj/v2"
	"github.com/ghodss/yaml"
	"github.com/mattn/go-sqlite3"
)

func loadHelperFuncs(conn *sqlite3.SQLiteConn) error {
	// str_split(inputString, splitCharacter, index) string
	split := func(s, c string, i int) string {
		split := strings.Split(s, c)
		if i < len(split) {
			return split[i]
		}
		return ""
	}
	yml2json := func(s string) (string, error) {
		json, err := yaml.YAMLToJSON([]byte(s))
		return string(json), err
	}
	toml2json := func(s string) (string, error) {
		var x interface{}
		if _, err := toml.Decode(s, &x); err != nil {
			return "", err
		}
		jsonFromToml, err := json.Marshal(x)
		if err != nil {
			return "", err
		}
		return string(jsonFromToml), nil
	}
	xml2json := func(s string) (string, error) {
		mv, err := xml_parser.NewMapXml([]byte(s))
		if err != nil {
			return "", err
		}
		jsonFromXml, err := mv.Json()
		if err != nil {
			return "", err
		}
		return string(jsonFromXml), nil
	}
	if err := conn.RegisterFunc("str_split", split, true); err != nil {
		return err
	}
	if err := conn.RegisterFunc("yml_to_json", yml2json, true); err != nil {
		return err
	}
	if err := conn.RegisterFunc("toml_to_json", toml2json, true); err != nil {
		return err
	}
	if err := conn.RegisterFunc("xml_to_json", xml2json, true); err != nil {
		return err
	}

	return nil
}
