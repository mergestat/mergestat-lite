package funcs

import (
	"github.com/go-enry/go-enry/v2"
	"go.riyazali.net/sqlite"
)

type EnryDetectLanguage struct{}

func (y *EnryDetectLanguage) Args() int           { return 2 }
func (y *EnryDetectLanguage) Deterministic() bool { return true }

func (y *EnryDetectLanguage) Apply(context *sqlite.Context, value ...sqlite.Value) {

	path := value[0].Text()
	contents := []byte(value[1].Text())

	lang := enry.GetLanguage(path, contents)
	if lang == "" {
		context.ResultNull()
		return
	}
	context.ResultText(lang)
}
