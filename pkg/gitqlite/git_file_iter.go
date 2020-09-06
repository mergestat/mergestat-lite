package gitqlite

import (
	"io"

	git "github.com/libgit2/git2go/v30"
)

type treeEntryWithPath struct {
	*git.TreeEntry
	path string
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

func NewCommitTreeIter(repo *git.Repository) (*commitFileIter, error) {
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

	iter.treeEntries = make([]*treeEntryWithPath, 0)
	iter.currentTreeEntryIndex = 0
	err = tree.Walk(func(path string, treeEntry *git.TreeEntry) int {
		if treeEntry.Type == git.ObjectBlob {
			iter.treeEntries = append(iter.treeEntries, &treeEntryWithPath{treeEntry, path})
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
	iter.currentCommit.Free()
	iter.commitIter.Free()
}
