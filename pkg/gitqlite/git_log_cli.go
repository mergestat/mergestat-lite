package gitqlite

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/augmentable-dev/askgit/pkg/gitlog"
	"github.com/mattn/go-sqlite3"
)

type gitLogCLIModule struct{}

type gitLogCLITable struct {
	repoPath string
}

func (m *gitLogCLIModule) Create(c *sqlite3.SQLiteConn, args []string) (sqlite3.VTab, error) {
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
	return &gitLogCLITable{repoPath: repoPath}, nil
}

func (m *gitLogCLIModule) Connect(c *sqlite3.SQLiteConn, args []string) (sqlite3.VTab, error) {
	return m.Create(c, args)
}

func (m *gitLogCLIModule) DestroyModule() {}

func (v *gitLogCLITable) Open() (sqlite3.VTabCursor, error) {
	return &commitCLICursor{repoPath: v.repoPath}, nil
}

func (v *gitLogCLITable) BestIndex(cst []sqlite3.InfoConstraint, ob []sqlite3.InfoOrderBy) (*sqlite3.IndexResult, error) {
	// TODO this should actually be implemented!
	dummy := make([]bool, len(cst))
	return &sqlite3.IndexResult{Used: dummy}, nil
}

func (v *gitLogCLITable) Disconnect() error {
	return nil
}
func (v *gitLogCLITable) Destroy() error { return nil }

type commitCLICursor struct {
	repoPath string
	iter     *gitlog.CommitIter
	current  *gitlog.Commit
}

func (vc *commitCLICursor) Filter(idxNum int, idxStr string, vals []interface{}) error {
	iter, err := gitlog.Execute(vc.repoPath)
	if err != nil {
		return err
	}
	vc.iter = iter

	commit, err := iter.Next()
	if err != nil {
		return err
	}

	vc.current = commit
	return nil
}

func (vc *commitCLICursor) Next() error {
	commit, err := vc.iter.Next()
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

func (vc *commitCLICursor) EOF() bool {
	return vc.current == nil
}

func (vc *commitCLICursor) Column(c *sqlite3.SQLiteContext, col int) error {
	current := vc.current
	switch col {
	case 0:
		//commit id
		c.ResultText(current.SHA)
	case 1:
		//commit message
		c.ResultText(current.Message)
	case 2:
		//commit summary
		c.ResultText(strings.Split(current.Message, "\n")[0])
	case 3:
		//commit author name
		c.ResultText(current.AuthorName)
	case 4:
		//commit author email
		c.ResultText(current.AuthorEmail)
	case 5:
		//author when
		c.ResultText(current.AuthorWhen.Format(time.RFC3339Nano))
	case 6:
		//committer name
		c.ResultText(current.CommitterName)
	case 7:
		//committer email
		c.ResultText(current.CommitterEmail)
	case 8:
		//committer when
		c.ResultText(current.CommitterWhen.Format(time.RFC3339Nano))
	case 9:
		//parent_id
		parentID := strings.Split(current.ParentID, " ")[0]
		if strings.Trim(parentID, " ") == "" {
			c.ResultNull()
		} else {
			c.ResultText(parentID)
		}
	case 10:
		//parent_count
		parentIDs := strings.Split(current.ParentID, " ")
		if len(parentIDs) > 0 && parentIDs[0] == "" {
			c.ResultInt(0)
		} else {
			c.ResultInt(len(parentIDs))
		}
	case 11:
		//tree_id
		c.ResultText(current.TreeID)
	case 12:
		c.ResultInt(current.Additions)
	case 13:
		c.ResultInt(current.Deletions)

	}
	return nil
}

func (vc *commitCLICursor) Rowid() (int64, error) {
	return int64(0), nil
}

func (vc *commitCLICursor) Close() error {
	return nil
}
