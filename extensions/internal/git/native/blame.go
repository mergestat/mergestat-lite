package native

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/askgitdev/askgit/extensions/internal/git/utils"
	"github.com/askgitdev/askgit/extensions/services"
	"github.com/augmentable-dev/vtab"
	"github.com/go-git/go-git/v5/storage/filesystem"
	libgit2 "github.com/libgit2/git2go/v31"
	"github.com/pkg/errors"
	"go.riyazali.net/sqlite"
)

var blameCols = []vtab.Column{
	{Name: "line_no", Type: "TEXT", NotNull: false, Hidden: false, Filters: nil, OrderBy: vtab.NONE},
	{Name: "commit_hash", Type: "TEXT", NotNull: false, Hidden: false, Filters: nil, OrderBy: vtab.NONE},

	{Name: "repository", Type: "TEXT", NotNull: true, Hidden: true, Filters: []*vtab.ColumnFilter{{Op: sqlite.INDEX_CONSTRAINT_EQ, OmitCheck: true}}, OrderBy: vtab.NONE},
	{Name: "rev", Type: "TEXT", NotNull: true, Hidden: true, Filters: []*vtab.ColumnFilter{{Op: sqlite.INDEX_CONSTRAINT_EQ, Required: true, OmitCheck: true}}, OrderBy: vtab.NONE},
	{Name: "file_path", Type: "TEXT", NotNull: true, Hidden: true, Filters: []*vtab.ColumnFilter{{Op: sqlite.INDEX_CONSTRAINT_EQ, Required: true, OmitCheck: true}}, OrderBy: vtab.NONE},
}

// NewBlameModule returns the implementation of a table-valued-function for accessing git blame
func NewBlameModule(locator services.RepoLocator, ctx services.Context) sqlite.Module {
	return vtab.NewTableFunc("blame", blameCols, func(constraints []*vtab.Constraint, order []*sqlite.OrderBy) (vtab.Iterator, error) {
		var repoPath, rev, filePath string
		for _, constraint := range constraints {
			if constraint.Op == sqlite.INDEX_CONSTRAINT_EQ {
				switch constraint.ColIndex {
				case 2:
					repoPath = constraint.Value.Text()
				case 3:
					rev = constraint.Value.Text()
				case 4:
					filePath = constraint.Value.Text()
				}
			}
		}

		if filePath == "" {
			return nil, fmt.Errorf("blame table requires a file path")
		}

		if repoPath == "" {
			var err error
			repoPath, err = utils.GetDefaultRepoFromCtx(ctx)
			if err != nil {
				return nil, err
			}
		}

		return newBlameIter(locator, repoPath, rev, filePath)
	})
}

func newBlameIter(locator services.RepoLocator, repoPath, rev, filePath string) (*blameIter, error) {
	iter := &blameIter{
		repoPath: repoPath,
		rev:      rev,
		filePath: filePath,
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
		return nil, fmt.Errorf("blame table only supported on filesystem backed git repos")
	}

	repo, err := libgit2.OpenRepository(fsStorer.Filesystem().Root())
	if err != nil {
		return nil, err
	}
	defer repo.Free()

	var commitID *libgit2.Oid
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

	opts, err := libgit2.DefaultBlameOptions()
	if err != nil {
		return nil, err
	}

	opts.NewestCommit = commitID

	blame, err := repo.BlameFile(filePath, &opts)
	if err != nil {
		return nil, err
	}
	defer func() {
		err := blame.Free()
		if err != nil {
			fmt.Println(err) // TODO(patrickdevivo) figure out a better handling here
		}
	}()

	iter.lines = make([]*blamedLine, 0)
	fileLine := 1
	for {
		hunk, err := blame.HunkByLine(fileLine)
		if err != nil {
			if errors.Is(err, libgit2.ErrInvalid) {
				break
			}
			return nil, err
		}
		iter.lines = append(iter.lines, &blamedLine{
			hunk:   &hunk,
			lineNo: fileLine,
		})
		fileLine++
	}

	return iter, nil
}

type blamedLine struct {
	lineNo int
	hunk   *libgit2.BlameHunk
}

type blameIter struct {
	repoPath string
	rev      string
	filePath string
	lines    []*blamedLine
	index    int
}

func (i *blameIter) Column(ctx *sqlite.Context, c int) error {
	currentLine := i.lines[i.index]
	switch c {
	case 0:
		ctx.ResultInt(currentLine.lineNo)
	case 1:
		ctx.ResultText(currentLine.hunk.OrigCommitId.String())
	}
	return nil
}

func (i *blameIter) Next() (vtab.Row, error) {
	i.index++
	if i.index >= len(i.lines) {
		return nil, io.EOF
	}
	return i, nil
}
