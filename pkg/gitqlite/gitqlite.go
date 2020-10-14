package gitqlite

import (
	"crypto/md5"
	"database/sql"
	"fmt"
	"os/exec"
	"strings"

	git "github.com/libgit2/git2go/v30"
	"github.com/mattn/go-sqlite3"
)

// GitQLite loads git repositories into sqlite
type GitQLite struct {
	DB       *sql.DB
	RepoPath string
}
type Options struct {
	UseGitCLI bool
}

func init() {
	sql.Register("gitqlite", &sqlite3.SQLiteDriver{
		ConnectHook: func(conn *sqlite3.SQLiteConn) error {
			err := conn.CreateModule("git_log", &gitLogModule{})
			if err != nil {
				return err
			}

			err = conn.CreateModule("git_log_cli", &gitLogCLIModule{})
			if err != nil {
				return err
			}

			err = conn.CreateModule("git_tree", &gitTreeModule{})
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
			err = conn.CreateModule("git_stats_cli", &gitStatsCLIModule{})
			if err != nil {
				return err
			}

			err = loadHelperFuncs(conn)
			if err != nil {
				return err
			}

			return nil
		},
	})
}

// New creates an instance of GitQLite
func New(repoPath string, options *Options) (*GitQLite, error) {
	// see https://github.com/mattn/go-sqlite3/issues/204
	// also mentioned in the FAQ of the README: https://github.com/mattn/go-sqlite3#faq
	db, err := sql.Open("gitqlite", fmt.Sprintf("file:%x?mode=memory", md5.Sum([]byte(repoPath))))
	if err != nil {
		return nil, err
	}
	_, err = git.OpenRepository(repoPath)
	if err != nil {
		return nil, err
	}

	g := &GitQLite{DB: db, RepoPath: repoPath}

	err = g.ensureTables(options)
	if err != nil {
		return nil, err
	}
	return g, nil
}

// creates the virtual tables inside of the *sql.DB
func (g *GitQLite) ensureTables(options *Options) error {

	_, err := exec.LookPath("git")
	localGitExists := err == nil
	g.RepoPath = strings.ReplaceAll(g.RepoPath, "'", "''")
	if !options.UseGitCLI || !localGitExists {
		_, err := g.DB.Exec(fmt.Sprintf("CREATE VIRTUAL TABLE IF NOT EXISTS commits USING git_log('%s');", g.RepoPath))
		if err != nil {
			return err
		}
	} else {
		_, err := g.DB.Exec(fmt.Sprintf("CREATE VIRTUAL TABLE IF NOT EXISTS commits USING git_log_cli('%s');", g.RepoPath))
		if err != nil {
			return err
		}
		_, err = g.DB.Exec(fmt.Sprintf("CREATE VIRTUAL TABLE IF NOT EXISTS stats USING git_stats_cli('%s');", g.RepoPath))
		if err != nil {
			return err
		}

	}

	_, err = g.DB.Exec(fmt.Sprintf("CREATE VIRTUAL TABLE IF NOT EXISTS files USING git_tree('%s');", g.RepoPath))
	if err != nil {
		return err
	}
	_, err = g.DB.Exec(fmt.Sprintf("CREATE VIRTUAL TABLE IF NOT EXISTS tags USING git_tag('%s');", g.RepoPath))
	if err != nil {
		return err
	}
	_, err = g.DB.Exec(fmt.Sprintf("CREATE VIRTUAL TABLE IF NOT EXISTS branches USING git_branch('%s');", g.RepoPath))
	if err != nil {
		return err
	}

	return nil
}

func loadHelperFuncs(conn *sqlite3.SQLiteConn) error {
	// str_split(inputString, splitCharacter, index) string
	split := func(s, c string, i int) string {
		split := strings.Split(s, c)
		if i < len(split) {
			return split[i]
		}
		return ""
	}

	if err := conn.RegisterFunc("str_split", split, true); err != nil {
		return err
	}

	return nil
}
