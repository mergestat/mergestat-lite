package helpers

import (
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/augmentable-dev/vtab"
	"go.riyazali.net/sqlite"
)

var grepCols = []vtab.Column{
	{Name: "line_no", Type: "INT", OrderBy: vtab.NONE},
	{Name: "line", Type: "TEXT", OrderBy: vtab.NONE},

	{Name: "contents", Type: "TEXT", Hidden: true, Filters: []*vtab.ColumnFilter{{Op: sqlite.INDEX_CONSTRAINT_EQ, OmitCheck: true}}, OrderBy: vtab.NONE},
	{Name: "search", Type: "TEXT", Hidden: true, Filters: []*vtab.ColumnFilter{{Op: sqlite.INDEX_CONSTRAINT_EQ, OmitCheck: true}}, OrderBy: vtab.NONE},
	{Name: "preceeding", Type: "INT", Hidden: true, Filters: []*vtab.ColumnFilter{{Op: sqlite.INDEX_CONSTRAINT_EQ, OmitCheck: true}}, OrderBy: vtab.NONE},
	{Name: "proceeding", Type: "INT", Hidden: true, Filters: []*vtab.ColumnFilter{{Op: sqlite.INDEX_CONSTRAINT_EQ, OmitCheck: true}}, OrderBy: vtab.NONE},
}

// NewStatsModule returns the implementation of a table-valued-function for grep
func NewGrepModule() sqlite.Module {
	return vtab.NewTableFunc("grep", grepCols, func(constraints []*vtab.Constraint, order []*sqlite.OrderBy) (vtab.Iterator, error) {
		var contents, search string
		before, after := 0, 0
		for _, constraint := range constraints {
			if constraint.Op == sqlite.INDEX_CONSTRAINT_EQ {
				switch constraint.ColIndex {
				case 2:
					contents = constraint.Value.Text()
				case 3:
					search = constraint.Value.Text()
				case 4:
					before = constraint.Value.Int()
				case 5:
					after = constraint.Value.Int()
				}
			}
		}

		// TODO(patrickdevivo) not entirely sure if we should fail/error on this or let it be
		if search == "" {
			return nil, fmt.Errorf("no search string provided")
		}

		return newGrepIter(contents, search, before, after)
	})
}

func newGrepIter(contents, search string, preceeding, proceeding int) (*grepIter, error) {
	iter := &grepIter{
		contents:    contents,
		preceeding:  preceeding,
		proceeding:  proceeding,
		splitString: strings.Split(contents, "\n"),
		index:       -1,
	}

	if r, err := regexp.Compile(search); err != nil {
		return nil, err
	} else {
		iter.search = r
	}

	return iter, nil
}

type grepIter struct {
	contents    string
	search      *regexp.Regexp
	preceeding  int
	proceeding  int
	splitString []string
	index       int
}

func (i *grepIter) Column(ctx vtab.Context, c int) error {
	switch c {
	case 0:
		ctx.ResultInt(i.index + 1)
	case 1:
		min := 0
		if min < i.index-i.preceeding {
			min = i.index - i.preceeding
		}
		max := len(i.splitString) - 1
		if max > i.index+i.proceeding {
			max = i.index + i.proceeding
		}
		ctx.ResultText(strings.Join(i.splitString[min:max+1], "\n"))
	}
	return nil
}

func (i *grepIter) Next() (vtab.Row, error) {
	i.index++
	for i.index < len(i.splitString) && !(i.search.MatchString(i.splitString[i.index])) {
		i.index++
	}
	if i.index >= len(i.splitString) {
		return nil, io.EOF
	}
	return i, nil
}
