package cmd

import (
	"database/sql"
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"
)

var (
	exports []string
	append  bool
)

type export struct {
	table string
	query string
}

func init() {
	exportCmd.Flags().StringArrayVarP(&exports, "exports", "e", []string{}, "queries to export, supplied as string pairs")
	exportCmd.Flags().BoolVarP(&append, "append", "a", false, "append mode: insert into tables rather than creating new ones")
}

var exportCmd = &cobra.Command{
	Use:   "export [sqlite db file]",
	Short: "Export queries into a SQLite db file",
	Long:  `Use this command to export queries into a SQLite database file on disk`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		var err error

		if len(exports) == 0 {
			handleExitError(fmt.Errorf("please supply at least one export pair"))
		}

		if len(exports)%2 != 0 {
			handleExitError(fmt.Errorf("expected even number of export pairs, got %d", len(exports)))
		}

		pairs := make([]export, len(exports)/2)
		for e := 0; e < len(exports)-1; e += 2 {
			if e == 0 {
				pairs[e] = export{exports[e], exports[e+1]}
			} else {
				pairs[e-1] = export{exports[e], exports[e+1]}
			}
		}

		var fileName string
		if fileName, err = filepath.Abs(args[0]); err != nil {
			handleExitError(fmt.Errorf("failed to resolve file path: %v", err))
		}

		var db *sql.DB
		if db, err = sql.Open("sqlite3", fileName); err != nil {
			handleExitError(fmt.Errorf("failed to open sqlite database: %v", err))
		}

		for _, pair := range pairs {
			if append {
				var tableAlreadyExists bool
				if row := db.QueryRow("SELECT EXISTS (SELECT * FROM sqlite_master WHERE type='table' AND name = ?)", pair.table); row.Err() != nil {
					handleExitError(fmt.Errorf("failed to execute query: %v", err))
				} else {
					if err := row.Scan(&tableAlreadyExists); err != nil {
						handleExitError(fmt.Errorf("failed to scan row: %v", err))
					}
				}

				if !tableAlreadyExists {
					if _, err = db.Exec(fmt.Sprintf("CREATE TABLE %s AS %s", pair.table, pair.query)); err != nil {
						handleExitError(fmt.Errorf("failed to execute query: %v", err))
					}
				} else {
					var tx *sql.Tx
					if tx, err = db.BeginTx(cmd.Context(), &sql.TxOptions{}); err != nil {
						handleExitError(fmt.Errorf("failed to start transaction: %v", err))
					}

					if _, err = tx.Exec(fmt.Sprintf("INSERT INTO %s %s", pair.table, pair.query)); err != nil {
						handleExitError(fmt.Errorf("failed to execute query: %v", err))
					}

					if err = tx.Commit(); err != nil {
						handleExitError(err)
					}
				}

			} else {
				if _, err = db.Exec(fmt.Sprintf("CREATE TABLE %s AS %s", pair.table, pair.query)); err != nil {
					handleExitError(fmt.Errorf("failed to execute query: %v", err))
				}
			}
		}

	},
}
