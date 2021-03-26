package git

import (
	"fmt"
	"go.riyazali.net/sqlite"

	git "github.com/libgit2/git2go/v31"
)

// TagsModule implements sqlite.Module that provides "tags" table
type TagsModule struct{}

func (m *TagsModule) Connect(_ *sqlite.Conn, args []string, declare func(string) error) (sqlite.VirtualTable, error) {
	// TODO(@riyaz): parse args to extract repo
	var repo = "."

	var schema = fmt.Sprintf(`CREATE TABLE %q (full_name TEXT, name TEXT, lightweight BOOL, target TEXT, 
			tagger_name TEXT, tagger_email TEXT, message TEXT, target_type TEXT)`, args[0])
	return &gitTagsTable{repoPath: repo}, declare(schema)
}

type gitTagsTable struct{ repoPath string }

func (v *gitTagsTable) BestIndex(_ *sqlite.IndexInfoInput) (*sqlite.IndexInfoOutput, error) {
	// TODO this should actually be implemented!
	return &sqlite.IndexInfoOutput{}, nil
}

func (v *gitTagsTable) Open() (sqlite.VirtualCursor, error) {
	repo, err := git.OpenRepository(v.repoPath)
	if err != nil {
		return nil, err
	}

	return &tagsCursor{repo: repo}, nil
}

func (v *gitTagsTable) Disconnect() error { return nil }
func (v *gitTagsTable) Destroy() error    { return nil }

type currentTag struct {
	name string
	id   *git.Oid
}

type tagsCursor struct {
	repo  *git.Repository
	index int
	tags  []*currentTag
}

func (vc *tagsCursor) Column(c *sqlite.Context, col int) error {
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
		c.ResultInt(boolToInt(isLightweight))
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

func (vc *tagsCursor) Filter(_ int, _ string, _ ...sqlite.Value) error {
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

func (vc *tagsCursor) Next() error           { vc.index++; return nil }
func (vc *tagsCursor) Eof() bool             { return vc.index >= len(vc.tags) }
func (vc *tagsCursor) Rowid() (int64, error) { return int64(0), nil }
func (vc *tagsCursor) Close() error          { return nil }
