package gitqlite

import (
	"bytes"
	"fmt"
	"log"
	"time"

	git "github.com/libgit2/git2go/v30"
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
			parent_count INT,
			tree_id TEXT,
			additions INT,
			deletions INT
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
	repo, err := git.OpenRepository(v.repoPath)
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
	current    *git.Commit
	commitIter *git.RevWalk
}

func (vc *commitCursor) Column(c *sqlite3.SQLiteContext, col int) error {
	commit := vc.current
	author := commit.Author()
	committer := commit.Committer()

	switch col {
	case 0:
		//commit id
		c.ResultText(commit.Id().String())
	case 1:
		//commit message
		c.ResultText(commit.Message())
	case 2:
		//commit summary
		c.ResultText(commit.Summary())
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
		if int(commit.ParentCount()) > 0 {
			p := commit.Parent(0)
			c.ResultText(p.Id().String())
			p.Free()
		} else {
			c.ResultNull()
		}
	case 10:
		//parent_count
		c.ResultInt(int(commit.ParentCount()))
	case 11:
		//tree_id
		c.ResultText(commit.TreeId().String())

	case 12:
		additions, _, err := statCalc(vc.repo, commit)
		if err != nil {
			return err
		}
		c.ResultInt(additions)
	case 13:
		_, deletions, err := statCalc(vc.repo, commit)
		if err != nil {
			return err
		}
		c.ResultInt(deletions)
	}
	return nil
}

func (vc *commitCursor) Filter(idxNum int, idxStr string, vals []interface{}) error {
	revWalk, err := vc.repo.Walk()
	if err != nil {
		return err
	}

	err = revWalk.PushHead()
	if err != nil {
		return err
	}

	revWalk.Sorting(git.SortNone)

	vc.commitIter = revWalk

	id := new(git.Oid)
	err = revWalk.Next(id)
	if err != nil {
		return err
	}

	commit, err := vc.repo.LookupCommit(id)
	if err != nil {
		return err
	}

	vc.current = commit

	return nil
}

func (vc *commitCursor) Next() error {
	id := new(git.Oid)
	err := vc.commitIter.Next(id)
	if err != nil {
		if id.IsZero() {
			vc.current.Free()
			vc.current = nil
			return nil
		}
		return err
	}

	commit, err := vc.repo.LookupCommit(id)
	if err != nil {
		return err
	}
	vc.current.Free()
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
	vc.repo.Free()
	return nil
}

// statCalc calculates the number of additions/deletions and returns in format additions, deletions
func statCalc(r *git.Repository, c *git.Commit) (int, int, error) {
	tree, err := c.Tree()
	if err != nil {
		return 0, 0, err
	}
	defer tree.Free()

	if c.ParentCount() == 0 {
		var additions int
		err = tree.Walk(func(path string, treeEntry *git.TreeEntry) int {
			if treeEntry.Type == git.ObjectBlob {
				blob, err := r.LookupBlob(treeEntry.Id)
				if err != nil {
					return 1
				}
				defer blob.Free()
				contents := blob.Contents()
				lineSep := []byte{'\n'}
				additions += bytes.Count(contents, lineSep)
			}
			return 0
		})
		if err != nil {
			return 0, 0, err
		}
		return additions, 0, nil

	}

	parent := c.Parent(0)
	parentTree, err := parent.Tree()
	if err != nil {
		return 0, 0, err
	}
	defer parent.Free()
	defer parentTree.Free()

	diffOpt, err := git.DefaultDiffOptions()
	if err != nil {
		return 0, 0, err
	}

	diff, err := r.DiffTreeToTree(parentTree, tree, &diffOpt)
	if err != nil {
		return 0, 0, err
	}
	defer func() {
		err := diff.Free()
		if err != nil {
			log.Fatal(err)
		}
	}()

	diffFindOpt, err := git.DefaultDiffFindOptions()
	if err != nil {
		return 0, 0, err
	}

	err = diff.FindSimilar(&diffFindOpt)
	if err != nil {
		return 0, 0, err
	}

	stats, err := diff.Stats()
	if err != nil {
		return 0, 0, err
	}
	defer func() {
		err := stats.Free()
		if err != nil {
			log.Fatal(err)
		}
	}()

	return stats.Insertions(), stats.Deletions(), nil
}
