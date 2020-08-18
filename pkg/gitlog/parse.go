package gitlog

import (
	"bufio"
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
	Files          string
}
type Result []*Commit

type CommitIter struct {
	reader  io.ReadCloser
	scanner *bufio.Scanner
	current *Commit
}

func newCommitIter(reader io.ReadCloser) *CommitIter {
	return &CommitIter{
		reader:  reader,
		scanner: bufio.NewScanner(reader),
		current: nil,
	}
}

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

func (iter *CommitIter) Next() (*Commit, error) {
	for iter.scanner.Scan() {
		line := iter.scanner.Text()
		switch {
		case strings.HasPrefix(line, commit):
			current := iter.current // save the previous current
			iter.current = &Commit{ // set the iterator's current to the new commit we see
				SHA: strings.TrimPrefix(line, commit),
			}
			if current != nil { // if we're seeing a new commit but already have a current commit, we've finished a commit and should return it
				return current, nil
			}
		case strings.HasPrefix(line, tree):
			iter.current.TreeID = strings.TrimPrefix(line, tree)
		case strings.HasPrefix(line, parent):
			iter.current.ParentID = strings.TrimPrefix(line, parent)
		case strings.HasPrefix(line, author):
			s := strings.TrimPrefix(line, author)
			spl := strings.Split(s, " ")
			email := strings.Trim(spl[len(spl)-1], "<>")
			name := strings.Join(spl[:len(spl)-1], " ")
			iter.current.AuthorEmail = strings.Trim(email, "<>")
			iter.current.AuthorName = strings.TrimSpace(name)
		case strings.HasPrefix(line, authorDate):
			authorDateString := strings.TrimPrefix(line, authorDate)
			aD, err := time.Parse(time.RFC3339, authorDateString)
			if err != nil {
				return nil, err
			}
			iter.current.AuthorWhen = aD
		case strings.HasPrefix(line, committer):
			s := strings.TrimPrefix(line, committer)
			spl := strings.Split(s, " ")
			email := strings.Trim(spl[len(spl)-1], "<>")
			name := strings.Join(spl[:len(spl)-1], " ")
			iter.current.CommitterEmail = strings.Trim(email, "<>")
			iter.current.CommitterName = strings.TrimSpace(name)
		case strings.HasPrefix(line, commitDate):
			commitDateString := strings.TrimPrefix(line, commitDate)
			cD, err := time.Parse(time.RFC3339, commitDateString)
			if err != nil {
				return nil, err
			}
			iter.current.CommitterWhen = cD
		case strings.HasPrefix(line, message):
			iter.current.Message = strings.TrimPrefix(line, message)
		case strings.TrimSpace(line) == "": // ignore empty lines
		default:
			s := strings.Split(line, "\t")
			if s[0] != "-" {
				additions, err := strconv.Atoi(s[0])
				if err != nil {
					return nil, err
				}
				iter.current.Additions += additions
			}
			if s[1] != "-" {
				deletions, err := strconv.Atoi(s[1])
				if err != nil {
					return nil, err
				}
				iter.current.Deletions += deletions
			}
			if s[2] != "-" {
				iter.current.Files += s[2] + " "
			}
		}
	}

	if iter.current != nil {
		c := iter.current
		iter.current = nil
		return c, nil
	}

	err := iter.reader.Close()
	if err != nil {
		return nil, err
	}
	return nil, io.EOF
}

func Execute(repoPath string) (*CommitIter, error) {
	gitPath, err := exec.LookPath("git")
	if err != nil {
		return nil, err
	}

	args := []string{"log"}
	args = append(args, "--format=commit %H%ntree %T%nparent %P%nAuthor: %an %ae%nAuthorDate: %aI%nCommit: %cn %ce%nCommitDate: %cI%nMessage: %s", "--numstat")

	cmd := exec.Command(gitPath, args...)
	cmd.Dir = repoPath

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	return newCommitIter(stdout), nil
}
