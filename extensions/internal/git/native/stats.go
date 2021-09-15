package native

import (
	"context"
	"fmt"
	"io"

	"github.com/askgitdev/askgit/extensions/internal/git/utils"
	"github.com/augmentable-dev/vtab"
	"github.com/go-git/go-git/v5/storage/filesystem"
	libgit2 "github.com/libgit2/git2go/v31"
	"go.riyazali.net/sqlite"
)

var statsCols = []vtab.Column{
	{Name: "file_path", Type: "TEXT", NotNull: false, Hidden: false, Filters: nil, OrderBy: vtab.NONE},
	{Name: "additions", Type: "INT", NotNull: false, Hidden: false, Filters: nil, OrderBy: vtab.NONE},
	{Name: "deletions", Type: "INT", NotNull: false, Hidden: false, Filters: nil, OrderBy: vtab.NONE},

	{Name: "repository", Type: "TEXT", NotNull: true, Hidden: true, Filters: []*vtab.ColumnFilter{{Op: sqlite.INDEX_CONSTRAINT_EQ, OmitCheck: true}}, OrderBy: vtab.NONE},
	{Name: "rev", Type: "TEXT", NotNull: true, Hidden: true, Filters: []*vtab.ColumnFilter{{Op: sqlite.INDEX_CONSTRAINT_EQ, Required: true, OmitCheck: true}}, OrderBy: vtab.NONE},
	{Name: "to_rev", Type: "TEXT", NotNull: true, Hidden: true, Filters: []*vtab.ColumnFilter{{Op: sqlite.INDEX_CONSTRAINT_EQ, OmitCheck: true}}, OrderBy: vtab.NONE},
}

// NewStatsModule returns the implementation of a table-valued-function for git stats
func NewStatsModule(options *utils.ModuleOptions) sqlite.Module {
	return vtab.NewTableFunc("stats", statsCols, func(constraints []*vtab.Constraint, order []*sqlite.OrderBy) (vtab.Iterator, error) {
		var repoPath, rev, toRev string
		for _, constraint := range constraints {
			if constraint.Op == sqlite.INDEX_CONSTRAINT_EQ {
				switch constraint.ColIndex {
				case 3:
					repoPath = constraint.Value.Text()
				case 4:
					rev = constraint.Value.Text()
				case 5:
					toRev = constraint.Value.Text()
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

		return newStatsIter(options, repoPath, rev, toRev)
	})
}

func newStatsIter(options *utils.ModuleOptions, repoPath, rev, toRev string) (*statsIter, error) {
	logger := options.Logger.With().
		Str("module", "git-stats").
		Str("repo-path", repoPath).
		Logger()
	defer func() {
		logger.Debug().Msg("creating stats iterator")
	}()

	iter := &statsIter{
		repoPath: repoPath,
		index:    -1,
	}

	r, err := options.Locator.Open(context.Background(), repoPath)
	if err != nil {
		return nil, err
	}

	fsStorer, ok := r.Storer.(*filesystem.Storage)
	if !ok {
		return nil, fmt.Errorf("stats table only supported on filesystem backed git repos")
	}

	repo, err := libgit2.OpenRepository(fsStorer.Filesystem().Root())
	if err != nil {
		return nil, err
	}
	defer repo.Free()

	var fromCommit *libgit2.Commit
	// if no rev is supplied, use HEAD
	if rev == "" {
		head, err := repo.Head()
		if err != nil {
			return nil, err
		}
		fromCommit, err = repo.LookupCommit(head.Target())
		if err != nil {
			return nil, err
		}
	} else {
		obj, err := repo.RevparseSingle(rev)
		if err != nil {
			return nil, err
		}
		defer obj.Free()

		if obj.Type() != libgit2.ObjectCommit {
			return nil, fmt.Errorf("invalid rev, could not resolve to a commit")
		}

		fromCommit, err = repo.LookupCommit(obj.Id())
		if err != nil {
			return nil, err
		}
	}
	defer fromCommit.Free()
	logger = logger.With().Str("from-revision", fromCommit.Id().String()).Logger()

	tree, err := fromCommit.Tree()
	if err != nil {
		return nil, err
	}
	defer tree.Free()

	var toCommit *libgit2.Commit
	if toRev == "" {
		toCommit = fromCommit.Parent(0)
	} else {
		id, err := libgit2.NewOid(toRev)
		if err != nil {
			return nil, fmt.Errorf("invalid to_rev: %v", err)
		}

		toCommit, err = repo.LookupCommit(id)
		if err != nil {
			return nil, err
		}
	}

	var toTree *libgit2.Tree
	if toCommit == nil {
		toTree = &libgit2.Tree{}
		logger = logger.With().Str("to-revision", "").Logger()
	} else {
		toTree, err = toCommit.Tree()
		if err != nil {
			return nil, err
		}
		defer toCommit.Free()
		logger = logger.With().Str("to-revision", toCommit.Id().String()).Logger()
	}
	defer toTree.Free()

	diffOpts, err := libgit2.DefaultDiffOptions()
	if err != nil {
		return nil, err
	}

	diff, err := repo.DiffTreeToTree(toTree, tree, &diffOpts)
	if err != nil {
		return nil, err
	}
	defer func() {
		err := diff.Free()
		if err != nil {
			// TODO what should we do here?
			fmt.Println(err)
		}
	}()

	diffFindOpts, err := libgit2.DefaultDiffFindOptions()
	if err != nil {
		return nil, err
	}

	err = diff.FindSimilar(&diffFindOpts)
	if err != nil {
		return nil, err
	}

	iter.stats = make([]*stat, 0)
	err = diff.ForEach(func(delta libgit2.DiffDelta, progress float64) (libgit2.DiffForEachHunkCallback, error) {
		stat := &stat{filePath: delta.NewFile.Path}
		iter.stats = append(iter.stats, stat)
		return func(hunk libgit2.DiffHunk) (libgit2.DiffForEachLineCallback, error) {
			return func(line libgit2.DiffLine) error {
				switch line.Origin {
				case libgit2.DiffLineAddition:
					stat.additions++
				case libgit2.DiffLineDeletion:
					stat.deletions++
				}
				return nil
			}, nil
		}, nil
	}, libgit2.DiffDetailLines)
	if err != nil {
		return nil, err
	}

	return iter, nil
}

type stat struct {
	filePath  string
	additions int
	deletions int
}

type statsIter struct {
	repoPath string
	stats    []*stat
	index    int
}

func (i *statsIter) Column(ctx *sqlite.Context, c int) error {
	currentStat := i.stats[i.index]
	switch c {
	case 0:
		ctx.ResultText(currentStat.filePath)
	case 1:
		ctx.ResultInt(currentStat.additions)
	case 2:
		ctx.ResultInt(currentStat.deletions)
	}
	return nil
}

func (i *statsIter) Next() (vtab.Row, error) {
	i.index++
	if i.index >= len(i.stats) {
		return nil, io.EOF
	}
	return i, nil
}
