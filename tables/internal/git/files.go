package git

import (
	"context"
	"fmt"
	"io"
	"os"
	"path"

	"github.com/askgitdev/askgit/tables/services"
	"github.com/augmentable-dev/vtab"
	"github.com/go-git/go-git/v5/storage/filesystem"
	libgit2 "github.com/libgit2/git2go/v31"
	"go.riyazali.net/sqlite"
)

// NewFilesModule returns the implementation of a table-valued-function for accessing the content of files in git
func NewFilesModule(locator services.RepoLocator, ctx services.Context) sqlite.Module {
	return vtab.NewTableFunc("files", filesCols, func(constraints []*vtab.Constraint, order []*sqlite.OrderBy) (vtab.Iterator, error) {
		var repoPath, ref string
		for _, constraint := range constraints {
			if constraint.Op == sqlite.INDEX_CONSTRAINT_EQ {
				switch constraint.ColIndex {
				case 3:
					repoPath = constraint.Value.Text()
				case 4:
					ref = constraint.Value.Text()
				}
			}
		}

		if repoPath == "" {
			var err error
			repoPath, err = getDefaultRepoFromCtx(ctx)
			if err != nil {
				return nil, err
			}
		}

		return newFilesIter(locator, repoPath, ref)
	})
}

func newFilesIter(locator services.RepoLocator, repoPath, ref string) (*filesIter, error) {
	iter := &filesIter{
		repoPath: repoPath,
		ref:      ref,
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
	// if no ref is supplied, use HEAD
	if ref == "" {
		head, err := repo.Head()
		if err != nil {
			return nil, err
		}
		commitID = head.Target()
	} else {
		commitID, err = libgit2.NewOid(ref)
		if err != nil {
			return nil, fmt.Errorf("invalid ref: %v", err)
		}
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

var filesCols = []vtab.Column{
	{Name: "path", Type: sqlite.SQLITE_TEXT, NotNull: false, Hidden: false, Filters: nil, OrderBy: vtab.NONE},
	{Name: "executable", Type: sqlite.SQLITE_INTEGER, NotNull: false, Hidden: false, Filters: nil, OrderBy: vtab.NONE},
	{Name: "contents", Type: sqlite.SQLITE_BLOB, NotNull: false, Hidden: false, Filters: nil, OrderBy: vtab.NONE},

	{Name: "repository", Type: sqlite.SQLITE_TEXT, NotNull: true, Hidden: true, Filters: []*vtab.ColumnFilter{{Op: sqlite.INDEX_CONSTRAINT_EQ, OmitCheck: true}}, OrderBy: vtab.NONE},
	{Name: "ref", Type: sqlite.SQLITE_TEXT, NotNull: true, Hidden: true, Filters: []*vtab.ColumnFilter{{Op: sqlite.INDEX_CONSTRAINT_EQ, Required: true, OmitCheck: true}}, OrderBy: vtab.NONE},
}

type file struct {
	id         *libgit2.Oid
	path       string
	executable bool
}

type filesIter struct {
	repoPath string
	ref      string
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
