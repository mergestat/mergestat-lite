package gitlog

import (
	"bufio"
	"context"
	"io"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type Commit struct {
	SHA            string
	ParentID       string
	TreeID         string
	Message        string
	AuthorName     string
	AuthorEmail    string
	AuthorWhen     time.Time
	CommitterName  string
	CommitterEmail string
	CommitterWhen  time.Time
	Additions      int
	Deletions      int
}
type Result []*Commit

func parseLog(reader io.Reader) (Result, error) {
	scanner := bufio.NewScanner(reader)
	res := make(Result, 0)

	// line prefixes for the `fuller` formatted output
	const (
		commit     = "commit "
		tree       = "tree "
		parent     = "parent "
		author     = "Author: "
		authorDate = "AuthorDate: "
		message    = "Message: "
		committer  = "Commit: "
		commitDate = "CommitDate: "
	)

	var currentCommit *Commit
	for scanner.Scan() {
		line := scanner.Text()
		switch {
		case strings.HasPrefix(line, commit):
			if currentCommit != nil { // if we're seeing a new commit but already have a current commit, we've finished a commit
				res = append(res, currentCommit)
			}
			currentCommit = &Commit{
				SHA: strings.TrimPrefix(line, commit),
			}
		case strings.HasPrefix(line, tree):
			currentCommit.TreeID = strings.TrimPrefix(line, tree)
		case strings.HasPrefix(line, parent):
			currentCommit.ParentID = strings.TrimPrefix(line, parent)
		case strings.HasPrefix(line, author):
			s := strings.TrimPrefix(line, author)
			spl := strings.Split(s, " ")
			email := strings.Trim(spl[len(spl)-1], "<>")
			name := strings.Join(spl[:len(spl)-1], " ")
			currentCommit.AuthorEmail = strings.Trim(email, "<>")
			currentCommit.AuthorName = strings.TrimSpace(name)
		case strings.HasPrefix(line, authorDate):
			authorDateString := strings.TrimPrefix(line, authorDate)
			aD, err := time.Parse(time.RFC3339, authorDateString)
			if err != nil {
				return nil, err
			}
			currentCommit.AuthorWhen = aD
		case strings.HasPrefix(line, committer):
			s := strings.TrimPrefix(line, committer)
			spl := strings.Split(s, " ")
			email := strings.Trim(spl[len(spl)-1], "<>")
			name := strings.Join(spl[:len(spl)-1], " ")
			currentCommit.CommitterEmail = strings.Trim(email, "<>")
			currentCommit.CommitterName = strings.TrimSpace(name)
		case strings.HasPrefix(line, commitDate):
			commitDateString := strings.TrimPrefix(line, commitDate)
			cD, err := time.Parse(time.RFC3339, commitDateString)
			if err != nil {
				return nil, err
			}
			currentCommit.CommitterWhen = cD
		case strings.HasPrefix(line, message):
			currentCommit.Message = strings.TrimPrefix(line, message)
		case strings.TrimSpace(line) == "": // ignore empty lines
		default:
			s := strings.Split(line, "\t")
			var additions int
			var deletions int
			var err error
			if s[0] != "-" {
				additions, err = strconv.Atoi(s[0])
				if err != nil {
					return nil, err
				}
			}
			if s[1] != "-" {
				deletions, err = strconv.Atoi(s[1])
				if err != nil {
					return nil, err
				}
			}
			currentCommit.Additions = additions
			currentCommit.Deletions = deletions
		}
	}
	if currentCommit != nil {
		res = append(res, currentCommit)
	}

	return res, nil
}

func Execute(repoPath string) (Result, error) {
	gitPath, err := exec.LookPath("git")
	if err != nil {
		return nil, err
	}

	args := []string{"log"}
	args = append(args, "--format=commit %H%ntree %T%nparent %P%nAuthor: %an %ae%nAuthorDate: %aI%nCommit: %cn %ce%nCommitDate: %cI%nMessage: %s", "--numstat")

	cmd := exec.CommandContext(context.Background(), gitPath, args...)
	cmd.Dir = repoPath

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	// stderr, err := cmd.StderrPipe()
	// if err != nil {
	// 	return nil, err
	// }

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	res, err := parseLog(stdout)
	if err != nil {
		return nil, err
	}

	// stderr, err = ioutil.ReadAll(stderr)
	// if err != nil {
	// 	return nil, err
	// }

	if err := cmd.Wait(); err != nil {
		return nil, err
	}
	return res, nil
}
