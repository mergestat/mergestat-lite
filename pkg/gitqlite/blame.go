package gitqlite

import (
	"bytes"
	"fmt"

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
	fileIndex           int
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
		c.ResultText(vc.iterator.treeEntries[vc.fileIndex].Name)

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
	// get the blame by adding the directory (.path) and the filename (.name)
	blame, err := vc.repo.BlameFile(iterator.treeEntries[0].path+iterator.treeEntries[0].Name, &opts)
	if err != nil {
		return err
	}
	// get the current file to extract its contents
	currentFile, err := vc.repo.LookupBlob(iterator.treeEntries[0].Id)
	if err != nil {
		return err
	}
	// break up the file by newlines as one would see in git blame output
	str := bytes.Split(currentFile.Contents(), []byte{'\n'})
	// the current contents to be iterated over and outputed
	vc.currentFileContents = str
	// the iterator that provides us with the tree entries
	vc.iterator = iterator
	// the index for which file we are in in the head commit tree
	vc.fileIndex = 0
	// the current blame file
	vc.current = blame
	// the current line for output in the table (1 indexed)
	vc.lineIter = 1
	return nil
}

func (vc *blameCursor) Next() error {
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