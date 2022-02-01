package summary

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"
	"text/tabwriter"

	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

// type has2DStringArrFunction interface {
// 	to2StringArr() [][]string
// }
func tableBuilder(headers []string, data ...interface{}) (*bytes.Buffer, error) {
	var b bytes.Buffer
	p := message.NewPrinter(language.English)
	w := tabwriter.NewWriter(&b, 0, 0, 3, ' ', tabwriter.TabIndent)
	// format header string
	p.Fprintf(w, strings.Join(headers, "\t"))
	p.Fprintln(w)
	// format table data
	for _, row := range data {
		columnValueReflection := reflect.ValueOf(row)
		if columnValueReflection.Kind() == reflect.Ptr {
			columnValueReflection = columnValueReflection.Elem()
		}
		// iterate over all values of the struct
		for i := 0; i < columnValueReflection.NumField(); i++ {
			columnValue := columnValueReflection.Field(i).Interface()
			if i < columnValueReflection.NumField()-1 {
				p.Fprintf(w, fmt.Sprintf("%v\t", columnValue))
			} else {
				p.Fprintln(w, fmt.Sprintf("%v", columnValue))
			}
		}
	}

	if err := w.Flush(); err != nil {
		return nil, err
	}

	return &b, nil
}
