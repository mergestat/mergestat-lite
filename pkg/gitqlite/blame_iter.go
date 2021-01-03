package gitqlite

import (
	"bytes"

	git "github.com/libgit2/git2go/v31"
)

type BlameIterator struct {
	repo                *git.Repository
	current             *git.Blame
	currentFileContents [][]byte
	iterator            *commitFileIter
	fileIndex           int
	lineIter            int
}

func NewBlameIterator(repo *git.Repository) (*BlameIterator, error) {
	opts, err := git.DefaultBlameOptions()
	if err != nil {
		return nil, err
	}

	head, err := repo.Head()
	if err != nil {
		return nil, err
	}
	defer head.Free()
	// get a new iterator from repo and use the head commit
	iterator, err := NewCommitFileIter(repo, &commitFileIterOptions{head.Target().String()})
	// get the blame by adding the directory (.path) and the filename (.name)
	if err != nil {
		return nil, err
	}
	blame, err := repo.BlameFile(iterator.treeEntries[0].path+iterator.treeEntries[0].Name, &opts)
	if err != nil {
		return nil, err
	}
	// get the current file to extract its contents
	currentFile, err := repo.LookupBlob(iterator.treeEntries[0].Id)
	if err != nil {
		return nil, err
	}
	// break up the file by newlines as one would see in git blame output
	str := bytes.Split(currentFile.Contents(), []byte{'\n'})
	return &BlameIterator{
		repo,
		blame,
		str,
		iterator,
		0,
		1,
	}, nil
}
func (vc *BlameIterator) Next() error {
	// increment the lineIterator count to the next line
	vc.lineIter++
	// check if the lineIter has overrun the file
	_, err := vc.current.HunkByLine(vc.lineIter)
	// if there is an error then go to the next file
	if err != nil {
		vc.fileIndex++
		if vc.fileIndex < len(vc.iterator.treeEntries) {
			opts, err := git.DefaultBlameOptions()
			if err != nil {
				return err
			}
			// look up blamefile as in filter
			blame, err := vc.repo.BlameFile(vc.iterator.treeEntries[vc.fileIndex].path+vc.iterator.treeEntries[vc.fileIndex].Name, &opts)
			if err != nil {
				return err
			}
			// look up blob as in filter
			currentFile, err := vc.repo.LookupBlob(vc.iterator.treeEntries[vc.fileIndex].Id)
			if err != nil {
				return err
			}
			// create string array to display as in filter
			str := bytes.Split(currentFile.Contents(), []byte{'\n'})
			vc.currentFileContents = str
			vc.current = blame
			vc.lineIter = 1
		} else {
			vc.current = nil
			return nil
		}

	}
	return nil
}
