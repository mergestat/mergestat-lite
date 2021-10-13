package github

import (
	"context"
	"io"
	"strings"

	"github.com/augmentable-dev/vtab"
	"github.com/rs/zerolog"
	"github.com/shurcooL/githubv4"
	"go.riyazali.net/sqlite"
)

/* TODO: Consider adding
* BranchProtectionRuleConflicts
* MatchingRefs
* PushAllowances
* ReviewDismissalAllowances
 */
type protectionRules struct {
	AllowsDeletions   bool
	AllowsForcePushes bool
	Creator           struct {
		Login string
	}
	DatabaseId                     int
	DismissesStaleReviews          bool
	IsAdminEnforced                bool
	Pattern                        string
	RequiredApprovingReviewCount   int
	RequiredStatusCheckContexts    []string
	RequiresApprovingReviews       bool
	RequiresCodeOwnerReviews       bool
	RequiresCommitSignatures       bool
	RequiresConversationResolution bool
	RequiresLinearHistory          bool
	RequiresStatusChecks           bool
	RequiresStrictStatusChecks     bool
	RestrictsPushes                bool
	RestrictsReviewDismissals      bool
}

type fetchBranchProtectionResults struct {
	Edges       []*protectionRules
	HasNextPage bool
	EndCursor   *githubv4.String
}

func (i *iterProtections) fetchProtections(ctx context.Context, startCursor *githubv4.String) (*fetchBranchProtectionResults, error) {
	var Query struct {
		Repository struct {
			Owner struct {
				Login string
			}
			Name                  string
			BranchProtectionRules struct {
				Nodes    []*protectionRules
				PageInfo struct {
					EndCursor   githubv4.String
					HasNextPage bool
				}
			} `graphql:"branchProtectionRules(first: $perpage, after: $protectionCursor)"`
		} `graphql:"repository(owner: $owner, name: $name)"`
	}
	variables := map[string]interface{}{
		"owner":            githubv4.String(i.owner),
		"name":             githubv4.String(i.name),
		"perpage":          githubv4.Int(i.PerPage),
		"protectionCursor": startCursor,
	}

	err := i.Client().Query(ctx, &Query, variables)
	if err != nil {
		return nil, err
	}

	return &fetchBranchProtectionResults{
		Query.Repository.BranchProtectionRules.Nodes,
		Query.Repository.BranchProtectionRules.PageInfo.HasNextPage,
		&Query.Repository.BranchProtectionRules.PageInfo.EndCursor,
	}, nil
}

type iterProtections struct {
	*Options
	owner   string
	name    string
	current int
	results *fetchBranchProtectionResults
}

func (i *iterProtections) logger() *zerolog.Logger {
	logger := i.Logger.With().Int("per-page", i.PerPage).Str("owner", i.owner).Str("name", i.name).Logger()
	return &logger
}

func (i *iterProtections) Column(ctx vtab.Context, c int) error {
	current := i.results.Edges[i.current]
	col := protectionCols[c]

	switch col.Name {
	case "allow_deletions":
		ctx.ResultInt(t1f0(current.AllowsDeletions))
	case "allows_force_pushes":
		ctx.ResultInt(t1f0(current.AllowsForcePushes))
	case "creator_login":
		ctx.ResultText(string(current.Creator.Login))
	case "database_id":
		ctx.ResultInt(current.DatabaseId)
	case "dismisses_stale_reviews":
		ctx.ResultInt(t1f0(current.DismissesStaleReviews))
	case "is_admin_enforced":
		ctx.ResultInt(t1f0(current.IsAdminEnforced))
	case "pattern":
		ctx.ResultText(current.Pattern)
	case "required_approving_review_count":
		ctx.ResultInt(current.RequiredApprovingReviewCount)
	case "required_status_check_contexts":
		ctx.ResultText(strings.Join(current.RequiredStatusCheckContexts, ", "))
	case "requires_approving_reviews":
		ctx.ResultInt(t1f0(current.RequiresApprovingReviews))
	case "requires_code_owners_reviews":
		ctx.ResultInt(t1f0(current.RequiresCodeOwnerReviews))
	case "requires_commit_signature":
		ctx.ResultInt(t1f0(current.RequiresCommitSignatures))
	case "requires_conversation_resolution":
		ctx.ResultInt(t1f0(current.RequiresConversationResolution))
	case "requires_linear_history":
		ctx.ResultInt(t1f0(current.RequiresLinearHistory))
	case "requires_status_checks":
		ctx.ResultInt(t1f0(current.RequiresStatusChecks))
	case "requires_strict_status_checks":
		ctx.ResultInt(t1f0(current.RequiresStrictStatusChecks))
	case "restricts_pushes":
		ctx.ResultInt(t1f0(current.RestrictsPushes))
	case "restricts_review_dismissal":
		ctx.ResultInt(t1f0(current.RestrictsReviewDismissals))
	}
	return nil
}

