package cmd

import (
	"errors"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mergestat/mergestat/cmd/blame"
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
	Short: "Print a summary of the blame context for a file",
	Long:  ``,
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		var pathPattern string
		if len(args) > 0 {
			pathPattern = args[0]
		} else {
			handleExitError(errors.New("please supply a file path pattern as first argument, use '%%' to represent any path"))
		}

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
