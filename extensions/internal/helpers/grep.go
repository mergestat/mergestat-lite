package helpers

import (
	"fmt"
	"io"
	"strings"

	"github.com/augmentable-dev/vtab"
	"go.riyazali.net/sqlite"
)

var grepCols = []vtab.Column{
	{Name: "line_no", Type: "INT", NotNull: false, Hidden: false, Filters: nil, OrderBy: vtab.NONE},
	{Name: "line", Type: "TEXT", NotNull: false, Hidden: false, Filters: nil, OrderBy: vtab.NONE},

	{Name: "contents", Type: "TEXT", NotNull: true, Hidden: true, Filters: []*vtab.ColumnFilter{{Op: sqlite.INDEX_CONSTRAINT_EQ, OmitCheck: true}}, OrderBy: vtab.NONE},
	{Name: "search", Type: "TEXT", NotNull: true, Hidden: true, Filters: []*vtab.ColumnFilter{{Op: sqlite.INDEX_CONSTRAINT_EQ, OmitCheck: true}}, OrderBy: vtab.NONE},
	{Name: "preceeding", Type: "INT", NotNull: true, Hidden: true, Filters: []*vtab.ColumnFilter{{Op: sqlite.INDEX_CONSTRAINT_EQ, OmitCheck: true}}, OrderBy: vtab.NONE},
	{Name: "preceeding", Type: "INT", NotNull: true, Hidden: true, Filters: []*vtab.ColumnFilter{{Op: sqlite.INDEX_CONSTRAINT_EQ, OmitCheck: true}}, OrderBy: vtab.NONE},
}

// NewStatsModule returns the implementation of a table-valued-function for git stats
func NewGrepModule() sqlite.Module {
	return vtab.NewTableFunc("grep", grepCols, func(constraints []*vtab.Constraint, order []*sqlite.OrderBy) (vtab.Iterator, error) {
		var contents, search string
		preceeding, proceeding := 0, 0
		for _, constraint := range constraints {
			if constraint.Op == sqlite.INDEX_CONSTRAINT_EQ {
				switch constraint.ColIndex {
				case 2:
					contents = constraint.Value.Text()
				case 3:
					search = constraint.Value.Text()
				case 4:
					preceeding = constraint.Value.Int()
				case 5:
					proceeding = constraint.Value.Int()
				}
			}
		}

		if contents == "" {
			return nil, fmt.Errorf("No Contents Provided")
		}
		if search == "" {
			return nil, fmt.Errorf("No search string provided")
		}

		return newGrepIter(contents, search, preceeding, proceeding)
	})
}

func newGrepIter(contents string, search string, preceeding int, proceeding int) (*grepAllIter, error) {

	iter := &grepAllIter{
		contents:   contents,
		search:     search,
		preceeding: preceeding,
		proceeding: proceeding,
		index:      -1,
	}
	iter.splitString = strings.Split(contents, "\n")

	return iter, nil
}

type grepAllIter struct {
	contents    string
	search      string
	preceeding  int
	proceeding  int
	splitString []string
	index       int
}

func (i *grepAllIter) Column(ctx vtab.Context, c int) error {
	switch c {
	case 0:
		ctx.ResultInt(i.index)
	case 1:
		min := 0
		if min < i.index-i.preceeding {
			min = i.index - i.preceeding
		}
		max := len(i.splitString) - 1
		if max > i.index+i.proceeding {
			max = i.index + i.proceeding
		}
		ret := ""
		for g := min; g <= max; g++ {
			ret += i.splitString[g] + "\n"

		}
		ctx.ResultText(ret)
	}
	return nil
}

func (i *grepAllIter) Next() (vtab.Row, error) {
	for i.index >= len(i.splitString) && !(strings.Contains(i.splitString[i.index], i.search)) {
		i.index++
		if i.index >= len(i.splitString) {
			return nil, io.EOF
		}
	}
	return i, nil
}
