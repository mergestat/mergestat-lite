package gitqlite

import (
	"fmt"
	"io"

	"github.com/go-git/go-git/plumbing/object"
	git "github.com/libgit2/git2go/v30"

	// "github.com/go-git/go-git/v5"
	// "github.com/go-git/go-git/v5/plumbing"
	// "github.com/go-git/go-git/v5/plumbing/object"
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

func (vc *LLTreeCursor) Column(c *sqlite3.SQLiteContext, col int) error {

	switch col {
	case 0:
		//commit id
		c.ResultText(vc.self.commit.Id().String())
	case 1:
		//tree id
		c.ResultText(vc.self.tree.Id().String())
	case 2:
		//tree name
		c.ResultText(vc.self.current.Name)
	case 3:
		//filemode
		c.ResultText(string(int(vc.self.current.Filemode)))
	case 4:
		//filetype
		c.ResultText(vc.self.current.Type.String())
	case 5:
		//File
		// change this to text with the blob section of go-git
		if vc.self.current.Type.String() == "Blob" {
			blob, err := vc.self.repo.LookupBlob(vc.self.current.Id)
			if err != nil {
				return err
			}
			contents := blob.Contents()
			c.ResultText(string(contents))
		} else {
			c.ResultText("NULL")
		}
	}
	return nil
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

func newTreeCommitIterator(commitIter object.CommitIter) (*treeCommitIterator, error) {
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

type LLTreeCursor struct {
	parent *LLTreeCursor
	self   *treeCursor
	eof    bool
}

type treeCursor struct {
	index    int
	repo     *git.Repository
	iterator *git.RevWalk
	oid      *git.Oid
	commit   *git.Commit
	tree     *git.Tree
	current  *git.TreeEntry
	eof      bool
}

func (v *gitTreeTable) Open() (sqlite3.VTabCursor, error) {
	repo, err := git.OpenRepository(v.repoPath)
	if err != nil {
		return nil, err
	}
	v.repo = repo

	v.repo = repo

	revWalk, err := repo.Walk()
	if err != nil {
		return nil, err
	}

	revWalk.Sorting(git.SortTime)
	err = revWalk.PushHead()
	if err != nil {
		return nil, err
	}

	var oid git.Oid
	err = revWalk.Next(&oid)
	if err != nil {
		return nil, err
	}
	commit, err := repo.LookupCommit(&oid)
	if err != nil {
		return nil, err
	}
	tree, err := commit.Tree()
	if err != nil {
		return nil, err
	}
	current := tree.EntryByIndex(0)

	return &LLTreeCursor{nil, &treeCursor{0, v.repo, revWalk, &oid, commit, tree, current, false}, false}, nil
}

func (vc *LLTreeCursor) Next() error {
	vc.self.index++
	//Iterates to next tree entry if the index is less than the entry count
	if vc.self.index < int(vc.self.tree.EntryCount()) {
		// go to next entry in the tree
		file := vc.self.tree.EntryByIndex(uint64(vc.self.index))
		//if next entry is a tree then go into that tree
		if file.Type.String() == "Tree" {
			tree, err := vc.self.repo.LookupTree(file.Id)
			if err != nil {
				return nil
			}
			//set parent to self
			vc.parent = vc
			//set self to new tree
			vc.self.tree = tree
			vc.self.index = 0
			vc.self.current = vc.self.tree.EntryByIndex(0)
			return nil

		}
		// if not EOF and not err go to next file in tree
		vc.self.current = file
		return nil
	} else {
		if vc.parent != nil {
			vc.self = vc.parent.self
			vc.parent = vc.parent.parent
			return nil
		}
		var oid git.Oid
		err := vc.self.iterator.Next(&oid)
		if git.IsErrorCode(err, git.ErrIterOver) {
			vc.self.current = nil
			vc.parent = nil
			vc.eof = true
			return nil
		}
		if err != nil {
			return err
		}
		commit, err := vc.self.repo.LookupCommit(&oid)
		if err != nil {
			return err
		}
		tree, err := commit.Tree()
		if err != nil {
			return err
		}
		vc.self.oid = &oid
		vc.self.commit = commit
		vc.self.tree = tree
		vc.self.current = tree.EntryByIndex(0)
		vc.self.index = 0
		return nil

	}
}
func (vc *LLTreeCursor) Filter(idxNum int, idxStr string, vals []interface{}) error {
	vc.self.index = 0
	return nil
}
func (vc *LLTreeCursor) EOF() bool {
	return vc.eof
}

func (vc *LLTreeCursor) Rowid() (int64, error) {
	return int64(vc.self.index), nil
}

func (vc *LLTreeCursor) Close() error {
	vc.parent = nil
	vc.self.iterator.Free()
	return nil
}
