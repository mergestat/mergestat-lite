package tui

import (
	"fmt"
	"path/filepath"
	"text/tabwriter"
	"time"

	"github.com/augmentable-dev/askgit/pkg/askgit"
	"github.com/jroimartin/gocui"
)

//Displays a selection of information into the Info view
func DisplayInformation(g *gocui.Gui, ag *askgit.AskGit, length time.Duration) error {
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

	row := ag.DB().QueryRow("select count(*) from commits")
	var commitCount int
	err = row.Scan(&commitCount)
	if err != nil {
		return err
	}
	fmt.Fprintln(w, "# Commits \t", commitCount, "\t")

	row = ag.DB().QueryRow("select count(distinct author_name) from commits")
	var distinctAuthorCount int
	err = row.Scan(&distinctAuthorCount)
	if err != nil {
		return err
	}
	fmt.Fprintln(w, "# Authors \t", distinctAuthorCount, "\t")

	row = ag.DB().QueryRow("select count(distinct name) from branches where remote = 1")
	var distinctRemotes int
	err = row.Scan(&distinctRemotes)
	if err != nil {
		return err
	}
	fmt.Fprintln(w, "# Remote branches \t", distinctRemotes, "\t")

	row = ag.DB().QueryRow("select count(distinct name) from branches where remote = 0")
	var distinctLocals int
	err = row.Scan(&distinctLocals)
	if err != nil {
		return err
	}
	fmt.Fprintln(w, "# Local branches \t", distinctLocals, "\t")

	fmt.Fprintln(w, "Query time (ms)\t", length.String(), "\t")
	w.Flush()
	return nil

}
