package enry

import (
	"github.com/go-enry/go-enry/v2"
	"go.riyazali.net/sqlite"
)

type EnryIsDocumentation struct{}

func (f *EnryIsDocumentation) Args() int           { return 1 }
func (f *EnryIsDocumentation) Deterministic() bool { return true }
func (f *EnryIsDocumentation) Apply(context *sqlite.Context, value ...sqlite.Value) {
	if enry.IsDocumentation(value[0].Text()) {
		context.ResultInt(1)
	} else {
		context.ResultInt(0)
	}
}
