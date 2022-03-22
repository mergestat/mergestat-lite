package options

import "github.com/shurcooL/githubv4"

// GitHubRateLimitResponse represents metadata about the caller's rate limit, returned by the GitHub GraphQL API
type GitHubRateLimitResponse struct {
	Cost      int
	Limit     int
	NodeCount int
	Remaining int
	ResetAt   githubv4.DateTime
	Used      int
}
