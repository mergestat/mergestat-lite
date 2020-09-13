package gitqlite

import (
	"fmt"

	git "github.com/libgit2/git2go/v30"

	//"github.com/go-git/go-git/v5"
	// "github.com/go-git/go-git/v5/plumbing"
	// "github.com/go-git/go-git/v5/plumbing/object"
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
			file TEXT,
			additions INT(10),
			deletions INT(10)
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
	repo       *git.Repository
	current    *git.Commit
	stats      *git.DiffStats
	statIndex  int
	commitIter *git.RevWalk
}

func (vc *StatsCursor) Column(c *sqlite3.SQLiteContext, col int) error {
	commit := vc.current

	switch col {
	case 0:
		//commit id
		c.ResultText(commit.Id().String())
	case 1:
		c.ResultInt(vc.stats.FilesChanged())

	case 2:
		additions := vc.stats.Insertions()
		c.ResultInt(additions)

	case 3:
		deletions := vc.stats.Deletions()
		c.ResultInt(deletions)
	}
	return nil
}

func (vc *StatsCursor) Filter(idxNum int, idxStr string, vals []interface{}) error {
	// headRef, err := vc.repo.Head()
	// if err != nil {
	// 	if err == plumbing.ErrReferenceNotFound {
	// 		return nil
	// 	}
	// 	return err
	// }

	// iter, err := vc.repo.Walk()
	// if err != nil {
	// 	return err
	// }
	// vc.commitIter = *iter

	// commit := vc.commitIter.Next()
	// stats, err := vc.repo.DiffTreeToTree()
	// if err != nil {
	// 	return err
	// }
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
	err = revWalk.Next(id)
	if err != nil {
		return err
	}
	commit, err = vc.repo.LookupCommit(id)
	if err != nil {
		return err
	}
	tree, err := commit.Tree()
	defaults, err := git.DefaultDiffOptions()
	if err != nil {
		return nil
	}
	diff, err := vc.repo.DiffTreeToTree(oldTree, tree, &defaults)
	diffStats, err := diff.Stats()
	diff.Free()
	vc.current = commit
	vc.stats = diffStats
	vc.current = commit
	vc.statIndex = 0

	return nil
}

func (vc *StatsCursor) Next() error {
	// go to next file
	//for file, err := vc.fileIter.Next();err != io.EOF &&
	// if there are stats left go to the next stat
	// if len(vc.stats) > vc.statIndex+1 {
	// 	vc.statIndex++
	// 	if vc.stats[vc.statIndex].Addition == 0 && vc.stats[vc.statIndex].Deletion == 0 {
	// 		return vc.Next()
	// 	}
	// 	return nil
	// }
	// vc.statIndex = 0

	// commit, err := vc.commitIter.Next()
	// if err != nil {
	// 	if err == io.EOF {
	// 		vc.current = nil
	// 		return nil
	// 	}
	// 	return err
	// }

	// // edge case of initial commit
	// if commit.NumParents() == 0 {
	// 	files, err := commit.Files()
	// 	if err != nil {
	// 		return err
	// 	}
	// 	var stat object.FileStats
	// 	for x, err := files.Next(); err != io.EOF; x, err = files.Next() {
	// 		lines, err := x.Lines()
	// 		if err != nil {
	// 			return err
	// 		}
	// 		stat = append(stat, object.FileStat{Name: x.Name, Addition: len(lines), Deletion: 0})
	// 	}
	// 	vc.stats = stat
	// 	//case for when out of stats... go to next commit
	// } else {
	// 	stats, err := commit.Stats()
	// 	if err != nil {
	// 		return err
	// 	}
	// 	if len(stats) == 0 {
	// 		vc.stats = stats
	// 		vc.current = commit
	// 		return vc.Next()
	// 	}
	// 	vc.stats = stats
	// }
	// vc.current = commit

	// return nil
	id := new(git.Oid)
	err := vc.commitIter.Next(id)
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
	vc.current.Free()
	vc.current = commit
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
	vc.current.Free()
	vc.stats.Free()
	return nil
}
