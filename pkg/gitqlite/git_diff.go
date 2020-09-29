package gitqlite

import (
	"fmt"
	"io"

	git "github.com/libgit2/git2go/v30"
	"github.com/mattn/go-sqlite3"
)

type gitDiffModule struct{}

type gitDiffTable struct {
	repoPath string
	repo     *git.Repository
}
type credit struct {
	filename     string
	linesChanged string
	author       string
	commitId     string
}

func (m *gitDiffModule) Create(c *sqlite3.SQLiteConn, args []string) (sqlite3.VTab, error) {
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
	return &gitDiffTable{repoPath: repoPath}, nil
}

func (m *gitDiffModule) Connect(c *sqlite3.SQLiteConn, args []string) (sqlite3.VTab, error) {
	return m.Create(c, args)
}

func (m *gitDiffModule) DestroyModule() {}

func (v *gitDiffTable) Open() (sqlite3.VTabCursor, error) {
	repo, err := git.OpenRepository(v.repoPath)
	if err != nil {
		return nil, err
	}
	v.repo = repo

	return &diffCursor{repo: v.repo}, nil
}

func (v *gitDiffTable) BestIndex(cst []sqlite3.InfoConstraint, ob []sqlite3.InfoOrderBy) (*sqlite3.IndexResult, error) {
	// TODO this should actually be implemented!
	dummy := make([]bool, len(cst))
	return &sqlite3.IndexResult{Used: dummy}, nil
}

func (v *gitDiffTable) Disconnect() error {
	v.repo = nil
	return nil
}
func (v *gitDiffTable) Destroy() error { return nil }

type diffCursor struct {
	repo        *git.Repository
	current     *git.Commit
	diff        *git.Diff
	deltaIndex  int
	commitIter  *git.RevWalk
	blames      []credit
	blamesIndex int
}

