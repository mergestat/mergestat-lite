package enry

import (
	"github.com/go-enry/go-enry/v2"
	"go.riyazali.net/sqlite"
)

type EnryIsGenerated struct{}

func (f *EnryIsGenerated) Args() int           { return 2 }
func (f *EnryIsGenerated) Deterministic() bool { return true }
func (f *EnryIsGenerated) Apply(context *sqlite.Context, value ...sqlite.Value) {
	if enry.IsGenerated(value[0].Text(), value[1].Blob()) {
		context.ResultInt(1)
	} else {
		context.ResultInt(0)
	}
}
