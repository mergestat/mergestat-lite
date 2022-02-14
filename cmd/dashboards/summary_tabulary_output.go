package dashboards

import (
	"bytes"
	"strings"
	"text/tabwriter"

	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

// type has2DStringArrFunction interface {
// 	to2StringArr() [][]string
// }
type hasDelimToStringArr interface {
	ToStringArr(delimiter ...string) []string
}

func TableBuilder(headers []string, data hasDelimToStringArr) (*bytes.Buffer, error) {
	var b bytes.Buffer
	p := message.NewPrinter(language.English)
	w := tabwriter.NewWriter(&b, 0, 0, 3, ' ', tabwriter.TabIndent)
	// format header string
	p.Fprintf(w, strings.Join(headers, "\t"))
	p.Fprintln(w)
	// format table data
	rows := data.ToStringArr()
	for _, row := range rows {
		p.Fprintln(w, row)
	}

	if err := w.Flush(); err != nil {
		return nil, err
	}

	return &b, nil
}
