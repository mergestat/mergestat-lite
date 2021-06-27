package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/askgitdev/askgit/pkg/askgit"
	"github.com/spf13/cobra"
)

var (
	exports []string
)

type export struct {
	table string
	query string
}

func init() {
	exportCmd.Flags().StringArrayVarP(&exports, "exports", "e", []string{}, "queries to export, supplied as string pairs")
}

var exportCmd = &cobra.Command{
	Use:  "export [sqlite db file]",
	Long: `Use this command to export queries into a SQLite database file on disk`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if len(exports) == 0 {
			handleError(errors.New("please supply at least one export pair"))
		}

		if len(exports)%2 != 0 {
			handleError(fmt.Errorf("expected even number of export pairs, got %d", len(exports)))
		}

		pairs := make([]export, len(exports)/2)
		for e := 0; e < len(exports)-1; e += 2 {
			if e == 0 {
				pairs[e] = export{exports[e], exports[e+1]}
			} else {
				pairs[e-1] = export{exports[e], exports[e+1]}
			}
		}

		exportFile, err := filepath.Abs(args[0])
		handleError(err)

		dir, cleanup := determineRepo()
		defer cleanup()

		ag, err := askgit.New(&askgit.Options{
			RepoPath:    dir,
			UseGitCLI:   useGitCLI,
			GitHubToken: os.Getenv("GITHUB_TOKEN"),
			DBFilePath:  exportFile,
		})
		handleError(err)

		for _, pair := range pairs {
			s := fmt.Sprintf("CREATE TABLE %s AS %s", pair.table, pair.query)

			_, err := ag.DB().Exec(s)
			handleError(err)
		}

	},
}
