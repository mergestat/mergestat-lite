package tui

import (
	"fmt"
	"path/filepath"
	"text/tabwriter"
	"time"

	"github.com/augmentable-dev/askgit/pkg/gitqlite"
	"github.com/jroimartin/gocui"
)

//Displays a selection of information into the Info view
func DisplayInformation(g *gocui.Gui, git *gitqlite.GitQLite, length time.Duration) error {
	out, err := g.View("Info")
	if err != nil {
		return err
	}
	w := tabwriter.NewWriter(out, 0, 0, 1, ' ', 0)
	out.Clear()
	path, err := filepath.Abs(usrInpt)
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
