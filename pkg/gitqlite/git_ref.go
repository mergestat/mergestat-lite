package gitqlite

import (
	"fmt"
	"io"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/storer"
	"github.com/mattn/go-sqlite3"
)

type gitRefModule struct{}

type gitRefTable struct {
	repoPath string
	repo     *git.Repository
}

func (m *gitRefModule) Create(c *sqlite3.SQLiteConn, args []string) (sqlite3.VTab, error) {
	err := c.DeclareVTab(fmt.Sprintf(`
			CREATE TABLE %q(
				name TEXT,
				type TEXT,
				hash TEXT
			)`, args[0]))
	if err != nil {
		return nil, err
	}
	repoPath := args[3][1 : len(args[3])-1]
	return &gitRefTable{repoPath: repoPath}, nil
}

func (m *gitRefModule) Connect(c *sqlite3.SQLiteConn, args []string) (sqlite3.VTab, error) {
	return m.Create(c, args)
}

func (m *gitRefModule) DestroyModule() {}

func (vc *refCursor) Column(c *sqlite3.SQLiteContext, col int) error {

	ref := vc.current
	nameNHash := ref.Strings()
	switch col {
	case 0:
		//RefName
		c.ResultText(nameNHash[0])

	case 1:
		//RefType
		c.ResultText(ref.Type().String())
	case 2:
		//RefHash
		c.ResultText(nameNHash[1])
	}
	return nil
}
func (v *gitRefTable) BestIndex(cst []sqlite3.InfoConstraint, ob []sqlite3.InfoOrderBy) (*sqlite3.IndexResult, error) {
	// TODO this should actually be implemented!
	dummy := make([]bool, len(cst))
	return &sqlite3.IndexResult{Used: dummy}, nil
}

func (v *gitRefTable) Disconnect() error {
	v.repo = nil
	return nil
}

func (v *gitRefTable) Destroy() error { return nil }

type refCursor struct {
	repo     *git.Repository
	iterator storer.ReferenceIter
	current  *plumbing.Reference
}

func (v *gitRefTable) Open() (sqlite3.VTabCursor, error) {
	repo, err := git.PlainOpen(v.repoPath)
	if err != nil {
		return nil, err
	}
	v.repo = repo

	return &refCursor{repo: v.repo}, nil
}

func (vc *refCursor) Filter(idxNum int, idxStr string, vals []interface{}) error {
	refIter, err := vc.repo.References()
	if err != nil {
		return err
	}

	vc.iterator = refIter

	current, err := refIter.Next()
	if err != nil {
		return err
	}

	vc.current = current
	return nil
}

func (vc *refCursor) Next() error {
	//Iterates to next ref
	file, err := vc.iterator.Next()
	if err != nil {
		if err == io.EOF {
			vc.current = nil
			return nil
		}
		return err
	}
	// if not EOF and not err go to next ref
	vc.current = file
	return nil
}

func (vc *refCursor) EOF() bool {
	return vc.current == nil
}

func (vc *refCursor) Rowid() (int64, error) {
	return int64(0), nil
}

func (vc *refCursor) Close() error {
	vc.iterator.Close()
	return nil
}
