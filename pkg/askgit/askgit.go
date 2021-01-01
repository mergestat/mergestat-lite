package askgit

import (
	"context"
	"crypto/md5"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path"
	"time"

	"github.com/augmentable-dev/askgit/pkg/ghqlite"
	"github.com/augmentable-dev/askgit/pkg/gitqlite"
	"github.com/gitsight/go-vcsurl"
	git "github.com/libgit2/git2go/v31"
	"github.com/mattn/go-sqlite3"
	"golang.org/x/time/rate"
)

type AskGit struct {
	db      *sql.DB
	options *Options
}

type Options struct {
	RepoPath    string
	UseGitCLI   bool
	GitHubToken string
	DBFilePath  string
}

type driverConnector struct {
	dsn string
	d   *sqlite3.SQLiteDriver
}

func newDriverConnector(dsn string, d *sqlite3.SQLiteDriver) (driver.Connector, error) {
	return &driverConnector{dsn, d}, nil
}

func (dc *driverConnector) Connect(context.Context) (driver.Conn, error) {
	conn, err := dc.d.Open(dc.dsn)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

func (dc *driverConnector) Driver() driver.Driver {
	return dc.d
}

// New creates an instance of AskGit
func New(options *Options) (*AskGit, error) {
	// TODO with the addition of the GitHub API virtual tables, repoPath should no longer be required for creating
	// as *AskGit instance, as the caller may just be interested in querying against the GitHub API (or some other
	// to be define virtual table that doesn't need a repo on disk).
	// This should be reformulated, as it means currently the askgit command requires a local git repo, even if the query
	// only executes agains the GitHub API

	// ensure the repository exists by trying to open it
	_, err := git.OpenRepository(options.RepoPath)
	if err != nil {
		return nil, err
	}

	var dataSource string
	if options.DBFilePath == "" {
		// see https://github.com/mattn/go-sqlite3/issues/204
		// also mentioned in the FAQ of the README: https://github.com/mattn/go-sqlite3#faq
		dataSource = fmt.Sprintf("file:%x?mode=memory&cache=shared", md5.Sum([]byte(options.RepoPath)))
	} else {
		dataSource = options.DBFilePath
	}

	a := &AskGit{options: options}

	d := sqlite3.SQLiteDriver{
		ConnectHook: func(conn *sqlite3.SQLiteConn) error {
			err := loadHelperFuncs(conn)
			if err != nil {
				return err
			}

			err = a.loadGitQLiteModules(conn)
			if err != nil {
				return err
			}

			err = a.loadGitHubModules(conn)
			if err != nil {
				return err
			}

			return nil
		},
	}

	dc, err := newDriverConnector(dataSource, &d)
	if err != nil {
		return nil, err
	}

	a.db = sql.OpenDB(dc)

	return a, nil
}

func (a *AskGit) DB() *sql.DB {
	return a.db
}

func (a *AskGit) RepoPath() string {
	return a.options.RepoPath
}

func (a *AskGit) loadGitQLiteModules(conn *sqlite3.SQLiteConn) error {
	_, err := exec.LookPath("git")
	localGitExists := err == nil

	if !a.options.UseGitCLI || !localGitExists {
		err = conn.CreateModule("commits", gitqlite.NewGitLogModule(&gitqlite.GitLogModuleOptions{RepoPath: a.RepoPath()}))
		if err != nil {
			return err
		}
	} else {
		err = conn.CreateModule("commits", gitqlite.NewGitLogCLIModule(&gitqlite.GitLogCLIModuleOptions{RepoPath: a.RepoPath()}))
		if err != nil {
			return err
		}
	}

	err = conn.CreateModule("stats", gitqlite.NewGitStatsModule(&gitqlite.GitStatsModuleOptions{RepoPath: a.RepoPath()}))
	if err != nil {
		return err
	}

	err = conn.CreateModule("files", gitqlite.NewGitFilesModule(&gitqlite.GitFilesModuleOptions{RepoPath: a.RepoPath()}))
	if err != nil {
		return err
	}

	err = conn.CreateModule("tags", gitqlite.NewGitTagsModule(&gitqlite.GitTagsModuleOptions{RepoPath: a.RepoPath()}))
	if err != nil {
		return err
	}

	err = conn.CreateModule("branches", gitqlite.NewGitBranchesModule(&gitqlite.GitBranchesModuleOptions{RepoPath: a.RepoPath()}))
	if err != nil {
		return err
	}

	return nil
}

func (a *AskGit) loadGitHubModules(conn *sqlite3.SQLiteConn) error {
	githubToken := os.Getenv("GITHUB_TOKEN")
	rateLimiter := rate.NewLimiter(rate.Every(2*time.Second), 1)

	err := conn.CreateModule("github_org_repos", ghqlite.NewReposModule(ghqlite.OwnerTypeOrganization, ghqlite.ReposModuleOptions{
		Token:       githubToken,
		RateLimiter: rateLimiter,
	}))
	if err != nil {
		return err
	}

	err = conn.CreateModule("github_user_repos", ghqlite.NewReposModule(ghqlite.OwnerTypeUser, ghqlite.ReposModuleOptions{
		Token:       githubToken,
		RateLimiter: rateLimiter,
	}))
	if err != nil {
		return err
	}

	err = conn.CreateModule("github_pull_requests", ghqlite.NewPullRequestsModule(ghqlite.PullRequestsModuleOptions{
		Token:       githubToken,
		RateLimiter: rateLimiter,
	}))
	if err != nil {
		return err
	}

	return nil
}

func CreateAuthenticationCallback(remote *vcsurl.VCS) *git.CloneOptions {
	cloneOptions := &git.CloneOptions{}

	if _, err := remote.Remote(vcsurl.SSH); err == nil { // if SSH, use "default" credentials
		// use FetchOptions instead of directly RemoteCallbacks
		// https://github.com/libgit2/git2go/commit/36e0a256fe79f87447bb730fda53e5cbc90eb47c
		cloneOptions.FetchOptions = &git.FetchOptions{
			RemoteCallbacks: git.RemoteCallbacks{
				CredentialsCallback: func(url string, username string, allowedTypes git.CredType) (*git.Cred, error) {
					usr, _ := user.Current()
					publicSSH := path.Join(usr.HomeDir, ".ssh/id_rsa.pub")
					privateSSH := path.Join(usr.HomeDir, ".ssh/id_rsa")

					cred, ret := git.NewCredSshKey("git", publicSSH, privateSSH, "")
					return cred, ret
				},
				CertificateCheckCallback: func(cert *git.Certificate, valid bool, hostname string) git.ErrorCode {
					return git.ErrOk
				},
			}}
	}
	return cloneOptions
}
