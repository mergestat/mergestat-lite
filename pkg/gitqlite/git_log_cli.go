package gitqlite

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/mattn/go-sqlite3"
)

type gitLogCLIModule struct{}

type gitLogCLITable struct {
	repoPath string
	repo     *git.Repository
}
type Commit struct {
	SHA            string
	Parent         string
	Tree           string
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

func (m *gitLogCLIModule) Create(c *sqlite3.SQLiteConn, args []string) (sqlite3.VTab, error) {
	err := c.DeclareVTab(fmt.Sprintf(`
		CREATE TABLE %q (
			id TEXT,
			message TEXT,
			summary TEXT,
			author_name TEXT,
			author_email TEXT,
			author_when DATETIME,
			committer_name TEXT,
			committer_email TEXT,
			committer_when DATETIME, 
			parent_id TEXT,
			parent_count INT(10),
			tree_id TEXT,
			additions INT(10),
			deletions INT(10)
		)`, args[0]))
	if err != nil {
		return nil, err
	}

	// the repoPath will be enclosed in double quotes "..." since ensureTables uses %q when setting up the table
	// we need to pop those off when referring to the actual directory in the fs
	repoPath := args[3][1 : len(args[3])-1]
	return &gitLogCLITable{repoPath: repoPath}, nil
}

func (m *gitLogCLIModule) Connect(c *sqlite3.SQLiteConn, args []string) (sqlite3.VTab, error) {
	return m.Create(c, args)
}

func (m *gitLogCLIModule) DestroyModule() {}

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
			currentCommit.Tree = strings.TrimPrefix(line, tree)
		case strings.HasPrefix(line, parent):
			currentCommit.Parent = strings.TrimPrefix(line, parent)
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
	fmt.Println(res)
	return res, nil
}

func (v *gitLogCLITable) Open() (sqlite3.VTabCursor, error) {
	gitPath, err := exec.LookPath("git")
	if err != nil {
		return nil, err
	}
	args := []string{"log"}
	args = append(args, "--format=commit %H %ntree %T%nparent %P%nAuthor: %an %ae%nAuthorDate: %aI%nCommit: %cn %ce%nCommitDate: %cI%nMessage: %s", "--numstat")
	cmd := exec.CommandContext(context.Background(), gitPath, args...)
	cmd.Dir = v.repoPath

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	res, err := parseLog(stdout)
	if err != nil {
		return nil, err
	}

	errs, err := ioutil.ReadAll(stderr)
	if err != nil {
		return nil, err
	}
	if err := cmd.Wait(); err != nil {
		fmt.Println(string(errs))
		return nil, err
	}
	return &commitCLICursor{0, res, res[0], false}, nil
}

func (v *gitLogCLITable) BestIndex(cst []sqlite3.InfoConstraint, ob []sqlite3.InfoOrderBy) (*sqlite3.IndexResult, error) {
	// TODO this should actually be implemented!
	dummy := make([]bool, len(cst))
	return &sqlite3.IndexResult{Used: dummy}, nil
}

func (v *gitLogCLITable) Disconnect() error {
	v.repo = nil
	return nil
}
func (v *gitLogCLITable) Destroy() error { return nil }

type commitCLICursor struct {
	index   int
	results Result
	current *Commit
	eof     bool
}

func (vc *commitCLICursor) Column(c *sqlite3.SQLiteContext, col int) error {

	switch col {
	case 0:
		//commit id
		c.ResultText(vc.current.SHA)
	case 1:
		//commit message
		c.ResultText(vc.current.Message)
	case 2:
		//commit summary
		c.ResultText(strings.Split(vc.current.Message, "\n")[0])
	case 3:
		//commit author name
		c.ResultText(vc.current.AuthorName)
	case 4:
		//commit author email
		c.ResultText(vc.current.AuthorEmail)
	case 5:
		//author when
		c.ResultText(vc.current.AuthorWhen.Format(time.RFC3339Nano))
	case 6:
		//committer name
		c.ResultText(vc.current.CommitterName)
	case 7:
		//committer email
		c.ResultText(vc.current.CommitterEmail)
	case 8:
		//committer when
		c.ResultText(vc.current.CommitterWhen.Format(time.RFC3339Nano))
	case 9:
		//parent_id
		c.ResultText(vc.current.Parent)
	case 10:
		//parent_count
		c.ResultInt(len(strings.Split(vc.current.Parent, " ")))
	case 11:
		//tree_id
		c.ResultText(vc.current.Tree)

	case 12:
		c.ResultInt(vc.current.Additions)
	case 13:
		c.ResultInt(vc.current.Deletions)

	}
	return nil
}

func (vc *commitCLICursor) Filter(idxNum int, idxStr string, vals []interface{}) error {
	vc.index = 0
	return nil
}

func (vc *commitCLICursor) Next() error {
	vc.index++
	if vc.index < len(vc.results) {
		vc.current = vc.results[vc.index]
		return nil

	} else {
		vc.current = nil
		vc.eof = true
		return nil
	}
}

func (vc *commitCLICursor) EOF() bool {
	return vc.eof
}

func (vc *commitCLICursor) Rowid() (int64, error) {
	return int64(vc.index), nil
}

func (vc *commitCLICursor) Close() error {
	return nil
}
