package ghqlite

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/google/go-github/github"
	"github.com/mattn/go-sqlite3"
	"golang.org/x/time/rate"
)

type IssuesModule struct {
	options IssuesModuleOptions
}

type IssuesModuleOptions struct {
	Token       string
	RateLimiter *rate.Limiter
}

func NewIssuesModule(options IssuesModuleOptions) *IssuesModule {
	if options.RateLimiter == nil {
		options.RateLimiter = rate.NewLimiter(rate.Every(time.Second), 2)
	}

	return &IssuesModule{options}
}

func (m *IssuesModule) EponymousOnlyModule() {}

func (m *IssuesModule) Create(c *sqlite3.SQLiteConn, args []string) (sqlite3.VTab, error) {
	err := c.DeclareVTab(fmt.Sprintf(`
		CREATE TABLE %s (
			repo_owner HIDDEN,
			repo_name HIDDEN,
			id INT,
			node_id TEXT PRIMARY KEY,
			number INT,
			author TEXT,
			body TEXT,
			body_text TEXT,
			closed BOOL,
			closed_at DATETIME,
			comments INT,
			created_at DATETIME,
			created_via_email BOOL,
			editor_login TEXT,
			includes_created_edit BOOL,
			is_read_by_viewer BOOL,
			labels INT,
			last_edited_at DATETIME,
			locked BOOL,
			milestone_number INT,
			milestone_progress REAL,
			participants_count INT,
			published_at DATETIME,
			reactions_count INT,
			state TEXT,
			title TEXT,
			updated_at DATETIME,
			url TEXT,
			user_content_edits_count INT,
			viewer_can_subscribe BOOL,
			viewer_can_update BOOL,
			viewer_did_author BOOL,
			viewer_subscription TEXT		
		) WITHOUT ROWID`, args[0]))
	if err != nil {
		return nil, err
	}

	return &issuesTable{m}, nil
}

func (m *IssuesModule) Connect(c *sqlite3.SQLiteConn, args []string) (sqlite3.VTab, error) {
	return m.Create(c, args)
}

func (m *IssuesModule) DestroyModule() {}

type issuesTable struct {
	module *IssuesModule
}

func (v *issuesTable) Open() (sqlite3.VTabCursor, error) {
	return &issuesCursor{v, "", "", nil, nil, false}, nil
}

func (v *issuesTable) Disconnect() error { return nil }
func (v *issuesTable) Destroy() error    { return nil }

type issuesCursor struct {
	table        *issuesTable
	repoOwner    string
	repoName     string
	iter         *RepoIssueIterator
	currentIssue *issue
	eof          bool
}

func (vc *issuesCursor) Column(c *sqlite3.SQLiteContext, col int) error {
	issue := vc.currentIssue
	repo := vc.iter.repo
	switch col {
	case 0:
		c.ResultText(vc.repoOwner)
	case 1:
		c.ResultText(vc.repoName)
	case 2:
		c.ResultInt64(int64(repo.Repository.DatabaseID))
	case 3:
		c.ResultInt64(int64(issue.DatabaseId))
	case 4:
		c.ResultInt(issue.Number)
	case 5:
		c.ResultText(issue.Author.Login)
	case 6:
		c.ResultText(issue.Body)
	case 7:
		c.ResultText(issue.BodyText)
	case 8:
		c.ResultBool(issue.Closed)
	case 9:
		t := issue.ClosedAt.Time
		if t.IsZero() {
			c.ResultNull()
		} else {
			c.ResultText(t.Format(time.RFC3339Nano))
		}
	case 10:

		c.ResultInt(issue.Comments.TotalCount)
	case 11:
		t := issue.CreatedAt.Time
		if t.IsZero() {
			c.ResultNull()
		} else {
			c.ResultText(t.Format(time.RFC3339Nano))
		}
	case 12:
		c.ResultBool(issue.CreatedViaEmail)
	case 13:
		c.ResultText(issue.Editor.Login)
	case 14:
		c.ResultBool(issue.IncludesCreatedEdit)
	case 15:
		c.ResultBool(issue.IsReadByViewer)
	case 16:
		c.ResultInt(issue.Labels.TotalCount)
	case 17:
		t := issue.LastEditedAt.Time
		if t.IsZero() {
			c.ResultNull()
		} else {
			c.ResultText(t.Format(time.RFC3339Nano))
		}
	case 18:
		c.ResultBool(issue.Locked)
	case 19:
		c.ResultInt(issue.Milestone.Number)
	case 20:
		c.ResultDouble(float64(issue.Milestone.ProgressPercentage))
	case 21:
		c.ResultInt(issue.Participants.TotalCount)
	case 22:
		t := issue.PublishedAt.Time
		if t.IsZero() {
			c.ResultNull()
		} else {
			c.ResultText(t.Format(time.RFC3339Nano))
		}
	case 23:
		c.ResultInt(issue.Reactions.TotalCount)
	case 24:
		c.ResultText(string(issue.State))
	case 25:
		c.ResultText(issue.Title)
	case 26:
		t := issue.UpdatedAt.Time
		if t.IsZero() {
			c.ResultNull()
		} else {
			c.ResultText(t.Format(time.RFC3339Nano))
		}
	case 27:
		c.ResultText(issue.Url.RawPath)
	case 28:
		c.ResultInt(issue.UserContentEdits.TotalCount)
	case 29:
		c.ResultBool(issue.ViewerCanSubscribe)
	case 30:
		c.ResultBool(issue.ViewerCanUpdate)
	case 31:
		c.ResultBool(issue.ViewerDidAuthor)
	case 32:
		c.ResultText(string(issue.ViewerSubscription))
	}

	return nil
}

