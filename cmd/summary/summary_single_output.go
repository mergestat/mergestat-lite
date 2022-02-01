package summary

import (
	"bytes"
	"fmt"
	"reflect"
	"text/tabwriter"

	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

func oneToOneOutputBuilder(names []string, data interface{}) (*bytes.Buffer, error) {
	var b bytes.Buffer
	p := message.NewPrinter(language.English)
	w := tabwriter.NewWriter(&b, 0, 0, 3, ' ', tabwriter.TabIndent)

	dataPointReflection := reflect.ValueOf(data)
	if dataPointReflection.Kind() == reflect.Ptr {
		dataPointReflection = dataPointReflection.Elem()
	}
	for i := 0; i < dataPointReflection.NumField(); i++ {
		dataPoint := dataPointReflection.Field(i).Interface()
		p.Fprintf(w, fmt.Sprintf("%v\t", names[i]))
		p.Fprintln(w, fmt.Sprintf("%v", dataPoint))
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
