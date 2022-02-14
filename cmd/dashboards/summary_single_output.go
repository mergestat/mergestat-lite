package dashboards

import (
	"bytes"
	"fmt"
	"text/tabwriter"

	"github.com/charmbracelet/bubbles/spinner"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

type hasToStringArr interface {
	ToStringArr() []string
}

func OneToOneOutputBuilder(names []string, data hasToStringArr) (*bytes.Buffer, error) {
	var b bytes.Buffer
	p := message.NewPrinter(language.English)
	w := tabwriter.NewWriter(&b, 0, 0, 3, ' ', tabwriter.TabIndent)
	stringData := data.ToStringArr()
	if len(stringData) != len(names) {
		return nil, fmt.Errorf("length of headers does not match length of data")
	}
	for i := 0; i < len(stringData); i++ {
		p.Fprintf(w, fmt.Sprintf("%s\t", names[i]))
		p.Fprintln(w, stringData[i])
	}
	p.Fprint(w, "\n\n")
	if err := w.Flush(); err != nil {
		return nil, err
	}

	return &b, nil
}

func LoadingSymbols(names []string, s spinner.Model) (*bytes.Buffer, error) {
	var b bytes.Buffer
	p := message.NewPrinter(language.English)
	w := tabwriter.NewWriter(&b, 0, 0, 3, ' ', tabwriter.TabIndent)
	for _, n := range names {
		p.Fprintf(w, "%s\t%s\n", n, s.View())
	}
	p.Fprintf(w, "\n\n")
	if err := w.Flush(); err != nil {
		return nil, err
	}
	return &b, nil
}