func (vc *diffCursor) Column(c *sqlite3.SQLiteContext, col int) error {
	switch col {
	case 0:
		//commit id
		if vc.blamesIndex < len(vc.blames) {
			c.ResultText(vc.blames[vc.blamesIndex].filename)
		} else {
			c.ResultText("problem")
		}
	case 1:
		if vc.blamesIndex < len(vc.blames) {

			c.ResultText(vc.blames[vc.blamesIndex].linesChanged)
		} else {
			c.ResultText("problem")
		}
	case 2:
		if vc.blamesIndex < len(vc.blames) {

			c.ResultText(vc.blames[vc.blamesIndex].author)
		} else {
			c.ResultText("problem")
		}
	case 3:
		if vc.blamesIndex < len(vc.blames) {

			c.ResultText(vc.blames[vc.blamesIndex].commitId)
		} else {
			c.ResultText("problem")
		}

	}

	return nil
}
func (vc *diffCursor) Filter(idxNum int, idxStr string, vals []interface{}) error {
	revWalk, err := vc.repo.Walk()
	if err != nil {
		return err
	}

	err = revWalk.PushHead()
	if err != nil {
		return err
	}

	revWalk.Sorting(git.SortNone)

	vc.commitIter = revWalk

	id := new(git.Oid)
	err = revWalk.Next(id)
	if err != nil {
		return err
	}

	commit, err := vc.repo.LookupCommit(id)
	if err != nil {
		return err
	}
	oldTree, err := commit.Tree()
	if err != nil {
		fmt.Println(err)
		return nil
	}
	err = revWalk.Next(id)
	if err != nil {
		return err
	}
	commit, err = vc.repo.LookupCommit(id)
	if err != nil {
		return err
	}
	tree, err := commit.Tree()
	if err != nil {
		fmt.Println(err)
		return nil
	}
	defaults, err := git.DefaultDiffOptions()
	if err != nil {
		fmt.Println(err)
		return nil
	}
	diff, err := vc.repo.DiffTreeToTree(oldTree, tree, &defaults)
	if err != nil {
		fmt.Println(err)
		return nil
	}

	oldTree.Free()
	tree.Free()
	vc.diff = diff
	vc.deltaIndex = 0
	vc.blamesIndex = 0
	vc.current = commit
	numDeltas, err := vc.diff.NumDeltas()
	if err != nil {
		fmt.Println(err)
		return nil
	}
	if vc.deltaIndex >= numDeltas {
		return vc.Next()
	}
	deltaHolder, err := diff.Delta(vc.deltaIndex)
	if err != nil {
		fmt.Printf("89 : %s", err)
		return err
	}
	blame, err := vc.repo.BlameFile(deltaHolder.NewFile.Path, &git.BlameOptions{})
	if err != nil {
		return err
	}
	for i := 0; i < blame.HunkCount(); i++ {
		hunk, err := blame.HunkByIndex(i)
		if err != nil {
		} else {
			linesChanged := fmt.Sprintf(" %d - %d", int(hunk.FinalStartLineNumber), int(hunk.FinalStartLineNumber+hunk.LinesInHunk-1))
			vc.blames = append(vc.blames, credit{deltaHolder.NewFile.Path, linesChanged, hunk.FinalSignature.Name, hunk.FinalCommitId.String()})
		}
	}
	return nil
}
func (vc *diffCursor) NextBlame() error {
	if vc.blamesIndex+2 < len(vc.blames) {
		vc.blamesIndex++
		return nil
	}
	return io.EOF
}
func (vc *diffCursor) nextDiffDelta() error {
	vc.blames = make([]credit, 0)

	numDeltas, err := vc.diff.NumDeltas()
	if err != nil {
		fmt.Println(err)
		return nil
	}
	if vc.deltaIndex < numDeltas-1 {
		vc.deltaIndex += 1
		deltaHolder, err := vc.diff.Delta(vc.deltaIndex)
		if err != nil {
			fmt.Printf("89 : %s", err)
			return err
		}
		blame, err := vc.repo.BlameFile(deltaHolder.NewFile.Path, &git.BlameOptions{})
		if err != nil {
			return err
		}
		for i := 0; i < blame.HunkCount(); i++ {
			hunk, err := blame.HunkByIndex(i)
			if err != nil {
			} else {
				linesChanged := fmt.Sprintf(" %d - %d", int(hunk.FinalStartLineNumber), int(hunk.FinalStartLineNumber+hunk.LinesInHunk-1))
				vc.blames = append(vc.blames, credit{deltaHolder.NewFile.Path, linesChanged, hunk.FinalSignature.Name, hunk.FinalCommitId.String()})
			}
		}
		vc.blamesIndex = 0
		return nil
	}
	return io.EOF
}
func (vc *diffCursor) Next() error {
	err := vc.NextBlame()
	if err == io.EOF {
		err = vc.nextDiffDelta()
	}
	if err == io.EOF {

		numDeltas, err := vc.diff.NumDeltas()
		if err != nil {
			fmt.Println(err)
			return nil
		}
		for vc.deltaIndex >= numDeltas-1 {
			err = vc.nextCommitDiff()
			if err != nil {
				return err
			}
			numDeltas, err = vc.diff.NumDeltas()
			if err != nil {
				return err
			}
			vc.deltaIndex = 0
		}
		deltaHolder, err := vc.diff.Delta(vc.deltaIndex)
		if err != nil {
			fmt.Printf("89 : %s", err)
			return err
		}
		blame, err := vc.repo.BlameFile(deltaHolder.NewFile.Path, &git.BlameOptions{})
		if err != nil {
			return err
		}
		for i := 0; i < blame.HunkCount(); i++ {
			hunk, err := blame.HunkByIndex(i)
			if err != nil {
			} else {
				start := hunk.FinalStartLineNumber
				end := hunk.LinesInHunk - 1 + start
				linesChanged := fmt.Sprintf("%d - %d", start, end)
				vc.blames = append(vc.blames, credit{deltaHolder.NewFile.Path, linesChanged, hunk.FinalSignature.Name, hunk.FinalCommitId.String()})
			}
		}
		vc.blamesIndex = 0
		return nil
	} else {
		return nil
	}
}
func (vc *diffCursor) nextCommitDiff() error {
	oldTree, err := vc.current.Tree()
	if err != nil {
		return err
	}
	id := new(git.Oid)
	err = vc.commitIter.Next(id)
	if err != nil {
		if id.IsZero() {
			vc.current.Free()
			vc.current = nil
			return nil
		}
		return err
	}

	commit, err := vc.repo.LookupCommit(id)
	if err != nil {
		return err
	}
	tree, err := commit.Tree()
	if err != nil {
		fmt.Println(err)
		return nil
	}
	defaults, err := git.DefaultDiffOptions()
	if err != nil {
		fmt.Println(err)
		return nil
	}
	diff, err := vc.repo.DiffTreeToTree(oldTree, tree, &defaults)
	if err != nil {
		fmt.Println(err)
		return nil
	}
	vc.diff = diff
	vc.current = commit

	oldTree.Free()
	tree.Free()
	return nil
}

func (vc *diffCursor) EOF() bool {
	return vc.current == nil
}

func (vc *diffCursor) Rowid() (int64, error) {
	return int64(0), nil
}

func (vc *diffCursor) Close() error {
	vc.commitIter.Free()
	//vc.current.Free()
	//vc.commitStats.Free()
	return nil
}
