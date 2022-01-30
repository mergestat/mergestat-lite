package summary

import (
	"bytes"
	"database/sql"
	"fmt"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jmoiron/sqlx"
	"github.com/mergestat/timediff"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

type CommitSummary struct {
	Total           int            `db:"total"`
	TotalNonMerges  int            `db:"total_non_merges"`
	FirstCommit     sql.NullString `db:"first_commit"`
	LastCommit      sql.NullString `db:"last_commit"`
	DistinctAuthors int            `db:"distinct_authors"`
	DistinctFiles   int            `db:"distinct_files"`
}

const preloadCommitsSQL = `
CREATE TABLE preloaded_commit_stats AS SELECT * FROM commits, stats('', commits.hash) WHERE file_path LIKE '%';
CREATE TABLE preloaded_commits AS SELECT hash, author_name, author_email, author_when, parents FROM preloaded_commit_stats GROUP BY hash;
CREATE TABLE preloaded_commits_summary AS SELECT
(SELECT count(*) FROM preloaded_commits) AS total,
(SELECT count(*) FROM preloaded_commits WHERE parents < 2) AS total_non_merges,
(SELECT author_when FROM preloaded_commits ORDER BY author_when ASC LIMIT 1) AS first_commit,
(SELECT author_when FROM preloaded_commits ORDER BY author_when DESC LIMIT 1) AS last_commit,
(SELECT count(distinct(author_email || author_name)) FROM preloaded_commits) AS distinct_authors,
(SELECT count(distinct(path)) FROM files WHERE path LIKE '%') AS distinct_files;
`

const commitSummarySQL = `
SELECT
	total,
	total_non_merges,
	first_commit,
	last_commit,
	distinct_authors,
	distinct_files
	FROM preloaded_commits_summary
`

type CommitAuthorSummary struct {
	AuthorName    string  `db:"author_name"`
	AuthorEmail   string  `db:"author_email"`
	Commits       int     `db:"commit_count"`
	CommitPercent float32 `db:"commit_percent"`
	Additions     int     `db:"additions"`
	Deletions     int     `db:"deletions"`
	DistinctFiles int     `db:"distinct_files"`
	FirstCommit   string  `db:"first_commit"`
	LastCommit    string  `db:"last_commit"`
}

const commitAuthorSummarySQL = `
SELECT author_name,
    author_email,
    count(distinct hash) AS commit_count,
    ROUND(CAST(count(distinct hash) as REAL) / total, 4) * 100 AS commit_percent,
    sum(additions) AS additions,
    sum(deletions) AS deletions,
    count(distinct file_path) AS distinct_files,
    CASE
        WHEN julianday('now', 'localtime') - julianday(min(author_when), 'localtime') > 730 THEN 
            PRINTF('%d years ago (%s)',ROUND((julianday('now', 'localtime') - julianday(min(author_when), 'localtime')) / 365),DATE(author_when))
        WHEN julianday('now', 'localtime') - julianday(min(author_when), 'localtime') > 365 THEN 
            PRINTF('1 year ago (%s)', DATE(author_when))
        WHEN julianday('now', 'localtime') - julianday(min(author_when), 'localtime') > 31 THEN 
        PRINTF('%d months ago (%s)',ROUND((julianday('now', 'localtime') - julianday(min(author_when), 'localtime')) /(365 / 12)), DATE(author_when))
        WHEN ROUND(julianday('now', 'localtime') - julianday(min(author_when), 'localtime')) = 1 THEN
            PRINTF('1 day ago (%s)',DATE(author_when))
        ELSE PRINTF('%d day ago (%s)',ROUND(julianday('now', 'localtime') - julianday(min(author_when), 'localtime')),DATE(author_when))
    END AS first_commit,
    CASE
        WHEN julianday('now', 'localtime') - julianday(max(author_when), 'localtime') > 730 THEN 
            PRINTF('%d years ago (%s)', ROUND((julianday('now', 'localtime') - julianday(max(author_when), 'localtime')) / 365), DATE(author_when))
        WHEN julianday('now', 'localtime') - julianday(max(author_when), 'localtime') > 365 THEN 
            PRINTF('1 year ago (%s)', DATE(author_when))
        WHEN julianday('now', 'localtime') - julianday(max(author_when), 'localtime') > 31 THEN 
            PRINTF('%d months ago (%s)',ROUND((julianday('now', 'localtime') - julianday(max(author_when))) /(365 / 12)), DATE(author_when))
        WHEN ROUND(julianday('now', 'localtime') - julianday(max(author_when), 'localtime')) = 1 THEN
            PRINTF('1 day ago (%s)',DATE(author_when))
        ELSE PRINTF('%d day(s) ago (%s)',ROUND(julianday('now', 'localtime') - julianday(max(author_when), 'localtime')),DATE(author_when))
    END AS last_commit
FROM preloaded_commit_stats,
    preloaded_commits_summary
GROUP BY author_name,
    author_email
ORDER BY commit_count DESC
LIMIT 25;
`

type TermUI struct {
	db                    *sqlx.DB
	pathPattern           string
	err                   error
	spinner               spinner.Model
	commitsPreloaded      bool
	commitSummary         *CommitSummary
	commitAuthorSummaries []CommitAuthorSummary
}

func NewTermUI(pathPattern string) (*TermUI, error) {
	var db *sqlx.DB
	var err error
	if db, err = sqlx.Open("sqlite3", "file::memory:?cache=shared"); err != nil {
		return nil, fmt.Errorf("failed to initialize database connection: %v", err)
	}
	db.SetMaxOpenConns(1)

	s := spinner.New()
	s.Spinner = spinner.Spinner{
		Frames: []string{".", "..", "..."},
		FPS:    300 * time.Millisecond,
	}

	if pathPattern == "" {
		pathPattern = "%"
	}

	return &TermUI{
		db:          db,
		pathPattern: pathPattern,
		spinner:     s,
	}, nil
}

func (t *TermUI) Init() tea.Cmd {
	return tea.Batch(
		t.spinner.Tick,
		t.preloadCommits,
		t.loadCommitSummary,
		t.loadAuthorCommitSummary,
	)
}

func (t *TermUI) preloadCommits() tea.Msg {
	if _, err := t.db.Exec(preloadCommitsSQL, t.pathPattern); err != nil {
		return err
	}

	t.commitsPreloaded = true
	return nil
}

func (t *TermUI) loadCommitSummary() tea.Msg {
	for !t.commitsPreloaded {
		time.Sleep(300 * time.Millisecond)
	}
	var commitSummary CommitSummary
	if err := t.db.QueryRowx(commitSummarySQL, t.pathPattern).StructScan(&commitSummary); err != nil {
		return err
	}

	t.commitSummary = &commitSummary
	return nil
}

func (t *TermUI) loadAuthorCommitSummary() tea.Msg {
	for !t.commitsPreloaded {
		time.Sleep(300 * time.Millisecond)
	}
	var commitAuthorSummaries []CommitAuthorSummary
	if err := t.db.Select(&commitAuthorSummaries, commitAuthorSummarySQL); err != nil {
		return err
	}

	t.commitAuthorSummaries = commitAuthorSummaries
	return nil
}

func (t *TermUI) renderCommitSummaryTable(boldHeader bool) string {
	var b bytes.Buffer
	p := message.NewPrinter(language.English)
	w := tabwriter.NewWriter(&b, 0, 0, 3, ' ', tabwriter.TabIndent)

	var total, totalNonMerges, distinctFiles, distinctAuthors, firstCommit, lastCommit string

	if t.commitSummary != nil {
		total = p.Sprintf("%d", t.commitSummary.Total)
		totalNonMerges = p.Sprintf("%d", t.commitSummary.TotalNonMerges)
		distinctFiles = p.Sprintf("%d", t.commitSummary.DistinctFiles)
		distinctAuthors = p.Sprintf("%d", t.commitSummary.DistinctAuthors)

		firstCommit, lastCommit = "<none>", "<none>"

		if t.commitSummary.FirstCommit.Valid {
			when, _ := time.Parse(time.RFC3339, t.commitSummary.FirstCommit.String)
			firstCommit = fmt.Sprintf("%s (%s)", timediff.TimeDiff(when), when.Format("2006-01-02"))
		}

		if t.commitSummary.LastCommit.Valid {
			when, _ := time.Parse(time.RFC3339, t.commitSummary.LastCommit.String)
			lastCommit = fmt.Sprintf("%s (%s)", timediff.TimeDiff(when), when.Format("2006-01-02"))
		}

	} else {
		total = t.spinner.View()
		totalNonMerges = t.spinner.View()
		distinctFiles = t.spinner.View()
		distinctAuthors = t.spinner.View()
		firstCommit = t.spinner.View()
		lastCommit = t.spinner.View()
	}

	var headingStyle = lipgloss.NewStyle().Bold(boldHeader)

	rows := []string{
		strings.Join([]string{headingStyle.Render("Commits"), total}, "\t"),
		strings.Join([]string{headingStyle.Render("Non-Merge Commits"), totalNonMerges}, "\t"),
		strings.Join([]string{headingStyle.Render("Files"), distinctFiles}, "\t"),
		strings.Join([]string{headingStyle.Render("Unique Authors"), distinctAuthors}, "\t"),
		strings.Join([]string{headingStyle.Render("First Commit"), firstCommit}, "\t"),
		strings.Join([]string{headingStyle.Render("Latest Commit"), lastCommit}, "\t"),
	}

	p.Fprintln(w, strings.Join(rows, "\n"))
	if err := w.Flush(); err != nil {
		return err.Error()
	}

	p.Fprintln(&b)
	p.Fprintln(&b)

	return b.String()
}

func (t *TermUI) renderCommitAuthorSummary() string {
	var b bytes.Buffer
	p := message.NewPrinter(language.English)

	if t.commitAuthorSummaries != nil && t.commitSummary != nil {

		if len(t.commitAuthorSummaries) == 0 {
			return "<no authors>"
		}

		headers := []string{
			"Author",
			"Author email",
			"Commits",
			"Commit %%",
			"Files Δ",
			"Additions",
			"Deletions",
			"First Commit",
			"Latest Commit",
		}

		// put t.commitAuthorSummaries in a struct. I dream of generics
		s := make([]interface{}, len(t.commitAuthorSummaries))
		for i, v := range t.commitAuthorSummaries {
			s[i] = v
		}
		formattedTable, err := tableBuilder(headers, s...)
		if err != nil {
			return err.Error()
		}
		b = *formattedTable

		d := t.commitSummary.DistinctAuthors - len(t.commitAuthorSummaries)
		if d == 1 {
			p.Fprintf(&b, "...1 more author\n")
		} else if d > 1 {
			p.Fprintf(&b, "...%d more authors\n", d)
		}
	} else {
		p.Fprintln(&b, "Loading authors", t.spinner.View())
	}

	return b.String()
}

func (t *TermUI) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case error:
		t.err = msg
		return t, tea.Quit

	case tea.KeyMsg:
		switch msg.String() {

		case "ctrl+c", "q":
			return t, tea.Quit
		}

	default:
		if t.commitSummary != nil && t.commitAuthorSummaries != nil && t.commitsPreloaded {
			return t, tea.Quit
		}
		var cmd tea.Cmd
		t.spinner, cmd = t.spinner.Update(msg)
		return t, cmd
	}

	return t, nil
}

func (t *TermUI) View() string {
	if t.err != nil {
		return t.err.Error()
	}

	var b bytes.Buffer
	fmt.Fprint(&b, t.renderCommitSummaryTable(true))
	fmt.Fprint(&b, t.renderCommitAuthorSummary())

	return b.String()
}

func (t *TermUI) PrintNoTTY() string {
	t.preloadCommits()
	t.loadCommitSummary()
	t.loadAuthorCommitSummary()

	if t.err != nil {
		return t.err.Error()
	}

	var b bytes.Buffer
	fmt.Fprint(&b, t.renderCommitSummaryTable(false))
	fmt.Fprint(&b, t.renderCommitAuthorSummary())

	return b.String()
}

func (t *TermUI) Close() error {
	return t.db.Close()
}
