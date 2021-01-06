package gitqlite

import (
	"bytes"
	"io"

	git "github.com/libgit2/git2go/v31"
)

type BlameIterator struct {
	repo                *git.Repository
	current             *git.Blame
	currentFileContents [][]byte
	file                *commitFile
	iterator            *commitFileIter
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
	file, err := iterator.Next()
	if err != nil {
		return nil, err
	}
	blame, err := repo.BlameFile(file.path+file.Name, &opts)
	if err != nil {
		return nil, err
	}
	// get the current file to extract its contents
	currentFile, err := file.AsBlob()
	if err != nil {
		return nil, err
	}
	// break up the file by newlines as one would see in git blame output
	str := bytes.Split(currentFile.Contents(), []byte{'\n'})
	return &BlameIterator{
		repo,
		blame,
		str,
		file,
		iterator,
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
		file, err := vc.iterator.Next()
		if err != nil {
			if err == io.EOF {
				vc.current = nil
				return nil
			}
			return err
		}
		opts, err := git.DefaultBlameOptions()
		if err != nil {
			return nil
		}
		blame, err := vc.repo.BlameFile(file.path+file.Name, &opts)
		if err != nil {
			return err
		}
		// get the current file to extract its contents
		currentFile, err := file.AsBlob()
		if err != nil {
			return err
		}
		// create string array to display as in filter
		str := bytes.Split(currentFile.Contents(), []byte{'\n'})
		vc.currentFileContents = str
		vc.current = blame
		vc.file = file
		vc.lineIter = 1
	}

	return nil
}
