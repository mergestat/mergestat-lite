package cmd

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mergestat/mergestat/cmd/summarize/blame"
	"github.com/spf13/cobra"
)

var (
	blameOutputJSON bool
)

func init() {
	blameCmd.Flags().BoolVar(&blameOutputJSON, "json", false, "output as JSON")
}

var blameCmd = &cobra.Command{
	Use:   "blame [file pattern]",
	Short: "Print a summary of blameable lines for a file path pattern",
	Long: `Prints a summary of the blameable lines for all files matching the supplied path pattern in the default repo (--repo or current directory).
Specify a file path pattern as the first argument to see aggregate blame data for all files that match the pattern.
Use '%' to match all file paths or as a wildcard (e.g. '%.go' for all .go files). You may specify a full file path (no wildcard) as well.
`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		pathPattern := args[0]

		var ui *blame.TermUI
		var err error
		if ui, err = blame.NewTermUI(pathPattern); err != nil {
			handleExitError(err)
		}
		defer ui.Close()

		if blameOutputJSON {
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
