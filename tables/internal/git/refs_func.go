package git

import (
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/pkg/errors"
	"go.riyazali.net/sqlite"
)

// CommitFromTagFn implements the COMMIT_FROM_TAG(...) sql function
type CommitFromTagFn struct{}

func (_ *CommitFromTagFn) Deterministic() bool { return true }
func (_ *CommitFromTagFn) Args() int           { return 1 }
func (_ *CommitFromTagFn) Apply(c *sqlite.Context, values ...sqlite.Value) {
	tag, ok := values[0].Pointer().(*object.Tag)
	if !ok {
		return
	}

	commit, err := tag.Commit()
	if err != nil {
		c.ResultError(errors.Wrap(err, "tag target is not a commit"))
		return
	}

	c.ResultText(commit.Hash.String())
}
