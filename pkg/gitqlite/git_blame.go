package gitqlite

import (
	"bytes"
	"fmt"

	git "github.com/libgit2/git2go/v30"
	"github.com/mattn/go-sqlite3"
)

type gitBlameModule struct{}

func NewGitBlameModule() *gitBlameModule {
	return &gitBlameModule{}
}

type gitBlameTable struct {
	repoPath string
}

func (m *gitBlameModule) Create(c *sqlite3.SQLiteConn, args []string) (sqlite3.VTab, error) {
	err := c.DeclareVTab(fmt.Sprintf(`
		CREATE TABLE %q (
			line_no TEXT,
			path TEXT,
			commit_id TEXT,
			contents TEXTS
		)`, args[0]))
	if err != nil {
		return nil, err
	}

	// the repoPath will be enclosed in double quotes "..." since ensureTables uses %q when setting up the table
	// we need to pop those off when referring to the actual directory in the fs
	repoPath := args[3][1 : len(args[3])-1]
	return &gitBlameTable{repoPath: repoPath}, nil
}

func (m *gitBlameModule) Connect(c *sqlite3.SQLiteConn, args []string) (sqlite3.VTab, error) {
	return m.Create(c, args)
}

func (m *gitBlameModule) DestroyModule() {}

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
	filenames           []string
	fileIds             []*git.Oid
	currentFileContents [][]byte
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
		c.ResultText(vc.filenames[vc.fileIter])
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
	revWalk, err := vc.repo.Walk()
	if err != nil {
		return err
	}
	err = revWalk.PushHead()
	if err != nil {
		return err
	}
	revWalk.Sorting(git.SortNone)

	oid := new(git.Oid)
	err = revWalk.Next(oid)
	if err != nil {
		return err
	}

	commit, err := vc.repo.LookupCommit(oid)
	if err != nil {
		return err
	}
	tree, err := commit.Tree()
	if err != nil {
		return err
	}

	var entries []string
	var ids []*git.Oid
	err = tree.Walk(func(s string, entry *git.TreeEntry) int {
		if entry.Type.String() == "Blob" {
			entries = append(entries, s+entry.Name)
			ids = append(ids, entry.Id)
		}
		return 0
	})
	if err != nil {
		return err
	}
	blame, err := vc.repo.BlameFile(entries[0], &opts)
	if err != nil {
		fmt.Println(err)
		return err
	}
	currentFile, err := vc.repo.LookupBlob(ids[0])
	if err != nil {
		fmt.Println(err)
		return err
	}
	str := bytes.Split(currentFile.Contents(), []byte{'\n'})

	vc.currentFileContents = str
	//fmt.Println(vc.currentFileContents)
	vc.fileIds = ids
	vc.filenames = entries
	vc.current = blame
	vc.lineIter = 1
	vc.fileIter = 1
	return nil
}

func (vc *blameCursor) Next() error {
	vc.lineIter++
	_, err := vc.current.HunkByLine(vc.lineIter)
	if err != nil {
		if vc.fileIter < len(vc.filenames)-1 {
			opts, err := git.DefaultBlameOptions()
			if err != nil {
				return err
			}
			vc.fileIter++
			blame, err := vc.repo.BlameFile(vc.filenames[vc.fileIter], &opts)
			if err != nil {
				fmt.Println(err)
				return err
			}
			currentFile, err := vc.repo.LookupBlob(vc.fileIds[vc.fileIter])
			if err != nil {
				fmt.Println(err)
				return err
			}
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
