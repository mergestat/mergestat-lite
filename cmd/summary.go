package cmd

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mergestat/mergestat/cmd/summary"
	"github.com/spf13/cobra"
)

var summaryCmd = &cobra.Command{
	Use:  "summary",
	Long: "prints a summary of commit activity in the default repository.",
	Args: cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		var ui *summary.TermUI
		var err error
		if ui, err = summary.NewTermUI(); err != nil {
			handleExitError(err)
		}
		defer ui.Close()

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
