package ghqlite

import (
	"encoding/json"
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
			state TEXT,
			locked BOOL,
			title TEXT,
			user_login TEXT,
			body TEXT,
			labels TEXT,
			active_lock_reason TEXT,
			created_at DATETIME,
			updated_at DATETIME,
			closed_at DATETIME,
			merged_at DATETIME,
			merge_commit_sha TEXT,
			assignee_login TEXT,
			assignees TEXT,
			url TEXT,
			html_url TEXT,
			comments_url TEXT,
			events_url TEXT,
			labels_url TEXT,
			repository_url TEXT,
			comments INT,
			milestone TEXT,
			reactions INT			
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
	currentIssue *github.Issue
	eof          bool
}

func (vc *issuesCursor) Column(c *sqlite3.SQLiteContext, col int) error {
	issue := vc.currentIssue
	switch col {
	case 0:
		c.ResultText(vc.repoOwner)
	case 1:
		c.ResultText(vc.repoName)
	case 2:
		c.ResultInt64(issue.GetID())
	case 3:
		c.ResultText(issue.GetNodeID())
	case 4:
		c.ResultInt(issue.GetNumber())
	case 5:
		c.ResultText(issue.GetState())
	case 6:
		c.ResultBool(issue.GetActiveLockReason() != "")
	case 7:
		c.ResultText(issue.GetTitle())
	case 8:
		c.ResultText(issue.GetUser().GetLogin())
	case 9:
		c.ResultText(issue.GetBody())
	case 10:
		str, err := json.Marshal(issue.Labels)
		if err != nil {
			return err
		}
		c.ResultText(string(str))
	case 11:
		c.ResultText(issue.GetActiveLockReason())
	case 12:
		t := issue.GetCreatedAt()
		if t.IsZero() {
			c.ResultNull()
		} else {
			c.ResultText(t.Format(time.RFC3339Nano))
		}
	case 13:
		t := issue.GetUpdatedAt()
		if t.IsZero() {
			c.ResultNull()
		} else {
			c.ResultText(t.Format(time.RFC3339Nano))
		}
	case 14:
		t := issue.GetClosedAt()
		if t.IsZero() {
			c.ResultNull()
		} else {
			c.ResultText(t.Format(time.RFC3339Nano))
		}
	case 15:
		t := issue.GetUpdatedAt()
		if t.IsZero() {
			c.ResultNull()
		} else {
			c.ResultText(t.Format(time.RFC3339Nano))
		}
	case 16:
		c.ResultText(issue.GetClosedBy().GetLogin())
	case 17:
		c.ResultText(issue.GetAssignee().GetLogin())
	case 18:
		str, err := json.Marshal(issue.Assignees)
		if err != nil {
			return err
		}
		c.ResultText(string(str))
	case 19:
		c.ResultText(issue.GetURL())
	case 20:
		c.ResultText(issue.GetHTMLURL())
	case 21:
		c.ResultText(issue.GetCommentsURL())
	case 22:
		c.ResultText(issue.GetEventsURL())
	case 23:
		c.ResultText(issue.GetLabelsURL())
	case 24:
		c.ResultText(issue.GetRepositoryURL())
	case 25:
		c.ResultInt(issue.GetComments())
	case 26:
		c.ResultText(issue.GetMilestone().GetDescription())
	case 27:
		c.ResultInt(issue.GetReactions().GetTotalCount())
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
	return vc.currentIssue.GetID(), nil
}

func (vc *issuesCursor) Close() error {
	return nil
}
