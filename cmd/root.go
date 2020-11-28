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
	git "github.com/libgit2/git2go/v30"
	"github.com/spf13/cobra"
)

//define flags in here
var (
	repo        string
	format      string
	useGitCLI   bool
	cui         bool
	presetQuery string
	githubOrg   string
	githubUser  string
)

func init() {
	rootCmd.PersistentFlags().StringVar(&repo, "repo", ".", "path to git repository (defaults to current directory). A remote repo may be specified, it will be cloned to a temporary directory before query execution.")
	rootCmd.PersistentFlags().StringVar(&format, "format", "table", "specify the output format. Options are 'csv' 'tsv' 'table' 'single' and 'json'")
	rootCmd.PersistentFlags().BoolVar(&useGitCLI, "use-git-cli", false, "whether to use the locally installed git command (if it's available). Defaults to false.")
	rootCmd.PersistentFlags().BoolVarP(&cui, "interactive", "i", false, "whether to run in interactive mode, which displays a terminal UI")
	rootCmd.PersistentFlags().StringVar(&presetQuery, "preset", "", "used to pick a preset query")
	rootCmd.PersistentFlags().StringVar(&githubOrg, "github-org", "", "used to specify a GitHub *org* to query against when scanning repos")
	rootCmd.PersistentFlags().StringVar(&githubUser, "github-user", "", "used to specify a GitHub *user* to query against when scanning repos")
}

func handleError(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use: `askgit "SELECT * FROM commits"`,
	Long: `
  askgit is a CLI for querying git repositories with SQL, using SQLite virtual tables.
  Example queries can be found in the GitHub repo: https://github.com/augmentable-dev/askgit`,
	Short: `query your github repos with SQL`,
	Run: func(cmd *cobra.Command, args []string) {
		cwd, err := os.Getwd()
		handleError(err)

		// if a repo path is not supplied as a flag, use the current directory
		if repo == "" {
			if len(args) > 1 {
				repo = args[1]
			} else {
				repo = cwd
			}
		}
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
		var dir string

		// if the repo can be parsed as a remote git url, clone it to a temporary directory and use that as the repo path
		if remote, err := vcsurl.Parse(repo); err == nil { // if it can be parsed
			dir, err = ioutil.TempDir("", "repo")
			handleError(err)
			cloneOptions := askgit.CreateAuthenticationCallback(remote)
			_, err = git.Clone(repo, dir, cloneOptions)
			handleError(err)

			defer func() {
				err := os.RemoveAll(dir)
				handleError(err)
			}()
		}

		if dir == "" {
			dir, err = filepath.Abs(repo)
		} else {
			dir, err = filepath.Abs(dir)
		}

		if err != nil {
			handleError(err)
		}

		ag, err := askgit.New(dir, &askgit.Options{
			UseGitCLI:   useGitCLI,
			GitHubOrg:   githubOrg,
			GitHubUser:  githubUser,
			GitHubToken: os.Getenv("GITHUB_TOKEN"),
		})
		handleError(err)

		if cui {
			tui.RunGUI(ag, query)
			return
		}

		rows, err := ag.DB().Query(query)
		handleError(err)

		err = askgit.DisplayDB(rows, os.Stdout, format)
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
