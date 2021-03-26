package git

import (
	"errors"
	"strings"

	git "github.com/libgit2/git2go/v31"
)

type BlameIterator struct {
	repo                 *git.Repository
	fileIter             *commitFileIter
	currentBlamedLines   []*BlamedLine
	currentBlamedLineIdx int
}

type BlamedLine struct {
	File     string
	Line     int
	CommitID string
	Content  string
}

func NewBlameIterator(repo *git.Repository) (*BlameIterator, error) {
	head, err := repo.Head()
	if err != nil {
		return nil, err
	}
	defer head.Free()

	// get a new iterator from repo and use the head commit
	fileIter, err := NewCommitFileIter(repo, &commitFileIterOptions{head.Target().String()})
	if err != nil {
		return nil, err
	}

	return &BlameIterator{
		repo,
		fileIter,
		nil,
		0,
	}, nil
}

func (iter *BlameIterator) nextFile() error {
	iter.currentBlamedLines = make([]*BlamedLine, 0)

	// grab the next file
	file, err := iter.fileIter.Next()
	if err != nil {
		return err
	}
	defer file.Free()

	// blame the file
	opts, err := git.DefaultBlameOptions()
	if err != nil {
		return err
	}
	blame, err := iter.repo.BlameFile(file.path+file.Name, &opts)
	if err != nil {
		return err
	}
	defer func() {
		err := blame.Free()
		if err != nil {
			panic(err)
		}
	}()

	// store the lines of the file, used as we iterate over hunks
	fileContents := file.Contents()
	lines := strings.Split(string(fileContents), "\n")

	// iterate over the blame hunks
	fileLine := 1

	for {
		hunk, err := blame.HunkByLine(fileLine)
		if err != nil {
			if errors.Is(err, git.ErrInvalid) {
				break
			}
			return err
		}
		blamedLine := &BlamedLine{
			File:     file.path + file.Name,
			CommitID: hunk.OrigCommitId.String(),
			Line:     fileLine,
			Content:  lines[fileLine-1],
		}
		// add it to the list for the current file
		iter.currentBlamedLines = append(iter.currentBlamedLines, blamedLine)
		// increment the file line by 1
		fileLine++

	}
	iter.currentBlamedLineIdx = 0

	return nil
}

func (iter *BlameIterator) Next() (*BlamedLine, error) {
	// if there's no currently blamed lines, grab the next file
	if iter.currentBlamedLines == nil {
		err := iter.nextFile()
		if err != nil {
			return nil, err
		}
	}

	// if we've exceeded the number of lines then go to next file
	if iter.currentBlamedLineIdx >= len(iter.currentBlamedLines) {
		err := iter.nextFile()
		if err != nil {
			return nil, err
		}
	}

	// if there's no blamed lines
	if len(iter.currentBlamedLines) == 0 {
		return iter.Next()
	}

	blamedLine := iter.currentBlamedLines[iter.currentBlamedLineIdx]
	iter.currentBlamedLineIdx++

	return blamedLine, nil
}
