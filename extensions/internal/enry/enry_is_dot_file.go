package enry

import (
	"github.com/go-enry/go-enry/v2"
	"go.riyazali.net/sqlite"
)

type EnryIsDotFile struct{}

func (f *EnryIsDotFile) Args() int           { return 1 }
func (f *EnryIsDotFile) Deterministic() bool { return true }
func (f *EnryIsDotFile) Apply(context *sqlite.Context, value ...sqlite.Value) {
	if enry.IsDotFile(value[0].Text()) {
		context.ResultInt(1)
	} else {
		context.ResultInt(0)
	}
}
