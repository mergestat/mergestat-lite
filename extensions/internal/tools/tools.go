// Package tools provides utilities to help with writing
// integration test suites for mergestat sqlite module.
package tools

import (
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
	_ "github.com/mergestat/mergestat/pkg/sqlite"
)

func RowContent(rows *sql.Rows) (colCount int, contents [][]string, err error) {
	columns, err := rows.Columns()
	if err != nil {
		return colCount, nil, err
	}

	colCount = len(columns)

	pointers := make([]interface{}, len(columns))
	container := make([]sql.NullString, len(columns))

	for i := range pointers {
		pointers[i] = &container[i]
	}

	for rows.Next() {
		err = rows.Scan(pointers...)
		if err != nil {
			return colCount, nil, err
		}

		r := make([]string, len(columns))
		for i, c := range container {
			if c.Valid {
				r[i] = c.String
			} else {
				r[i] = "NULL"
			}
		}
		contents = append(contents, r)
	}
	return colCount, contents, rows.Err()

}
