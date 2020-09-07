package gitqlite

import (
	"fmt"

	git "github.com/libgit2/git2go/v30"
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
	repo, err := git.OpenRepository(v.repoPath)
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

type currentTag struct {
	name string
	id   *git.Oid
}

type tagCursor struct {
	repo  *git.Repository
	index int
	tags  []*currentTag
}

func (vc *tagCursor) Column(c *sqlite3.SQLiteContext, col int) error {
	current := vc.tags[vc.index]

	ref, err := vc.repo.References.Lookup(current.name)
	if err != nil {
		return err
	}
	defer ref.Free()

	isLightweight := false
	tag, err := vc.repo.LookupTag(current.id)
	if err != nil {
		isLightweight = true
	} else {
		defer tag.Free()
	}

	switch col {
	case 0:
		c.ResultText(ref.Name())
	case 1:
		c.ResultText(ref.Shorthand())
	case 2:
		c.ResultBool(isLightweight)
	case 3:
		if tag != nil {
			c.ResultText(tag.Target().Id().String())
		} else {
			c.ResultText(ref.Target().String())
		}
	case 4:
		if tag != nil {
			c.ResultText(tag.Tagger().Name)
		} else {
			c.ResultNull()
		}
	case 5:
		if tag != nil {
			c.ResultText(tag.Tagger().Email)
		} else {
			c.ResultNull()
		}
	case 6:
		if tag != nil {
			c.ResultText(tag.Message())
		} else {
			c.ResultNull()
		}
	case 7:
		if tag != nil {
			c.ResultText(tag.TargetType().String())
		} else {
			c.ResultNull()
		}
	}
	return nil

}

func (vc *tagCursor) Filter(idxNum int, idxStr string, vals []interface{}) error {
	tags := make([]*currentTag, 0)
	vc.repo.Tags.Foreach(func(name string, id *git.Oid) error {
		tags = append(tags, &currentTag{name, id})
		return nil
	})
	vc.tags = tags

	return nil
}

func (vc *tagCursor) Next() error {
	vc.index++
	return nil
}

func (vc *tagCursor) EOF() bool {
	return vc.index >= len(vc.tags)
}

func (vc *tagCursor) Rowid() (int64, error) {
	return int64(0), nil
}

func (vc *tagCursor) Close() error {
	return nil
}
