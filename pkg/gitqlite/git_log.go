package gitqlite

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"path"
	"path/filepath"
	"time"

	"github.com/gitsight/go-vcsurl"
	git "github.com/libgit2/git2go/v31"
	"github.com/mattn/go-sqlite3"
)

type GitLogModule struct {
	options commitTableModuleOptions
}

func NewGitLogModule() *GitLogModule {
	return &GitLogModule{}
}

type commitTableModuleOptions struct {
	repoPath string
}
type commitsTable struct {
	module *GitLogModule
}

func (m *GitLogModule) EponymousOnlyModule() {}
func (m *GitLogModule) Connect(c *sqlite3.SQLiteConn, args []string) (sqlite3.VTab, error) {
	return m.Create(c, args)
}

func (m *GitLogModule) Create(c *sqlite3.SQLiteConn, args []string) (sqlite3.VTab, error) {
	//print("create")
	err := c.DeclareVTab(fmt.Sprintf(`
	CREATE TABLE %q (
		repoName HIDDEN,
		id TEXT,
		message TEXT,
		summary TEXT,
		author_name TEXT,
		author_email TEXT,
		author_when DATETIME,
		committer_name TEXT,
		committer_email TEXT,
		committer_when DATETIME, 
		parent_id TEXT,
		parent_count INT,
		tree_id TEXT
	)`, args[0]))
	if err != nil {
		return nil, err
	}

	// the repoPath will be enclosed in double quotes "..." since ensureTables uses %q when setting up the table
	// we need to pop those off when referring to the actual directory in the fs
	return &commitsTable{m}, nil
}

func (m *GitLogModule) DestroyModule() {}

func (v *commitsTable) Open() (sqlite3.VTabCursor, error) {

	return &commitCursor{nil, "", nil, nil}, nil
}

func (v *commitsTable) Disconnect() error {
	return nil
}
func (v *commitsTable) Destroy() error { return nil }

type commitCursor struct {
	repo       *git.Repository
	repoName   string
	current    *git.Commit
	commitIter *git.RevWalk
}

func (vc *commitCursor) Column(c *sqlite3.SQLiteContext, col int) error {
	//print("column")
	commit := vc.current
	author := commit.Author()
	committer := commit.Committer()

	switch col {
	case 0:
		c.ResultText(vc.repoName)
	case 1:
		//commit id
		c.ResultText(commit.Id().String())
	case 2:
		//commit message
		c.ResultText(commit.Message())
	case 3:
		//commit summary
		c.ResultText(commit.Summary())
	case 4:
		//commit author name
		c.ResultText(author.Name)
	case 5:
		//commit author email
		c.ResultText(author.Email)
	case 6:
		//author when
		c.ResultText(author.When.Format(time.RFC3339Nano))
	case 7:
		//committer name
		c.ResultText(committer.Name)
	case 8:
		//committer email
		c.ResultText(committer.Email)
	case 9:
		//committer when
		c.ResultText(committer.When.Format(time.RFC3339Nano))
	case 10:
		//parent_id
		if int(commit.ParentCount()) > 0 {
			p := commit.Parent(0)
			c.ResultText(p.Id().String())
			p.Free()
		} else {
			c.ResultNull()
		}
	case 11:
		//parent_count
		c.ResultInt(int(commit.ParentCount()))
	case 12:
		//tree_id
		c.ResultText(commit.TreeId().String())
	}
	return nil
}

func (v *commitsTable) BestIndex(cst []sqlite3.InfoConstraint, ob []sqlite3.InfoOrderBy) (*sqlite3.IndexResult, error) {
	//print("index")
	used := make([]bool, len(cst))
	// TODO this loop construct won't work well for multiple constraints...
	for c, constraint := range cst {
		switch {
		case constraint.Usable && constraint.Column == 0 && constraint.Op == sqlite3.OpEQ:
			used[c] = true
			return &sqlite3.IndexResult{Used: used, IdxNum: 1, IdxStr: "commit-by-id", EstimatedCost: 1.0, EstimatedRows: 1}, nil
		}
	}
	//return &sqlite3.IndexResult{Used: used, EstimatedCost: 100}, nil
	return &sqlite3.IndexResult{
		IdxNum: 1,
		IdxStr: "default",
		Used:   used,
	}, nil
}

func (vc *commitCursor) Filter(idxNum int, idxStr string, vals []interface{}) error {
	//print("filter")
	vc.repoName = vals[0].(string)
	var (
		dir string

		err error
	)
	if remote, err := vcsurl.Parse(vc.repoName); err == nil { // if it can be parsed
		dir, err = ioutil.TempDir("", "repo")

		if err != nil {
			return err
		}
		cloneOptions := CreateAuthenticationCallback(remote)
		_, err = git.Clone(vc.repoName, dir, cloneOptions)
		if err != nil {
			//print(err)
			return err
		}

		defer func() {
			_ = os.RemoveAll(dir)
		}()
	}
	//println(dir)

	if dir == "" {
		dir, err = filepath.Abs(vc.repoName)
	} else {
		dir, err = filepath.Abs(dir)
	}
	if err != nil {
		return err
	}
	//println(dir)
	//println(vc.repoName)
	repo, err := git.OpenRepository(dir)
	if err != nil {
		return err
	}
	vc.repo = repo
	//println(vc.repoName, vc.repo)
	//print(idxNum)
	switch idxNum - 1 {
	case 0:
		// no index is used, walk over all commits
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

		vc.current = commit
	case 1:
		// commit-by-id - lookup a commit by the ID used in the query
		revWalk, err := vc.repo.Walk()
		if err != nil {
			return err
		}
		// nothing is pushed to this revWalk
		vc.commitIter = revWalk

		id, err := git.NewOid(vals[0].(string))
		if err != nil {
			return err
		}
		commit, err := vc.repo.LookupCommit(id)
		if err != nil {
			return err
		}
		vc.current = commit
	}

	return nil
}

func (vc *commitCursor) Next() error {
	//print("next")
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

func (vc *commitCursor) EOF() bool {
	return vc.current == nil
}

func (vc *commitCursor) Rowid() (int64, error) {
	return int64(0), nil
}

func (vc *commitCursor) Close() error {
	// vc.commitIter.Free()
	// vc.repo.Free()
	return nil
}
func CreateAuthenticationCallback(remote *vcsurl.VCS) *git.CloneOptions {
	//print("callback")
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
