package gitqlite

import (
	"io"

	git "github.com/libgit2/git2go/v31"
)

type treeEntryWithPath struct {
	*git.TreeEntry
	path   string
	treeID string
}

type commitFile struct {
	*git.Blob
	*treeEntryWithPath
	commitID string
}

type commitFileIter struct {
	repo                  *git.Repository
	commitIter            *git.RevWalk
	currentCommit         *git.Commit
	treeEntries           []*treeEntryWithPath
	currentTreeEntryIndex int
}

type commitFileIterOptions struct {
	commitID string
}

func newCommitFileIter(repo *git.Repository, opt *commitFileIterOptions) (*commitFileIter, error) {
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

		return &commitFileIter{
			repo:                  repo,
			commitIter:            revWalk,
			currentCommit:         nil,
			treeEntries:           make([]*treeEntryWithPath, 10),
			currentTreeEntryIndex: 100, // init with an index greater than above array, so that the first call to Next() sets up the first commit, rather than trying to return a current Blob
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

		tree, err := commit.Tree()
		if err != nil {
			return nil, err
		}
		defer tree.Free()

		treeEntries := make([]*treeEntryWithPath, 0)
		treeID := tree.Id().String()
		err = tree.Walk(func(path string, treeEntry *git.TreeEntry) int {
			if treeEntry.Type == git.ObjectBlob {
				treeEntries = append(treeEntries, &treeEntryWithPath{treeEntry, path, treeID})
			}
			return 0
		})
		if err != nil {
			return nil, err
		}

		return &commitFileIter{
			repo:                  repo,
			commitIter:            nil,
			currentCommit:         commit,
			treeEntries:           treeEntries,
			currentTreeEntryIndex: 0,
		}, nil
	}
}

func (iter *commitFileIter) Next() (*commitFile, error) {
	defer func() {
		iter.currentTreeEntryIndex++
	}()

	if iter.currentTreeEntryIndex < len(iter.treeEntries) {

		f := iter.treeEntries[iter.currentTreeEntryIndex]
		blob, err := iter.repo.LookupBlob(f.Id)
		if err != nil {
			return nil, err
		}

		return &commitFile{blob, f, iter.currentCommit.Id().String()}, nil
	}

	// if the commitIter is nil, there are no commits to iterate over, end
	// this assumes that a currentCommit was set when this was first called, with treeEntries already populated
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

	tree, err := commit.Tree()
	if err != nil {
		return nil, err
	}
	defer tree.Free()

	iter.treeEntries = make([]*treeEntryWithPath, 0)
	iter.currentTreeEntryIndex = 0
	treeID := tree.Id().String()
	err = tree.Walk(func(path string, treeEntry *git.TreeEntry) int {
		if treeEntry.Type == git.ObjectBlob {
			iter.treeEntries = append(iter.treeEntries, &treeEntryWithPath{treeEntry, path, treeID})
		}
		return 0
	})
	if err != nil {
		return nil, err
	}

	f := iter.treeEntries[iter.currentTreeEntryIndex]
	blob, err := iter.repo.LookupBlob(f.Id)
	if err != nil {
		return nil, err
	}

	return &commitFile{blob, f, iter.currentCommit.Id().String()}, nil
}

func (iter *commitFileIter) Close() {
	if iter == nil {
		return
	}
	iter.currentCommit.Free()
	if iter.commitIter != nil {
		iter.commitIter.Free()
	}
}
