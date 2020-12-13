package ghqlite

import (
	"encoding/json"
	"errors"
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
			node_id TEXT,
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
			head_repo_name,
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
		)`, args[0]))
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
	return &pullRequestsCursor{v, "", "", nil, nil, false}, nil
}

func (v *pullRequestsTable) Disconnect() error { return nil }
func (v *pullRequestsTable) Destroy() error    { return nil }

type pullRequestsCursor struct {
	table     *pullRequestsTable
	repoOwner string
	repoName  string
	iter      *RepoPullRequestIterator
	currentPR *github.PullRequest
	eof       bool
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
		c.ResultText(pr.GetCreatedAt().Format(time.RFC3339Nano))
	case 13:
		c.ResultText(pr.GetUpdatedAt().Format(time.RFC3339Nano))
	case 14:
		c.ResultText(pr.GetClosedAt().Format(time.RFC3339Nano))
	case 15:
		c.ResultText(pr.GetMergedAt().Format(time.RFC3339Nano))
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
		c.ResultBool(pr.GetMerged())
	case 32:
		c.ResultBool(pr.GetMergeable())
	case 33:
		c.ResultText(pr.GetMergeableState())
	case 34:
		c.ResultText(pr.GetMergedBy().GetLogin())
	case 35:
		c.ResultInt(pr.GetComments())
	case 36:
		c.ResultBool(pr.GetMaintainerCanModify())
	case 37:
		c.ResultInt(pr.GetCommits())
	case 38:
		c.ResultInt(pr.GetAdditions())
	case 39:
		c.ResultInt(pr.GetDeletions())
	case 40:
		c.ResultInt(pr.GetChangedFiles())
	}

	return nil
}

func (v *pullRequestsTable) BestIndex(constraints []sqlite3.InfoConstraint, ob []sqlite3.InfoOrderBy) (*sqlite3.IndexResult, error) {
	used := make([]bool, len(constraints))
	var repoOwnerCstUsed, repoNameCstUsed bool
	idxNameVals := make([]string, 0)
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
			idxNameVals = append(idxNameVals, "repo_owner")
		case 1: // repo_name
			if cst.Op != sqlite3.OpEQ { // if there's no equality constraint, fail
				return nil, sqlite3.ErrConstraint
			}
			// if the constraint is usable, use it, otherwise fail
			if used[c] = cst.Usable; !used[c] {
				return nil, sqlite3.ErrConstraint
			}
			repoNameCstUsed = true
			idxNameVals = append(idxNameVals, "repo_name")
		case 5:
			if cst.Usable && cst.Op == sqlite3.OpEQ {
				used[c] = true
			}
			idxNameVals = append(idxNameVals, "state")
		}
	}

	if !repoOwnerCstUsed {
		return nil, errors.New("must supply a repo owner")
	}

	if !repoNameCstUsed {
		return nil, errors.New("must supply a repo name")
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
	}, nil
}

func (vc *pullRequestsCursor) Filter(idxNum int, idxStr string, vals []interface{}) error {
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
