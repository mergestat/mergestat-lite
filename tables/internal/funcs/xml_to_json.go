package funcs

import (
	"github.com/clbanning/mxj/v2"
	"go.riyazali.net/sqlite"
)

// XmlToJson implements xml_to_json sql function.
// The function signature of the equivalent sql function is:
//     xml_to_json(string) string
type XmlToJson struct{}

func (y *XmlToJson) Args() int           { return 1 }
func (y *XmlToJson) Deterministic() bool { return true }

func (y *XmlToJson) Apply(context *sqlite.Context, value ...sqlite.Value) {
	var err error
	var m mxj.Map

	if m, err = mxj.NewMapXml(value[0].Blob()); err != nil {
		context.ResultError(err)
	} else {
		if json, err := m.Json(); err != nil {
			context.ResultError(err)
		} else {
			context.ResultText(string(json))
		}
	}
}
