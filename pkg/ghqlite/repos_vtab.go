package ghqlite

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/go-github/github"
	"github.com/mattn/go-sqlite3"
	"golang.org/x/time/rate"
)

type ReposModule struct{}

func NewReposModule() *ReposModule {
	return &ReposModule{}
}

func (m *ReposModule) Create(c *sqlite3.SQLiteConn, args []string) (sqlite3.VTab, error) {
	err := c.DeclareVTab(fmt.Sprintf(`
		CREATE TABLE %s (
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

	return &reposTable{args[3], args[4][1 : len(args[4])-1]}, nil
}

func (m *ReposModule) Connect(c *sqlite3.SQLiteConn, args []string) (sqlite3.VTab, error) {
	return m.Create(c, args)
}

func (m *ReposModule) DestroyModule() {}

type reposTable struct {
	owner string
	token string
}

func (v *reposTable) Open() (sqlite3.VTabCursor, error) {
	return &reposCursor{v, nil, nil, false}, nil
}

func (v *reposTable) BestIndex(cst []sqlite3.InfoConstraint, ob []sqlite3.InfoOrderBy) (*sqlite3.IndexResult, error) {
	return &sqlite3.IndexResult{}, nil
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
		c.ResultInt64(repo.GetID())
	case 1:
		c.ResultText(repo.GetNodeID())
	case 2:
		c.ResultText(repo.GetName())
	case 3:
		c.ResultText(repo.GetFullName())
	case 4:
		c.ResultText(repo.GetOwner().GetLogin())
	case 5:
		c.ResultBool(repo.GetPrivate())
	case 6:
		c.ResultText(repo.GetDescription())
	case 7:
		c.ResultBool(repo.GetFork())
	case 8:
		c.ResultText(repo.GetHomepage())
	case 9:
		c.ResultText(repo.GetLanguage())
	case 10:
		c.ResultInt(repo.GetForksCount())
	case 11:
		c.ResultInt(repo.GetStargazersCount())
	case 12:
		c.ResultInt(repo.GetWatchersCount())
	case 13:
		c.ResultInt(repo.GetSize())
	case 14:
		c.ResultText(repo.GetDefaultBranch())
	case 15:
		c.ResultInt(repo.GetOpenIssuesCount())
	case 16:
		str, err := json.Marshal(repo.Topics)
		if err != nil {
			return err
		}
		c.ResultText(string(str))
	case 17:
		c.ResultBool(repo.GetHasIssues())
	case 18:
		c.ResultBool(repo.GetHasProjects())
	case 19:
		c.ResultBool(repo.GetHasWiki())
	case 20:
		c.ResultBool(repo.GetHasPages())
	case 21:
		c.ResultBool(repo.GetHasDownloads())
	case 22:
		c.ResultBool(repo.GetArchived())
	case 23:
		c.ResultText(repo.PushedAt.Format(time.RFC3339Nano))
	case 24:
		c.ResultText(repo.CreatedAt.Format(time.RFC3339Nano))
	case 25:
		c.ResultText(repo.UpdatedAt.Format(time.RFC3339Nano))
	case 26:
		str, err := json.Marshal(repo.GetPermissions())
		if err != nil {
			return err
		}
		c.ResultText(string(str))
	}
	return nil
}

func (vc *reposCursor) Filter(idxNum int, idxStr string, vals []interface{}) error {
	var rateLimiter *rate.Limiter
	if vc.table.token == "" {
		rateLimiter = rate.NewLimiter(rate.Every(time.Minute), 30)
	} else {
		rateLimiter = rate.NewLimiter(rate.Every(time.Minute), 80)
	}
	iter := NewRepoIterator(vc.table.owner, OwnerTypeOrganization, vc.table.token, &RepoIteratorOptions{
		PerPage:      100,
		PreloadPages: 10,
		RateLimiter:  rateLimiter,
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
