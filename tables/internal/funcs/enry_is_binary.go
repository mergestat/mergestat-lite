package funcs

import (
	"github.com/go-enry/go-enry/v2"
	"go.riyazali.net/sqlite"
)

type EnryIsBinary struct{}

func (f *EnryIsBinary) Args() int           { return 1 }
func (f *EnryIsBinary) Deterministic() bool { return true }

func (f *EnryIsBinary) Apply(context *sqlite.Context, value ...sqlite.Value) {

	contents := []byte(value[0].Text())

	if enry.IsBinary(contents) {
		context.ResultInt(1)
	} else {
		context.ResultInt(0)
	}
}
