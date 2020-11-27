package gitqlite

import (
	"fmt"

	git "github.com/libgit2/git2go/v30"
	"github.com/mattn/go-sqlite3"
)

type GitBranchesModule struct{}

func NewGitBranchesModule() *GitBranchesModule {
	return &GitBranchesModule{}
}

type gitBranchesTable struct {
	repoPath string
}

func (m *GitBranchesModule) Create(c *sqlite3.SQLiteConn, args []string) (sqlite3.VTab, error) {
	err := c.DeclareVTab(fmt.Sprintf(`
		CREATE TABLE %q (
			name TEXT,
			remote BOOL,
			target TEXT,
			head BOOL
		)`, args[0]))
	if err != nil {
		return nil, err
	}

	// the repoPath will be enclosed in double quotes "..." since ensureTables uses %q when setting up the table
	// we need to pop those off when referring to the actual directory in the fs
	repoPath := args[3][1 : len(args[3])-1]
	return &gitBranchesTable{repoPath: repoPath}, nil
}

func (m *GitBranchesModule) Connect(c *sqlite3.SQLiteConn, args []string) (sqlite3.VTab, error) {
	return m.Create(c, args)
}

func (m *GitBranchesModule) DestroyModule() {}

func (v *gitBranchesTable) Open() (sqlite3.VTabCursor, error) {
	repo, err := git.OpenRepository(v.repoPath)
	if err != nil {
		return nil, err
	}

	return &branchesCursor{repo: repo}, nil

}

func (v *gitBranchesTable) BestIndex(cst []sqlite3.InfoConstraint, ob []sqlite3.InfoOrderBy) (*sqlite3.IndexResult, error) {
	// TODO this should actually be implemented!
	dummy := make([]bool, len(cst))
	return &sqlite3.IndexResult{Used: dummy}, nil
}

func (v *gitBranchesTable) Disconnect() error {
	return nil
}

func (v *gitBranchesTable) Destroy() error { return nil }

type currentBranch struct {
	*git.Branch
	branchType git.BranchType
}

type branchesCursor struct {
	repo    *git.Repository
	current *currentBranch
	iter    *git.BranchIterator
}

func (vc *branchesCursor) Column(c *sqlite3.SQLiteContext, col int) error {
	branch := vc.current
	switch col {
	case 0:
		//branch name
		name, err := branch.Name()
		if err != nil {
			return err
		}
		c.ResultText(name)
	case 1:
		c.ResultBool(branch.IsRemote())
	case 2:
		switch branch.Type() {
		case git.ReferenceSymbolic:
			c.ResultText(branch.SymbolicTarget())
		case git.ReferenceOid:
			c.ResultText(branch.Target().String())
		}
	case 3:
		isHead, err := branch.IsHead()
		if err != nil {
			return err
		}
		c.ResultBool(isHead)
	}
	return nil
}

func (vc *branchesCursor) Filter(idxNum int, idxStr string, vals []interface{}) error {
	branchIter, err := vc.repo.NewBranchIterator(git.BranchAll)
	if err != nil {
		return err
	}

	vc.iter = branchIter

	branch, branchType, err := vc.iter.Next()
	if err != nil {
		return err
	}

	vc.current = &currentBranch{branch, branchType}

	return nil
}

func (vc *branchesCursor) Next() error {
	branch, branchType, err := vc.iter.Next()
	if err != nil {
		if branch == nil {
			vc.current = nil
			return nil
		}
		return err
	}

	vc.current.Free()
	vc.current = &currentBranch{branch, branchType}
	return nil
}

func (vc *branchesCursor) EOF() bool {
	return vc.current == nil
}

func (vc *branchesCursor) Rowid() (int64, error) {
	return int64(0), nil
}

func (vc *branchesCursor) Close() error {
	if vc.current != nil {
		vc.current.Free()
	}
	vc.iter.Free()
	return nil
}
