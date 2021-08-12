package enry

import (
	"github.com/go-enry/go-enry/v2"
	"go.riyazali.net/sqlite"
)

type EnryDetectLanguage struct{}

func (f *EnryDetectLanguage) Args() int           { return 2 }
func (f *EnryDetectLanguage) Deterministic() bool { return true }
func (f *EnryDetectLanguage) Apply(context *sqlite.Context, value ...sqlite.Value) {
	if lang := enry.GetLanguage(value[0].Text(), value[1].Blob()); lang == "" {
		context.ResultNull()
		return
	} else {
		context.ResultText(lang)
	}
}
