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
			target TEXT,
			type TEXT,
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

	return &branchCursor{repo: v.repo}, nil

}

func remoteBranches(s storer.ReferenceStorer) (storer.ReferenceIter, error) {
	refs, err := s.IterReferences()
	if err != nil {
		return nil, err
	}

	return storer.NewReferenceFilteredIter(func(ref *plumbing.Reference) bool {
		return ref.Name().IsRemote()
	}, refs), nil
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
	repo             *git.Repository
	current          *plumbing.Reference
	localBranchIter  storer.ReferenceIter
	remoteBranchIter storer.ReferenceIter
}

func (vc *branchCursor) Column(c *sqlite3.SQLiteContext, col int) error {
	branchRef := vc.current
	branch, err := vc.repo.Branch(branchRef.Name().Short())
	if err != nil {
		switch col {
		case 0:
			//branch name
			c.ResultText(branchRef.Name().Short())
		case 1:
			//branch remote
			c.ResultText(branchRef.Name().String())
		case 2:
			//branch merge
			c.ResultText(branchRef.Target().String())
		case 3:
			//branchger rebase
			c.ResultText(branchRef.Type().String())
		case 4:
			//Branch hash
			c.ResultText(branchRef.Hash().String())
		case 5:
			//BranchRef Name
			c.ResultText(branchRef.Name().String())
		}
		return nil
	} else {

		switch col {
		case 0:
			//branch name
			c.ResultText(branch.Name)
		case 1:
			//branch remote
			c.ResultText(branch.Remote)
		case 2:
			//branch merge
			c.ResultText(branchRef.Target().String())
		case 3:
			//branchger rebase
			c.ResultText(branchRef.Type().String())
		case 4:
			//Branch hash
			c.ResultText(branchRef.Hash().String())
		case 5:
			//BranchRef Name
			c.ResultText(branchRef.Name().String())
		}
		return nil
	}
}

func (vc *branchCursor) Filter(idxNum int, idxStr string, vals []interface{}) error {
	localBranchIterator, err := vc.repo.Branches()
	if err != nil {
		return err
	}

	vc.localBranchIter = localBranchIterator

	remoteBranchIterator, err := remoteBranches(vc.repo.Storer)
	if err != nil {
		return err
	}

	vc.remoteBranchIter = remoteBranchIterator

	branchRef, err := localBranchIterator.Next()
	if err != nil {
		if err == io.EOF {
			return nil
		}
		return err
	}

	vc.current = branchRef

	return nil
}

func (vc *branchCursor) Next() error {
	branchRef, err := vc.localBranchIter.Next()
	if err != nil {
		if err == io.EOF {
			branchRef, err = vc.remoteBranchIter.Next()
			if err != nil {
				if err == io.EOF {
					vc.current = nil
					return nil
				}
				return err
			}
			vc.current = branchRef
			return nil

		} else {
			return err
		}
	}
	vc.current = branchRef
	return nil
}

func (vc *branchCursor) EOF() bool {
	return vc.current == nil
}

func (vc *branchCursor) Rowid() (int64, error) {
	return int64(0), nil
}

func (vc *branchCursor) Close() error {
	if vc.localBranchIter != nil {
		vc.localBranchIter.Close()
	}
	if vc.remoteBranchIter != nil {
		vc.remoteBranchIter.Close()
	}
	return nil
}
