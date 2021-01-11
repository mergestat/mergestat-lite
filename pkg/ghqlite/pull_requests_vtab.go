package ghqlite

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/google/go-github/github"
	"github.com/mattn/go-sqlite3"
	"golang.org/x/time/rate"
)

type PullRequestsModule struct {
	options PullRequestsModuleOptions
}

type PullRequestsModuleOptions struct {
	Token       string
	RateLimiter *rate.Limiter
}

func NewPullRequestsModule(options PullRequestsModuleOptions) *PullRequestsModule {
	if options.RateLimiter == nil {
		options.RateLimiter = rate.NewLimiter(rate.Every(time.Second), 2)
	}

	return &PullRequestsModule{options}
}

func (m *PullRequestsModule) EponymousOnlyModule() {}

func (m *PullRequestsModule) Create(c *sqlite3.SQLiteConn, args []string) (sqlite3.VTab, error) {
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
			requested_reviewer_logins TEXT,
			head_label TEXT,
			head_ref TEXT,
			head_sha TEXT,
			head_repo_owner TEXT,
			head_repo_name TEXT,
			base_label TEXT,
			base_ref TEXT,
			base_sha TEXT,
			base_repo_owner TEXT,
			base_repo_name TEXT,
			author_association TEXT,
			merged BOOL,
			mergeable BOOL,
			mergeable_state BOOL,
			merged_by_login TEXT,
			comments INT,
			maintainer_can_modify BOOL,
			commits INT,
			additions INT,
			deletions INT,
			changed_files INT
		) WITHOUT ROWID`, args[0]))
	if err != nil {
		return nil, err
	}

	return &pullRequestsTable{m}, nil
}

func (m *PullRequestsModule) Connect(c *sqlite3.SQLiteConn, args []string) (sqlite3.VTab, error) {
	return m.Create(c, args)
}

func (m *PullRequestsModule) DestroyModule() {}

type pullRequestsTable struct {
	module *PullRequestsModule
}

func (v *pullRequestsTable) Open() (sqlite3.VTabCursor, error) {
	return &pullRequestsCursor{v, "", "", nil, nil, false, false}, nil
}

func (v *pullRequestsTable) Disconnect() error { return nil }
func (v *pullRequestsTable) Destroy() error    { return nil }

type pullRequestsCursor struct {
	table              *pullRequestsTable
	repoOwner          string
	repoName           string
	iter               *RepoPullRequestIterator
	currentPR          *github.PullRequest
	extraFieldsFetched bool
	eof                bool
}

// TODO this is a little odd, but some fields of a PR are not available when the PR
// is retrieved from a .List call (.../pulls), but they're useful to have in the table
// this retrieves the PR as a single .Get (.../pull/:number), which does return the "extra" fields
// this is likely a case where using the GraphQL API would benefit, as we wouldn't have to make
// an additional API call for every row (PR) in the table, when accessing any "extra" fields
// this should still respect the rate limit of the iterator
func (vc *pullRequestsCursor) getCurrentPRExtraFields() (*github.PullRequest, error) {
	if !vc.extraFieldsFetched {
		err := vc.iter.githubIter.options.RateLimiter.Wait(context.Background())
		if err != nil {
			return nil, err
		}

		pr, _, err := vc.iter.githubIter.options.Client.PullRequests.Get(context.Background(), vc.repoOwner, vc.repoName, vc.currentPR.GetNumber())
		if err != nil {
			return nil, err
		}

		vc.currentPR = pr
		vc.extraFieldsFetched = true
	}

	return vc.currentPR, nil
}

func (vc *pullRequestsCursor) Column(c *sqlite3.SQLiteContext, col int) error {
	pr := vc.currentPR
	switch col {
	case 0:
		c.ResultText(vc.repoOwner)
	case 1:
		c.ResultText(vc.repoName)
	case 2:
		c.ResultInt64(pr.GetID())
	case 3:
		c.ResultText(pr.GetNodeID())
	case 4:
		c.ResultInt(pr.GetNumber())
	case 5:
		c.ResultText(pr.GetState())
	case 6:
		c.ResultBool(pr.GetActiveLockReason() != "")
	case 7:
		c.ResultText(pr.GetTitle())
	case 8:
		c.ResultText(pr.GetUser().GetLogin())
	case 9:
		c.ResultText(pr.GetBody())
	case 10:
		str, err := json.Marshal(pr.Labels)
		if err != nil {
			return err
		}
		c.ResultText(string(str))
	case 11:
		c.ResultText(pr.GetActiveLockReason())
	case 12:
		t := pr.GetCreatedAt()
		if t.IsZero() {
			c.ResultNull()
		} else {
			c.ResultText(t.Format(time.RFC3339Nano))
		}
	case 13:
		t := pr.GetUpdatedAt()
		if t.IsZero() {
			c.ResultNull()
		} else {
			c.ResultText(t.Format(time.RFC3339Nano))
		}
	case 14:
		t := pr.GetClosedAt()
		if t.IsZero() {
			c.ResultNull()
		} else {
			c.ResultText(t.Format(time.RFC3339Nano))
		}
	case 15:
		t := pr.GetMergedAt()
		if t.IsZero() {
			c.ResultNull()
		} else {
			c.ResultText(t.Format(time.RFC3339Nano))
		}
	case 16:
		c.ResultText(pr.GetMergeCommitSHA())
	case 17:
		c.ResultText(pr.GetAssignee().GetLogin())
	case 18:
		str, err := json.Marshal(pr.Assignees)
		if err != nil {
			return err
		}
		c.ResultText(string(str))
	case 19:
		str, err := json.Marshal(pr.RequestedReviewers)
		if err != nil {
			return err
		}
		c.ResultText(string(str))
	case 20:
		c.ResultText(pr.GetHead().GetLabel())
	case 21:
		c.ResultText(pr.GetHead().GetRef())
	case 22:
		c.ResultText(pr.GetHead().GetSHA())
	case 23:
		c.ResultText(pr.Head.GetRepo().GetOwner().GetLogin())
	case 24:
		c.ResultText(pr.Head.GetRepo().GetName())
	case 25:
		c.ResultText(pr.GetBase().GetSHA())
	case 26:
		c.ResultText(pr.GetBase().GetRef())
	case 27:
		c.ResultText(pr.GetBase().GetSHA())
	case 28:
		c.ResultText(pr.Base.GetRepo().GetOwner().GetLogin())
	case 29:
		c.ResultText(pr.Base.GetRepo().GetName())
	case 30:
		c.ResultText(pr.GetAuthorAssociation())
	case 31:
		pr, err := vc.getCurrentPRExtraFields()
		if err != nil {
			return err
		}
		c.ResultBool(pr.GetMerged())
	case 32:
		pr, err := vc.getCurrentPRExtraFields()
		if err != nil {
			return err
		}
		c.ResultBool(pr.GetMergeable())
	case 33:
		pr, err := vc.getCurrentPRExtraFields()
		if err != nil {
			return err
		}
		c.ResultText(pr.GetMergeableState())
	case 34:
		pr, err := vc.getCurrentPRExtraFields()
		if err != nil {
			return err
		}
		c.ResultText(pr.GetMergedBy().GetLogin())
	case 35:
		pr, err := vc.getCurrentPRExtraFields()
		if err != nil {
			return err
		}
		c.ResultInt(pr.GetComments())
	case 36:
		c.ResultBool(pr.GetMaintainerCanModify())
	case 37:
		pr, err := vc.getCurrentPRExtraFields()
		if err != nil {
			return err
		}
		c.ResultInt(pr.GetCommits())
	case 38:
		pr, err := vc.getCurrentPRExtraFields()
		if err != nil {
			return err
		}
		c.ResultInt(pr.GetAdditions())
	case 39:
		pr, err := vc.getCurrentPRExtraFields()
		if err != nil {
			return err
		}
		c.ResultInt(pr.GetDeletions())
	case 40:
		pr, err := vc.getCurrentPRExtraFields()
		if err != nil {
			return err
		}
		c.ResultInt(pr.GetChangedFiles())
	}

	return nil
}

func (v *pullRequestsTable) BestIndex(constraints []sqlite3.InfoConstraint, ob []sqlite3.InfoOrderBy) (*sqlite3.IndexResult, error) {
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

func (vc *pullRequestsCursor) Filter(idxNum int, idxStr string, vals []interface{}) error {
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

	iter := NewRepoPullRequestIterator(vc.repoOwner, vc.repoName, RepoPullRequestIteratorOptions{
		GitHubIteratorOptions: GitHubIteratorOptions{
			Token:        vc.table.module.options.Token,
			PerPage:      100,
			PreloadPages: 5,
			RateLimiter:  vc.table.module.options.RateLimiter,
		},
		PullRequestListOptions: github.PullRequestListOptions{
			State:     state,
			Sort:      orderBy,
			Direction: direction,
		},
	})
	vc.iter = iter
	return vc.Next()
}

func (vc *pullRequestsCursor) Next() error {
	nextPR, err := vc.iter.Next()
	if err != nil {
		return err
	}
	if nextPR == nil {
		vc.eof = true
		return nil
	}
	vc.currentPR = nextPR
	vc.extraFieldsFetched = false

	return nil
}

func (vc *pullRequestsCursor) EOF() bool {
	return vc.eof
}

func (vc *pullRequestsCursor) Rowid() (int64, error) {
	return vc.currentPR.GetID(), nil
}

func (vc *pullRequestsCursor) Close() error {
	return nil
}
