package tui

import (
	"fmt"
	"time"

	"github.com/augmentable-dev/askgit/pkg/askgit"
	"github.com/jroimartin/gocui"
)

var (
	viewArr = []string{"Query", "Output"}
)

func SetCurrentViewOnTop(g *gocui.Gui, name string) (*gocui.View, error) {
	if _, err := g.SetCurrentView(name); err != nil {
		return nil, err
	}
	return g.SetViewOnTop(name)
}

//Clear's the query view
func ClearQuery(g *gocui.Gui, v *gocui.View) error {
	q, err := g.View("Query")
	if err != nil {
		return err
	}
	q.Clear()
	q.Rewind()
	return nil
}

// goes to the next view in the viewArr
func NextView(g *gocui.Gui, v *gocui.View) error {
	nextIndex := (active + 1) % len(viewArr)
	name := viewArr[nextIndex]
	if _, err := SetCurrentViewOnTop(g, name); err != nil {
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

//handles Left click.
func HandleClick(g *gocui.Gui, v *gocui.View) error {
	if v.Name() == "Default" {
		_, y := v.Cursor()
		key := v.BufferLines()[y]
		if val, ok := Queries[key]; ok {
			input, err := g.View("Query")
			if err != nil {
				return err
			}
			input.Clear()
			fmt.Fprint(input, val)

		}
	} else if v.Name() != "Info" && v.Name() != "Keybinds" {
		if _, err := g.SetCurrentView(v.Name()); err != nil {
			return err
		}
		if v.Name() == "Query" {
			g.Cursor = true
		} else {
			g.Cursor = false
		}
		err := HandleCursor(g, v)
		if err != nil {
			return nil
		}
	}
	return nil
}

//Makes sure Cursor is not more right or more down than allowed
func HandleCursor(g *gocui.Gui, v *gocui.View) error {
	if v.Buffer() == "" {
		err := v.SetCursor(0, 0)
		if err != nil {
			return err
		}
		return nil
	}

	b := v.BufferLines()
	var y int
	var x int
	xb, yb := v.Cursor()
	y = len(b) - 1
	if y < 0 {
		y = 0
	}
	if yb > y {
		yb = y
	}
	x = len(b[yb])
	if x < 0 {
		x = 0
	}
	if xb > x {
		xb = x
	}

	err := v.SetCursor(xb, yb)
	if err != nil {
		fmt.Fprintf(v, "%s, xb: %d, yb: %d x: %d, y: %d", err, xb, yb, x, y)
		return nil
	}
	return nil
}

//Run's the query
func RunQuery(g *gocui.Gui, v *gocui.View) error {
	input, err := g.View("Query")
	if err != nil {
		return err
	}
	if input.Buffer() != "" {
		out, err := g.View("Output")
		if err != nil {
			return err
		}
		out.Clear()
		err = out.SetOrigin(0, 0)
		if err != nil {
			return err
		}
		query = input.Buffer()
		start := time.Now()
		rows, err := ag.DB().Query(query)
		if err != nil {
			fmt.Fprint(out, err)
			return nil
		}

		err = askgit.DisplayDB(rows, out, "", true)
		if err != nil {
			return err
		}
		total := time.Since(start)
		err = DisplayInformation(g, ag, total)
		if err != nil {
			return err
		}
	}
	return nil
}

//Goes to the previous line
func PreviousLine(g *gocui.Gui, v *gocui.View) error {

	x, y := v.Origin()
	err := v.SetOrigin(x, y-1)
	if err != nil {
		//do nothing print for lint
		fmt.Print()
	}

	return nil
}

func NextLine(g *gocui.Gui, v *gocui.View) error {

	x, y := v.Origin()
	err := v.SetOrigin(x, y+1)
	if err != nil {
		//do nothing print for lint

		fmt.Print()
	}

	return nil
}
func GoLeft(g *gocui.Gui, v *gocui.View) error {

	x, y := v.Origin()
	var err error
	if x-1 > 0 {
		err = v.SetOrigin(x-1, y)
	}
	if err != nil {
		return err
	}

	return nil
}
func GoRight(g *gocui.Gui, v *gocui.View) error {

	x, y := v.Origin()
	err := v.SetOrigin(x+1, y)
	if err != nil {
		return err
	}
	return nil
}
func JumpRight(g *gocui.Gui, v *gocui.View) error {
	x, y := v.Origin()
	width, _ := v.Size()
	err := v.SetOrigin(x+width, y)
	if err != nil {
		return err
	}
	return nil
}
func JumpLeft(g *gocui.Gui, v *gocui.View) error {
	x, y := v.Origin()
	width, _ := v.Size()
	var err error
	if x-width > 0 {
		err = v.SetOrigin(x-width, y)
	} else {
		err = v.SetOrigin(0, y)
	}
	if err != nil {
		return err
	}
	return nil
}
func JumpUp(g *gocui.Gui, v *gocui.View) error {
	x, y := v.Origin()
	_, height := v.Size()
	var err error
	if y-height >= 0 {
		err = v.SetOrigin(x, y-height)
	} else {
		err = v.SetOrigin(x, 0)
	}
	if err != nil {
		return err
	}
	return nil
}
func JumpDown(g *gocui.Gui, v *gocui.View) error {
	x, y := v.Origin()
	_, height := v.Size()
	err := v.SetOrigin(x, y+height)
	if err != nil {
		return err
	}
	return nil
}
