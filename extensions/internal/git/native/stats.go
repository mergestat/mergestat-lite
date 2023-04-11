package native

import (
	"context"
	"fmt"
	"io"

	"github.com/augmentable-dev/vtab"
	"github.com/go-git/go-git/v5/storage/filesystem"
	libgit2 "github.com/libgit2/git2go/v34"
	"github.com/mergestat/mergestat-lite/extensions/internal/git/utils"
	"go.riyazali.net/sqlite"
)

var statsCols = []vtab.Column{
	{Name: "file_path", Type: "TEXT", NotNull: false, Hidden: false, Filters: nil, OrderBy: vtab.NONE},
	{Name: "additions", Type: "INT", NotNull: false, Hidden: false, Filters: nil, OrderBy: vtab.NONE},
	{Name: "deletions", Type: "INT", NotNull: false, Hidden: false, Filters: nil, OrderBy: vtab.NONE},

	{Name: "old_file_mode", Type: "TEXT", NotNull: true, Hidden: false, Filters: nil, OrderBy: vtab.NONE},
	{Name: "new_file_mode", Type: "TEXT", NotNull: true, Hidden: false, Filters: nil, OrderBy: vtab.NONE},

	{Name: "repository", Type: "TEXT", NotNull: true, Hidden: true, Filters: []*vtab.ColumnFilter{{Op: sqlite.INDEX_CONSTRAINT_EQ, OmitCheck: true}}, OrderBy: vtab.NONE},
	{Name: "rev", Type: "TEXT", NotNull: true, Hidden: true, Filters: []*vtab.ColumnFilter{{Op: sqlite.INDEX_CONSTRAINT_EQ, OmitCheck: true}}, OrderBy: vtab.NONE},
	{Name: "to_rev", Type: "TEXT", NotNull: true, Hidden: true, Filters: []*vtab.ColumnFilter{{Op: sqlite.INDEX_CONSTRAINT_EQ, OmitCheck: true}}, OrderBy: vtab.NONE},
}

type GitFileModeObjectType string

const (
	GitFileModeObjectTypeUnknown     GitFileModeObjectType = "unknown"
	GitFileModeObjectTypeNone        GitFileModeObjectType = "none"
	GitFileModeObjectTypeRegularFile GitFileModeObjectType = "regular_file"
	GitFileModeOjectTypeSymbolicLink GitFileModeObjectType = "symbolic_link"
	GitFileModeOjectTypeGitLink      GitFileModeObjectType = "git_link"
)

// gitFileModeObjectTypeFromUint16 takes a git stats file mode and returns the GitFileModeObjectType.
// See here for more info on the modes: https://unix.stackexchange.com/questions/450480/file-permission-with-six-bytes-in-git-what-does-it-mean
func gitFileModeObjectTypeFromUint16(mode uint16) GitFileModeObjectType {
	switch mode >> 12 {
	case 0:
		return GitFileModeObjectTypeNone
	case 0b1110:
		return GitFileModeOjectTypeGitLink
	case 0b1010:
		return GitFileModeOjectTypeSymbolicLink
	case 0b1000:
		return GitFileModeObjectTypeRegularFile
	default:
		return GitFileModeObjectTypeUnknown
	}
}

// NewStatsModule returns the implementation of a table-valued-function for git stats
func NewStatsModule(options *utils.ModuleOptions) sqlite.Module {
	return vtab.NewTableFunc("stats", statsCols, func(constraints []*vtab.Constraint, order []*sqlite.OrderBy) (vtab.Iterator, error) {
		var repoPath, rev, toRev string
		for _, constraint := range constraints {
			if constraint.Op == sqlite.INDEX_CONSTRAINT_EQ {
				switch statsCols[constraint.ColIndex].Name {
				case "repository":
					repoPath = constraint.Value.Text()
				case "rev":
					rev = constraint.Value.Text()
				case "to_rev":
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
			return nil, fmt.Errorf("invalid revision, could not resolve to a commit")
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
		stat := &stat{filePath: delta.NewFile.Path, oldFileMode: gitFileModeObjectTypeFromUint16(delta.OldFile.Mode), newFileMode: gitFileModeObjectTypeFromUint16(delta.NewFile.Mode)}
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
	filePath    string
	additions   int
	deletions   int
	oldFileMode GitFileModeObjectType
	newFileMode GitFileModeObjectType
}

type statsIter struct {
	repoPath string
	stats    []*stat
	index    int
}

func (i *statsIter) Column(ctx vtab.Context, c int) error {
	currentStat := i.stats[i.index]
	switch statsCols[c].Name {
	case "file_path":
		ctx.ResultText(currentStat.filePath)
	case "additions":
		ctx.ResultInt(currentStat.additions)
	case "deletions":
		ctx.ResultInt(currentStat.deletions)
	case "old_file_mode":
		ctx.ResultText(string(currentStat.oldFileMode))
	case "new_file_mode":
		ctx.ResultText(string(currentStat.newFileMode))
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
