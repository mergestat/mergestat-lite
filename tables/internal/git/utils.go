package git

import (
	"encoding/base64"
	"io"
	"os"
	"strings"

	"github.com/askgitdev/askgit/tables/services"
	"github.com/go-git/go-git/v5/plumbing"
)

// returns true if error is an end-of-file error
func eof(err error) bool { return err == io.EOF }

func enc(buf []byte) string          { return base64.StdEncoding.EncodeToString(buf) }
func dec(str string) ([]byte, error) { return base64.StdEncoding.DecodeString(str) }

func isRemoteBranch(ref plumbing.ReferenceName) bool {
	return ref.IsRemote() &&
		plumbing.ReferenceName(strings.Replace(ref.String(), "remotes", "heads", 1)).IsBranch()
}

// getDefaultRepoFromCtx looks up the defaultRepoPath key in the supplied context and returns it if set,
// otherwise it returns the current working directory
func getDefaultRepoFromCtx(ctx services.Context) (repoPath string, err error) {
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
