package askgit

import (
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/jedib0t/go-pretty/table"
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

	output := container[0].String

	_, err = write.Write([]byte(output))
	if err != nil {
		return err
	}

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
	cols := make([]interface{}, len(columns))
	for i, v := range columns {
		cols[i] = v
	}
	pointers := make([]interface{}, len(columns))
	container := make([]sql.NullString, len(columns))
	for i := range pointers {
		pointers[i] = &container[i]
	}
	cmd := exec.Command("stty", "size")
	cmd.Stdin = os.Stdin
	out, err := cmd.Output()
	if err != nil {
		println(err.Error)
		return err
	}
	val := strings.TrimSpace(strings.Split(string(out), " ")[1])

	width, err := strconv.Atoi(val)
	if err != nil {
		println(err.Error())
		return err
	}
	t := table.NewWriter()
	t.Style().Options.SeparateRows = true
	t.SetAllowedRowLength(width)
	t.AppendHeader(cols)
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

	fmt.Println(t.Render())
	return nil
}
