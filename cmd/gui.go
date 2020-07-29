package cmd

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/augmentable-dev/gitqlite/pkg/gitqlite"
	"github.com/go-git/go-git/v5"

	"github.com/jroimartin/gocui"
)

var (
	viewArr  = []string{"Query", "Info", "Output"}
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
func clearQuery(g *gocui.Gui, v *gocui.View) error {
	q, err := g.View("Query")
	if err != nil {
		return err
	}
	q.Clear()
	q.Rewind()
	return nil
}
func nextView(g *gocui.Gui, v *gocui.View) error {
	nextIndex := (active + 1) % len(viewArr)
	name := viewArr[nextIndex]
	if v.Name() == "Query" && v.Buffer() != "" {
		out, err := g.View("Output")
		if err != nil {
			return err
		}
		out.Clear()
		query = v.Buffer()
		path, err := getRepo(repoPath)
		if err != nil {
			return err
		}
		git, err := gitqlite.New(path, &gitqlite.Options{
			SkipGitCLI: skipGitCLI,
		})
		if err != nil {
			return err
		}
		start := time.Now()
		rows, err := git.DB.Query(query)
		if err != nil {
			return err
		}
		err = displayDB(rows, out)
		if err != nil {
			return err
		}
		total := time.Since(start)
		err = displayInformation(g, git, total)
		if err != nil {
			return err
		}
	}

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
	if v, err := g.SetView("Query", 0, 0, maxX/2-1, maxY*3/10); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Title = "Query"
		v.Editable = true
		v.Wrap = true
		if _, err = setCurrentViewOnTop(g, "Query"); err != nil {
			return err
		}
	}
	if v, err := g.SetView("Info", maxX/2+1, 0, maxX-1, maxY*3/10); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Title = "Info"
		v.Editable = true
		v.Wrap = true
	}
	if v, err := g.SetView("Output", 0, maxY*3/10+1, maxX, maxY-1); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Title = "Output"
		v.Wrap = false
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
	if err := g.SetKeybinding("", gocui.KeyCtrlQ, gocui.ModNone, clearQuery); err != nil {
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
func displayInformation(g *gocui.Gui, git *gitqlite.GitQLite, length time.Duration) error {
	out, err := g.View("Info")
	if err != nil {
		return err
	}
	out.Clear()
	path, err := filepath.Abs(repoPath)
	if err != nil {
		return err
	}
	fmt.Fprint(out, "Repo: "+path+"\n")
	rows, err := git.DB.Query("Select id from commits")
	if err != nil {
		return err
	}
	index := 0
	for rows.Next() {
		index++
	}
	fmt.Fprintf(out, "Number of commits %d\n", index)

	rows, err = git.DB.Query("Select distinct author_name from commits")
	if err != nil {
		return err
	}
	index = 0
	for rows.Next() {
		index++
	}
	fmt.Fprintf(out, "Number of authors %d\n", index)

	fmt.Fprintln(out, "Time taken to execute query"+length.String())
	return nil

}
