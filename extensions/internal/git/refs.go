package git

import (
	"context"
	"regexp"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/storer"
	"github.com/mergestat/mergestat/extensions/internal/git/utils"
	"github.com/pkg/errors"
	"go.riyazali.net/sqlite"
)

var remoteName = regexp.MustCompile(`(?m)refs\/remotes\/([^\/]*)\/.+`)

// NewRefModule returns a new virtual table for listing git refs
func NewRefModule(opt *utils.ModuleOptions) sqlite.Module {
	return &refModule{opt}
}

type refModule struct {
	*utils.ModuleOptions
}

func (mod *refModule) Connect(_ *sqlite.Conn, _ []string, declare func(string) error) (sqlite.VirtualTable, error) {
	const schema = `
		CREATE TABLE refs (
			name		TEXT,
			type		TEXT,
			remote		TEXT,
			full_name	TEXT,
			hash		TEXT,
			target		TEXT,
			
			repository	HIDDEN,
			tag			HIDDEN,
			PRIMARY KEY ( name )
		) WITHOUT ROWID`

	return &gitRefTable{ModuleOptions: mod.ModuleOptions}, declare(schema)
}

type gitRefTable struct {
	*utils.ModuleOptions
}

func (tab *gitRefTable) Disconnect() error { return nil }
func (tab *gitRefTable) Destroy() error    { return nil }
func (tab *gitRefTable) Open() (sqlite.VirtualCursor, error) {
	return &gitRefCursor{ModuleOptions: tab.ModuleOptions}, nil
}

func (tab *gitRefTable) BestIndex(input *sqlite.IndexInfoInput) (*sqlite.IndexInfoOutput, error) {
	var bitmap []byte
	var out = &sqlite.IndexInfoOutput{}
	out.ConstraintUsage = make([]*sqlite.ConstraintUsage, len(input.Constraints))

	for i, constraint := range input.Constraints {
		// if repository is provided, it must be usable
		if constraint.ColumnIndex == 6 && !constraint.Usable {
			return nil, sqlite.SQLITE_CONSTRAINT
		}

		if !constraint.Usable {
			continue // we do not support unusable constraint at all
		}

		if constraint.ColumnIndex == 6 && constraint.Op == sqlite.INDEX_CONSTRAINT_EQ {
			bitmap = append(bitmap, byte(1<<4|constraint.ColumnIndex))
			out.ConstraintUsage[i] = &sqlite.ConstraintUsage{ArgvIndex: 1, Omit: true}
		}
	}

	// validate passed in constraint to ensure there combination stays logical
	out.IndexString = enc(bitmap)
	return out, nil
}

type gitRefCursor struct {
	*utils.ModuleOptions

	repo *git.Repository

	ref  *plumbing.Reference
	refs storer.ReferenceIter
}

func (cur *gitRefCursor) Filter(_ int, s string, values ...sqlite.Value) (err error) {
	logger := cur.Logger.With().Str("module", "git-ref").Logger()
	defer func() {
		logger.Debug().Msg("running git refs filter")
	}()

	// values extracted from constraints
	var path string

	var bitmap, _ = dec(s)
	for i, val := range values {
		switch b := bitmap[i]; b {
		case 0b00010110:
			path = val.Text()
		}
	}

	var repo *git.Repository
	{ // open the git repository
		if path == "" {
			path, err = utils.GetDefaultRepoFromCtx(cur.Context)
			if err != nil {
				return err
			}
		}

		if repo, err = cur.Locator.Open(context.Background(), path); err != nil {
			return errors.Wrapf(err, "failed to open %q", path)
		}
		cur.repo = repo
		logger = logger.With().Str("repo-disk-path", path).Logger()
	}

	if cur.refs, err = repo.References(); err != nil {
		return errors.Wrap(err, "failed to create iterator")
	}

	return cur.Next()
}

func (cur *gitRefCursor) Column(c *sqlite.Context, col int) error {
	ref := cur.ref
	switch col {
	case 0:
		c.ResultText(ref.Name().Short())
	case 1:
		if ref.Name().IsBranch() || isRemoteBranch(ref.Name()) {
			c.ResultText("branch")
		} else if ref.Name().IsTag() {
			c.ResultText("tag")
		} else if ref.Name().IsNote() {
			c.ResultText("note")
		}
	case 2:
		if ref.Name().IsRemote() {
			if matches := remoteName.FindStringSubmatch(ref.Name().String()); len(matches) >= 2 {
				c.ResultText(matches[1])
			}
		}
	case 3:
		c.ResultText(ref.Name().String())
	case 4:
		// tags come in two flavors, lightweight and annotated
		// a lighteight tag points *directly* to a commit SHA
		// vs an annotated tag which points to an object that contains metadata
		// https://git-scm.com/book/en/v2/Git-Basics-Tagging
		if ref.Name().IsTag() {
			if tag, err := cur.repo.TagObject(ref.Hash()); err != nil && err != plumbing.ErrObjectNotFound {
				return errors.Wrap(err, "failed to fetch tag object")
			} else if tag != nil && tag.TargetType == plumbing.CommitObject {
				// if the hash of the current ref is a tag object
				// output the SHA of its target
				c.ResultText(tag.Target.String())
			} else {
				// otherwise, output the hash of the ref
				c.ResultText(ref.Hash().String())
			}
		}
	case 5:
		c.ResultText(ref.Target().String())
	case 7:
		if ref.Name().IsTag() {
			if tag, err := cur.repo.TagObject(ref.Hash()); err != nil && err != plumbing.ErrObjectNotFound {
				return errors.Wrap(err, "failed to fetch tag object")
			} else if tag != nil && tag.TargetType == plumbing.CommitObject {
				c.ResultPointer(tag)
			}
		}
	}

	return nil
}

func (cur *gitRefCursor) Next() (err error) {
	if cur.ref, err = cur.refs.Next(); err != nil {
		if !eof(err) {
			return err
		}
	}
	return nil
}

func (cur *gitRefCursor) Eof() bool             { return cur.ref == nil }
func (cur *gitRefCursor) Rowid() (int64, error) { return int64(0), nil }
func (cur *gitRefCursor) Close() error {
	if cur.refs != nil {
		cur.refs.Close()
	}
	return nil
}
