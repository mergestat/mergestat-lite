package gitqlite

import (
	"fmt"
	"io"
	"strings"

	"github.com/mattn/go-sqlite3"
)

type Module struct {
	options *ModuleOptions
}
type genericIter interface {
	Next() ([]interface{}, error)
}

type ModuleOptions struct {
	Iterator genericIter
	Row      []interface{}
}

func NewModule(options *ModuleOptions) *Module {
	return &Module{options}
}

type Table struct {
	Iter genericIter
	Row  []interface{}
}

func (m *Module) EponymousOnlyModule() {}

func (m *Module) Create(c *sqlite3.SQLiteConn, args []string) (sqlite3.VTab, error) {
	//println("create")
	createString := fmt.Sprintf(`CREATE TABLE %s (`, args[0])
	for _, value := range m.options.Row {
		createString += fmt.Sprintf(`%s TEXT,`, value)
	}
	createString = strings.TrimRight(createString, ",")
	createString += ")"
	//fmt.Println(createString)
	err := c.DeclareVTab(createString)
	if err != nil {
		//print(err.Error())
		return nil, err
	}
	//println("endCreate")
	return &Table{Iter: m.options.Iterator, Row: m.options.Row}, nil
}

func (m *Module) Connect(c *sqlite3.SQLiteConn, args []string) (sqlite3.VTab, error) {
	//println("connect")
	return m.Create(c, args)
}

func (m *Module) DestroyModule() {}

func (v *Table) Open() (sqlite3.VTabCursor, error) {
	//println("open")

	return &cursor{iter: v.Iter, row: v.Row}, nil
}

func (v *Table) BestIndex(cst []sqlite3.InfoConstraint, ob []sqlite3.InfoOrderBy) (*sqlite3.IndexResult, error) {
	//println("bestIndex")
	dummy := make([]bool, len(cst))
	//print(fmt.Sprint(dummy))
	return &sqlite3.IndexResult{Used: dummy}, nil
}

func (v *Table) Disconnect() error {
	//println("disconn")
	return nil
}
func (v *Table) Destroy() error {
	//println("destroy")

	return nil
}

type cursor struct {
	iter genericIter
	row  []interface{}
}

func (vc *cursor) Column(c *sqlite3.SQLiteContext, col int) error {
	//println("column")
	//println(fmt.Sprintf("%v", vc.row))
	if col < len(vc.row) {
		c.ResultText(fmt.Sprint(vc.row[col]))
	} else {
		c.ResultText("")
	}

	return nil
}
func (vc *cursor) Filter(idxNum int, idxStr string, vals []interface{}) error {
	//println("filter")
	row, err := vc.iter.Next()
	if err != nil {
		//print(err.Error())
		return err
	}
	vc.row = row
	return nil
}
func (vc *cursor) Next() error {
	//println("next")

	row, err := vc.iter.Next()
	if err != nil {
		if err == io.EOF {
			vc.row = nil
			return nil
		}
		return err
	}

	vc.row = row
	return nil

}

func (vc *cursor) EOF() bool {
	////println("EOF")
	return vc.row == nil
}

func (vc *cursor) Rowid() (int64, error) {
	//println("rowid")
	return int64(0), nil
}

func (vc *cursor) Close() error {
	//println("CLOSE")
	return nil
}
