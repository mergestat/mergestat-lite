package gitqlite

import (
	"fmt"
	"io"

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
			file_path TEXT,
			commit_id TEXT,
			line_contents TEXT
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
	current *BlamedLine
	iter    *BlameIterator
}

func (vc *blameCursor) Column(c *sqlite3.SQLiteContext, col int) error {
	blamedLine := vc.current
	switch col {
	case 0:
		c.ResultInt(blamedLine.Line)
	case 1:
		c.ResultText(blamedLine.File)
	case 2:
		c.ResultText(blamedLine.CommitID)
	case 3:
		c.ResultText(blamedLine.Content)
	}

	return nil

}

func (vc *blameCursor) Filter(idxNum int, idxStr string, vals []interface{}) error {
	iterator, err := NewBlameIterator(vc.repo)
	if err != nil {
		return err
	}

	blamedLine, err := iterator.Next()
	if err != nil {
		if err == io.EOF {
			vc.current = nil
			return nil
		}
		return err
	}

	vc.iter = iterator
	vc.current = blamedLine
	return nil
}

func (vc *blameCursor) Next() error {
	blamedLine, err := vc.iter.Next()
	if err != nil {
		if err == io.EOF {
			vc.current = nil
			return nil
		}
		return err
	}
	vc.current = blamedLine
	return nil
}

func (vc *blameCursor) EOF() bool {
	return vc.current == nil
}

func (vc *blameCursor) Rowid() (int64, error) {
	return int64(0), nil
}

func (vc *blameCursor) Close() error {
	return nil
}
