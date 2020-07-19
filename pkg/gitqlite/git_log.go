package gitqlite

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/mattn/go-sqlite3"
)

type gitLogModule struct{}

type gitLogTable struct {
	repoPath string
	repo     *git.Repository
}

func (m *gitLogModule) Create(c *sqlite3.SQLiteConn, args []string) (sqlite3.VTab, error) {
	err := c.DeclareVTab(fmt.Sprintf(`
		CREATE TABLE %q (
			id TEXT,
			message TEXT,
			summary TEXT,
			author_name TEXT,
			author_email TEXT,
			author_when DATETIME,
			committer_name TEXT,
			committer_email TEXT,
			committer_when DATETIME, 
			parent_id TEXT,
			parent_count INT(10),
			tree_id TEXT,
			additions INT(10),
			deletions INT(10)
		)`, args[0]))
	if err != nil {
		return nil, err
	}

	// the repoPath will be enclosed in double quotes "..." since ensureTables uses %q when setting up the table
	// we need to pop those off when referring to the actual directory in the fs
	repoPath := args[3][1 : len(args[3])-1]
	return &gitLogTable{repoPath: repoPath}, nil
}

func (m *gitLogModule) Connect(c *sqlite3.SQLiteConn, args []string) (sqlite3.VTab, error) {
	return m.Create(c, args)
}

func (m *gitLogModule) DestroyModule() {}

func (v *gitLogTable) Open() (sqlite3.VTabCursor, error) {
	repo, err := git.PlainOpen(v.repoPath)
	if err != nil {
		return nil, err
	}
	v.repo = repo

	return &commitCursor{repo: v.repo}, nil
}

func (v *gitLogTable) BestIndex(cst []sqlite3.InfoConstraint, ob []sqlite3.InfoOrderBy) (*sqlite3.IndexResult, error) {
	// TODO this should actually be implemented!
	dummy := make([]bool, len(cst))
	return &sqlite3.IndexResult{Used: dummy}, nil
}

func (v *gitLogTable) Disconnect() error {
	v.repo = nil
	return nil
}
func (v *gitLogTable) Destroy() error { return nil }

type commitCursor struct {
	repo       *git.Repository
	current    *object.Commit
	commitIter object.CommitIter
}

func (vc *commitCursor) Column(c *sqlite3.SQLiteContext, col int) error {
	commit := vc.current
	author := commit.Author
	committer := commit.Committer

	switch col {
	case 0:
		//commit id
		c.ResultText(commit.ID().String())
	case 1:
		//commit message
		c.ResultText(commit.Message)
	case 2:
		//commit summary
		c.ResultText(strings.Split(commit.Message, "\n")[0])
	case 3:
		//commit author name
		c.ResultText(author.Name)
	case 4:
		//commit author email
		c.ResultText(author.Email)
	case 5:
		//author when
		c.ResultText(author.When.Format(time.RFC3339Nano))
	case 6:
		//committer name
		c.ResultText(committer.Name)
	case 7:
		//committer email
		c.ResultText(committer.Email)
	case 8:
		//committer when
		c.ResultText(committer.When.Format(time.RFC3339Nano))
	case 9:
		//parent_id
		if int(commit.NumParents()) > 0 {
			p, err := commit.Parent(0)
			if err != nil {
				return err
			}
			c.ResultText(p.ID().String())
		} else {
			c.ResultNull()
		}
	case 10:
		//parent_count
		c.ResultInt(int(commit.NumParents()))
	case 11:
		//tree_id
		tree, err := vc.current.Tree()
		if err != nil {
			return err
		}
		c.ResultText(tree.ID().String())

	case 12:
		if int(commit.NumParents()) > 0 {
			additions, _, err := statCalc(commit)
			if err != nil {
				return err
			}
			c.ResultInt(additions)
		} else {
			c.ResultInt(0)
		}
	case 13:
		if int(commit.NumParents()) > 0 {
			_, deletions, err := statCalc(commit)
			if err != nil {
				return err
			}
			c.ResultInt(deletions)
		} else {
			c.ResultInt(0)
		}

	}
	return nil
}

func (vc *commitCursor) Filter(idxNum int, idxStr string, vals []interface{}) error {
	headRef, err := vc.repo.Head()
	if err != nil {
		if err == plumbing.ErrReferenceNotFound {
			return nil
		}
		return err
	}

	iter, err := vc.repo.Log(&git.LogOptions{
		From:  headRef.Hash(),
		Order: git.LogOrderCommitterTime,
	})
	if err != nil {
		return err
	}
	vc.commitIter = iter

	commit, err := iter.Next()
	if err != nil {
		return err
	}

	vc.current = commit

	return nil
}

func (vc *commitCursor) Next() error {
	commit, err := vc.commitIter.Next()
	if err != nil {
		if err == io.EOF {
			vc.current = nil
			return nil
		}
		return err
	}

	vc.current = commit
	return nil
}

func (vc *commitCursor) EOF() bool {
	return vc.current == nil
}

func (vc *commitCursor) Rowid() (int64, error) {
	return int64(0), nil
}

func (vc *commitCursor) Close() error {
	if vc.commitIter != nil {
		vc.commitIter.Close()
	}
	vc.current = nil
	return nil
}

//statCalc calculates the number of additions/deletions and returns in format additions, deletions
func statCalc(c *object.Commit) (int, int, error) {
	p, err := c.Parent(0)
	if err != nil {
		return 0, 0, err
	}
	parentTree, err := p.Tree()
	if err != nil {
		return 0, 0, err
	}
	tree, err := c.Tree()
	if err != nil {
		return 0, 0, err
	}

	patch, err := parentTree.Patch(tree)
	if err != nil {
		return 0, 0, err
	}
	additions, deletions := 0, 0
	stats := patch.Stats()
	for i := range stats {
		additions += stats[i].Addition
		deletions += stats[i].Deletion
	}
	return additions, deletions, nil
}
