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
	iterator            *commitFileIter
	file                *commitFile
	lineIter            int
}
type BlameHunk struct {
	lineNo       int
	lineContents []byte
	fileName     string
	commitID     string
}

func NewBlameIterator(repo *git.Repository) (*BlameIterator, *BlameHunk, error) {
	opts, err := git.DefaultBlameOptions()
	if err != nil {
		return nil, nil, err
	}

	head, err := repo.Head()
	if err != nil {
		return nil, nil, err
	}
	defer head.Free()
	// get a new iterator from repo and use the head commit
	iterator, err := NewCommitFileIter(repo, &commitFileIterOptions{head.Target().String()})
	// get the blame by adding the directory (.path) and the filename (.name)
	if err != nil {
		return nil, nil, err
	}
	file, err := iterator.Next()
	if err != nil {
		return nil, nil, err
	}
	blame, err := repo.BlameFile(file.path+file.Name, &opts)
	if err != nil {
		return nil, nil, err
	}
	// get the current file to extract its contents
	currentFile, err := file.AsBlob()
	if err != nil {
		return nil, nil, err
	}
	// break up the file by newlines as one would see in git blame output
	str := bytes.Split(currentFile.Contents(), []byte{'\n'})
	return &BlameIterator{
			repo,
			blame,
			str,
			iterator,
			file,
			1,
		}, &BlameHunk{
			1,
			str[0],
			file.path + file.Name,
			file.commitID,
		}, nil
}
func (vc *BlameIterator) Next() (*BlameHunk, error) {
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
				return nil, nil
			}
			return nil, err
		}
		opts, err := git.DefaultBlameOptions()
		if err != nil {
			return nil, nil
		}

		blame, err := vc.repo.BlameFile(file.path+file.Name, &opts)
		if err != nil {
			return nil, err
		}
		// get the current file to extract its contents
		currentFile, err := file.AsBlob()
		if err != nil {
			return nil, err
		}
		// create string array to display as in filter
		str := bytes.Split(currentFile.Contents(), []byte{'\n'})
		vc.currentFileContents = str
		vc.file = file
		vc.current = blame
		vc.lineIter = 1
	}

	return &BlameHunk{
		vc.lineIter,
		vc.currentFileContents[vc.lineIter-1],
		vc.file.path + vc.file.Name,
		vc.file.commitID,
	}, nil
}
