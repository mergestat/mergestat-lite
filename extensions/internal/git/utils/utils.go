package utils

import (
	"os"

	"github.com/askgitdev/askgit/extensions/services"
)

// GetDefaultRepoFromCtx looks up the defaultRepoPath key in the supplied context and returns it if set,
// otherwise it returns the current working directory
func GetDefaultRepoFromCtx(ctx services.Context) (repoPath string, err error) {
	var ok bool
	if repoPath, ok = ctx["defaultRepoPath"]; !ok || repoPath == "" {
		if wd, err := os.Getwd(); err != nil {
			return "", err
		} else {
			repoPath = wd
		}
	}
	return
}
