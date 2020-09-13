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
			files_changed INT(10),
			additions TEXT,
			deletions TEXT,
			files TEXT,
			stuff TEXT
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
	statIndex   int
	stats       []string
	commitIter  *git.RevWalk
}

func (vc *StatsCursor) Column(c *sqlite3.SQLiteContext, col int) error {
	commit := vc.current

	switch col {
	case 0:
		//commit id
		c.ResultText(commit.Id().String())
	case 1:
		c.ResultInt(vc.commitStats.FilesChanged())

	case 2:
		c.ResultText(vc.stats[vc.statIndex])

	case 3:
		//deletions := vc.commitStats.Deletions()
		c.ResultText(vc.stats[vc.statIndex+1])
	case 4:
		// statString, err := vc.stats.String(4, 0)
		// if err != nil {
		// 	return err
		// }
		// //info := strings.Split(statString, "|")
		// //filename := info[0]
		// stattos := strings.Split(statString, " ")
		// stattos = trimTheFat(stattos)
		// fmt.Println(stattos)
		// c.ResultText(fmt.Sprint(stattos))
		c.ResultText(vc.stats[vc.statIndex+2])
	case 5:
		stuff, err := vc.commitStats.String(4, 0)
		if err != nil {
			return err
		}
		c.ResultText(stuff)
	}

	return nil
}
func trimTheFat(s []string) []string {
	var newArr []string
	for _, x := range s {
		if x != "" {
			newArr = append(newArr, x)
		}
	}
	return newArr
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
		return nil
	}
	defaults, err := git.DefaultDiffOptions()
	if err != nil {
		return nil
	}
	diff, err := vc.repo.DiffTreeToTree(oldTree, tree, &defaults)
	if err != nil {
		return nil
	}
	diffStats, err := diff.Stats()
	if err != nil {
		return nil
	}
	statsString, err := diffStats.String(4, 0)
	if err != nil {
		return nil
	}
	statsString = strings.ReplaceAll(statsString, "\n", " ")
	stats := trimTheFat(strings.Split(statsString, " "))
	diff.Free()
	oldTree.Free()
	tree.Free()
	vc.stats = stats
	vc.statIndex = 0
	vc.commitStats = diffStats
	vc.current = commit
	if len(vc.stats) == 0 {
		vc.Next()
	}
	return nil
}

func (vc *StatsCursor) Next() error {
	// if vc.statIndex+5 < len(vc.stats) {
	// 	vc.statIndex += 3
	// 	return nil
	// }
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
		return nil
	}
	defaults, err := git.DefaultDiffOptions()
	if err != nil {
		return nil
	}
	diff, err := vc.repo.DiffTreeToTree(oldTree, tree, &defaults)
	if err != nil {
		return nil
	}
	diffStats, err := diff.Stats()
	if err != nil {
		return err
	}
	statsString, err := diffStats.String(4, 0)
	if err != nil {
		return nil
	}
	statsString = strings.ReplaceAll(statsString, "\n", " ")
	stats := trimTheFat(strings.Split(statsString, " "))

	err = diff.Free()
	if err != nil {
		return nil
	}
	oldTree.Free()
	tree.Free()
	diff.Free()
	vc.commitStats.Free()
	vc.statIndex = 0
	vc.commitStats = diffStats
	vc.stats = stats
	vc.current.Free()
	vc.current = commit
	if len(stats) == 0 {
		vc.Next()
	}
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
