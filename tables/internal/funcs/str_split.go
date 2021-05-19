package funcs

import (
	"go.riyazali.net/sqlite"
	"strings"
)

// StringSplit implements str_split scalar sql function.
// The function signature of the equivalent sql function is:
//     str_split(input, separator, index) string
type StringSplit struct{}

func (s *StringSplit) Args() int           { return 3 }
func (s *StringSplit) Deterministic() bool { return true }

func (s *StringSplit) Apply(context *sqlite.Context, value ...sqlite.Value) {
	var i = value[2].Int()
	if split := strings.Split(value[0].Text(), value[1].Text()); i < len(split) {
		context.ResultText(split[i])
	} else {
		context.ResultNull()
	}
}
