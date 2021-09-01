package native

import (
	"context"
	"fmt"
	"io"
	"os"
	"path"

	"github.com/askgitdev/askgit/extensions/internal/git/utils"
	"github.com/askgitdev/askgit/extensions/services"
	"github.com/augmentable-dev/vtab"
	"github.com/go-git/go-git/v5/storage/filesystem"
	libgit2 "github.com/libgit2/git2go/v31"
	"go.riyazali.net/sqlite"
)

var filesCols = []vtab.Column{
	{Name: "path", Type: "TEXT", NotNull: false, Hidden: false, Filters: nil, OrderBy: vtab.NONE},
	{Name: "executable", Type: "INT", NotNull: false, Hidden: false, Filters: nil, OrderBy: vtab.NONE},
	{Name: "contents", Type: "BLOB", NotNull: false, Hidden: false, Filters: nil, OrderBy: vtab.NONE},

	{Name: "repository", Type: "TEXT", NotNull: true, Hidden: true, Filters: []*vtab.ColumnFilter{{Op: sqlite.INDEX_CONSTRAINT_EQ, OmitCheck: true}}, OrderBy: vtab.NONE},
	{Name: "rev", Type: "TEXT", NotNull: true, Hidden: true, Filters: []*vtab.ColumnFilter{{Op: sqlite.INDEX_CONSTRAINT_EQ, Required: true, OmitCheck: true}}, OrderBy: vtab.NONE},
}

// NewFilesModule returns the implementation of a table-valued-function for accessing the content of files in git
func NewFilesModule(options *utils.ModuleOptions) sqlite.Module {
	return vtab.NewTableFunc("files", filesCols, func(constraints []*vtab.Constraint, order []*sqlite.OrderBy) (vtab.Iterator, error) {
		var repoPath, rev string
		for _, constraint := range constraints {
			if constraint.Op == sqlite.INDEX_CONSTRAINT_EQ {
				switch constraint.ColIndex {
				case 3:
					repoPath = constraint.Value.Text()
				case 4:
					rev = constraint.Value.Text()
				}
			}
		}

		if repoPath == "" {
			var err error
			repoPath, err = utils.GetDefaultRepoFromCtx(options.Context)
			if err != nil {
				return nil, err
			}
		}

		return newFilesIter(options.Locator, repoPath, rev)
	})
}

func newFilesIter(locator services.RepoLocator, repoPath, rev string) (*filesIter, error) {
	iter := &filesIter{
		repoPath: repoPath,
		rev:      rev,
		index:    -1,
	}

	if repoPath == "" {
		if wd, err := os.Getwd(); err != nil {
			return nil, err
		} else {
			repoPath = wd
		}
	}

	r, err := locator.Open(context.Background(), repoPath)
	if err != nil {
		return nil, err
	}

	fsStorer, ok := r.Storer.(*filesystem.Storage)
	if !ok {
		return nil, fmt.Errorf("file table only supported on filesystem backed git repos")
	}

	repo, err := libgit2.OpenRepository(fsStorer.Filesystem().Root())
	if err != nil {
		return nil, err
	}
	iter.repo = repo

	var commitID *libgit2.Oid
	var commit *libgit2.Commit
	// if no rev is supplied, use HEAD
	if rev == "" {
		head, err := repo.Head()
		if err != nil {
			return nil, err
		}
		commitID = head.Target()
	} else {
		obj, err := repo.RevparseSingle(rev)
		if err != nil {
			return nil, err
		}
		defer obj.Free()

		if obj.Type() != libgit2.ObjectCommit {
			return nil, fmt.Errorf("invalid rev, could not resolve to a commit")
		}

		commitID = obj.Id()
	}
	commit, err = repo.LookupCommit(commitID)
	if err != nil {
		return nil, err
	}
	defer commit.Free()

	tree, err := commit.Tree()
	if err != nil {
		return nil, err
	}

	iter.files = make([]*file, 0, tree.EntryCount())
	err = tree.Walk(func(p string, treeEntry *libgit2.TreeEntry) int {
		if treeEntry.Type != libgit2.ObjectBlob {
			return 0
		}
		iter.files = append(iter.files, &file{
			id:         treeEntry.Id,
			path:       path.Join(p, treeEntry.Name),
			executable: treeEntry.Filemode == libgit2.FilemodeBlobExecutable,
		})
		return 0
	})
	if err != nil {
		return nil, err
	}

	return iter, nil
}

type file struct {
	id         *libgit2.Oid
	path       string
	executable bool
}

type filesIter struct {
	repoPath string
	rev      string
	files    []*file
	index    int
	repo     *libgit2.Repository
}

func (i *filesIter) Column(ctx *sqlite.Context, c int) error {
	currentFile := i.files[i.index]
	switch c {
	case 0:
		ctx.ResultText(currentFile.path)
	case 1:
		if currentFile.executable {
			ctx.ResultInt(1)
		} else {
			ctx.ResultInt(0)
		}
	case 2:
		blob, err := i.repo.LookupBlob(currentFile.id)
		if err != nil {
			return err
		}
		defer blob.Free()
		ctx.ResultText(string(blob.Contents()))
	}
	return nil
}

func (i *filesIter) Next() (vtab.Row, error) {
	i.index++
	if i.index >= len(i.files) {
		if i.repo != nil {
			i.repo.Free()
		}
		return nil, io.EOF
	}
	return i, nil
}
