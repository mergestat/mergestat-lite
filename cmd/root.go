package cmd

import (
	"database/sql"
	"github.com/augmentable-dev/askgit/pkg/display"
	. "github.com/augmentable-dev/askgit/pkg/query"
	"github.com/spf13/cobra"
	"io/ioutil"
	"log"
	"os"
)

var format string      // output format flag
var presetQuery string // named / preset query flag

func init() {
	// local (root command only) flags
	rootCmd.Flags().StringVarP(&format, "format", "f", "table", "specify the output format. Options are 'csv' 'tsv' 'table' 'single' and 'json'")
	rootCmd.Flags().StringVarP(&presetQuery, "preset", "p", "", "used to pick a preset query")

	// add the export sub command
	rootCmd.AddCommand(exportCmd)
}

var rootCmd = &cobra.Command{
	Use:  `display "SELECT * FROM commits"`,
	Args: cobra.MaximumNArgs(2),
	Long: `
  display is a CLI for querying git repositories with SQL, using SQLite virtual tables.
  Example query can be found in the GitHub repo: https://github.com/augmentable-dev/display`,
	Short: `query your github repos with SQL`,
	Run: func(cmd *cobra.Command, args []string) {
		var err error

		var info os.FileInfo
		if info, err = os.Stdin.Stat(); err != nil {
			log.Fatalf("failed to read stdin stat: %v", err)
		}

		var query string
		if len(args) > 0 {
			query = args[0]
		} else if isPiped(info) {
			var stdin []byte
			if stdin, err = ioutil.ReadAll(os.Stdin); err != nil {
				log.Fatalf("failed to read from stdin: %v", err)
			}
			query = string(stdin)
		} else if presetQuery != "" {
			var found bool
			if query, found = Find(presetQuery); !found {
				log.Fatalf("unknown preset query: %s", presetQuery)
			}
		} else {
			if err = cmd.Help(); err != nil {
				log.Fatal(err.Error())
			}
			os.Exit(0)
		}

		var db *sql.DB
		if db, err = sql.Open("sqlite3", ":memory:"); err != nil {
			log.Fatalf("failed to initialize database connection: %v", err)
		}

		var rows *sql.Rows
		if rows, err = db.Query(query); err != nil {
			log.Fatalf("query execution failed: %v", err)
		}
		defer rows.Close()

		if err = display.WriteTo(rows, os.Stdout, format, false); err != nil {
			log.Fatalf("failed to output resultset: %v", err)
		}
	},
}

func isPiped(info os.FileInfo) bool { return info.Mode()&os.ModeCharDevice == 0 }

// Execute executes the root command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatalf("execution failed: %v", err)
	}
}
