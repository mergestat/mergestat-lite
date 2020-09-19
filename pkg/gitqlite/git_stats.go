package gitqlite

import (
	"fmt"
	"strings"

	git "github.com/libgit2/git2go/v30"
	"github.com/mattn/go-sqlite3"
)

type gitStatsModule struct{}

type gitStatsTable struct {
	repoPath string
	repo     *git.Repository
}

func (m *gitStatsModule) Create(c *sqlite3.SQLiteConn, args []string) (sqlite3.VTab, error) {
	err := c.DeclareVTab(fmt.Sprintf(`
		CREATE TABLE %q (
			commit_id TEXT,
			files_changed TEXT,
			additions TEXT,
			deletions TEXT
		)`, args[0]))
	if err != nil {
		return nil, err
	}

	// the repoPath will be enclosed in double quotes "..." since ensureTables uses %q when setting up the table
	// we need to pop those off when referring to the actual directory in the fs
	repoPath := args[3][1 : len(args[3])-1]
	return &gitStatsTable{repoPath: repoPath}, nil
}

func (m *gitStatsModule) Connect(c *sqlite3.SQLiteConn, args []string) (sqlite3.VTab, error) {
	return m.Create(c, args)
}

func (m *gitStatsModule) DestroyModule() {}

func (v *gitStatsTable) Open() (sqlite3.VTabCursor, error) {
	repo, err := git.OpenRepository(v.repoPath)
	if err != nil {
		return nil, err
	}
	v.repo = repo

	return &StatsCursor{repo: v.repo}, nil
}

func (v *gitStatsTable) BestIndex(cst []sqlite3.InfoConstraint, ob []sqlite3.InfoOrderBy) (*sqlite3.IndexResult, error) {
	// TODO this should actually be implemented!
	dummy := make([]bool, len(cst))
	return &sqlite3.IndexResult{Used: dummy}, nil
}

func (v *gitStatsTable) Disconnect() error {
	v.repo = nil
	return nil
}
func (v *gitStatsTable) Destroy() error { return nil }

type StatsCursor struct {
	repo        *git.Repository
	current     *git.Commit
	commitStats *git.DiffStats
	diff        *git.Diff
	deltaIndex  int
	commitIter  *git.RevWalk
}

func (vc *StatsCursor) Column(c *sqlite3.SQLiteContext, col int) error {
	commit := vc.current
	additions := 0
	deletions := 0
	filename := ""
	holder, err := vc.diff.Patch(vc.deltaIndex)
	if err != nil {
		fmt.Printf("84 : %s, %d", err, vc.deltaIndex)

		return err
	}
	deltaHolder, err := vc.diff.Delta(vc.deltaIndex)
	if err != nil {
		fmt.Printf("89 : %s", err)
		return err
	}
	h, err := holder.String()
	if err != nil {
		fmt.Printf("94 : %s", err)
		return err
	}
	filename = deltaHolder.NewFile.Path
	fileLines := strings.Split(h, "\n")
	for _, line := range fileLines {
		if strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---") {
			deletions++
		} else if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
			additions++
		}
		if err != nil {
			fmt.Println(err)
			return err
		}

	}
	switch col {
	case 0:
		//commit id
		c.ResultText(commit.Id().String())
	case 1:
		c.ResultText(filename)
	case 2:
		c.ResultInt(additions)
	case 3:
		c.ResultInt(deletions)

	}

	return nil
}

func (vc *StatsCursor) Filter(idxNum int, idxStr string, vals []interface{}) error {
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
	err = diff.FindSimilar(&git.DiffFindOptions{})
	if err != nil {
		fmt.Println(err)
		return nil
	}
	diffStats, err := diff.Stats()
	if err != nil {
		fmt.Println(err)
		return nil
	}
	oldTree.Free()
	tree.Free()
	vc.diff = diff
	vc.deltaIndex = 0
	vc.commitStats = diffStats
	vc.current = commit
	return nil
}

func (vc *StatsCursor) Next() error {
	numDeltas, err := vc.diff.NumDeltas()
	if err != nil {
		fmt.Println(err)
		return nil
	}
	if vc.deltaIndex < numDeltas-1 {
		vc.deltaIndex += 1
		return nil
	}
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
	diffStats, err := diff.Stats()
	if err != nil {
		return err
	}
	err = diff.FindSimilar(&git.DiffFindOptions{})
	if err != nil {
		fmt.Println(err)
		return nil
	}
	vc.diff = diff
	vc.deltaIndex = 0
	vc.commitStats = diffStats
	vc.current = commit
	numDeltas, err = vc.diff.NumDeltas()
	if err != nil {
		fmt.Println(err)
		return nil
	}
	if vc.deltaIndex >= numDeltas {
		vc.Next()
	}
	oldTree.Free()
	tree.Free()
	vc.commitStats.Free()
	return nil
}

func (vc *StatsCursor) EOF() bool {
	return vc.current == nil
}

func (vc *StatsCursor) Rowid() (int64, error) {
	return int64(0), nil
}

func (vc *StatsCursor) Close() error {
	vc.commitIter.Free()
	//vc.current.Free()
	//vc.commitStats.Free()
	return nil
}
