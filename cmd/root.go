package cmd

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/augmentable-dev/askgit/pkg/askgit"
	"github.com/augmentable-dev/askgit/pkg/tui"
	"github.com/gitsight/go-vcsurl"
	git "github.com/libgit2/git2go/v31"
	"github.com/spf13/cobra"
)

//define flags in here
var (
	repo        string
	format      string
	useGitCLI   bool
	cui         bool
	presetQuery string
)

func init() {
	// global flags
	rootCmd.PersistentFlags().StringVarP(&repo, "repo", "r", ".", "path to git repository (defaults to current directory).\nA remote repo may be specified, it will be cloned to a temporary directory before query execution.")
	rootCmd.PersistentFlags().BoolVar(&useGitCLI, "use-git-cli", false, "whether to use the locally installed git command (if it's available). Defaults to false.")

	// local (root command only) flags
	rootCmd.Flags().StringVarP(&format, "format", "f", "table", "specify the output format. Options are 'csv' 'tsv' 'table' 'single' and 'json'")
	rootCmd.Flags().BoolVarP(&cui, "interactive", "i", false, "whether to run in interactive mode, which displays a terminal UI")
	rootCmd.Flags().StringVarP(&presetQuery, "preset", "p", "", "used to pick a preset query")

	// add the export sub command
	rootCmd.AddCommand(exportCmd)
}

func handleError(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func determineRepo() (string, func()) {
	// the directory on disk of the repository
	var dir string

	cwd, err := os.Getwd()
	handleError(err)

	// if a repo path is not supplied as a flag, use the current directory
	if repo == "" {
		repo = cwd
	}

	// if the repo can be parsed as a remote git url, clone it to a temporary directory and use that as the repo path
	if remote, err := vcsurl.Parse(repo); err == nil { // if it can be parsed
		dir, err = ioutil.TempDir("", "repo")
		handleError(err)

		cloneOptions := askgit.CreateAuthenticationCallback(remote)
		_, err = git.Clone(repo, dir, cloneOptions)
		handleError(err)

		dir, err = filepath.Abs(dir)
		handleError(err)

		return dir, func() {
			err := os.RemoveAll(dir)
			handleError(err)
		}
	} else {
		dir, err = filepath.Abs(repo)
		handleError(err)

		return dir, func() {}
	}
}

var rootCmd = &cobra.Command{
	Use:  `askgit "SELECT * FROM commits"`,
	Args: cobra.MaximumNArgs(2),
	Long: `
  askgit is a CLI for querying git repositories with SQL, using SQLite virtual tables.
  Example queries can be found in the GitHub repo: https://github.com/augmentable-dev/askgit`,
	Short: `query your github repos with SQL`,
	Run: func(cmd *cobra.Command, args []string) {
		info, err := os.Stdin.Stat()
		handleError(err)

		var query string
		if len(args) > 0 {
			query = args[0]
		} else if info.Mode()&os.ModeCharDevice == 0 {
			query, err = readStdin()
			handleError(err)
		} else if cui {
			query = ""
		} else if presetQuery != "" {
			if val, ok := tui.Queries[presetQuery]; ok {
				query = val
			} else {
				handleError(fmt.Errorf("Unknown Preset Query: %s", presetQuery))
			}
		} else {
			err = cmd.Help()
			handleError(err)
			os.Exit(0)
		}

		dir, cleanup := determineRepo()
		defer cleanup()

		ag, err := askgit.New(&askgit.Options{
			RepoPath:    dir,
			UseGitCLI:   useGitCLI,
			GitHubToken: os.Getenv("GITHUB_TOKEN"),
		})
		handleError(err)

		if cui {
			tui.RunGUI(ag, query)
			return
		}

		rows, err := ag.DB().Query(query)
		handleError(err)

		err = askgit.DisplayDB(rows, os.Stdout, format, false)
		handleError(err)
	},
}

// Execute runs the root command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		handleError(err)
	}
}

func readStdin() (string, error) {
	reader := bufio.NewReader(os.Stdin)
	output, err := ioutil.ReadAll(reader)
	if err != nil {
		return "", err
	}
	returnString := string(output)
	return returnString, nil
}
