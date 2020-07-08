package gitqlite

import (
	"database/sql"
	"fmt"

	"github.com/go-git/go-git/v5"
	"github.com/mattn/go-sqlite3"
)

// GitQLite loads git repositories into sqlite
type GitQLite struct {
	DB       *sql.DB
	RepoPath string
}

func init() {
	sql.Register("gitqlite", &sqlite3.SQLiteDriver{
		ConnectHook: func(conn *sqlite3.SQLiteConn) error {
			err := conn.CreateModule("git_log", &gitLogModule{})
			if err != nil {
				return err
			}

			err = conn.CreateModule("git_tree", &gitTreeModule{})
			if err != nil {
				return err
			}

			err = conn.CreateModule("git_ref", &gitRefModule{})
			if err != nil {
				return err
			}
			err = conn.CreateModule("git_tag", &gitTagModule{})
			if err != nil {
				return err
			}
			err = conn.CreateModule("git_branch", &gitBranchModule{})
			if err != nil {
				return err
			}

			return nil
		},
	})
}

// New creates an instance of GitQLite
func New(repoPath string) (*GitQLite, error) {
	// see https://github.com/mattn/go-sqlite3/issues/204
	// also mentioned in the FAQ of the README: https://github.com/mattn/go-sqlite3#faq
	db, err := sql.Open("gitqlite", "file::memory:?cache=shared")
	if err != nil {
		return nil, err
	}

	_, err = git.PlainOpen(repoPath)
	if err != nil {
		return nil, err
	}

	g := &GitQLite{DB: db, RepoPath: repoPath}

	err = g.ensureTables()
	if err != nil {
		return nil, err
	}

	return g, nil
}

// creates the virtual tables inside of the *sql.DB
func (g *GitQLite) ensureTables() error {
	_, err := g.DB.Exec(fmt.Sprintf("CREATE VIRTUAL TABLE commits USING git_log(%q);", g.RepoPath))
	if err != nil {
		return err
	}
	_, err = g.DB.Exec(fmt.Sprintf("CREATE VIRTUAL TABLE files USING git_tree(%q);", g.RepoPath))
	if err != nil {
		return err
	}
	_, err = g.DB.Exec(fmt.Sprintf("CREATE VIRTUAL TABLE refs USING git_ref(%q);", g.RepoPath))
	if err != nil {
		return err
	}
	_, err = g.DB.Exec(fmt.Sprintf("CREATE VIRTUAL TABLE tags USING git_tag(%q);", g.RepoPath))
	if err != nil {
		return err
	}
	_, err = g.DB.Exec(fmt.Sprintf("CREATE VIRTUAL TABLE branches USING git_branch(%q);", g.RepoPath))
	if err != nil {
		return err
	}

	return nil
}
