package funcs

import (
	"github.com/go-enry/go-enry/v2"
	"go.riyazali.net/sqlite"
)

type EnryIsConfiguration struct{}

func (f *EnryIsConfiguration) Args() int           { return 1 }
func (f *EnryIsConfiguration) Deterministic() bool { return true }

func (f *EnryIsConfiguration) Apply(context *sqlite.Context, value ...sqlite.Value) {

	path := value[0].Text()

	if enry.IsConfiguration(path) {
		context.ResultInt(1)
	} else {
		context.ResultInt(0)
	}
}
