package git

import (
	"fmt"
	"go.riyazali.net/sqlite"
	"time"

	git "github.com/libgit2/git2go/v31"
)

type LogModule struct{}

func (l LogModule) Connect(_ *sqlite.Conn, args []string, declare func(string) error) (sqlite.VirtualTable, error) {
	// TODO(@riyaz): parse args to extract repo
	var repo = "."

	var schema = fmt.Sprintf(`CREATE TABLE %s (id TEXT, message TEXT, summary TEXT, 
			author_name TEXT, author_email TEXT, author_when DATETIME, 
			committer_name TEXT, committer_email TEXT, committer_when DATETIME, 
			parent_id TEXT, parent_count INT)`, args[0])

	return &gitLogTable{repoPath: repo}, declare(schema)
}

type gitLogTable struct {
	repoPath string
}

func (v *gitLogTable) BestIndex(input *sqlite.IndexInfoInput) (*sqlite.IndexInfoOutput, error) {
	var usage = make([]*sqlite.ConstraintUsage, len(input.Constraints))

	// TODO this loop construct won't work well for multiple constraints...
	for c, constraint := range input.Constraints {
		if constraint.Usable && constraint.ColumnIndex == 0 && constraint.Op == sqlite.INDEX_CONSTRAINT_EQ {
			usage[c] = &sqlite.ConstraintUsage{ArgvIndex: 1, Omit: false}
			return &sqlite.IndexInfoOutput{ConstraintUsage: usage, IndexNumber: 1, IndexString: "commit-by-id", EstimatedCost: 1.0, EstimatedRows: 1}, nil
		}
	}

	return &sqlite.IndexInfoOutput{ConstraintUsage: usage, EstimatedCost: 100}, nil
}

func (v *gitLogTable) Open() (sqlite.VirtualCursor, error) {
	repo, err := git.OpenRepository(v.repoPath)
	if err != nil {
		return nil, err
	}
	return &commitCursor{repo: repo}, nil
}

func (v *gitLogTable) Disconnect() error { return nil }
func (v *gitLogTable) Destroy() error    { return nil }

type commitCursor struct {
	repo       *git.Repository
	current    *git.Commit
	commitIter *git.RevWalk
}

func (vc *commitCursor) Filter(i int, _ string, values ...sqlite.Value) error {
	switch i {
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

		id, err := git.NewOid(values[0].Text())
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

func (vc *commitCursor) Column(c *sqlite.Context, col int) error {
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

func (vc *commitCursor) Eof() bool             { return vc.current == nil }
func (vc *commitCursor) Rowid() (int64, error) { return int64(0), nil }
func (vc *commitCursor) Close() error          { vc.commitIter.Free(); vc.repo.Free(); return nil }
