package github

import (
	"strconv"

	"github.com/askgitdev/askgit/tables/services"
)

// GetGitHubTokenFromCtx looks up the githubToken key in the supplied context and returns it if set
func GetGitHubTokenFromCtx(ctx services.Context) string {
	return ctx["githubToken"]
}

// GetGithubReqPerSecondFromCtx looks up the githubReqPerSec key in the supplied context and returns it if set,
// otherwise it returns a default of 1
func GetGithubReqPerSecondFromCtx(ctx services.Context) int {
	defaultValue := 1
	if githubReqPerSec, ok := ctx["githubReqPerSec"]; ok && githubReqPerSec != "" {
		if i, err := strconv.Atoi(githubReqPerSec); err != nil {
			return i
		} else {
			return defaultValue
		}
	} else {
		return defaultValue
	}
}
