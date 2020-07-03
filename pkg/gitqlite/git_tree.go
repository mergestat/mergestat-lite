package gitqlite

import (
	"fmt"
	"io"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
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
				name TEXT,
				mode TEXT,
				type TEXT,
				contents TEXT
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
	commit := vc.iterator.commit
	treeHeadEntry := commit.TreeHash.String() // should be structure Name String Id *Oid Type ObjectType Filemode Filemode
	currentFile := vc.current
	fileMode, err := currentFile.Mode.ToOSFileMode()
	if err != nil {
		return err
	}
	fileContents, err := currentFile.Contents()
	if err != nil {
		return err
	}
	name := currentFile.Name

	switch col {
	case 0:
		//commit id
		c.ResultText(commit.ID().String())
	case 1:
		//tree id
		c.ResultText(treeHeadEntry)
	case 2:
		//tree name
		c.ResultText(name)
	case 3:
		//filemode
		c.ResultText(fileMode.String())
	case 4:
		//filetype
		c.ResultText(currentFile.Type().String())
	case 5:
		//File
		// change this to text with the blob section of go-git

		c.ResultText(fileContents)
	}
	return nil
	//return errors.New("something messed up in column")
}
func (v *gitTreeTable) BestIndex(cst []sqlite3.InfoConstraint, ob []sqlite3.InfoOrderBy) (*sqlite3.IndexResult, error) {
	// TODO this should actually be implemented!
	dummy := make([]bool, len(cst))
	return &sqlite3.IndexResult{Used: dummy}, nil
}

func (v *gitTreeTable) Disconnect() error {
	v.repo = nil
	return nil
}

func (v *gitTreeTable) Destroy() error { return nil }

type treeCommitIterator struct {
	commitIter   object.CommitIter
	commit       *object.Commit
	treeFileIter *object.FileIter
}

func NewTreeCommitIterator(commitIter object.CommitIter) (*treeCommitIterator, error) {
	commit, err := commitIter.Next()
	if err != nil {
		return nil, err
	}

	tree, err := commit.Tree()
	if err != nil {
		return nil, err
	}

	treeFileIter := tree.Files()

	return &treeCommitIterator{commitIter, commit, treeFileIter}, nil
}

func (iter *treeCommitIterator) Next() (*object.File, error) {
	file, err := iter.treeFileIter.Next()
	if err != nil {
		// if it is the last file in the tree go to the next commit

		if err == io.EOF {

			commit, err := iter.commitIter.Next()

			if err != nil {

				//if it is the last file in the last tree of the commits return EOF
				if err == io.EOF {
					return nil, io.EOF
				}
				return nil, err
			}
			iter.commit = commit
			tree, err := commit.Tree()
			if err != nil {
				return nil, err
			}
			fileIter := tree.Files()
			iter.treeFileIter = fileIter
			file, err := iter.treeFileIter.Next()
			if err != nil {
				return nil, err
			}
			return file, nil
		}
		return nil, err
	}
	return file, nil

}

func (iter *treeCommitIterator) Close() {
	iter.treeFileIter.Close()
	iter.commitIter.Close()
}

type treeCursor struct {
	index    int
	repo     *git.Repository
	iterator *treeCommitIterator
	current  *object.File
	eof      bool
}

func (v *gitTreeTable) Open() (sqlite3.VTabCursor, error) {
	repo, err := git.PlainOpen(v.repoPath)
	if err != nil {
		return nil, err
	}
	v.repo = repo

	headRef, err := v.repo.Head()
	if err != nil {
		return nil, err
	}
	iter, err := v.repo.Log(&git.LogOptions{
		From:  headRef.Hash(),
		Order: git.LogOrderCommitterTime,
	})
	if err != nil {
		return nil, err
	}

	treeCommitIter, err := NewTreeCommitIterator(iter)
	if err != nil {
		return nil, err
	}

	current, err := treeCommitIter.Next()
	if err != nil {
		return nil, err
	}

	return &treeCursor{0, v.repo, treeCommitIter, current, false}, nil
}

func (vc *treeCursor) Next() error {
	vc.index++
	//Iterates to next file
	file, err := vc.iterator.Next()
	if err != nil {
		if err == io.EOF {
			vc.current = nil
			vc.eof = true
			return nil
		}
		return err
	}
	// if not EOF and not err go to next file in tree
	vc.current = file
	return nil
}
func (vc *treeCursor) Filter(idxNum int, idxStr string, vals []interface{}) error {
	vc.index = 0
	return nil
}
func (vc *treeCursor) EOF() bool {
	return vc.eof
}

func (vc *treeCursor) Rowid() (int64, error) {
	return int64(vc.index), nil
}

func (vc *treeCursor) Close() error {
	vc.iterator.Close()
	return nil
}
