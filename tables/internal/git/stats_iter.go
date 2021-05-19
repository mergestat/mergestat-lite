package git

import (
	"io"

	git "github.com/libgit2/git2go/v31"
)

type commitStat struct {
	commitID  string
	file      string
	additions int
	deletions int
}

type commitStatsIter struct {
	repo                   *git.Repository
	commitIter             *git.RevWalk
	currentCommit          *git.Commit
	commitStats            []*commitStat
	currentCommitStatIndex int
}

type commitStatsIterOptions struct {
	commitID string
}

func stats(commit *git.Commit) ([]*commitStat, error) {

	stats := make([]*commitStat, 0)

	repo := commit.Owner()
	tree, err := commit.Tree()
	if err != nil {
		return nil, err
	}
	defer tree.Free()

	var parentTree *git.Tree
	parent := commit.Parent(0)
	if parent == nil {
		parentTree = &git.Tree{}
	} else {
		parentTree, err = parent.Tree()
		if err != nil {
			return nil, err
		}
		defer parentTree.Free()
	}

	diffOpts, err := git.DefaultDiffOptions()
	if err != nil {
		return nil, err
	}
	diff, err := repo.DiffTreeToTree(parentTree, tree, &diffOpts)
	if err != nil {
		return nil, err
	}
	diffFindOpts, err := git.DefaultDiffFindOptions()
	if err != nil {
		return nil, err
	}
	err = diff.FindSimilar(&diffFindOpts)
	if err != nil {
		return nil, err
	}

	err = diff.ForEach(func(delta git.DiffDelta, progress float64) (git.DiffForEachHunkCallback, error) {
		stat := &commitStat{
			commitID: commit.Id().String(),
			file:     delta.NewFile.Path,
		}
		stats = append(stats, stat)
		return func(hunk git.DiffHunk) (git.DiffForEachLineCallback, error) {
			return func(line git.DiffLine) error {
				switch line.Origin {
				case git.DiffLineAddition:
					stat.additions++
				case git.DiffLineDeletion:
					stat.deletions++
				}
				return nil
			}, nil
		}, nil
	}, git.DiffDetailLines)

	if err != nil {
		return nil, err
	}
	return stats, nil
}

func NewCommitStatsIter(repo *git.Repository, opt *commitStatsIterOptions) (*commitStatsIter, error) {
	if opt.commitID == "" {
		revWalk, err := repo.Walk()
		if err != nil {
			return nil, err
		}

		err = revWalk.PushHead()
		if err != nil {
			return nil, err
		}

		revWalk.Sorting(git.SortNone)

		return &commitStatsIter{
			repo:                   repo,
			commitIter:             revWalk,
			currentCommit:          nil,
			commitStats:            make([]*commitStat, 0),
			currentCommitStatIndex: 100, // init with an index greater than above array, so that the first call to Next() sets up the first commit, rather than trying to return a current Blob
		}, nil

	} else {
		commitID, err := git.NewOid(opt.commitID)
		if err != nil {
			return nil, err
		}

		commit, err := repo.LookupCommit(commitID)
		if err != nil {
			return nil, err
		}

		commitStats, err := stats(commit)
		if err != nil {
			return nil, err
		}

		return &commitStatsIter{
			repo:                   repo,
			commitIter:             nil,
			currentCommit:          commit,
			commitStats:            commitStats,
			currentCommitStatIndex: 0,
		}, nil
	}
}

func (iter *commitStatsIter) Next() (*commitStat, error) {
	defer func() {
		iter.currentCommitStatIndex++
	}()

	if iter.currentCommitStatIndex < len(iter.commitStats) {
		return iter.commitStats[iter.currentCommitStatIndex], nil
	}

	// if the commitIter is nil, there are no commits to iterate over, end
	// this assumes that a currentCommit was set when this was first called, with commitStats already populated
	if iter.commitIter == nil {
		return nil, io.EOF
	}

	id := new(git.Oid)
	err := iter.commitIter.Next(id)
	if err != nil {
		if id.IsZero() {
			return nil, io.EOF
		}

		return nil, err
	}

	commit, err := iter.repo.LookupCommit(id)
	if err != nil {
		return nil, err
	}

	iter.currentCommit = commit

	commitStats, err := stats(commit)
	if err != nil {
		return nil, err
	}

	iter.commitStats = commitStats
	iter.currentCommitStatIndex = 0

	if len(commitStats) == 0 {
		return iter.Next()
	}

	return commitStats[iter.currentCommitStatIndex], nil
}

func (iter *commitStatsIter) Close() {
	if iter == nil {
		return
	}
	iter.currentCommit.Free()
	if iter.commitIter != nil {
		iter.commitIter.Free()
	}
}
