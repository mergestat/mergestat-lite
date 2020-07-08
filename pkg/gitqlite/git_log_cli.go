package gitqlite

import (
	"fmt"
	"strings"
	"time"

	"github.com/augmentable-dev/gitqlite/pkg/gitlog"
	"github.com/go-git/go-git/v5"
	"github.com/mattn/go-sqlite3"
)

type gitLogCLIModule struct{}

type gitLogCLITable struct {
	repoPath string
	repo     *git.Repository
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
	return &gitLogCLITable{repoPath: repoPath}, nil
}

func (m *gitLogCLIModule) Connect(c *sqlite3.SQLiteConn, args []string) (sqlite3.VTab, error) {
	return m.Create(c, args)
}

func (m *gitLogCLIModule) DestroyModule() {}

func (v *gitLogCLITable) Open() (sqlite3.VTabCursor, error) {
	res, err := gitlog.Execute(v.repoPath)
	if err != nil {
		return nil, err
	}
	return &commitCLICursor{0, res, res[0], false}, nil
}

func (v *gitLogCLITable) BestIndex(cst []sqlite3.InfoConstraint, ob []sqlite3.InfoOrderBy) (*sqlite3.IndexResult, error) {
	// TODO this should actually be implemented!
	dummy := make([]bool, len(cst))
	return &sqlite3.IndexResult{Used: dummy}, nil
}

func (v *gitLogCLITable) Disconnect() error {
	v.repo = nil
	return nil
}
func (v *gitLogCLITable) Destroy() error { return nil }

type commitCLICursor struct {
	index   int
	results gitlog.Result
	current *gitlog.Commit
	eof     bool
}

func (vc *commitCLICursor) Column(c *sqlite3.SQLiteContext, col int) error {

	switch col {
	case 0:
		//commit id
		c.ResultText(vc.current.SHA)
	case 1:
		//commit message
		c.ResultText(vc.current.Message)
	case 2:
		//commit summary
		c.ResultText(strings.Split(vc.current.Message, "\n")[0])
	case 3:
		//commit author name
		c.ResultText(vc.current.AuthorName)
	case 4:
		//commit author email
		c.ResultText(vc.current.AuthorEmail)
	case 5:
		//author when
		c.ResultText(vc.current.AuthorWhen.Format(time.RFC3339Nano))
	case 6:
		//committer name
		c.ResultText(vc.current.CommitterName)
	case 7:
		//committer email
		c.ResultText(vc.current.CommitterEmail)
	case 8:
		//committer when
		c.ResultText(vc.current.CommitterWhen.Format(time.RFC3339Nano))
	case 9:
		//parent_id
		c.ResultText(vc.current.ParentID)
	case 10:
		//parent_count
		c.ResultInt(len(strings.Split(vc.current.ParentID, " ")))
	case 11:
		//tree_id
		c.ResultText(vc.current.TreeID)

	case 12:
		c.ResultInt(vc.current.Additions)
	case 13:
		c.ResultInt(vc.current.Deletions)

	}
	return nil
}

func (vc *commitCLICursor) Filter(idxNum int, idxStr string, vals []interface{}) error {
	vc.index = 0
	return nil
}

func (vc *commitCLICursor) Next() error {
	vc.index++
	if vc.index < len(vc.results) {
		vc.current = vc.results[vc.index]
		return nil

	} else {
		vc.current = nil
		vc.eof = true
		return nil
	}
}

func (vc *commitCLICursor) EOF() bool {
	return vc.eof
}

func (vc *commitCLICursor) Rowid() (int64, error) {
	return int64(vc.index), nil
}

func (vc *commitCLICursor) Close() error {
	return nil
}
