package gitqlite

import (
	"fmt"
	"io"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
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
			tagger_name TEXT,
			tagger_email TEXT,
			message TEXT,
			pgp TEXT,
			target_type TEXT,
			target TEXT
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
	tagIterator, err := v.repo.TagObjects()
	if err != nil {
		return nil, err
	}
	tag, err := tagIterator.Next()
	if err != nil {
		if err == plumbing.ErrReferenceNotFound || err == io.EOF {
			return &tagCursor{0, v.repo, nil, nil, true}, nil
		}
		return nil, err
	}

	return &tagCursor{0, v.repo, tag, tagIterator, false}, nil

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
	index   int
	repo    *git.Repository
	current *object.Tag
	tagIter *object.TagIter
	eof     bool
}

func (vc *tagCursor) Column(c *sqlite3.SQLiteContext, col int) error {

	tag := vc.current

	switch col {
	case 0:
		//tag id
		c.ResultText(tag.Hash.String())
	case 1:
		//tag name
		c.ResultText(tag.Name)
	case 2:
		//tag tagger name
		c.ResultText(tag.Tagger.Name)
	case 3:
		//tagger email
		c.ResultText(tag.Tagger.Email)
	case 4:
		//message
		c.ResultText(tag.Message)
	case 5:
		//PGP
		c.ResultText(tag.PGPSignature)
	case 6:
		//target_type
		c.ResultText(tag.TargetType.String())
	case 7:
		//target
		c.ResultText(tag.Target.String())

	}
	return nil

}

func (vc *tagCursor) Filter(idxNum int, idxStr string, vals []interface{}) error {
	vc.index = 0
	return nil
}

func (vc *tagCursor) Next() error {
	vc.index++

	tag, err := vc.tagIter.Next()
	if err != nil {
		if err == io.EOF {
			vc.current = nil
			vc.eof = true
			return nil
		}
		return err
	}

	vc.current = tag
	return nil
}

func (vc *tagCursor) EOF() bool {
	return vc.eof
}

func (vc *tagCursor) Rowid() (int64, error) {
	return int64(vc.index), nil
}

func (vc *tagCursor) Close() error {
	if vc.tagIter != nil {
		vc.tagIter.Close()
	}
	return nil
}
