package gitqlite

import (
	"fmt"

	git "github.com/libgit2/git2go/v30"
	"github.com/mattn/go-sqlite3"
)

type gitStatsModule struct{}

type gitStatsTable struct {
	repoPath string
	repo     *git.Repository
}
type fileStats struct {
	commitId  string
	fileName  string
	additions int
	deletions int
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
	repo   *git.Repository
	stats  []fileStats
	sindex int
}

func (vc *StatsCursor) Column(c *sqlite3.SQLiteContext, col int) error {

	switch col {
	case 0:
		//commit id
		c.ResultText(vc.stats[vc.sindex].commitId)
	case 1:
		c.ResultText(vc.stats[vc.sindex].fileName)
	case 2:
		c.ResultInt(vc.stats[vc.sindex].additions)
	case 3:
		c.ResultInt(vc.stats[vc.sindex].deletions)

	}

	return nil
}
func (vc *StatsCursor) Filter(idxNum int, idxStr string, vals []interface{}) error {
	revWalk, err := vc.repo.Walk()
	if err != nil {
		return err
	}
	defer revWalk.Free()
	err = revWalk.PushHead()
	if err != nil {
		fmt.Println(err)
		return err
	}
	vc.sindex = -1
	var prevCommit *git.Commit
	err = revWalk.Iterate(func(commit *git.Commit) bool {
		err := calcStats(commit, prevCommit, vc)
		if err != nil {
			return false
		}
		prevCommit = commit
		return true
	})

	if err != nil {
		fmt.Println(err)
		return err
	}
	vc.sindex = 0
	return nil
}

func (vc *StatsCursor) Next() error {
	if vc.sindex < len(vc.stats) {
		vc.sindex++
	}
	return nil
}

func (vc *StatsCursor) EOF() bool {
	return vc.sindex == len(vc.stats)
}

func (vc *StatsCursor) Rowid() (int64, error) {
	return int64(0), nil
}

func (vc *StatsCursor) Close() error {
	vc.stats = nil
	return nil
}
func calcStats(commit, prevCommit *git.Commit, vc *StatsCursor) error {
	if commit == nil || prevCommit == nil {
		return nil
	}
	repo := commit.Owner()
	tree, err := commit.Tree()
	if err != nil {
		return err
	}
	defer tree.Free()
	prevTree, err := prevCommit.Tree()
	if err != nil {
		return err
	}
	defer prevTree.Free()
	diffOpts, err := git.DefaultDiffOptions()
	if err != nil {
		return err
	}
	diff, err := repo.DiffTreeToTree(tree, prevTree, &diffOpts)
	if err != nil {
		return err
	}
	diffFindOpts, err := git.DefaultDiffFindOptions()
	if err != nil {
		return err
	}
	err = diff.FindSimilar(&diffFindOpts)
	if err != nil {
		return err
	}

	err = diff.ForEach(func(delta git.DiffDelta, progress float64) (git.DiffForEachHunkCallback, error) {
		perHunkCB := func(hunk git.DiffHunk) (git.DiffForEachLineCallback, error) {
			perLineCB := func(line git.DiffLine) error {
				if vc.sindex == -1 {
					if line.Origin == git.DiffLineAddition {
						vc.stats = append(vc.stats, fileStats{prevCommit.Id().String(), delta.NewFile.Path, 1, 0})
						vc.sindex++
					} else if line.Origin == git.DiffLineDeletion {
						vc.stats = append(vc.stats, fileStats{prevCommit.Id().String(), delta.NewFile.Path, 0, 1})
						vc.sindex++
					}
				} else {
					if vc.stats[vc.sindex].fileName == delta.NewFile.Path {
						if line.Origin == git.DiffLineAddition {
							vc.stats[vc.sindex].additions++
						} else if line.Origin == git.DiffLineDeletion {
							vc.stats[vc.sindex].deletions++
						}
					} else {
						if line.Origin == git.DiffLineAddition {
							vc.stats = append(vc.stats, fileStats{prevCommit.Id().String(), delta.NewFile.Path, 1, 0})
							vc.sindex++

						} else if line.Origin == git.DiffLineDeletion {
							vc.stats = append(vc.stats, fileStats{prevCommit.Id().String(), delta.NewFile.Path, 0, 1})
							vc.sindex++

						}
					}
				}
				return nil
			}
			return perLineCB, nil
		}
		return perHunkCB, nil
	}, git.DiffDetailLines)
	if err != nil {
		return err
	}
	// for x, j := range stats {
	// 	fmt.Println(x, j)
	// }
	return nil
}
