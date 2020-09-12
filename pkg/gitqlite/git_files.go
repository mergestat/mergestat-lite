package gitqlite

import (
	"fmt"
	"io"
	"path"

	git "github.com/libgit2/git2go/v30"
	"github.com/mattn/go-sqlite3"
)

type gitTreeModule struct{}

type gitTreeTable struct {
	repoPath string
	repo     *git.Repository
}

func (m *gitTreeModule) Create(c *sqlite3.SQLiteConn, args []string) (sqlite3.VTab, error) {
	err := c.DeclareVTab(fmt.Sprintf(`
			CREATE TABLE %q(
				commit_id TEXT,
				tree_id TEXT,
				file_id TEXT,
				name TEXT,
				contents TEXT,
				executable BOOL
			)`, args[0]))
	if err != nil {
		return nil, err
	}
	repoPath := args[3][1 : len(args[3])-1]
	return &gitTreeTable{repoPath: repoPath}, nil
}

func (m *gitTreeModule) Connect(c *sqlite3.SQLiteConn, args []string) (sqlite3.VTab, error) {
	return m.Create(c, args)
}

func (m *gitTreeModule) DestroyModule() {}

func (vc *treeCursor) Column(c *sqlite3.SQLiteContext, col int) error {
	file := vc.current

	switch col {
	case 0:
		//commit id
		c.ResultText(file.commitID)
	case 1:
		//tree id
		c.ResultText(file.treeID)
	case 2:
		//file id
		c.ResultText(file.Blob.Id().String())
	case 3:
		//tree name
		c.ResultText(path.Join(file.path, file.Name))
	case 4:
		c.ResultText(string(file.Contents()))
	case 5:
		c.ResultBool(file.Filemode == git.FilemodeBlobExecutable)
	}

	return nil
}

func (v *gitTreeTable) Disconnect() error {
	v.repo = nil
	return nil
}

func (v *gitTreeTable) Destroy() error { return nil }

type treeCursor struct {
	repo     *git.Repository
	iterator *commitFileIter
	current  *commitFile
}

func (v *gitTreeTable) Open() (sqlite3.VTabCursor, error) {
	repo, err := git.OpenRepository(v.repoPath)
	if err != nil {
		return nil, err
	}
	v.repo = repo

	return &treeCursor{repo: v.repo}, nil
}

func (v *gitTreeTable) BestIndex(cst []sqlite3.InfoConstraint, ob []sqlite3.InfoOrderBy) (*sqlite3.IndexResult, error) {
	used := make([]bool, len(cst))
	// TODO implement an index for file name glob patterns?
	// TODO this loop construct won't work well for multiple constraints...
	for c, constraint := range cst {
		switch {
		case constraint.Usable && constraint.Column == 0 && constraint.Op == sqlite3.OpEQ:
			used[c] = true
			return &sqlite3.IndexResult{Used: used, IdxNum: 1, IdxStr: "files-by-commit-id", EstimatedCost: 1.0, EstimatedRows: 1}, nil
		}
	}

	return &sqlite3.IndexResult{Used: used, EstimatedCost: 100}, nil
}

func (vc *treeCursor) Filter(idxNum int, idxStr string, vals []interface{}) error {
	var opt *commitFileIterOptions

	switch idxNum {
	case 0:
		opt = &commitFileIterOptions{}
	case 1:
		opt = &commitFileIterOptions{commitID: vals[0].(string)}
	}

	iter, err := NewCommitFileIter(vc.repo, opt)
	if err != nil {
		return err
	}

	vc.iterator = iter

	file, err := vc.iterator.Next()
	if err != nil {
		if err == io.EOF {
			vc.current = nil
			return nil
		}
		return err
	}

	vc.current = file
	return nil
}

func (vc *treeCursor) Next() error {
	//Iterates to next file
	file, err := vc.iterator.Next()
	if err != nil {
		if err == io.EOF {
			vc.current = nil
			return nil
		}
		return err
	}
	vc.current = file
	return nil
}

func (vc *treeCursor) EOF() bool {
	return vc.current == nil
}

func (vc *treeCursor) Rowid() (int64, error) {
	return int64(0), nil
}

func (vc *treeCursor) Close() error {
	vc.iterator.Close()
	return nil
}
