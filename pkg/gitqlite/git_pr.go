package gitqlite

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/google/go-github/github"
	"github.com/mattn/go-sqlite3"
	"golang.org/x/oauth2"
)

type gitPrModule struct{}

type gitPrTable struct {
	repoPath string
	repo     *git.Repository
}

func (m *gitPrModule) Create(c *sqlite3.SQLiteConn, args []string) (sqlite3.VTab, error) {
	err := c.DeclareVTab(fmt.Sprintf(`
		CREATE TABLE %q (
			number TEXT,
			title TEXT,
			author TEXT,
			pr_when DATETIME,
			state TEXT,
			mergeSHA TEXT
		)`, args[0]))
	if err != nil {
		return nil, err
	}

	// the repoPath will be enclosed in double quotes "..." since ensureTables uses %q when setting up the table
	// we need to pop those off when referring to the actual directory in the fs
	repoPath := args[3][1 : len(args[3])-1]
	return &gitPrTable{repoPath: repoPath}, nil
}

func (m *gitPrModule) Connect(c *sqlite3.SQLiteConn, args []string) (sqlite3.VTab, error) {
	return m.Create(c, args)
}

func (m *gitPrModule) DestroyModule() {}

func (v *gitPrTable) Open() (sqlite3.VTabCursor, error) {
	repo, err := git.PlainOpen(v.repoPath)
	if err != nil {
		return nil, err
	}
	v.repo = repo

	return &prCursor{repoName: v.repoPath}, nil
}

func (v *gitPrTable) BestIndex(cst []sqlite3.InfoConstraint, ob []sqlite3.InfoOrderBy) (*sqlite3.IndexResult, error) {
	// TODO this should actually be implemented!
	dummy := make([]bool, len(cst))
	return &sqlite3.IndexResult{Used: dummy}, nil
}

func (v *gitPrTable) Disconnect() error {
	v.repo = nil
	return nil
}
func (v *gitPrTable) Destroy() error { return nil }

type prCursor struct {
	repoName string
	pr       []*github.PullRequest
	current  *github.PullRequest
	index    int
}

func (vc *prCursor) Column(c *sqlite3.SQLiteContext, col int) error {
	pr := vc.current

	switch col {
	case 0:
		//pr id
		c.ResultText(string(int((pr.GetID()))))
	case 1:
		//pr title
		c.ResultText(pr.GetTitle())
	case 2:
		//pr autho name
		c.ResultText(pr.User.GetLogin())
	case 3:
		//when pr created
		c.ResultText(pr.GetCreatedAt().String())
	case 4:
		//pr state
		c.ResultText(pr.GetState())
	case 5:
		//merge commit sha
		c.ResultText(pr.GetMergeCommitSHA())
	}
	return nil
}

func (vc *prCursor) Filter(idxNum int, idxStr string, vals []interface{}) error {

	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: os.Getenv("GITHUB_AUTH_TOKEN")},
	)
	tc := oauth2.NewClient(ctx, ts)

	client := github.NewClient(tc)
	info := strings.Split(vc.repoName, "/")
	fmt.Fprintln(os.Stdout, info)

	owner, repo := info[len(info)-2], info[len(info)-1]
	fmt.Fprintln(os.Stdout, owner, repo)

	pr, _, err := client.PullRequests.List(context.Background(), owner, repo, &github.PullRequestListOptions{State: "all", ListOptions: github.ListOptions{PerPage: 50}})
	if err != nil {
		fmt.Println(err)
		return err
	}
	vc.pr = pr
	vc.index = 0
	return nil
}

func (vc *prCursor) Next() error {
	if vc.index < len(vc.pr)-1 {
		vc.index++
		vc.current = vc.pr[vc.index]
	} else {
		vc.current = nil
		return nil
	}
	return nil
}

func (vc *prCursor) EOF() bool {
	return vc.current == nil
}

func (vc *prCursor) Rowid() (int64, error) {
	return int64(0), nil
}

func (vc *prCursor) Close() error {
	vc.current = nil
	return nil
}
