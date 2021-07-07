// Package tools provides utilities to hel with writing
// integration test suites for askgit sqlite module.
package tools

import (
	"database/sql"
	_ "github.com/augmentable-dev/askgit/pkg/sqlite"
	_ "github.com/mattn/go-sqlite3"
)

func RowContent(rows *sql.Rows) (colCount int, contents [][]string, err error) {
	columns, err := rows.Columns()
	if err != nil {
		return colCount, nil, err
	}

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
