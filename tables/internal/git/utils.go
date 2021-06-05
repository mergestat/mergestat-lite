package git

import (
	"encoding/base64"
	"github.com/go-git/go-git/v5/plumbing"
	"io"
	"strings"
)

// returns true if error is an end-of-file error
func eof(err error) bool { return err == io.EOF }

func enc(buf []byte) string          { return base64.StdEncoding.EncodeToString(buf) }
func dec(str string) ([]byte, error) { return base64.StdEncoding.DecodeString(str) }

func isRemoteBranch(ref plumbing.ReferenceName) bool {
	return ref.IsRemote() &&
		plumbing.ReferenceName(strings.Replace(ref.String(), "remotes", "heads", 1)).IsBranch()
}