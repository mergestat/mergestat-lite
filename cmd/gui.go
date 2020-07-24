package cmd

import (
	"database/sql"
	"encoding/csv"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/augmentable-dev/gitqlite/pkg/gitqlite"
	"github.com/go-git/go-git/v5"
	"github.com/olekukonko/tablewriter"

	"github.com/jroimartin/gocui"
)

var (
	viewArr  = []string{"Query", "Output"}
	active   = 0
	query    = ""
	repoPath = ""
)

func setCurrentViewOnTop(g *gocui.Gui, name string) (*gocui.View, error) {
	if _, err := g.SetCurrentView(name); err != nil {
		return nil, err
	}
	return g.SetViewOnTop(name)
}

func nextView(g *gocui.Gui, v *gocui.View) error {
	nextIndex := (active + 1) % len(viewArr)
	name := viewArr[nextIndex]
	if v.Name() == "Query" && v.Buffer() != "" {

		query = v.Buffer()
		path, err := getRepo(repoPath)
		if err != nil {
			return err
		}
		err = display(g, path)
		if err != nil {
			return err
		}
	}
	// if v.Name() == "Repo" && v.Buffer() != "" {
	// 	repoPath = v.Buffer()
	// 	path, err := getRepo(repoPath)
	// 	if err != nil {
	// 		return err
	// 	}
	// 	err = display(g, path)
	// 	if err != nil {
	// 		return err
	// 	}

	// } else if v.Name() == "Repo" {
	// 	var (
	// 		err  error
	// 		path string
	// 	)

	// 	path, err = getRepo(repoPath)
	// 	if err != nil {
	// 		return err
	// 	}

	// 	err = display(g, path)
	// 	if err != nil {
	// 		return err
	// 	}

	// }
	if _, err := setCurrentViewOnTop(g, name); err != nil {
		return err
	}

	g.Cursor = true
	v.Rewind()
	active = nextIndex
	return nil
}

func layout(g *gocui.Gui) error {
	maxX, maxY := g.Size()
	if v, err := g.SetView("Query", 0, 0, maxX-1, 2); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Title = "Query"
		v.Editable = true
		v.Autoscroll = true
		v.Wrap = true
		if _, err = setCurrentViewOnTop(g, "Query"); err != nil {
			return err
		}
	}

	// if v, err := g.SetView("Repo", maxX/2-1, 0, maxX-1, 2); err != nil {
	// 	if err != gocui.ErrUnknownView {
	// 		return err
	// 	}
	// 	v.Title = "Repo"
	// 	v.Autoscroll = true
	// 	v.Wrap = true
	// 	v.Editable = true
	// }
	if v, err := g.SetView("Output", 0, 3, maxX, maxY-1); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Title = "Output"
		v.Wrap = true
		v.Editable = true
	}
	return nil
}

func quit(g *gocui.Gui, v *gocui.View) error {
	return gocui.ErrQuit
}
func RunGUI(repo string) {
	g, err := gocui.NewGui(gocui.OutputNormal)
	if err != nil {
		log.Panicln(err)
	}
	defer g.Close()
	repoPath = repo
	g.Highlight = true
	g.Cursor = true
	g.SelFgColor = gocui.ColorGreen

	g.SetManagerFunc(layout)

	if err := g.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, quit); err != nil {
		log.Panicln(err)
	}
	if err := g.SetKeybinding("", gocui.KeyTab, gocui.ModNone, nextView); err != nil {
		log.Panicln(err)
	}

	if err := g.MainLoop(); err != nil && err != gocui.ErrQuit {
		log.Panicln(err)
	}
}
func getRepo(remote string) (string, error) {

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
func display(g *gocui.Gui, path string) error {
	out, err := g.View("Output")
	if err != nil {
		return err
	}

	if err != nil {
		return err
	}
	instance, err := gitqlite.New(path, &gitqlite.Options{})
	if err != nil {
		return err
	}
	defer instance.DB.Close()
	rows, err := instance.DB.Query(query)
	if err != nil {
		return err
	}
	file, err := filepath.Abs(repoPath)
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "Repo: "+file+"\nQuery: "+query)
	columns, err := rows.Columns()
	if err != nil {
		return err
	}
	if len(columns) < 9 {
		err := table(rows, out)
		if err != nil {
			return err
		}

	} else {
		err := displayCsv(rows, out)
		if err != nil {
			return err
		}
	}
	return nil
}
func table(rows *sql.Rows, v *gocui.View) error {
	columns, err := rows.Columns()
	if err != nil {
		return err
	}
	pointers := make([]interface{}, len(columns))
	container := make([]sql.NullString, len(columns))

	for i := range pointers {
		pointers[i] = &container[i]
	}
	table := tablewriter.NewWriter(v)
	for rows.Next() {

		err := rows.Scan(pointers...)
		if err != nil {
			return err
		}

		r := make([]string, len(columns))
		for i, c := range container {
			if c.Valid {
				r[i] = c.String
			} else {
				r[i] = "NULL"
			}
		}

		table.Append(r)
		if err != nil {
			return err
		}
	}
	table.Render()
	return nil
}
func displayCsv(rows *sql.Rows, v *gocui.View) error {

	columns, err := rows.Columns()
	if err != nil {
		return err
	}
	w := csv.NewWriter(v)

	err = w.Write(columns)
	if err != nil {
		return err
	}
	pointers := make([]interface{}, len(columns))
	container := make([]sql.NullString, len(columns))

	for i := range pointers {
		pointers[i] = &container[i]
	}
	for rows.Next() {
		err := rows.Scan(pointers...)
		if err != nil {
			return err
		}

		r := make([]string, len(columns))
		for i, c := range container {
			if c.Valid {
				r[i] = c.String
			}
		}

		err = w.Write(r)
		if err != nil {
			return err
		}
	}
	w.Flush()
	return nil
}
