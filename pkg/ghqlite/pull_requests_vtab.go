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
			node_id INT PRIMARY KEY,
			number INT,
			active_lock_reason TEXT,
			assignees TEXT,
			additions INT,
			author TEXT,
			author_association TEXT,
			body TEXT,
			body_text TEXT,
			body_html TEXT,
			base_ref_oid TEXT,
			base_ref_name TEXT,
			baseref_name TEXT,
			base_repository_name TEXT,
			base_repository_owner TEXT,
			can_be_rebased BOOL,
			checks_rescource_path TEXT,
			checks_url TEXT,
			comment_count INT,
			commits_count INT,
			changed_files INT,
			closed BOOL,
			closed_at DATETIME,
			created_at DATETIME,
			created_via_email BOOL,
			deletions INT,
			editor_login TEXT,
			files_count INT,
			head_repository_name TEXT,
			head_repository_owner TEXT,
			head_ref_oid TEXT,
			head_ref_name TEXT,
			includes_created_edit BOOL,
			is_cross_repository BOOL,
			is_draft BOOL,
			labels INT,
			last_edited_at DATETIME,
			locked BOOL,
			merged_at DATETIME,
			merge_commit_oid TEXT,
			merged BOOL,
			mergeable TEXT,
			merged_by TEXT,
			maintainer_can_modify BOOL,
			merge_state_statuses TEXT,
			milestone_number INT,
			participant_count INT,
			permalink TEXT,
			published_at DATETIME,
			review_decision TEXT,
			review_requests TEXT,
			review_threads_count INT,
			reviews_count INT,
			state TEXT,
			title TEXT,
			updated_at DATETIME,
			url TEXT,
			user_content_edits_count INT
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
	currentPR          *pullRequest
	extraFieldsFetched bool
	eof                bool
}

// TODO this is a little odd, but some fields of a PR are not available when the PR
// is retrieved from a .List call (.../pulls), but they're useful to have in the table
// this retrieves the PR as a single .Get (.../pull/:number), which does return the "extra" fields
// this is likely a case where using the GraphQL API would benefit, as we wouldn't have to make
// an additional API call for every row (PR) in the table, when accessing any "extra" fields
// this should still respect the rate limit of the iterator
// func (vc *pullRequestsCursor) getCurrentPRExtraFields() (*github.PullRequest, error) {
// 	if !vc.extraFieldsFetched {
// 		err := vc.iter.githubIter.options.RateLimiter.Wait(context.Background())
// 		if err != nil {
// 			return nil, err
// 		}

// 		pr, _, err := vc.iter.githubIter.options.Client.PullRequests.Get(context.Background(), vc.repoOwner, vc.repoName, vc.currentPR.GetNumber())
// 		if err != nil {
// 			return nil, err
// 		}

// 		vc.currentPR = pr
// 		vc.extraFieldsFetched = true
// 	}

// 	return vc.currentPR, nil
// }

