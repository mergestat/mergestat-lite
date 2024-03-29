package display

import (
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"io"

	"github.com/jedib0t/go-pretty/table"
	"golang.org/x/term"
)

func WriteTo(rows *sql.Rows, w io.Writer, format string, interactive bool) error {
	switch format {
	case "single":
		err := single(rows, w)
		if err != nil {
			return err
		}
	case "csv":
		err := csvDisplay(rows, ',', false, w)
		if err != nil {
			return err
		}
	case "csv-noheader":
		err := csvDisplay(rows, ',', true, w)
		if err != nil {
			return err
		}
	case "tsv":
		err := csvDisplay(rows, '\t', false, w)
		if err != nil {
			return err
		}
	case "tsv-noheader":
		err := csvDisplay(rows, '\t', true, w)
		if err != nil {
			return err
		}
	case "json":
		err := jsonDisplay(rows, w)
		if err != nil {
			return err
		}
	case "ndjson":
		err := ndjsonDisplay(rows, w)
		if err != nil {
			return err
		}
	default:
		err := tableDisplay(rows, w, interactive)
		if err != nil {
			return err
		}

	}

	return rows.Err()
}

func single(rows *sql.Rows, write io.Writer) error {
	columns, err := rows.Columns()
	if err != nil {
		return err
	}

	pointers := make([]interface{}, len(columns))
	container := make([]sql.NullString, len(columns))

	for i := range pointers {
		pointers[i] = &container[i]
	}
	rows.Next()

	err = rows.Scan(pointers...)
	if err != nil {
		return err
	}

	output := container[0].String

	_, err = write.Write([]byte(output))
	if err != nil {
		return err
	}

	return nil
}

func csvDisplay(rows *sql.Rows, commaChar rune, noHeader bool, writer io.Writer) error {
	columns, err := rows.Columns()
	if err != nil {
		return err
	}
	w := csv.NewWriter(writer)
	w.Comma = commaChar

	if !noHeader {
		err = w.Write(columns)
		if err != nil {
			return err
		}
	}

	pointers := make([]interface{}, len(columns))
	container := make([]sql.NullString, len(columns))

	for i := range pointers {
		pointers[i] = &container[i]
	}
	for rows.Next() {
		err := rows.Scan(pointers...)
		if err != nil {
			return err
		}

		r := make([]string, len(columns))
		for i, c := range container {
			if c.Valid {
				r[i] = c.String
			}
		}

		err = w.Write(r)
		if err != nil {
			return err
		}
	}
	w.Flush()
	return nil
}

func ndjsonDisplay(rows *sql.Rows, writer io.Writer) error {
	columns, err := rows.Columns()
	if err != nil {
		return err
	}

	values := make([]interface{}, len(columns))
	for i := range values {
		values[i] = new(interface{})
	}

	enc := json.NewEncoder(writer)

	for rows.Next() {
		err = rows.Scan(values...)
		if err != nil {
			return err
		}

		dest := make(map[string]interface{})

		for i, column := range columns {
			dest[column] = *(values[i].(*interface{}))
		}

		err := enc.Encode(dest)
		if err != nil {
			return err
		}

	}

	return nil
}

func jsonDisplay(rows *sql.Rows, writer io.Writer) error {
	buffer := make([]interface{}, 0)

	columns, err := rows.Columns()
	if err != nil {
		return err
	}

	values := make([]interface{}, len(columns))
	for i := range values {
		values[i] = new(interface{})
	}

	for rows.Next() {
		err = rows.Scan(values...)
		if err != nil {
			return err
		}

		dest := make(map[string]interface{})

		for i, column := range columns {
			dest[column] = *(values[i].(*interface{}))
		}

		buffer = append(buffer, dest)
	}

	if out, err := json.Marshal(buffer); err != nil {
		return err
	} else {
		if _, err := writer.Write(out); err != nil {
			return err
		}
	}

	return nil
}

func tableDisplay(rows *sql.Rows, write io.Writer, overflow bool) error {
	columns, err := rows.Columns()
	if err != nil {
		return err
	}
	cols := make([]interface{}, len(columns))
	for i, v := range columns {
		cols[i] = v
	}
	pointers := make([]interface{}, len(columns))
	container := make([]sql.NullString, len(columns))
	for i := range pointers {
		pointers[i] = &container[i]
	}

	t := table.NewWriter()
	t.Style().Options.SeparateRows = true
	if !overflow {
		width, _, err := term.GetSize(0)
		if err != nil {
			//  TODO - getting terminal size seems to fail with `operation not supported by device` in tests
			//  as a workaround for now, set a default width instead of returning an error, if one is encountered
			width = 500
		}
		t.SetAllowedRowLength(width)
	}
	t.AppendHeader(cols)
	t.SetOutputMirror(write)

	for rows.Next() {
		err := rows.Scan(pointers...)
		if err != nil {
			return err
		}

		r := make([]interface{}, len(columns))
		for i, c := range container {
			if c.Valid {
				r[i] = c.String
			} else {
				r[i] = "NULL"
			}
		}

		t.AppendRow(r)
		if err != nil {
			return err
		}
	}

	t.Render()
	return nil
}
