package github

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/augmentable-dev/vtab"
	"github.com/rs/zerolog"
	"github.com/shurcooL/githubv4"
	"go.riyazali.net/sqlite"
)

type fetchOrgAuditLogResults struct {
	AuditLogs   []*auditLogEntries
	HasNextPage bool
	EndCursor   *githubv4.String
}
type auditLogEntry struct {
	Typename   string `graphql:"__typename"`
	Id         githubv4.ID
	Action     githubv4.String
	ActorLogin string
	CreatedAt  githubv4.DateTime
	User       struct {
		Email   string
		Company string
		Id      githubv4.ID
		Login   string
		Name    string
	}
	UserLogin string
}
type auditLogEntries struct {
	Node auditLogEntry
}

func (i *iterOrgAuditLogs) fetchOrgRepos(ctx context.Context, startCursor *githubv4.String) (*fetchOrgAuditLogResults, error) {
	var reposQuery struct {
		Organization struct {
			Login    string
			AuditLog struct {
				TotalCount int
				Edges      []*auditLogEntries
				PageInfo   struct {
					EndCursor   githubv4.String
					HasNextPage bool
				}
			} `graphql:"auditLog(first: $perPage, after: $auditLogCursor)"`
		} `graphql:"organization(login: $login)"`
	}
	variables := map[string]interface{}{
		"login":          githubv4.String(i.login),
		"perPage":        githubv4.Int(i.PerPage),
		"auditLogCursor": startCursor,
		//TODO implement ordering
		//"auditLogOrder":  i.auditOrder,
	}

	err := i.Client().Query(ctx, &reposQuery, variables)
	if err != nil {
		return nil, err
	}

	return &fetchOrgAuditLogResults{
		reposQuery.Organization.AuditLog.Edges,
		reposQuery.Organization.AuditLog.PageInfo.HasNextPage,
		&reposQuery.Organization.AuditLog.PageInfo.EndCursor,
	}, nil

}

type iterOrgAuditLogs struct {
	*Options
	login        string
	affiliations string
	current      int
	results      *fetchOrgAuditLogResults
	//TODO implement ordering
	//auditOrder   *githubv4.AuditLogOrder
}

func (i *iterOrgAuditLogs) logger() *zerolog.Logger {
	logger := i.Logger.With().Int("per-page", i.PerPage).Str("login", i.login).Str("affiliations", i.affiliations).Logger()
	//TODO implement auditOrder ordering
	// if i.auditOrder != nil {
	// 	logger = logger.With().Str("order_by", string(i.auditOrder.Field)).Str("order_dir", string(i.auditOrder.Direction)).Logger()
	// }
	return &logger
}

func (i *iterOrgAuditLogs) Column(ctx vtab.Context, c int) error {
	current := i.results.AuditLogs[i.current]
	switch orgReposCols[c].Name {
	case "login":
		ctx.ResultText(i.login)
	case "audit_entry_action":
		ctx.ResultText(string(current.Node.Action))
	case "audit_entry_id":
		ctx.ResultText(fmt.Sprint(current.Node.Id))
	case "audit_entry_type":
		ctx.ResultText(current.Node.Typename)
	case "audit_entry_user_company":
		ctx.ResultText(current.Node.User.Company)
	case "audit_entry_user_email":
		ctx.ResultText(current.Node.User.Email)
	case "audit_entry_user_id":
		ctx.ResultText(fmt.Sprint(current.Node.User.Id))
	case "audit_entry_user_login":
		ctx.ResultText(current.Node.User.Login)
	case "audit_entry_user_name":
		ctx.ResultText(current.Node.User.Name)
	case "audit_log_count":
		ctx.ResultInt(i.current)
	}
	return nil
}

func (i *iterOrgAuditLogs) Next() (vtab.Row, error) {
	i.current += 1

	if i.results == nil || i.current >= len(i.results.AuditLogs) {
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
			l.Info().Msgf("fetching page of org audit entries for %s", i.login)
			results, err := i.fetchOrgRepos(context.Background(), cursor)
			if err != nil {
				return nil, err
			}

			i.results = results
			i.current = 0

		} else {
			return nil, io.EOF
		}
	}

	return i, nil
}

var orgReposCols = []vtab.Column{
	{Name: "login", Type: "TEXT", Hidden: true, Filters: []*vtab.ColumnFilter{{Op: sqlite.INDEX_CONSTRAINT_EQ, OmitCheck: true}}},
	{Name: "audit_entry_action", Type: "TEXT"},
	{Name: "audit_entry_actor_login", Type: "TEXT"},
	{Name: "audit_entry_id", Type: "TEXT"},
	{Name: "audit_entry_type", Type: "TEXT"},
	{Name: "audit_entry_user_company", Type: "TEXT"},
	{Name: "audit_entry_user_email", Type: "TEXT"},
	{Name: "audit_entry_user_id", Type: "TEXT"},
	{Name: "audit_entry_user_login", Type: "TEXT"},
	{Name: "audit_entry_user_name", Type: "TEXT"},
	{Name: "audit_log_count", Type: "INT"},
	{Name: "user_login", Type: "TEXT"},
}

func NewOrgReposModule(opts *Options) sqlite.Module {
	return vtab.NewTableFunc("github_org_repos", orgReposCols, func(constraints []*vtab.Constraint, orders []*sqlite.OrderBy) (vtab.Iterator, error) {
		var login, affiliations string
		for _, constraint := range constraints {
			if constraint.Op == sqlite.INDEX_CONSTRAINT_EQ {
				switch constraint.ColIndex {
				case 0:
					login = constraint.Value.Text()

				case 1:
					affiliations = strings.ToUpper(constraint.Value.Text())
				}
			}
		}

		// var auditOrder *githubv4.AuditLogOrder
		// // for now we can only support single field order bys
		// if len(orders) == 1 {
		// 	auditOrder = &githubv4.AuditLogOrder{}
		// 	order := orders[0]
		// 	switch orgReposCols[order.ColumnIndex].Name {
		// 	case "ASC":
		// 		createdAt := *githubv4.AuditLogOrderFieldCreatedAt
		// 		auditOrder.Field = *createdAt
		// 	case "DESC":
		// 		auditOrder.Field = githubv4.A
		// 	case "CREATGED_AT":
		// 		auditOrder.Field = githubv4.RepositoryOrderFieldUpdatedAt
		// 	case "pushed_at":
		// 		auditOrder.Field = githubv4.RepositoryOrderFieldPushedAt
		// 	case "stargazer_count":
		// 		auditOrder.Field = githubv4.RepositoryOrderFieldStargazers
		// 	}
		// 	auditOrder.Direction = orderByToGitHubOrder(order.Desc)
		// }
		iter := &iterOrgAuditLogs{opts, login, affiliations, -1, nil} //, auditOrder}
		iter.logger().Info().Msgf("starting GitHub audit_log iterator for %s", login)
		return iter, nil
	}, vtab.EarlyOrderByConstraintExit(true))
}
