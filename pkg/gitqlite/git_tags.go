package gitqlite

import (
	"fmt"

	git "github.com/libgit2/git2go/v31"
	"github.com/mattn/go-sqlite3"
)

type GitTagsModule struct {
	options *GitTagsModuleOptions
}

type GitTagsModuleOptions struct {
	RepoPath string
}

func NewGitTagsModule(options *GitTagsModuleOptions) *GitTagsModule {
	return &GitTagsModule{options}
}

type gitTagsTable struct {
	repoPath string
	repo     *git.Repository
}

func (m *GitTagsModule) EponymousOnlyModule() {}

func (m *GitTagsModule) Create(c *sqlite3.SQLiteConn, args []string) (sqlite3.VTab, error) {
	err := c.DeclareVTab(fmt.Sprintf(`
		CREATE TABLE %q (
			full_name TEXT,
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

	return &gitTagsTable{repoPath: m.options.RepoPath}, nil
}

func (m *GitTagsModule) Connect(c *sqlite3.SQLiteConn, args []string) (sqlite3.VTab, error) {
	return m.Create(c, args)
}

func (m *GitTagsModule) DestroyModule() {}

func (v *gitTagsTable) Open() (sqlite3.VTabCursor, error) {
	repo, err := git.OpenRepository(v.repoPath)
	if err != nil {
		return nil, err
	}
	v.repo = repo

	return &tagsCursor{repo: v.repo}, nil

}

func (v *gitTagsTable) BestIndex(cst []sqlite3.InfoConstraint, ob []sqlite3.InfoOrderBy) (*sqlite3.IndexResult, error) {
	// TODO this should actually be implemented!
	dummy := make([]bool, len(cst))
	return &sqlite3.IndexResult{Used: dummy}, nil
}

func (v *gitTagsTable) Disconnect() error {
	v.repo = nil
	return nil
}
func (v *gitTagsTable) Destroy() error { return nil }

type currentTag struct {
	name string
	id   *git.Oid
}

type tagsCursor struct {
	repo  *git.Repository
	index int
	tags  []*currentTag
}

func (vc *tagsCursor) Column(c *sqlite3.SQLiteContext, col int) error {
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

func (vc *tagsCursor) Filter(idxNum int, idxStr string, vals []interface{}) error {
	tags := make([]*currentTag, 0)
	err := vc.repo.Tags.Foreach(func(name string, id *git.Oid) error {
		tags = append(tags, &currentTag{name, id})
		return nil
	})
	if err != nil {
		return err
	}

	vc.tags = tags

	return nil
}

func (vc *tagsCursor) Next() error {
	vc.index++
	return nil
}

func (vc *tagsCursor) EOF() bool {
	return vc.index >= len(vc.tags)
}

func (vc *tagsCursor) Rowid() (int64, error) {
	return int64(0), nil
}

func (vc *tagsCursor) Close() error {
	return nil
}
