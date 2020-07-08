package gitqlite

import (
	"fmt"
	"io"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/storer"
	"github.com/mattn/go-sqlite3"
)

type gitBranchModule struct{}

type gitBranchTable struct {
	repoPath string
	repo     *git.Repository
}

func (m *gitBranchModule) Create(c *sqlite3.SQLiteConn, args []string) (sqlite3.VTab, error) {
	err := c.DeclareVTab(fmt.Sprintf(`
		CREATE TABLE %q (
			name TEXT,
			remote TEXT,
			merge TEXT,
			rebase TEXT,
			hash TEXT,
			ref_name TEXT
		)`, args[0]))
	if err != nil {
		return nil, err
	}

	// the repoPath will be enclosed in double quotes "..." since ensureTables uses %q when setting up the table
	// we need to pop those off when referring to the actual directory in the fs
	repoPath := args[3][1 : len(args[3])-1]
	return &gitBranchTable{repoPath: repoPath}, nil
}

func (m *gitBranchModule) Connect(c *sqlite3.SQLiteConn, args []string) (sqlite3.VTab, error) {
	return m.Create(c, args)
}

func (m *gitBranchModule) DestroyModule() {}

func (v *gitBranchTable) Open() (sqlite3.VTabCursor, error) {
	repo, err := git.PlainOpen(v.repoPath)
	if err != nil {
		return nil, err
	}
	v.repo = repo
	branchIterator, err := v.repo.Branches()
	if err != nil {
		return nil, err
	}
	branchRef, err := branchIterator.Next()
	if err != nil {
		if err == io.EOF {
			return &branchCursor{0, v.repo, nil, nil, true}, nil
		}
		return nil, err
	}
	_, err = v.repo.Branch((branchRef.Name().Short()))
	if err != nil {
		if err == plumbing.ErrReferenceNotFound {
			return &branchCursor{0, v.repo, nil, nil, true}, nil
		}
		return nil, err
	}

	return &branchCursor{0, v.repo, branchRef, branchIterator, false}, nil

}

func (v *gitBranchTable) BestIndex(cst []sqlite3.InfoConstraint, ob []sqlite3.InfoOrderBy) (*sqlite3.IndexResult, error) {
	// TODO this should actually be implemented!
	dummy := make([]bool, len(cst))
	return &sqlite3.IndexResult{Used: dummy}, nil
}

func (v *gitBranchTable) Disconnect() error {
	v.repo = nil
	return nil
}
func (v *gitBranchTable) Destroy() error { return nil }

type branchCursor struct {
	index      int
	repo       *git.Repository
	current    *plumbing.Reference
	branchIter storer.ReferenceIter
	eof        bool
}

func (vc *branchCursor) Column(c *sqlite3.SQLiteContext, col int) error {

	branchRef := vc.current
	branch, err := vc.repo.Branch(branchRef.Name().Short())
	if err != nil {
		return nil
	}

	switch col {
	case 0:
		//branch name
		c.ResultText(branch.Name)
	case 1:
		//branch remote
		c.ResultText(branch.Remote)
	case 2:
		//branch merge
		c.ResultText(branch.Merge.String())
	case 3:
		//branchger rebase
		c.ResultText(branch.Rebase)

	case 4:
		//Branch hash
		c.ResultText(branchRef.Hash().String())
	case 5:
		//BranchRef Name
		c.ResultText(branchRef.Name().String())
	}
	return nil

}

func (vc *branchCursor) Filter(idxNum int, idxStr string, vals []interface{}) error {
	vc.index = 0
	return nil
}

func (vc *branchCursor) Next() error {
	vc.index++

	branchRef, err := vc.branchIter.Next()
	if err != nil {
		if err == io.EOF {
			vc.current = nil
			vc.eof = true
			return nil
		}
		return err
	}
	vc.current = branchRef
	return nil
}

func (vc *branchCursor) EOF() bool {
	return vc.eof
}

func (vc *branchCursor) Rowid() (int64, error) {
	return int64(vc.index), nil
}

func (vc *branchCursor) Close() error {
	if vc.branchIter != nil {
		vc.branchIter.Close()
	}
	return nil
}
