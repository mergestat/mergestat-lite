package gitqlite

import (
	"fmt"
	"io"

	git "github.com/libgit2/git2go/v30"
	"github.com/mattn/go-sqlite3"
)

type gitBlameModule struct{}

type gitBlameTable struct {
	repoPath string
	repo     *git.Repository
}
type credit struct {
	filename     string
	linesChanged string
	author       string
	commitId     string
}

func (m *gitBlameModule) Create(c *sqlite3.SQLiteConn, args []string) (sqlite3.VTab, error) {
	err := c.DeclareVTab(fmt.Sprintf(`
		CREATE TABLE %q (
			filename TEXT,
			lines_changed TEXT,
			author TEXT,
			commit_id TEXT
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
	v.repo = repo

	return &blameCursor{repo: v.repo}, nil
}

func (v *gitBlameTable) BestIndex(cst []sqlite3.InfoConstraint, ob []sqlite3.InfoOrderBy) (*sqlite3.IndexResult, error) {
	// TODO this should actually be implemented!
	dummy := make([]bool, len(cst))
	return &sqlite3.IndexResult{Used: dummy}, nil
}

func (v *gitBlameTable) Disconnect() error {
	v.repo = nil
	return nil
}
func (v *gitBlameTable) Destroy() error { return nil }

type blameCursor struct {
	repo        *git.Repository
	current     *commitFile
	fileIter    *commitFileIter
	blames      []credit
	blamesIndex int
}

func (vc *blameCursor) Column(c *sqlite3.SQLiteContext, col int) error {
	switch col {
	case 0:
		//commit id

		c.ResultText(vc.blames[vc.blamesIndex].filename)

	case 1:

		c.ResultText(vc.blames[vc.blamesIndex].linesChanged)

	case 2:

		c.ResultText(vc.blames[vc.blamesIndex].author)

	case 3:

		c.ResultText(vc.blames[vc.blamesIndex].commitId)

	}

	return nil
}
func (vc *blameCursor) Filter(idxNum int, idxStr string, vals []interface{}) error {
	var opt *commitFileIterOptions

	switch idxNum {
	case 0:
		opt = &commitFileIterOptions{}
	case 1:
		opt = &commitFileIterOptions{commitID: vals[0].(string)}
	}
	iter, err := NewCommitFileIter(vc.repo, opt)
	if err != nil {
		return err
	}

	vc.fileIter = iter

	file, err := vc.fileIter.Next()
	if err != nil {
		if err == io.EOF {
			vc.current = nil
			return nil
		}
		return err
	}
	blame, err := vc.repo.BlameFile(file.path+file.Name, &git.BlameOptions{})
	if err != nil {
		fmt.Println(err)
		return err
	}
	for i := 0; i < blame.HunkCount(); i++ {
		hunk, err := blame.HunkByIndex(i)
		if err != nil {
			fmt.Println(err)
		} else {
			linesChanged := fmt.Sprintf(" %d - %d", int(hunk.FinalStartLineNumber), int(hunk.FinalStartLineNumber+hunk.LinesInHunk-1))
			vc.blames = append(vc.blames, credit{file.path + file.Name, linesChanged, hunk.FinalSignature.Name, hunk.FinalCommitId.String()})
		}
	}
	vc.current = file
	return nil
}
func (vc *blameCursor) NextBlame() error {
	if vc.blamesIndex+2 < len(vc.blames) {
		vc.blamesIndex++
		return nil
	}
	return io.EOF
}
func (vc *blameCursor) Next() error {
	err := vc.NextBlame()
	if err == io.EOF {
		file, err := vc.fileIter.Next()
		if err != nil {
			if err == io.EOF {
				vc.current = nil
				return nil
			}
			return err
		}
		blame, err := vc.repo.BlameFile(file.path+file.Name, &git.BlameOptions{})
		if err != nil {
			return err
		}
		vc.blames = nil
		for i := 0; i < blame.HunkCount(); i++ {
			hunk, err := blame.HunkByIndex(i)
			if err != nil {
			} else {
				start := hunk.FinalStartLineNumber
				end := hunk.LinesInHunk - 1 + start
				linesChanged := fmt.Sprintf("%d - %d", start, end)
				vc.blames = append(vc.blames, credit{file.path + file.Name, linesChanged, hunk.FinalSignature.Name, hunk.FinalCommitId.String()})
			}
		}
		vc.blamesIndex = 0
		return nil
	} else {
		return nil
	}
}

func (vc *blameCursor) EOF() bool {
	return vc.current == nil
}

func (vc *blameCursor) Rowid() (int64, error) {
	return int64(0), nil
}

func (vc *blameCursor) Close() error {
	//vc.commitIter.Free()
	//vc.current.Free()
	//vc.commitStats.Free()
	return nil
}
