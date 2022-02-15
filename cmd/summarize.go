package cmd

import (
	"github.com/spf13/cobra"
)

func init() {
	summarizeCmd.AddCommand(summarizeCommitsCmd, summarizeBlameCmd)
}

var summarizeCmd = &cobra.Command{
	Use:     "summarize [command]",
	Short:   "Generate various summary reports",
	Aliases: []string{"summary"},
}
