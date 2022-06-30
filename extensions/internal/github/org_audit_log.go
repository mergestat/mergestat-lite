package github

import (
	"context"
	"encoding/json"
	"io"
	"strings"
	"time"

	"github.com/augmentable-dev/vtab"
	"github.com/rs/zerolog"
	"github.com/shurcooL/githubv4"
	"go.riyazali.net/sqlite"
)

type fetchOrgAuditLogResults struct {
	AuditLogs   []*auditLogEntry
	HasNextPage bool
	EndCursor   *githubv4.String
}

type auditLogEntry struct {
	Typename     string `graphql:"__typename"`
	NodeFragment struct {
		Id string
	} `graphql:"... on Node"`
	Entry auditLogEntryContents `graphql:"... on AuditEntry"`
}

type auditLogEntryContents struct {
	Action string
	Actor  struct {
		Type string `graphql:"__typename"`
	}
	ActorLogin    string
	ActorIp       string
	ActorLocation struct {
		City        string `json:"city"`
		Country     string `json:"country"`
		CountryCode string `json:"countryCode"`
		Region      string `json:"region"`
		RegionCode  string `json:"regionCode"`
	}
	CreatedAt     githubv4.DateTime
	OperationType string
	UserLogin     string
}

func (i *iterOrgAuditLogs) fetchOrgAuditRepos(ctx context.Context, startCursor *githubv4.String) (*fetchOrgAuditLogResults, error) {
	var reposQuery struct {
		Organization struct {
			Login    string
			AuditLog struct {
				TotalCount int
				Nodes      []*auditLogEntry
				PageInfo   struct {
					EndCursor   githubv4.String
					HasNextPage bool
				}
			} `graphql:"auditLog(first: $perPage, after: $auditLogCursor, orderBy: $auditLogOrder)"`
		} `graphql:"organization(login: $login)"`
	}
	variables := map[string]interface{}{
		"login":          githubv4.String(i.login),
		"perPage":        githubv4.Int(i.PerPage),
		"auditLogCursor": startCursor,
		"auditLogOrder":  i.auditOrder,
	}

	err := i.Client().Query(ctx, &reposQuery, variables)
	if err != nil {
		return nil, err
	}

	return &fetchOrgAuditLogResults{
		reposQuery.Organization.AuditLog.Nodes,
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
	auditOrder   *githubv4.AuditLogOrder
}

func (i *iterOrgAuditLogs) logger() *zerolog.Logger {
	logger := i.Logger.With().Int("per-page", i.PerPage).Str("login", i.login).Logger()
	if i.auditOrder != nil {
		logger = logger.With().Str("order_by", string(*i.auditOrder.Field)).Str("order_dir", string(*i.auditOrder.Direction)).Logger()
	}
	return &logger
}

func (i *iterOrgAuditLogs) Column(ctx vtab.Context, c int) error {
	current := i.results.AuditLogs[i.current]

	switch orgAuditCols[c].Name {
	case "login":
		ctx.ResultText(i.login)
	case "id":
		ctx.ResultText(current.NodeFragment.Id)
	case "entry_type":
		ctx.ResultText(current.Typename)
	case "action":
		ctx.ResultText(current.Entry.Action)
	case "actor_type":
		ctx.ResultText(current.Entry.Actor.Type)
	case "actor_login":
		ctx.ResultText(current.Entry.ActorLogin)
	case "actor_ip":
		ctx.ResultText(current.Entry.ActorIp)
	case "actor_location":
		if s, err := json.Marshal(current.Entry.ActorLocation); err != nil {
			return err
		} else {
			ctx.ResultText(string(s))
		}
	case "created_at":
		t := current.Entry.CreatedAt
		if t.IsZero() {
			ctx.ResultNull()
		} else {
			ctx.ResultText(t.Format(time.RFC3339Nano))
		}
	case "operation_type":
		ctx.ResultText(current.Entry.OperationType)
	case "user_login":
		ctx.ResultText(current.Entry.UserLogin)
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
			results, err := i.fetchOrgAuditRepos(context.Background(), cursor)
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

var orgAuditCols = []vtab.Column{
	{Name: "login", Type: "TEXT", Hidden: true, Filters: []*vtab.ColumnFilter{{Op: sqlite.INDEX_CONSTRAINT_EQ, OmitCheck: true}}},
	{Name: "id", Type: "TEXT"},
	{Name: "entry_type", Type: "TEXT"},
	{Name: "action", Type: "TEXT"},
	{Name: "actor_type", Type: "TEXT"},
	{Name: "actor_login", Type: "TEXT"},
	{Name: "actor_ip", Type: "TEXT"},
	{Name: "actor_location", Type: "JSON"},
	{Name: "created_at", Type: "DATETIME", OrderBy: vtab.ASC | vtab.DESC},
	{Name: "operation_type", Type: "TEXT"},
	{Name: "user_login", Type: "TEXT"},
}

func NewOrgAuditModule(opts *Options) sqlite.Module {
	return vtab.NewTableFunc("github_audit_repos", orgAuditCols, func(constraints []*vtab.Constraint, orders []*sqlite.OrderBy) (vtab.Iterator, error) {
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

		var auditOrder *githubv4.AuditLogOrder
		// for now we can only support single field order bys
		if len(orders) == 1 {
			order := orders[0]
			switch orgAuditCols[order.ColumnIndex].Name {
			case "created_at":
				createdAt := githubv4.AuditLogOrderFieldCreatedAt
				dir := orderByToGitHubOrder(order.Desc)
				auditOrder = &githubv4.AuditLogOrder{
					Field: &createdAt,
				}
				auditOrder.Direction = &dir
			}
		}
		iter := &iterOrgAuditLogs{opts, login, affiliations, -1, nil, auditOrder}
		iter.logger().Info().Msgf("starting GitHub audit_log iterator for %s", login)
		return iter, nil
	}, vtab.EarlyOrderByConstraintExit(true))
}
