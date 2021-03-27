// Package tools provides utilities to hel with writing
// integration test suites for askgit sqlite module.
package tools

import (
	"database/sql"
	_ "github.com/augmentable-dev/askgit/pkg/sqlite"
	git "github.com/libgit2/git2go/v31"
	_ "github.com/mattn/go-sqlite3"
	"io/ioutil"
	"os"
)

func Clone(url string) (_ *git.Repository, _ func() error, err error) {
	var dir string
	if dir, err = ioutil.TempDir("", "repo"); err != nil {
		return nil, nil, err
	}

	var repo *git.Repository
	if repo, err = git.Clone(url, dir, &git.CloneOptions{}); err != nil {
		return nil, nil, err
	}

	return repo, func() error { return os.RemoveAll(dir) }, nil
}

func RowCount(rows *sql.Rows) int {
	var count = 0
	for rows.Next() {
		count++
	}
	return count
}

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
