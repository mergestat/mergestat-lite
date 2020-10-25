package gitqlite

import (
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"

	"github.com/olekukonko/tablewriter"
)

func DisplayDB(rows *sql.Rows, w io.Writer, format string) error {

	switch format {
	case "single":
		err := single(rows, w)
		if err != nil {
			return err
		}
	case "csv":
		err := csvDisplay(rows, ',', w)
		if err != nil {
			return err
		}
	case "tsv":
		err := csvDisplay(rows, '\t', w)
		if err != nil {
			return err
		}
	case "json":
		err := jsonDisplay(rows, w)
		if err != nil {
			return err
		}
	//TODO: switch between table and csv dependent on num columns(suggested num for table 5<=
	default:
		err := tableDisplay(rows, w)
		if err != nil {
			return err
		}

	}
	return nil
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

	r := make([]string, len(columns))
	for i, c := range container {
		if c.Valid {
			r[i] = c.String
		}
	}

	fmt.Println(r[0])

	return nil
}

func csvDisplay(rows *sql.Rows, commaChar rune, write io.Writer) error {

	columns, err := rows.Columns()
	if err != nil {
		return err
	}
	w := csv.NewWriter(write)
	w.Comma = commaChar

	err = w.Write(columns)
	if err != nil {
		return err
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

func jsonDisplay(rows *sql.Rows, write io.Writer) error {
	columns, err := rows.Columns()
	if err != nil {
		return err
	}

	values := make([]interface{}, len(columns))
	for i := range values {
		values[i] = new(interface{})
	}

	enc := json.NewEncoder(write)

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
func tableDisplay(rows *sql.Rows, write io.Writer) error {
	columns, err := rows.Columns()
	if err != nil {
		return err
	}
	pointers := make([]interface{}, len(columns))
	container := make([]sql.NullString, len(columns))

	for i := range pointers {
		pointers[i] = &container[i]
	}
	table := tablewriter.NewWriter(write)
	table.SetHeader(columns)
	for rows.Next() {
		err := rows.Scan(pointers...)
		if err != nil {
			return err
		}

		r := make([]string, len(columns))
		for i, c := range container {
			if c.Valid {
				r[i] = c.String
			} else {
				r[i] = "NULL"
			}
		}

		table.Append(r)
		if err != nil {
			return err
		}
	}

	table.Render()
	return nil
}
