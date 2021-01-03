package gitqlite

import (
	"bytes"
	"fmt"
	"io"

	git "github.com/libgit2/git2go/v31"
	"github.com/mattn/go-sqlite3"
)

type GitBlameModule struct {
	options *GitBlameModuleOptions
}

type GitBlameModuleOptions struct {
	RepoPath string
}

func NewGitBlameModule(options *GitBlameModuleOptions) *GitBlameModule {
	return &GitBlameModule{options}
}

type gitBlameTable struct {
	repoPath string
}

func (m *GitBlameModule) EponymousOnlyModule() {}

func (m *GitBlameModule) Create(c *sqlite3.SQLiteConn, args []string) (sqlite3.VTab, error) {
	err := c.DeclareVTab(fmt.Sprintf(`
		CREATE TABLE %q (
			line_no TEXT,
			path TEXT,
			commit_id TEXT,
			contents TEXT
		)`, args[0]))
	if err != nil {
		return nil, err
	}

	return &gitBlameTable{repoPath: m.options.RepoPath}, nil
}

func (m *GitBlameModule) Connect(c *sqlite3.SQLiteConn, args []string) (sqlite3.VTab, error) {
	return m.Create(c, args)
}

func (m *GitBlameModule) DestroyModule() {}

func (v *gitBlameTable) Open() (sqlite3.VTabCursor, error) {
	repo, err := git.OpenRepository(v.repoPath)
	if err != nil {
		return nil, err
	}

	return &blameCursor{repo: repo}, nil

}

func (v *gitBlameTable) BestIndex(cst []sqlite3.InfoConstraint, ob []sqlite3.InfoOrderBy) (*sqlite3.IndexResult, error) {
	// TODO this should actually be implemented!
	dummy := make([]bool, len(cst))
	return &sqlite3.IndexResult{Used: dummy}, nil
}

func (v *gitBlameTable) Disconnect() error {
	return nil
}

func (v *gitBlameTable) Destroy() error { return nil }

type blameCursor struct {
	repo                *git.Repository
	current             *git.Blame
	currentFileContents [][]byte
	iterator            *commitFileIter
	fileIter            int
	lineIter            int
}

func (vc *blameCursor) Column(c *sqlite3.SQLiteContext, col int) error {
	line, err := vc.current.HunkByLine(vc.lineIter)
	if err != nil {
		return err
	}

	switch col {
	case 0:
		//branch name
		c.ResultText(fmt.Sprint(vc.lineIter))
	case 1:
		c.ResultText(vc.iterator.treeEntries[vc.iterator.currentTreeEntryIndex].Name)
	case 2:
		c.ResultText(line.FinalCommitId.String())
	case 3:
		c.ResultText(string(vc.currentFileContents[vc.lineIter-1]) + " ")

	}

	return nil

}

func (vc *blameCursor) Filter(idxNum int, idxStr string, vals []interface{}) error {
	opts, err := git.DefaultBlameOptions()
	if err != nil {
		return err
	}

	head, err := vc.repo.Head()
	if err != nil {
		return err
	}
	defer head.Free()
	// get a new iterator from repo and use the head commit
	iterator, err := NewCommitFileIter(vc.repo, &commitFileIterOptions{head.Target().String()})
	// limit the 'search' to blobs as tree entry's will not have blame's
	for iterator.treeEntries[iterator.currentTreeEntryIndex].Type.String() != "Blob" {
		iterator.Next()
	}
	// get the blame by adding the directory (.path) and the filename (.name)
	blame, err := vc.repo.BlameFile(iterator.treeEntries[iterator.currentTreeEntryIndex].path+iterator.treeEntries[iterator.currentTreeEntryIndex].Name, &opts)
	if err != nil {
		fmt.Println(err)
		return err
	}
	// get the current file to extract its contents
	currentFile, err := vc.repo.LookupBlob(iterator.treeEntries[iterator.currentTreeEntryIndex].Id)
	if err != nil {
		fmt.Println(err)
		return err
	}
	// break up the file by newlines as one would a blame file
	str := bytes.Split(currentFile.Contents(), []byte{'\n'})

	vc.currentFileContents = str
	vc.iterator = iterator
	vc.current = blame
	vc.lineIter = 1
	vc.fileIter = 0
	return nil
}

func (vc *blameCursor) Next() error {
	// increment the lineIterator count to the next line
	vc.lineIter++
	// check if the lineIter has overrun the file
	_, err := vc.current.HunkByLine(vc.lineIter)
	// if there is an error then go to the next file
	if err != nil {
		// iterate to next file
		_, err := vc.iterator.Next()
		// make sure that it is a file and not a tree entry
		for vc.iterator.treeEntries[vc.iterator.currentTreeEntryIndex].Type.String() != "Blob" {
			_, err = vc.iterator.Next()
		}
		// if the error isn't an EOF flag then create the next currentFileContents
		if err != io.EOF {
			opts, err := git.DefaultBlameOptions()
			if err != nil {
				panic(err)
				return err
			}
			// look up blamefile as in filter
			blame, err := vc.repo.BlameFile(vc.iterator.treeEntries[vc.iterator.currentTreeEntryIndex].path+vc.iterator.treeEntries[vc.iterator.currentTreeEntryIndex].Name, &opts)
			if err != nil {
				fmt.Println(err)
				panic(err)
				return err
			}
			// look up blob as in filter
			currentFile, err := vc.repo.LookupBlob(vc.iterator.treeEntries[vc.iterator.currentTreeEntryIndex].Id)
			if err != nil {
				fmt.Println(err)
				panic(err)
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

func (vc *blameCursor) EOF() bool {
	return vc.current == nil
}

func (vc *blameCursor) Rowid() (int64, error) {
	return int64(0), nil
}

func (vc *blameCursor) Close() error {
	if vc.current != nil {
		err := vc.current.Free()
		if err != nil {
			return nil
		}
	}

	return nil
}
