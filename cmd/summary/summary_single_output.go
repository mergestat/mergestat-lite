package summary

import (
	"bytes"
	"fmt"
	"text/tabwriter"

	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

type hasToStringArr interface {
	toStringArr() []string
}

func oneToOneOutputBuilder(names []string, data hasToStringArr) (*bytes.Buffer, error) {
	var b bytes.Buffer
	p := message.NewPrinter(language.English)
	w := tabwriter.NewWriter(&b, 0, 0, 3, ' ', tabwriter.TabIndent)
	stringData := data.toStringArr()
	if len(stringData) != len(names) {
		return nil, fmt.Errorf("length of headers does not match length of data")
	}
	for i := 0; i < len(stringData); i++ {
		p.Fprintf(w, fmt.Sprintf("%v\t", names[i]))
		p.Fprintln(w, fmt.Sprintf("%v", stringData[i]))
	}
	p.Fprint(w, "\n\n")
	if err := w.Flush(); err != nil {
		return nil, err
	}

	return &b, nil
}

func loadingSymbols(names []string, t *TermUI) (*bytes.Buffer, error) {
	var b bytes.Buffer
	p := message.NewPrinter(language.English)
	w := tabwriter.NewWriter(&b, 0, 0, 3, ' ', tabwriter.TabIndent)
	for _, n := range names {
		p.Fprintf(w, "%s\t%s\n", n, t.spinner.View())
	}
	p.Fprintf(w, "\n\n")
	if err := w.Flush(); err != nil {
		return nil, err
	}
	return &b, nil
}
