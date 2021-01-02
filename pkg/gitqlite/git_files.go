package gitqlite

import (
	"fmt"
	"io"
	"path"

	git "github.com/libgit2/git2go/v31"
	"github.com/mattn/go-sqlite3"
)

type GitFilesModule struct {
	options *GitFilesModuleOptions
}

type GitFilesModuleOptions struct {
	RepoPath string
}

func NewGitFilesModule(options *GitFilesModuleOptions) *GitFilesModule {
	return &GitFilesModule{options}
}

type gitFilesTable struct {
	repoPath string
}

func (m *GitFilesModule) EponymousOnlyModule() {}

func (m *GitFilesModule) Create(c *sqlite3.SQLiteConn, args []string) (sqlite3.VTab, error) {
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

	return &gitFilesTable{repoPath: m.options.RepoPath}, nil
}

func (m *GitFilesModule) Connect(c *sqlite3.SQLiteConn, args []string) (sqlite3.VTab, error) {
	return m.Create(c, args)
}

func (m *GitFilesModule) DestroyModule() {}

func (vc *gitFilesCursor) Column(c *sqlite3.SQLiteContext, col int) error {
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

func (v *gitFilesTable) Disconnect() error {
	return nil
}

func (v *gitFilesTable) Destroy() error { return nil }

type gitFilesCursor struct {
	repo     *git.Repository
	iterator *commitFileIter
	current  *commitFile
}

func (v *gitFilesTable) Open() (sqlite3.VTabCursor, error) {
	repo, err := git.OpenRepository(v.repoPath)
	if err != nil {
		return nil, err
	}

	return &gitFilesCursor{repo: repo}, nil
}

func (v *gitFilesTable) BestIndex(cst []sqlite3.InfoConstraint, ob []sqlite3.InfoOrderBy) (*sqlite3.IndexResult, error) {
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

func (vc *gitFilesCursor) Filter(idxNum int, idxStr string, vals []interface{}) error {
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

func (vc *gitFilesCursor) Next() error {
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

func (vc *gitFilesCursor) EOF() bool {
	return vc.current == nil
}

func (vc *gitFilesCursor) Rowid() (int64, error) {
	return int64(0), nil
}

func (vc *gitFilesCursor) Close() error {
	vc.iterator.Close()
	return nil
}
