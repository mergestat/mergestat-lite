package gitqlite

import (
	"fmt"
	"io"

	"github.com/augmentable-dev/askgit/pkg/gitlog"
	"github.com/mattn/go-sqlite3"
)

type gitStatsCLIModule struct{}

type gitStatsCLITable struct {
	repoPath string
}

func (m *gitStatsCLIModule) Create(c *sqlite3.SQLiteConn, args []string) (sqlite3.VTab, error) {
	err := c.DeclareVTab(fmt.Sprintf(`
		CREATE TABLE %q (
			commit_id TEXT,
			file TEXT,
			additions INT(10),
			deletions INT(10)
		)`, args[0]))
	if err != nil {
		return nil, err
	}

	// the repoPath will be enclosed in double quotes "..." since ensureTables uses %q when setting up the table
	// we need to pop those off when referring to the actual directory in the fs
	repoPath := args[3][1 : len(args[3])-1]
	return &gitStatsCLITable{repoPath: repoPath}, nil
}

func (m *gitStatsCLIModule) Connect(c *sqlite3.SQLiteConn, args []string) (sqlite3.VTab, error) {
	return m.Create(c, args)
}

func (m *gitStatsCLIModule) DestroyModule() {}

func (v *gitStatsCLITable) Open() (sqlite3.VTabCursor, error) {
	return &statsCLICursor{repoPath: v.repoPath}, nil
}

func (v *gitStatsCLITable) BestIndex(cst []sqlite3.InfoConstraint, ob []sqlite3.InfoOrderBy) (*sqlite3.IndexResult, error) {
	// TODO this should actually be implemented!
	dummy := make([]bool, len(cst))
	return &sqlite3.IndexResult{Used: dummy}, nil
}

func (v *gitStatsCLITable) Disconnect() error {
	return nil
}
func (v *gitStatsCLITable) Destroy() error { return nil }

type statsCLICursor struct {
	repoPath  string
	iter      *gitlog.CommitIter
	current   *gitlog.Commit
	statIndex int
}

func (vc *statsCLICursor) Filter(idxNum int, idxStr string, vals []interface{}) error {
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

func (vc *statsCLICursor) Next() error {
	if vc.statIndex+1 < len(vc.current.Files) {
		vc.statIndex++
		return nil
	}

	commit, err := vc.iter.Next()
	if err != nil {
		if err == io.EOF {
			vc.current = nil
			return nil
		}
		return err
	}

	vc.statIndex = 0

	vc.current = commit
	if len(vc.current.Files) == 0 {
		err = vc.Next()
		if err != nil {
			return err
		}
	}
	return nil
}

func (vc *statsCLICursor) EOF() bool {
	return vc.current == nil
}

func (vc *statsCLICursor) Column(c *sqlite3.SQLiteContext, col int) error {
	current := vc.current
	switch col {
	case 0:
		//commit id
		c.ResultText(current.SHA)

	case 1:
		if len(current.Files) > vc.statIndex {
			c.ResultText(current.Files[vc.statIndex])
		} else {
			c.ResultText("")
		}
	case 2:
		if len(current.Deletions) > vc.statIndex {

			c.ResultInt(current.Deletions[vc.statIndex])
		} else {
			c.ResultInt(0)
		}
	case 3:
		if len(current.Additions) > vc.statIndex {

			c.ResultInt(current.Additions[vc.statIndex])
		} else {
			c.ResultInt(0)
		}
	}
	return nil
}

func (vc *statsCLICursor) Rowid() (int64, error) {
	return int64(0), nil
}

func (vc *statsCLICursor) Close() error {
	return nil
}