func (i *iterProtections) Next() (vtab.Row, error) {
	i.current += 1

	if i.results == nil || i.current >= len(i.results.Edges) {
		if i.results == nil || i.results.HasNextPage {
			err := i.RateLimiter.Wait(context.Background())
			if err != nil {
				return nil, err
			}

			var cursor *githubv4.String
			if i.results != nil {
				cursor = i.results.EndCursor
			}

			l := i.logger().With().Interface("cursor", cursor).Logger()
			l.Info().Msgf("fetching page of repo_protections for %s/%s", i.owner, i.name)
			results, err := i.fetchProtections(context.Background(), cursor)
			if err != nil {
				return nil, err
			}

			i.results = results
			i.current = 0

			if len(i.results.Edges) == 0 {
				return nil, io.EOF
			}
		} else {
			return nil, io.EOF
		}
	}

	return i, nil
}

var protectionCols = []vtab.Column{
	{Name: "owner", Type: "TEXT", NotNull: true, Hidden: true, Filters: []*vtab.ColumnFilter{{Op: sqlite.INDEX_CONSTRAINT_EQ, OmitCheck: true}}},
	{Name: "reponame", Type: "TEXT", NotNull: true, Hidden: true, Filters: []*vtab.ColumnFilter{{Op: sqlite.INDEX_CONSTRAINT_EQ, OmitCheck: true}}},
	{Name: "allow_deletions", Type: "BOOLEAN"},
	{Name: "allows_force_pushes", Type: "BOOLEAN"},
	{Name: "creator_login", Type: "TEXT"},
	{Name: "database_id", Type: "INT"},
	{Name: "dismisses_stale_reviews", Type: "BOOLEAN"},
	{Name: "is_admin_enforced", Type: "BOOLEAN"},
	{Name: "pattern", Type: "TEXT"},
	{Name: "required_approving_review_count", Type: "INT"},
	{Name: "required_status_check_contexts", Type: "BOOLEAN"},
	{Name: "requires_approving_reviews", Type: "BOOLEAN"},
	{Name: "requires_code_owners_reviews", Type: "BOOLEAN"},
	{Name: "requires_commit_signature", Type: "BOOLEAN"},
	{Name: "requires_conversation_resolution", Type: "BOOLEAN"},
	{Name: "requires_linear_history", Type: "BOOLEAN"},
	{Name: "requires_status_checks", Type: "BOOLEAN"},
	{Name: "requires_strict_status_checks", Type: "BOOLEAN"},
	{Name: "restricts_pushes", Type: "BOOLEAN"},
	{Name: "restricts_review_dismissal", Type: "BOOLEAN"},
}

func NewProtectionsModule(opts *Options) sqlite.Module {
	return vtab.NewTableFunc("github_repo_prs", protectionCols, func(constraints []*vtab.Constraint, orders []*sqlite.OrderBy) (vtab.Iterator, error) {
		var fullNameOrOwner, name string
		for _, constraint := range constraints {
			if constraint.Op == sqlite.INDEX_CONSTRAINT_EQ {
				switch constraint.ColIndex {
				case 0:
					fullNameOrOwner = constraint.Value.Text()
				case 1:
					name = constraint.Value.Text()
				}
			}
		}

		owner, name, err := repoOwnerAndName(name, fullNameOrOwner)
		if err != nil {
			return nil, err
		}

		iter := &iterProtections{opts, owner, name, -1, nil}
		iter.logger().Info().Msgf("starting GitHub repo_protections iterator for %s/%s", owner, name)
		return iter, nil
	}, vtab.EarlyOrderByConstraintExit(true))
}