func (v *issuesTable) BestIndex(constraints []sqlite3.InfoConstraint, ob []sqlite3.InfoOrderBy) (*sqlite3.IndexResult, error) {
	used := make([]bool, len(constraints))
	var repoOwnerCstUsed, repoNameCstUsed bool
	idxNameVals := make([]string, 0)
	cost := 1000.0
	for c, cst := range constraints {
		if !cst.Usable {
			continue
		}
		if cst.Op != sqlite3.OpEQ {
			continue
		}
		switch cst.Column {
		case 0: // repo_owner
			used[c] = true
			repoOwnerCstUsed = true
			idxNameVals = append(idxNameVals, "repo_owner")
		case 1: // repo_name
			used[c] = true
			repoNameCstUsed = true
			idxNameVals = append(idxNameVals, "repo_name")
		case 5:
			used[c] = true
			idxNameVals = append(idxNameVals, "state")
		}
	}

	if repoOwnerCstUsed && repoNameCstUsed {
		cost = 0
	}

	var idxNum int
	var alreadyOrdered bool
	if len(ob) == 1 {
		switch ob[0].Column {
		case 12: // created_at
			alreadyOrdered = true
			if ob[0].Desc {
				idxNum = -ob[0].Column
			} else {
				idxNum = ob[0].Column
			}
		}
	}

	return &sqlite3.IndexResult{
		IdxNum:         idxNum,
		IdxStr:         strings.Join(idxNameVals, ","),
		Used:           used,
		AlreadyOrdered: alreadyOrdered,
		EstimatedCost:  cost,
	}, nil
}

func (vc *issuesCursor) Filter(idxNum int, idxStr string, vals []interface{}) error {
	vc.eof = false
	state := "all"
	for c, cstVal := range strings.Split(idxStr, ",") {
		switch cstVal {
		case "repo_owner":
			vc.repoOwner = vals[c].(string)
		case "repo_name":
			vc.repoName = vals[c].(string)
		case "state":
			state = vals[c].(string)
		}
	}

	orderBy := "created"
	switch math.Abs(float64(idxNum)) {
	case 12:
		orderBy = "created"
	case 13:
		orderBy = "updated"
	}

	var direction string
	if idxNum <= 0 {
		direction = "desc"
	} else {
		direction = "asc"
	}

	iter := NewRepoIssueIterator(vc.repoOwner, vc.repoName, RepoIssueIteratorOptions{
		GitHubIteratorOptions: GitHubIteratorOptions{
			Token:        vc.table.module.options.Token,
			PerPage:      100,
			PreloadPages: 5,
			RateLimiter:  vc.table.module.options.RateLimiter,
		},
		IssueListByRepoOptions: github.IssueListByRepoOptions{
			State:     state,
			Sort:      orderBy,
			Direction: direction,
		},
	})
	vc.iter = iter
	return vc.Next()
}

func (vc *issuesCursor) Next() error {
	nextissue, err := vc.iter.Next()

	if err != nil {
		return err
	}
	if nextissue == nil {
		vc.eof = true
		return nil
	}
	vc.currentIssue = nextissue
	return nil
}

func (vc *issuesCursor) EOF() bool {
	return vc.eof
}

func (vc *issuesCursor) Rowid() (int64, error) {
	return int64(vc.currentIssue.DatabaseId), nil
}

func (vc *issuesCursor) Close() error {
	return nil
}
