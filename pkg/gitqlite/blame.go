package gitqlite

import (
	"fmt"

	git "github.com/libgit2/git2go/v31"
	"github.com/mattn/go-sqlite3"
)

type GitBlameModule struct {
	options *GitBlameModuleOptions
}

type GitBlameModuleOptions struct {
	RepoPath string
}

func NewGitBlameModule(options *GitBlameModuleOptions) *GitBlameModule {
	return &GitBlameModule{options}
}

type gitBlameTable struct {
	repoPath string
}

func (m *GitBlameModule) EponymousOnlyModule() {}

func (m *GitBlameModule) Create(c *sqlite3.SQLiteConn, args []string) (sqlite3.VTab, error) {
	err := c.DeclareVTab(fmt.Sprintf(`
		CREATE TABLE %q (
			line_no INT,
			path TEXT,
			commit_id TEXT,
			contents TEXT
		)`, args[0]))
	if err != nil {
		return nil, err
	}

	return &gitBlameTable{repoPath: m.options.RepoPath}, nil
}

func (m *GitBlameModule) Connect(c *sqlite3.SQLiteConn, args []string) (sqlite3.VTab, error) {
	return m.Create(c, args)
}

func (m *GitBlameModule) DestroyModule() {}

func (v *gitBlameTable) Open() (sqlite3.VTabCursor, error) {
	repo, err := git.OpenRepository(v.repoPath)
	if err != nil {
		return nil, err
	}

	return &blameCursor{repo: repo}, nil

}

func (v *gitBlameTable) BestIndex(cst []sqlite3.InfoConstraint, ob []sqlite3.InfoOrderBy) (*sqlite3.IndexResult, error) {
	// TODO this should actually be implemented!
	dummy := make([]bool, len(cst))
	return &sqlite3.IndexResult{Used: dummy}, nil
}

func (v *gitBlameTable) Disconnect() error {
	return nil
}

func (v *gitBlameTable) Destroy() error { return nil }

type blameCursor struct {
	repo    *git.Repository
	current *BlameHunk
	iter    *BlameIterator
}

func (vc *blameCursor) Column(c *sqlite3.SQLiteContext, col int) error {

	switch col {
	case 0:
		//branch name
		c.ResultInt(vc.current.lineNo)
	case 1:
		c.ResultText(vc.current.fileName)

	case 2:
		c.ResultText(vc.current.commitID)
	case 3:
		c.ResultText(string(vc.current.lineContents) + " ")
	}

	return nil

}

func (vc *blameCursor) Filter(idxNum int, idxStr string, vals []interface{}) error {
	iterator, hunk, err := NewBlameIterator(vc.repo)
	if err != nil {
		return err
	}
	vc.iter = iterator
	vc.current = hunk
	return nil
}

func (vc *blameCursor) Next() error {
	hunk, err := vc.iter.Next()
	if err != nil {
		print(err.Error())
		return err
	}
	vc.current = hunk
	return nil
}

func (vc *blameCursor) EOF() bool {
	return vc.iter.current == nil
}

func (vc *blameCursor) Rowid() (int64, error) {
	return int64(0), nil
}

func (vc *blameCursor) Close() error {
	if vc.iter.current != nil {
		err := vc.iter.current.Free()
		if err != nil {
			return nil
		}
	}

	return nil
}
