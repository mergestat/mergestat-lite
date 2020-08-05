package cmd

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/augmentable-dev/gitqlite/pkg/gitqlite"
	"github.com/go-git/go-git/v5"
	"gopkg.in/yaml.v2"

	"github.com/jroimartin/gocui"
)

var (
	viewArr  = []string{"Query", "Selection", "Output"}
	active   = 0
	query    = ""
	repoPath = ""
	conf     ymlConfig
)

type ymlConfig struct {
	Details []string
	Queries []string
}

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
	if _, err := setCurrentViewOnTop(g, name); err != nil {
		return err
	}
	// since going to next this actually sets g.Cursor to true for the Query view
	if v.Name() == "Output" {
		g.Cursor = true
	} else {
		g.Cursor = false
	}
	v.Rewind()
	active = nextIndex
	return nil
}

func handleClick(g *gocui.Gui, v *gocui.View) error {
	if v.Name() != "Info" && v.Name() != "Default" {
		if _, err := g.SetCurrentView(v.Name()); err != nil {
			return err
		}
		if v.Name() == "Query" {
			g.Cursor = true
		} else {
			g.Cursor = false
		}
	}
	return nil
}

func runQuery(g *gocui.Gui, v *gocui.View) error {
	input, err := g.View("Query")
	if err != nil {
		return err
	}
	choice, err := g.View("Selection")
	if err != nil {
		return err
	}
	if choice.Buffer() != "" {
		x := strings.Trim(choice.Buffer(), "\n ")
		i64, err := strconv.ParseInt(x, 10, 32)
		if err != nil {
			fmt.Fprint(choice, err)
			return nil
		}
		i := int(i64)
		input.Clear()
		fmt.Fprint(input, conf.Queries[i])
	}
	if input.Buffer() != "" {
		out, err := g.View("Output")
		if err != nil {
			return err
		}
		out.Clear()
		query = input.Buffer()
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
			fmt.Fprint(out, err)
			return nil
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
	return nil
}

func previousLine(g *gocui.Gui, v *gocui.View) error {

	x, y := v.Origin()
	err := v.SetOrigin(x, y-1)
	if err != nil {
		//do nothing print for lint
		fmt.Print()
	}

	return nil
}
func nextLine(g *gocui.Gui, v *gocui.View) error {

	x, y := v.Origin()
	err := v.SetOrigin(x, y+1)
	if err != nil {
		//do nothing print for lint

		fmt.Print()
	}

	return nil
}
func goLeft(g *gocui.Gui, v *gocui.View) error {

	x, y := v.Origin()
	err := v.SetOrigin(x-1, y)
	if err != nil {
		//do nothing print for lint
		fmt.Print()
	}

	return nil
}
func goRight(g *gocui.Gui, v *gocui.View) error {

	x, y := v.Origin()
	err := v.SetOrigin(x+1, y)
	if err != nil {
		//do nothing print for lint
		fmt.Print()
	}

	return nil
}

func layout(g *gocui.Gui) error {
	maxX, maxY := g.Size()
	if v, err := g.SetView("Query", 0, 0, maxX/2-1, maxY*4/10); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Title = "Query"
		v.Editable = true
		v.Wrap = true
		fmt.Fprint(v, query)
		if _, err = setCurrentViewOnTop(g, "Query"); err != nil {
			return err
		}

	}
	if v, err := g.SetView("Info", maxX*7/10+1, maxY*4/10+1, maxX-1, maxY-1); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Title = "Info"
		v.Wrap = true
		v.Editable = true
		fmt.Fprintln(v, "Keybinds: \n Ctrl+C\t: exit \n Ctrl+Space\t: execute query \n Ctrl+Q\t: clear query box")

	}
	if v, err := g.SetView("Output", 0, maxY*4/10+1, maxX*7/10, maxY-1); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Title = "Output"
		v.Wrap = false

	}
	if v, err := g.SetView("Default", maxX/2+1, 0, maxX-1, maxY*3/10-1); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Title = "Default's"
		blob, err := ioutil.ReadFile("cmd/conf.yml")
		if err != nil {
			return nil
		}
		if err := yaml.Unmarshal(blob, &conf); err != nil {
			return err
		}
		for i, s := range conf.Details {
			fmt.Fprintf(v, "%d: %s \n", i, s)
		}

	}
	if v, err := g.SetView("Selection", maxX/2+1, maxY*3/10, maxX-1, maxY*4/10); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Title = "Selection"
		v.Editable = true
	}
	return nil
}
func test(g *gocui.Gui, v *gocui.View) error {
	//for use with testing uses ctrl+t
	return nil
}
func quit(g *gocui.Gui, v *gocui.View) error {
	return gocui.ErrQuit
}
func RunGUI(repo string, q string) {
	g, err := gocui.NewGui(gocui.OutputNormal)
	if err != nil {
		log.Panicln(err)
	}
	defer g.Close()
	query = q
	repoPath = repo
	g.Highlight = true
	g.Cursor = true
	g.SelFgColor = gocui.ColorGreen
	g.Mouse = true

	g.SetManagerFunc(layout)

	if err := g.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, quit); err != nil {
		log.Panicln(err)
	}
	if err := g.SetKeybinding("", gocui.KeyTab, gocui.ModNone, nextView); err != nil {
		log.Panicln(err)
	}
	if err := g.SetKeybinding("", gocui.KeyCtrlSpace, gocui.ModNone, runQuery); err != nil {
		log.Panicln(err)
	}
	if err := g.SetKeybinding("", gocui.KeyCtrlQ, gocui.ModNone, clearQuery); err != nil {
		log.Panicln(err)
	}
	if err := g.SetKeybinding("", gocui.MouseLeft, gocui.ModNone, handleClick); err != nil {
		log.Panicln(err)
	}
	if err := g.SetKeybinding("", gocui.MouseWheelUp, gocui.ModNone, previousLine); err != nil {
		log.Panicln(err)
	}
	if err := g.SetKeybinding("", gocui.MouseWheelDown, gocui.ModNone, nextLine); err != nil {
		log.Panicln(err)
	}
	if err := g.SetKeybinding("", gocui.KeyArrowUp, gocui.ModNone, previousLine); err != nil {
		log.Panicln(err)
	}
	if err := g.SetKeybinding("", gocui.KeyArrowDown, gocui.ModNone, nextLine); err != nil {
		log.Panicln(err)
	}
	if err := g.SetKeybinding("Output", gocui.KeyArrowRight, gocui.ModNone, goRight); err != nil {
		log.Panicln(err)
	}
	if err := g.SetKeybinding("Output", gocui.KeyArrowLeft, gocui.ModNone, goLeft); err != nil {
		log.Panicln(err)
	}
	if err := g.SetKeybinding("", gocui.KeyCtrlT, gocui.ModNone, test); err != nil {
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
	w := tabwriter.NewWriter(out, 0, 0, 1, ' ', 0)
	out.Clear()
	path, err := filepath.Abs(repoPath)
	if err != nil {
		return err
	}
	fmt.Fprint(w, "Keybinds: \n Ctrl+C\t: exit \n Ctrl+Space\t: execute query \n Ctrl+Q\t: clear query box\n\n")
	fmt.Fprintln(w, "Repo \t: "+path+"\t")
	rows, err := git.DB.Query("Select id from commits")
	if err != nil {
		return err
	}
	index := 0
	for rows.Next() {
		index++
	}
	fmt.Fprintln(w, "Number of commits \t:", index, "\t")

	rows, err = git.DB.Query("Select distinct author_name from commits")
	if err != nil {
		return err
	}
	index = 0
	for rows.Next() {
		index++
	}
	fmt.Fprintln(w, "Number of authors \t:", index, "\t")

	fmt.Fprintln(w, "Time taken to execute query\t:", length.String(), "\t")
	w.Flush()
	return nil

}
