package git

import (
	"context"
	"fmt"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/storage/filesystem"
	"github.com/mergestat/mergestat/extensions/internal/git/utils"
	"github.com/pkg/errors"
	"go.riyazali.net/sqlite"
)

// CloneFn is essentially a no-op
type CloneFn struct {
	Options *utils.ModuleOptions
}

func NewCloneFn(opt *utils.ModuleOptions) *CloneFn {
	return &CloneFn{Options: opt}
}

func (*CloneFn) Deterministic() bool { return false }
func (*CloneFn) Args() int           { return 1 }
func (fn *CloneFn) Apply(c *sqlite.Context, values ...sqlite.Value) {
	path := values[0].Text()

	var err error
	if path == "" {
		path, err = utils.GetDefaultRepoFromCtx(fn.Options.Context)
		if err != nil {
			c.ResultError(err)
			return
		}
	}

	var repo *git.Repository
	if repo, err = fn.Options.Locator.Open(context.Background(), path); err != nil {
		c.ResultError(errors.Wrapf(err, "failed to open %q", path))
		return
	}

	fsStorer, ok := repo.Storer.(*filesystem.Storage)
	if !ok {
		c.ResultError(fmt.Errorf("clone scalar function can only open filesystem backed git repos"))
	}

	c.ResultText(fsStorer.Filesystem().Root())
}
