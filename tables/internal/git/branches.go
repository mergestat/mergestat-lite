package git

import (
	"fmt"
	"go.riyazali.net/sqlite"

	git "github.com/libgit2/git2go/v31"
)

type BranchesModule struct{}

func (m *BranchesModule) Connect(_ *sqlite.Conn, args []string, declare func(string) error) (sqlite.VirtualTable, error) {
	// TODO(@riyaz): parse args to extract repo
	var repo = "."

	var schema = fmt.Sprintf(`CREATE TABLE %q (name TEXT, remote BOOL, target TEXT, head BOOL)`, args[0])
	return &gitBranchesTable{repoPath: repo}, declare(schema)
}

type gitBranchesTable struct {
	repoPath string
}

func (v *gitBranchesTable) BestIndex(_ *sqlite.IndexInfoInput) (*sqlite.IndexInfoOutput, error) {
	// TODO: this should actually be implemented!
	return &sqlite.IndexInfoOutput{}, nil
}

func (v *gitBranchesTable) Open() (sqlite.VirtualCursor, error) {
	repo, err := git.OpenRepository(v.repoPath)
	if err != nil {
		return nil, err
	}

	return &branchesCursor{repo: repo}, nil
}

func (v *gitBranchesTable) Disconnect() error { return nil }
func (v *gitBranchesTable) Destroy() error    { return nil }

type currentBranch struct {
	*git.Branch
	branchType git.BranchType
}

type branchesCursor struct {
	repo    *git.Repository
	current *currentBranch
	iter    *git.BranchIterator
}

func (vc *branchesCursor) Filter(_ int, _ string, _ ...sqlite.Value) error {
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

func (vc *branchesCursor) Column(c *sqlite.Context, col int) error {
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
		c.ResultInt(boolToInt(branch.IsRemote()))
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
		c.ResultInt(boolToInt(isHead))
	}
	return nil
}

func (vc *branchesCursor) Close() error {
	if vc.current != nil {
		vc.current.Free()
	}
	vc.iter.Free()
	return nil
}

func (vc *branchesCursor) Eof() bool             { return vc.current == nil }
func (vc *branchesCursor) Rowid() (int64, error) { return int64(0), nil }
