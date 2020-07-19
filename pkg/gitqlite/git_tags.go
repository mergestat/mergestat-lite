package gitqlite

import (
	"fmt"
	"io"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/storer"
	"github.com/mattn/go-sqlite3"
)

type gitTagModule struct{}

type gitTagTable struct {
	repoPath string
	repo     *git.Repository
}

func (m *gitTagModule) Create(c *sqlite3.SQLiteConn, args []string) (sqlite3.VTab, error) {
	err := c.DeclareVTab(fmt.Sprintf(`
		CREATE TABLE %q (
			id TEXT,
			name TEXT,
			lightweight BOOL,
			target TEXT,
			tagger_name TEXT,
			tagger_email TEXT,
			message TEXT,
			pgp TEXT,
			target_type TEXT
		)`, args[0]))
	if err != nil {
		return nil, err
	}

	// the repoPath will be enclosed in double quotes "..." since ensureTables uses %q when setting up the table
	// we need to pop those off when referring to the actual directory in the fs
	repoPath := args[3][1 : len(args[3])-1]
	return &gitTagTable{repoPath: repoPath}, nil
}

func (m *gitTagModule) Connect(c *sqlite3.SQLiteConn, args []string) (sqlite3.VTab, error) {
	return m.Create(c, args)
}

func (m *gitTagModule) DestroyModule() {}

func (v *gitTagTable) Open() (sqlite3.VTabCursor, error) {
	repo, err := git.PlainOpen(v.repoPath)
	if err != nil {
		return nil, err
	}
	v.repo = repo

	return &tagCursor{repo: v.repo}, nil

}

func (v *gitTagTable) BestIndex(cst []sqlite3.InfoConstraint, ob []sqlite3.InfoOrderBy) (*sqlite3.IndexResult, error) {
	// TODO this should actually be implemented!
	dummy := make([]bool, len(cst))
	return &sqlite3.IndexResult{Used: dummy}, nil
}

func (v *gitTagTable) Disconnect() error {
	v.repo = nil
	return nil
}
func (v *gitTagTable) Destroy() error { return nil }

type tagCursor struct {
	repo    *git.Repository
	current *plumbing.Reference
	tagIter storer.ReferenceIter
}

func (vc *tagCursor) Column(c *sqlite3.SQLiteContext, col int) error {
	lightweight := false
	tag, err := vc.repo.TagObject(vc.current.Hash())
	if err != nil {
		lightweight = true
	}
	switch col {
	case 0:
		//tag id
		c.ResultText(vc.current.Hash().String())
	case 1:
		//tag name
		c.ResultText(vc.current.Name().String())
	case 2:
		//is tag lightweight
		c.ResultBool(lightweight)
	case 3:
		//tag tagger name
		c.ResultText(vc.current.Target().Short())
	case 4:
		//tagger email
		if err == nil {
			c.ResultText(tag.Tagger.Email)
		} else {
			c.ResultNull()
		}
	case 5:
		//message
		if err == nil {
			c.ResultText(tag.Message)
		} else {
			c.ResultNull()
		}
	case 6:
		//PGP
		if err == nil {
			c.ResultText(tag.PGPSignature)
		} else {
			c.ResultNull()
		}
	case 7:
		//target_type
		if err == nil {
			c.ResultText(tag.TargetType.String())
		} else {
			c.ResultNull()
		}
	case 8:
		//target
		if err == nil {
			c.ResultText(tag.Target.String())
		} else {
			c.ResultNull()
		}
	}
	return nil

}

func (vc *tagCursor) Filter(idxNum int, idxStr string, vals []interface{}) error {
	tagIterator, err := vc.repo.Tags()
	if err != nil {
		return err
	}

	vc.tagIter = tagIterator

	tag, err := tagIterator.Next()
	if err != nil {
		return err
	}

	vc.current = tag

	return nil
}

func (vc *tagCursor) Next() error {
	tag, err := vc.tagIter.Next()
	if err != nil {
		if err == io.EOF {
			vc.current = nil
			return nil
		}
		return err
	}

	vc.current = tag
	return nil
}

func (vc *tagCursor) EOF() bool {
	return vc.current == nil
}

func (vc *tagCursor) Rowid() (int64, error) {
	return int64(0), nil
}

func (vc *tagCursor) Close() error {
	if vc.tagIter != nil {
		vc.tagIter.Close()
	}
	return nil
}