func (vc *pullRequestsCursor) Column(c *sqlite3.SQLiteContext, col int) error {
	pr := vc.currentPR
	repo := vc.iter.repo
	switch col {
	case 0:
		c.ResultText(vc.repoOwner)
	case 1:
		c.ResultText(vc.repoName)
	case 2:
		c.ResultInt64(int64(repo.Repository.DatabaseID))
	case 3:
		c.ResultInt64(int64(pr.DatabaseID))
	case 4:
		c.ResultInt(int(pr.Number))
	case 5:
		c.ResultText(string(pr.ActiveLockReason))
	case 6:
		c.ResultText("Assignees")
	case 7:
		c.ResultInt(int(pr.Additions))
	case 8:
		c.ResultText(pr.Author.Login)
	case 9:
		c.ResultText(string(pr.AuthorAssociation))
	case 10:
		c.ResultText(pr.Body)
	case 11:
		c.ResultText(string(pr.BodyText))
	case 12:
		c.ResultText("BodyHTML")
	case 13:
		c.ResultText(string(pr.BaseRefOid))
	case 14:
		c.ResultText(pr.BaseRefName)
	case 15:
		c.ResultText(pr.BaseRef.Name)
	case 16:
		c.ResultText(pr.BaseRepository.Name)
	case 17:
		c.ResultText(pr.BaseRepository.Owner.Login)
	case 18:
		c.ResultBool(false) //pr.CanBeRebased)
	case 19:
		c.ResultText(pr.ChecksResourcePath.String())
	case 20:

		c.ResultText(pr.ChecksURL.String())
	case 21:

		c.ResultInt(pr.Comments.TotalCount)
	case 22:
		c.ResultInt(pr.Commits.TotalCount)
	case 23:
		c.ResultInt(pr.ChangedFiles)
	case 24:
		c.ResultBool(pr.Closed)
	case 25:
		t := pr.ClosedAt.Time
		if t.IsZero() {
			c.ResultNull()
		} else {
			c.ResultText(t.Format(time.RFC3339Nano))
		}
	case 26:
		t := pr.CreatedAt.Time
		if t.IsZero() {
			c.ResultNull()
		} else {
			c.ResultText(t.Format(time.RFC3339Nano))
		}
	case 27:
		c.ResultBool(pr.CreatedViaEmail)
	case 28:
		c.ResultInt(pr.Deletions)
	case 29:
		c.ResultText(pr.Editor.Login)
	case 30:
		c.ResultInt(pr.Files.TotalCount)
	case 31:
		c.ResultText(pr.HeadRepository.Name)

	case 32:
		c.ResultText(pr.HeadRepositoryOwner.Login)
	case 33:

		c.ResultText(string(pr.HeadRefOid))
	case 34:
		c.ResultText(pr.HeadRefName)
	case 35:

		c.ResultBool(pr.IncludesCreatedEdit)
	case 36:

		c.ResultBool(pr.IsCrossRepository)
	case 37:

		c.ResultBool(pr.IsDraft)
	case 38:
		c.ResultInt(pr.Labels.TotalCount)
	case 39:
		t := pr.LastEditedAt.Time
		if t.IsZero() {
			c.ResultNull()
		} else {
			c.ResultText(t.Format(time.RFC3339Nano))
		}
	case 40:

		c.ResultBool(bool(pr.Locked))
	case 41:
		t := pr.MergedAt.Time
		if t.IsZero() {
			c.ResultNull()
		} else {
			c.ResultText(t.Format(time.RFC3339Nano))
		}
	case 42:
		c.ResultText(string(pr.MergeCommit.Oid))
	case 43:
		c.ResultBool(bool(pr.Merged))
	case 44:
		c.ResultText(string(pr.Mergeable))
	case 45:
		c.ResultText(pr.MergedBy.Login)
	case 46:
		c.ResultBool(bool(pr.MaintainerCanModify))
	case 47:
		c.ResultText("MergeStateStatuses") //pr.MergeStateStatuses)
	case 48:
		c.ResultInt(pr.Milestone.Number)
	case 49:
		c.ResultInt(pr.Participants.TotalCount)
	case 50:
		c.ResultText(pr.Permalink.RawPath)
	case 51:
		t := pr.PublishedAt.Time
		if t.IsZero() {
			c.ResultNull()
		} else {
			c.ResultText(t.Format(time.RFC3339Nano))
		}
	case 52:
		c.ResultText(string(pr.ReviewDecision))
	case 53:
		c.ResultText("Review Requests")
	case 54:
		c.ResultInt(pr.ReviewThreads.TotalCount)
	case 55:
		c.ResultInt(pr.Reviews.TotalCount)
	case 56:
		c.ResultText(string(pr.State))
	case 57:
		c.ResultText(string(pr.Title))
	case 58:
		t := pr.UpdatedAt.Time
		if t.IsZero() {
			c.ResultNull()
		} else {
			c.ResultText(t.Format(time.RFC3339Nano))
		}
	case 59:
		c.ResultText(pr.Url.String())
	case 60:
		c.ResultInt(pr.UserContentEdits.TotalCount)
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
	pr, err := vc.iter.Next()
	if err != nil {
		return err
	}
	if pr == nil {
		vc.eof = true
		return nil
	}
	vc.currentPR = pr
	return nil
}

func (vc *pullRequestsCursor) EOF() bool {
	return vc.eof
}

func (vc *pullRequestsCursor) Rowid() (int64, error) {
	return int64(vc.currentPR.DatabaseID), nil
}

func (vc *pullRequestsCursor) Close() error {
	return nil
}
