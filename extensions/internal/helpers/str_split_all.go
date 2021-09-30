package helpers

import (
	"fmt"
	"io"
	"strings"

	"github.com/augmentable-dev/vtab"
	"go.riyazali.net/sqlite"
)

var strSplitCols = []vtab.Column{
	{Name: "line_no", Type: "INT", NotNull: false, Hidden: false, Filters: nil, OrderBy: vtab.NONE},
	{Name: "line", Type: "TEXT", NotNull: false, Hidden: false, Filters: nil, OrderBy: vtab.NONE},

	{Name: "contents", Type: "TEXT", NotNull: true, Hidden: true, Filters: []*vtab.ColumnFilter{{Op: sqlite.INDEX_CONSTRAINT_EQ, OmitCheck: true}}, OrderBy: vtab.NONE},
	{Name: "delimiter", Type: "TEXT", NotNull: true, Hidden: true, Filters: []*vtab.ColumnFilter{{Op: sqlite.INDEX_CONSTRAINT_EQ, OmitCheck: true}}, OrderBy: vtab.NONE},
}

// NewStatsModule returns the implementation of a table-valued-function for git stats
func NewStrSplitModule() sqlite.Module {
	return vtab.NewTableFunc("str_split_all", strSplitCols, func(constraints []*vtab.Constraint, order []*sqlite.OrderBy) (vtab.Iterator, error) {
		var contents, delimiter string
		for _, constraint := range constraints {
			if constraint.Op == sqlite.INDEX_CONSTRAINT_EQ {
				switch constraint.ColIndex {
				case 2:
					contents = constraint.Value.Text()
				case 3:
					delimiter = constraint.Value.Text()
				}
			}
		}

		if contents == "" {
			return nil, fmt.Errorf("No Contents Provided")
		}
		if delimiter == "" {
			return nil, fmt.Errorf("No delimiter Provided")
		}

		return newStatsIter(contents, delimiter)
	})
}

func newStatsIter(contents string, delimiter string) (*strSplitAllIter, error) {

	iter := &strSplitAllIter{
		contents:  contents,
		delimiter: delimiter,
		index:     -1,
	}
	iter.splitString = strings.Split(contents, delimiter)

	return iter, nil
}

type strSplitAllIter struct {
	contents    string
	delimiter   string
	splitString []string
	index       int
}

func (i *strSplitAllIter) Column(ctx vtab.Context, c int) error {
	switch c {
	case 0:
		ctx.ResultInt(i.index)
	case 1:
		ctx.ResultText(i.splitString[i.index])
	}
	return nil
}

func (i *strSplitAllIter) Next() (vtab.Row, error) {
	i.index++
	if i.index >= len(i.splitString) {
		return nil, io.EOF
	}
	return i, nil
}
