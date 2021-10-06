package helpers

import (
	"bufio"
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

// NewStrSplitModule returns the implementation of a table-valued-function for splitting string contents on a delimiter (default "\n")
func NewStrSplitModule() sqlite.Module {
	return vtab.NewTableFunc("str_split", strSplitCols, func(constraints []*vtab.Constraint, order []*sqlite.OrderBy) (vtab.Iterator, error) {
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
			return nil, fmt.Errorf("no Contents Provided")
		}
		if delimiter == "" {
			delimiter = "\n"
		}

		return newStrSplitIter(contents, delimiter)
	})
}

func newStrSplitIter(contents string, delimiter string) (*strSplitAllIter, error) {
	scanner := bufio.NewScanner(strings.NewReader(contents))

	// if a delimiter is provided, see here: https://stackoverflow.com/questions/33068644/how-a-scanner-can-be-implemented-with-a-custom-split/33069759
	split := func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		// Return nothing if at end of file and no data passed
		if atEOF && len(data) == 0 {
			return 0, nil, nil
		}

		// Find the index of the delimiter
		if i := strings.Index(string(data), delimiter); i >= 0 {
			return i + 1, data[0:i], nil
		}

		// If at end of file with data return the data
		if atEOF {
			return len(data), data, nil
		}

		return
	}
	scanner.Split(split)

	// TODO make the buffer size settable
	buf := make([]byte, 0, 1024*1024*512)
	scanner.Buffer(buf, 0)

	return &strSplitAllIter{
		contents: *scanner,
		index:    0,
	}, nil
}

type strSplitAllIter struct {
	contents bufio.Scanner
	index    int
}

func (i *strSplitAllIter) Column(ctx vtab.Context, c int) error {
	switch c {
	case 0:
		ctx.ResultInt(i.index)
	case 1:
		ctx.ResultText(i.contents.Text())
	}
	return nil
}

func (i *strSplitAllIter) Next() (vtab.Row, error) {

	i.index++
	keepGoing := i.contents.Scan()
	if !keepGoing {
		return nil, io.EOF
	}
	return i, nil
}
