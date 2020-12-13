package gitqlite

import (
	"fmt"
	"time"

	git "github.com/libgit2/git2go/v31"
	"github.com/mattn/go-sqlite3"
)

type GitLogModule struct{}

func NewGitLogModule() *GitLogModule {
	return &GitLogModule{}
}

type gitLogTable struct {
	repoPath string
	repo     *git.Repository
}

func (m *GitLogModule) Create(c *sqlite3.SQLiteConn, args []string) (sqlite3.VTab, error) {
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
			tree_id TEXT
		)`, args[0]))
	if err != nil {
		return nil, err
	}

	// the repoPath will be enclosed in double quotes "..." since ensureTables uses %q when setting up the table
	// we need to pop those off when referring to the actual directory in the fs
	repoPath := args[3][1 : len(args[3])-1]
	return &gitLogTable{repoPath: repoPath}, nil
}

func (m *GitLogModule) Connect(c *sqlite3.SQLiteConn, args []string) (sqlite3.VTab, error) {
	return m.Create(c, args)
}

func (m *GitLogModule) DestroyModule() {}

func (v *gitLogTable) Open() (sqlite3.VTabCursor, error) {
	repo, err := git.OpenRepository(v.repoPath)
	if err != nil {
		return nil, err
	}
	v.repo = repo

	return &commitCursor{repo: v.repo}, nil
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
	}
	return nil
}

func (v *gitLogTable) BestIndex(cst []sqlite3.InfoConstraint, ob []sqlite3.InfoOrderBy) (*sqlite3.IndexResult, error) {
	used := make([]bool, len(cst))
	// TODO this loop construct won't work well for multiple constraints...
	for c, constraint := range cst {
		switch {
		case constraint.Usable && constraint.Column == 0 && constraint.Op == sqlite3.OpEQ:
			used[c] = true
			return &sqlite3.IndexResult{Used: used, IdxNum: 1, IdxStr: "commit-by-id", EstimatedCost: 1.0, EstimatedRows: 1}, nil
		}
	}

	return &sqlite3.IndexResult{Used: used, EstimatedCost: 100}, nil
}

func (vc *commitCursor) Filter(idxNum int, idxStr string, vals []interface{}) error {
	switch idxNum {
	case 0:
		// no index is used, walk over all commits
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
	case 1:
		// commit-by-id - lookup a commit by the ID used in the query
		revWalk, err := vc.repo.Walk()
		if err != nil {
			return err
		}
		// nothing is pushed to this revWalk
		vc.commitIter = revWalk

		id, err := git.NewOid(vals[0].(string))
		if err != nil {
			return err
		}
		commit, err := vc.repo.LookupCommit(id)
		if err != nil {
			return err
		}
		vc.current = commit
	}

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
	vc.commitIter.Free()
	vc.repo.Free()
	return nil
}
