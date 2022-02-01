package cmd

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mergestat/mergestat/cmd/summary"
	"github.com/spf13/cobra"
)

var (
	dateFilterStart string
	dateFilterEnd   string
	outputJSON      bool
)

func init() {
	summaryCmd.Flags().StringVarP(&dateFilterStart, "start", "s", "", "specify a start date to filter by. Can be of format YYYY-MM-DD, or a SQLite \"date modifier,\" relative to 'now'")
	summaryCmd.Flags().StringVarP(&dateFilterEnd, "end", "e", "", "specify an end date to filter by. Can be of format YYYY-MM-DD, or a SQLite \"date modifier,\" relative to 'now'")
	summaryCmd.Flags().BoolVar(&outputJSON, "json", false, "output as JSON")
}

var summaryCmd = &cobra.Command{
	Use:   "summary [file pattern]",
	Short: "Print a summary of commit activity",
	Long: `Prints a summary of commit activity in the default repository (either the current directory or supplied by --repo).
Specify a file pattern as an argument to filter for commits that only modified a certain file or directory.
The path is used in a SQL LIKE clause, so use '%' as a wildcard.
Read more here: https://sqlite.org/lang_expr.html#the_like_glob_regexp_and_match_operators
`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		var pathPattern string
		if len(args) > 0 {
			pathPattern = args[0]
		}

		var ui *summary.TermUI
		var err error
		if ui, err = summary.NewTermUI(pathPattern, dateFilterStart, dateFilterEnd); err != nil {
			handleExitError(err)
		}
		defer ui.Close()

		if outputJSON {
			fmt.Println(ui.PrintJSON())
			return
		}

		// check if output is a terminal (https://rosettacode.org/wiki/Check_output_device_is_a_terminal#Go)
		if fileInfo, _ := os.Stdout.Stat(); (fileInfo.Mode() & os.ModeCharDevice) != 0 {
			if err := tea.NewProgram(ui).Start(); err != nil {
				handleExitError(err)
			}
		} else {
			fmt.Print(ui.PrintNoTTY())
		}
	},
}
