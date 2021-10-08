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
	Typename                                                string        `graphql:"__typename"`
	MembersCanDeleteReposClearAuditEntry                    entryContents `graphql:"... on MembersCanDeleteReposClearAuditEntry"`
	MembersCanDeleteReposDisableAuditEntry                  entryContents `graphql:"... on MembersCanDeleteReposDisableAuditEntry"`
	MembersCanDeleteReposEnableAuditEntry                   entryContents `graphql:"... on MembersCanDeleteReposEnableAuditEntry"`
	TeamRemoveRepositoryAuditEntry                          entryContents `graphql:"... on TeamRemoveRepositoryAuditEntry"`
	TeamRemoveMemberAuditEntry                              entryContents `graphql:"... on TeamRemoveMemberAuditEntry"`
	TeamChangeParentTeamAuditEntry                          entryContents `graphql:"... on TeamChangeParentTeamAuditEntry"`
	TeamAddRepositoryAuditEntry                             entryContents `graphql:"... on TeamAddRepositoryAuditEntry"`
	TeamAddMemberAuditEntry                                 entryContents `graphql:"... on TeamAddMemberAuditEntry"`
	RepositoryVisibilityChangeEnableAuditEntry              entryContents `graphql:"... on RepositoryVisibilityChangeEnableAuditEntry"`
	RepositoryVisibilityChangeDisableAuditEntry             entryContents `graphql:"... on RepositoryVisibilityChangeDisableAuditEntry"`
	RepoRemoveTopicAuditEntry                               entryContents `graphql:"... on RepoRemoveTopicAuditEntry"`
	RepoRemoveMemberAuditEntry                              entryContents `graphql:"... on RepoRemoveMemberAuditEntry"`
	RepoDestroyAuditEntry                                   entryContents `graphql:"... on RepoDestroyAuditEntry"`
	RepoCreateAuditEntry                                    entryContents `graphql:"... on RepoCreateAuditEntry"`
	RepoConfigUnlockAnonymousGitAccessAuditEntry            entryContents `graphql:"... on RepoConfigUnlockAnonymousGitAccessAuditEntry"`
	RepoConfigLockAnonymousGitAccessAuditEntry              entryContents `graphql:"... on RepoConfigLockAnonymousGitAccessAuditEntry"`
	RepoConfigEnableContributorsOnlyAuditEntry              entryContents `graphql:"... on RepoConfigEnableContributorsOnlyAuditEntry"`
	RepoConfigEnableCollaboratorsOnlyAuditEntry             entryContents `graphql:"... on RepoConfigEnableCollaboratorsOnlyAuditEntry"`
	RepoConfigEnableAnonymousGitAccessAuditEntry            entryContents `graphql:"... on RepoConfigEnableAnonymousGitAccessAuditEntry"`
	RepoConfigDisableSockpuppetDisallowedAuditEntry         entryContents `graphql:"... on RepoConfigDisableSockpuppetDisallowedAuditEntry"`
	RepoConfigDisableContributorsOnlyAuditEntry             entryContents `graphql:"... on RepoConfigDisableContributorsOnlyAuditEntry"`
	RepoConfigDisableAnonymousGitAccessAuditEntry           entryContents `graphql:"... on RepoConfigDisableAnonymousGitAccessAuditEntry"`
	RepoChangeMergeSettingAuditEntry                        entryContents `graphql:"... on RepoChangeMergeSettingAuditEntry"`
	RepoArchivedAuditEntry                                  entryContents `graphql:"... on RepoArchivedAuditEntry"`
	RepoAddTopicAuditEntry                                  entryContents `graphql:"... on RepoAddTopicAuditEntry"`
	RepoAddMemberAuditEntry                                 entryContents `graphql:"... on RepoAddMemberAuditEntry"`
	RepoAccessAuditEntry                                    entryContents `graphql:"... on RepoAccessAuditEntry"`
	PrivateRepositoryForkingDisableAuditEntry               entryContents `graphql:"... on PrivateRepositoryForkingDisableAuditEntry"`
	OrgUpdateMemberRepositoryInvitationPermissionAuditEntry entryContents `graphql:"... on OrgUpdateMemberRepositoryInvitationPermissionAuditEntry"`
	OrgUpdateMemberRepositoryCreationPermissionAuditEntry   entryContents `graphql:"... on OrgUpdateMemberRepositoryCreationPermissionAuditEntry"`
	OrgUpdateMemberAuditEntry                               entryContents `graphql:"... on OrgUpdateMemberAuditEntry"`
	OrgUpdateDefaultRepositoryPermissionAuditEntry          entryContents `graphql:"... on OrgUpdateDefaultRepositoryPermissionAuditEntry"`
	OrgUnblockUserAuditEntry                                entryContents `graphql:"... on OrgUnblockUserAuditEntry"`
	OrgRemoveOutsideCollaboratorAuditEntry                  entryContents `graphql:"... on OrgRemoveOutsideCollaboratorAuditEntry"`
	OrgRemoveMemberAuditEntry                               entryContents `graphql:"... on OrgRemoveMemberAuditEntry"`
	OrgRemoveBillingManagerAuditEntry                       entryContents `graphql:"... on OrgRemoveBillingManagerAuditEntry"`
	OrgOauthAppAccessRequestedAuditEntry                    entryContents `graphql:"... on OrgOauthAppAccessRequestedAuditEntry"`
	OrgOauthAppAccessDeniedAuditEntry                       entryContents `graphql:"... on OrgOauthAppAccessDeniedAuditEntry"`
	OrgInviteToBusinessAuditEntry                           entryContents `graphql:"... on OrgInviteToBusinessAuditEntry"`
	OrgInviteMemberAuditEntry                               entryContents `graphql:"... on OrgInviteMemberAuditEntry"`
	OrgEnableTwoFactorRequirementAuditEntry                 entryContents `graphql:"... on OrgEnableTwoFactorRequirementAuditEntry"`
	OrgEnableSamlAuditEntry                                 entryContents `graphql:"... on OrgEnableSamlAuditEntry"`
	OrgEnableOauthAppRestrictionsAuditEntry                 entryContents `graphql:"... on OrgEnableOauthAppRestrictionsAuditEntry"`
	OrgDisableSamlAuditEntry                                entryContents `graphql:"... on OrgDisableSamlAuditEntry"`
	OrgCreateAuditEntry                                     entryContents `graphql:"... on OrgCreateAuditEntry"`
	OrgConfigEnableCollaboratorsOnlyAuditEntry              entryContents `graphql:"... on OrgConfigEnableCollaboratorsOnlyAuditEntry"`
	OrgConfigDisableCollaboratorsOnlyAuditEntry             entryContents `graphql:"... on OrgConfigDisableCollaboratorsOnlyAuditEntry"`
	OrgBlockUserAuditEntry                                  entryContents `graphql:"... on OrgBlockUserAuditEntry"`
	OrgAddMemberAuditEntry                                  entryContents `graphql:"... on OrgAddMemberAuditEntry"`
	OrgAddBillingManagerAuditEntry                          entryContents `graphql:"... on OrgAddBillingManagerAuditEntry"`
	OauthApplicationCreateAuditEntry                        entryContents `graphql:"... on OauthApplicationCreateAuditEntry"`
	OrgDisableOauthAppRestrictionsAuditEntry                entryContents `graphql:"... on OrgDisableOauthAppRestrictionsAuditEntry"`
}
type entryContents struct {
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

func (i *iterOrgAuditLogs) fetchOrgAuditRepos(ctx context.Context, startCursor *githubv4.String) (*fetchOrgAuditLogResults, error) {
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
	var currentContents entryContents
	switch current.Node.Typename {
	case "MembersCanDeleteReposClearAuditEntry":
		currentContents = current.Node.MembersCanDeleteReposClearAuditEntry
	case "MembersCanDeleteReposDisableAuditEntry":
		currentContents = current.Node.MembersCanDeleteReposDisableAuditEntry
	case "MembersCanDeleteReposEnableAuditEntry":
		currentContents = current.Node.MembersCanDeleteReposEnableAuditEntry
	case "TeamRemoveRepositoryAuditEntry":
		currentContents = current.Node.TeamRemoveRepositoryAuditEntry
	case "TeamRemoveMemberAuditEntry":
		currentContents = current.Node.TeamRemoveMemberAuditEntry
	case "TeamChangeParentTeamAuditEntry":
		currentContents = current.Node.TeamChangeParentTeamAuditEntry
	case "TeamAddRepositoryAuditEntry":
		currentContents = current.Node.TeamAddRepositoryAuditEntry
	case "TeamAddMemberAuditEntry":
		currentContents = current.Node.TeamAddMemberAuditEntry
	case "RepositoryVisibilityChangeEnableAuditEntry":
		currentContents = current.Node.RepositoryVisibilityChangeEnableAuditEntry
	case "RepositoryVisibilityChangeDisableAuditEntry":
		currentContents = current.Node.RepositoryVisibilityChangeDisableAuditEntry
	case "RepoRemoveTopicAuditEntry":
		currentContents = current.Node.RepoRemoveTopicAuditEntry
	case "RepoRemoveMemberAuditEntry":
		currentContents = current.Node.RepoRemoveMemberAuditEntry
	case "RepoDestroyAuditEntry":
		currentContents = current.Node.RepoDestroyAuditEntry
	case "RepoCreateAuditEntry":
		currentContents = current.Node.RepoCreateAuditEntry
	case "RepoConfigUnlockAnonymousGitAccessAuditEntry":
		currentContents = current.Node.RepoConfigUnlockAnonymousGitAccessAuditEntry
	case "RepoConfigLockAnonymousGitAccessAuditEntry":
		currentContents = current.Node.RepoConfigLockAnonymousGitAccessAuditEntry
	case "RepoConfigEnableContributorsOnlyAuditEntry":
		currentContents = current.Node.RepoConfigEnableContributorsOnlyAuditEntry
	case "RepoConfigEnableCollaboratorsOnlyAuditEntry":
		currentContents = current.Node.RepoConfigEnableCollaboratorsOnlyAuditEntry
	case "RepoConfigEnableAnonymousGitAccessAuditEntry":
		currentContents = current.Node.RepoConfigEnableAnonymousGitAccessAuditEntry
	case "RepoConfigDisableSockpuppetDisallowedAuditEntry":
		currentContents = current.Node.RepoConfigDisableSockpuppetDisallowedAuditEntry
	case "RepoConfigDisableContributorsOnlyAuditEntry":
		currentContents = current.Node.RepoConfigDisableContributorsOnlyAuditEntry
	case "RepoConfigDisableAnonymousGitAccessAuditEntry":
		currentContents = current.Node.RepoConfigDisableAnonymousGitAccessAuditEntry
	case "RepoChangeMergeSettingAuditEntry":
		currentContents = current.Node.RepoChangeMergeSettingAuditEntry
	case "RepoArchivedAuditEntry":
		currentContents = current.Node.RepoArchivedAuditEntry
	case "RepoAddTopicAuditEntry":
		currentContents = current.Node.RepoAddTopicAuditEntry
	case "RepoAddMemberAuditEntry":
		currentContents = current.Node.RepoAddMemberAuditEntry
	case "RepoAccessAuditEntry":
		currentContents = current.Node.RepoAccessAuditEntry
	case "PrivateRepositoryForkingDisableAuditEntry":
		currentContents = current.Node.PrivateRepositoryForkingDisableAuditEntry
	case "OrgUpdateMemberRepositoryInvitationPermissionAuditEntry":
		currentContents = current.Node.OrgUpdateMemberRepositoryInvitationPermissionAuditEntry
	case "OrgUpdateMemberRepositoryCreationPermissionAuditEntry":
		currentContents = current.Node.OrgUpdateMemberRepositoryCreationPermissionAuditEntry
	case "OrgUpdateMemberAuditEntry":
		currentContents = current.Node.OrgUpdateMemberAuditEntry
	case "OrgUpdateDefaultRepositoryPermissionAuditEntry":
		currentContents = current.Node.OrgUpdateDefaultRepositoryPermissionAuditEntry
	case "OrgUnblockUserAuditEntry":
		currentContents = current.Node.OrgUnblockUserAuditEntry
	case "OrgRemoveOutsideCollaboratorAuditEntry":
		currentContents = current.Node.OrgRemoveOutsideCollaboratorAuditEntry
	case "OrgRemoveMemberAuditEntry":
		currentContents = current.Node.OrgRemoveMemberAuditEntry
	case "OrgRemoveBillingManagerAuditEntry":
		currentContents = current.Node.OrgRemoveBillingManagerAuditEntry
	case "OrgOauthAppAccessRequestedAuditEntry":
		currentContents = current.Node.OrgOauthAppAccessRequestedAuditEntry
	case "OrgOauthAppAccessDeniedAuditEntry":
		currentContents = current.Node.OrgOauthAppAccessDeniedAuditEntry
	case "OrgInviteToBusinessAuditEntry":
		currentContents = current.Node.OrgInviteToBusinessAuditEntry
	case "OrgInviteMemberAuditEntry":
		currentContents = current.Node.OrgInviteMemberAuditEntry
	case "OrgEnableTwoFactorRequirementAuditEntry":
		currentContents = current.Node.OrgEnableTwoFactorRequirementAuditEntry
	case "OrgEnableSamlAuditEntry":
		currentContents = current.Node.OrgEnableSamlAuditEntry
	case "OrgEnableOauthAppRestrictionsAuditEntry":
		currentContents = current.Node.OrgEnableOauthAppRestrictionsAuditEntry
	case "OrgDisableSamlAuditEntry":
		currentContents = current.Node.OrgDisableSamlAuditEntry
	case "OrgCreateAuditEntry":
		currentContents = current.Node.OrgCreateAuditEntry
	case "OrgConfigEnableCollaboratorsOnlyAuditEntry":
		currentContents = current.Node.OrgConfigEnableCollaboratorsOnlyAuditEntry
	case "OrgConfigDisableCollaboratorsOnlyAuditEntry":
		currentContents = current.Node.OrgConfigDisableCollaboratorsOnlyAuditEntry
	case "OrgBlockUserAuditEntry":
		currentContents = current.Node.OrgBlockUserAuditEntry
	case "OrgAddMemberAuditEntry":
		currentContents = current.Node.OrgAddMemberAuditEntry
	case "OrgAddBillingManagerAuditEntry":
		currentContents = current.Node.OrgAddBillingManagerAuditEntry
	case "OauthApplicationCreateAuditEntry":
		currentContents = current.Node.OauthApplicationCreateAuditEntry
	case "OrgDisableOauthAppRestrictionsAuditEntry":
		currentContents = current.Node.OrgDisableOauthAppRestrictionsAuditEntry
	}
	switch orgAuditCols[c].Name {
	case "login":
		ctx.ResultText(i.login)
	case "audit_entry_action":
		ctx.ResultText(string(currentContents.Action))
	case "audit_entry_id":
		ctx.ResultText(fmt.Sprint(currentContents.Id))
	case "audit_entry_type":
		ctx.ResultText(current.Node.Typename)
	case "audit_entry_user_company":
		ctx.ResultText(currentContents.User.Company)
	case "audit_entry_user_email":
		ctx.ResultText(currentContents.User.Email)
	case "audit_entry_user_id":
		ctx.ResultText(fmt.Sprint(currentContents.User.Id))
	case "audit_entry_user_login":
		ctx.ResultText(currentContents.User.Login)
	case "audit_entry_user_name":
		ctx.ResultText(currentContents.User.Name)
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
