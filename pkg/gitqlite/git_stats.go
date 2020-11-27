package gitqlite

import (
	"fmt"
	"io"

	git "github.com/libgit2/git2go/v30"
	"github.com/mattn/go-sqlite3"
)

type GitStatsModule struct{}

func NewGitStatsModule() *GitStatsModule {
	return &GitStatsModule{}
}

type gitStatsTable struct {
	repoPath string
	repo     *git.Repository
}

func (m *GitStatsModule) Create(c *sqlite3.SQLiteConn, args []string) (sqlite3.VTab, error) {
	err := c.DeclareVTab(fmt.Sprintf(`
			CREATE TABLE %q (
			commit_id TEXT,
			file TEXT,
			additions INT,
			deletions INT
			)`, args[0]))
	if err != nil {
		return nil, err
	}

	// the repoPath will be enclosed in double quotes "..." since ensureTables uses %q when setting up the table
	// we need to pop those off when referring to the actual directory in the fs
	repoPath := args[3][1 : len(args[3])-1]
	return &gitStatsTable{repoPath: repoPath}, nil
}

func (m *GitStatsModule) Connect(c *sqlite3.SQLiteConn, args []string) (sqlite3.VTab, error) {
	return m.Create(c, args)
}

func (m *GitStatsModule) DestroyModule() {}

func (v *gitStatsTable) Open() (sqlite3.VTabCursor, error) {
	repo, err := git.OpenRepository(v.repoPath)
	if err != nil {
		return nil, err
	}
	v.repo = repo

	return &statsCursor{repo: v.repo}, nil
}

func (v *gitStatsTable) BestIndex(cst []sqlite3.InfoConstraint, ob []sqlite3.InfoOrderBy) (*sqlite3.IndexResult, error) {
	used := make([]bool, len(cst))
	// TODO implement an index for file name glob patterns?
	// TODO this loop construct won't work well for multiple constraints...
	for c, constraint := range cst {
		switch {
		case constraint.Usable && constraint.Column == 0 && constraint.Op == sqlite3.OpEQ:
			used[c] = true
			return &sqlite3.IndexResult{Used: used, IdxNum: 1, IdxStr: "stats-by-commit-id", EstimatedCost: 1.0, EstimatedRows: 1}, nil
		}
	}

	return &sqlite3.IndexResult{Used: used, EstimatedCost: 100}, nil
}

func (v *gitStatsTable) Disconnect() error {
	v.repo = nil
	return nil
}
func (v *gitStatsTable) Destroy() error { return nil }

type statsCursor struct {
	repo     *git.Repository
	iterator *commitStatsIter
	current  *commitStat
}

func (vc *statsCursor) Column(c *sqlite3.SQLiteContext, col int) error {
	stat := vc.current
	switch col {
	case 0:
		//commit id
		c.ResultText(stat.commitID)
	case 1:
		c.ResultText(stat.file)
	case 2:
		c.ResultInt(stat.additions)
	case 3:
		c.ResultInt(stat.deletions)

	}

	return nil
}
func (vc *statsCursor) Filter(idxNum int, idxStr string, vals []interface{}) error {
	var opt *commitStatsIterOptions

	switch idxNum {
	case 0:
		opt = &commitStatsIterOptions{}
	case 1:
		opt = &commitStatsIterOptions{commitID: vals[0].(string)}
	}

	iter, err := NewCommitStatsIter(vc.repo, opt)
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

func (vc *statsCursor) Next() error {
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

func (vc *statsCursor) EOF() bool {
	return vc.current == nil
}

func (vc *statsCursor) Rowid() (int64, error) {
	return int64(0), nil
}

func (vc *statsCursor) Close() error {
	vc.iterator.Close()
	return nil
}
