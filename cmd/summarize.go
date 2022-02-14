package cmd

import (
	"github.com/spf13/cobra"
)

func init() {
	summarizeCmd.AddCommand(summarizeCommitsCmd, blameCmd)
}

var summarizeCmd = &cobra.Command{
	Use:     "summarize [command]",
	Aliases: []string{"summary"},
}
