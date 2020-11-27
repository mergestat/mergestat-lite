package askgit

import (
	"strings"

	"github.com/mattn/go-sqlite3"
)

func loadHelperFuncs(conn *sqlite3.SQLiteConn) error {
	// str_split(inputString, splitCharacter, index) string
	split := func(s, c string, i int) string {
		split := strings.Split(s, c)
		if i < len(split) {
			return split[i]
		}
		return ""
	}

	if err := conn.RegisterFunc("str_split", split, true); err != nil {
		return err
	}

	return nil
}
