package ghqlite

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/go-github/github"
	"github.com/mattn/go-sqlite3"
	"golang.org/x/time/rate"
)

type ReposModule struct {
	ownerType OwnerType
	options   ReposModuleOptions
}

func NewReposModule(ownerType OwnerType, options ReposModuleOptions) *ReposModule {
	if options.RateLimiter == nil {
		if options.RateLimiter == nil {
			options.RateLimiter = rate.NewLimiter(rate.Every(time.Second), 2)
		}
	}
	return &ReposModule{ownerType, options}
}

type ReposModuleOptions struct {
	Token       string
	RateLimiter *rate.Limiter
}

func (m *ReposModule) EponymousOnlyModule() {}

func (m *ReposModule) Create(c *sqlite3.SQLiteConn, args []string) (sqlite3.VTab, error) {
	err := c.DeclareVTab(fmt.Sprintf(`
		CREATE TABLE %s (
			repo_owner HIDDEN,
			id INT,
			node_id TEXT,
			name TEXT,
			full_name TEXT,
			owner TEXT,
			private BOOL,
			description TEXT,
			fork BOOL,
			homepage TEXT,
			language TEXT,
			forks_count INT,
			stargazers_count INT,
			watchers_count INT,
			size INT,
			default_branch TEXT,
			open_issues_count INT,
			topics TEXT,
			has_issues BOOL,
			has_projects BOOL,
			has_wiki BOOL,
			has_pages BOOL,
			has_downloads BOOL,
			archived BOOL,
			pushed_at DATETIME,
			created_at DATETIME,
			updated_at DATETIME,
			permissions TEXT
		)`, args[0]))
	if err != nil {
		return nil, err
	}
	return &reposTable{m}, nil
}

func (m *ReposModule) Connect(c *sqlite3.SQLiteConn, args []string) (sqlite3.VTab, error) {
	return m.Create(c, args)
}

func (m *ReposModule) DestroyModule() {}

type reposTable struct {
	module *ReposModule
}

func (v *reposTable) Open() (sqlite3.VTabCursor, error) {
	return &reposCursor{v, nil, nil, false}, nil
}

func (v *reposTable) Disconnect() error { return nil }
func (v *reposTable) Destroy() error    { return nil }

type reposCursor struct {
	table       *reposTable
	iter        *RepoIterator
	currentRepo *github.Repository
	eof         bool
}

func (vc *reposCursor) Column(c *sqlite3.SQLiteContext, col int) error {
	repo := vc.currentRepo
	switch col {
	case 0:
		c.ResultText(repo.GetOwner().GetName())
	case 1:
		c.ResultInt64(repo.GetID())
	case 2:
		c.ResultText(repo.GetNodeID())
	case 3:
		c.ResultText(repo.GetName())
	case 4:
		c.ResultText(repo.GetFullName())
	case 5:
		c.ResultText(repo.GetOwner().GetLogin())
	case 6:
		c.ResultBool(repo.GetPrivate())
	case 7:
		c.ResultText(repo.GetDescription())
	case 8:
		c.ResultBool(repo.GetFork())
	case 9:
		c.ResultText(repo.GetHomepage())
	case 10:
		c.ResultText(repo.GetLanguage())
	case 11:
		c.ResultInt(repo.GetForksCount())
	case 12:
		c.ResultInt(repo.GetStargazersCount())
	case 13:
		c.ResultInt(repo.GetWatchersCount())
	case 14:
		c.ResultInt(repo.GetSize())
	case 15:
		c.ResultText(repo.GetDefaultBranch())
	case 16:
		c.ResultInt(repo.GetOpenIssuesCount())
	case 17:
		str, err := json.Marshal(repo.Topics)
		if err != nil {
			return err
		}
		c.ResultText(string(str))
	case 18:
		c.ResultBool(repo.GetHasIssues())
	case 19:
		c.ResultBool(repo.GetHasProjects())
	case 20:
		c.ResultBool(repo.GetHasWiki())
	case 21:
		c.ResultBool(repo.GetHasPages())
	case 22:
		c.ResultBool(repo.GetHasDownloads())
	case 23:
		c.ResultBool(repo.GetArchived())
	case 24:
		c.ResultText(repo.PushedAt.Format(time.RFC3339Nano))
	case 25:
		c.ResultText(repo.CreatedAt.Format(time.RFC3339Nano))
	case 26:
		c.ResultText(repo.UpdatedAt.Format(time.RFC3339Nano))
	case 27:
		str, err := json.Marshal(repo.GetPermissions())
		if err != nil {
			return err
		}
		c.ResultText(string(str))
	}
	return nil
}

func (v *reposTable) BestIndex(constraints []sqlite3.InfoConstraint, ob []sqlite3.InfoOrderBy) (*sqlite3.IndexResult, error) {
	used := make([]bool, len(constraints))
	repoOwnerCstUsed := false
	for c, cst := range constraints {
		switch cst.Column {
		case 0: // repo_owner
			if cst.Op != sqlite3.OpEQ { // if there's no equality constraint, fail
				return nil, sqlite3.ErrConstraint
			}
			// if the constraint is usable, use it, otherwise fail
			if used[c] = cst.Usable; !used[c] {
				return nil, sqlite3.ErrConstraint
			}
			repoOwnerCstUsed = true
		}
	}

	if !repoOwnerCstUsed {
		return nil, errors.New("must supply a repo owner")
	}

	return &sqlite3.IndexResult{
		IdxNum: 0,
		IdxStr: "default",
		Used:   used,
	}, nil
}

func (vc *reposCursor) Filter(idxNum int, idxStr string, vals []interface{}) error {
	owner := vals[0].(string)
	iter := NewRepoIterator(owner, vc.table.module.ownerType, GitHubIteratorOptions{
		Token:        vc.table.module.options.Token,
		PerPage:      100,
		PreloadPages: 5,
		RateLimiter:  vc.table.module.options.RateLimiter,
	})
	vc.iter = iter
	return vc.Next()
}

func (vc *reposCursor) Next() error {
	nextRepo, err := vc.iter.Next()
	if err != nil {
		return err
	}
	if nextRepo == nil {
		vc.eof = true
		return nil
	}
	vc.currentRepo = nextRepo
	return nil
}

func (vc *reposCursor) EOF() bool {
	return vc.eof
}

func (vc *reposCursor) Rowid() (int64, error) {
	return vc.currentRepo.GetID(), nil
}

func (vc *reposCursor) Close() error {
	return nil
}
