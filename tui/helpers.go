package tui

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"text/tabwriter"
	"time"

	"github.com/augmentable-dev/askgit/pkg/gitqlite"
	"github.com/go-git/go-git/v5"

	"github.com/jroimartin/gocui"
)

func GetRepo(remote string) (string, error) {

	path, err := filepath.Abs(remote)
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		dir, err := ioutil.TempDir("", "repo")
		if err != nil {
			return "", err
		}
		_, err = git.PlainClone(dir, false, &git.CloneOptions{
			URL: remote,
		})
		if err != nil {
			return "", err
		}
		path = dir
	} else {
		repository, err := git.PlainOpen(path)
		if err != nil {
			return "", err
		}

		err = repository.Fetch(&git.FetchOptions{
			Force: true,
		})
		if err != nil {
			//do nothing
			fmt.Print()
		}
	}

	return path, nil
}
func DisplayInformation(g *gocui.Gui, git *gitqlite.GitQLite, length time.Duration) error {
	out, err := g.View("Info")
	if err != nil {
		return err
	}
	w := tabwriter.NewWriter(out, 0, 0, 1, ' ', 0)
	out.Clear()
	path, err := filepath.Abs(repoPath)
	if err != nil {
		return err
	}
	fmt.Fprintln(w, "Repo \t "+path+"\t")
	rows, err := git.DB.Query("Select id from commits")
	if err != nil {
		return err
	}
	index := 0
	for rows.Next() {
		index++
	}
	fmt.Fprintln(w, "# Commits \t", index, "\t")

	rows, err = git.DB.Query("Select distinct author_name from commits")
	if err != nil {
		return err
	}
	index = 0
	for rows.Next() {
		index++
	}
	fmt.Fprintln(w, "# Authors \t", index, "\t")

	rows, err = git.DB.Query("select Distinct name from branches where name like 'origin%'")
	if err != nil {
		return err
	}
	index = 0
	for rows.Next() {
		index++
	}
	fmt.Fprintln(w, "# Remote branches \t", index, "\t")

	rows, err = git.DB.Query("select Distinct name from branches where remote like 'origin'")
	if err != nil {
		return err
	}
	index = 0
	for rows.Next() {
		index++
	}
	fmt.Fprintln(w, "# Local branches \t", index, "\t")

	fmt.Fprintln(w, "Query time (ms)\t", length.String(), "\t")
	w.Flush()
	return nil

}
