package summary

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jmoiron/sqlx"
	dashboards "github.com/mergestat/mergestat/cmd/dashboards"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

type CommitSummary struct {
	Total           int            `db:"total"`
	TotalNonMerges  int            `db:"total_non_merges"`
	DistinctAuthors int            `db:"distinct_authors"`
	DistinctFiles   int            `db:"distinct_files"`
	FirstCommit     sql.NullString `db:"first_commit"`
	LastCommit      sql.NullString `db:"last_commit"`
}

func (cs *CommitSummary) ToStringArr() []string {
	stringArr := make([]string, 6)
	stringArr[0] = fmt.Sprint(cs.Total)
	stringArr[1] = fmt.Sprint(cs.TotalNonMerges)
	stringArr[2] = fmt.Sprint(cs.DistinctAuthors)
	stringArr[3] = fmt.Sprint(cs.DistinctFiles)
	stringArr[4] = cs.FirstCommit.String
	stringArr[5] = cs.LastCommit.String
	return stringArr
}

// This is a bit odd. We have two queries, one that includes filtering by file_path with a LIKE
// and another that skips file_path filtering completely.
// This is because it is possible to have "empty" commits - commits where no changes were made
// (and therefore there are no file_paths to filter on). For these commits, stats.* will be all NULLs.
// If a user does *not* specify a file pattern to match on, we want to include these empty commits
// in our calculations, therefore we don't mention file_path in the WHERE clause at all.
//
// This is because `file_path LIKE %` will not match when file_path IS NULL (empty commit).
// When a path filter is supplied by the user, we do apply it. Note that even supplying just a '%'
// will exclude empty commits from the resultset. This makes sense, because empty commits won't have
// changed any files in the specified pattern (they won't have changed any files at all).
const preloadCommitsWithFilePathPatternSQL = `
CREATE TABLE preloaded_commit_stats AS SELECT * FROM commits LEFT JOIN stats('', commits.hash) WHERE file_path LIKE $file_path AND author_when > date($start, $start_mod) AND author_when < date($end, $end_mod);
CREATE TABLE preloaded_commits AS SELECT hash, author_name, author_email, author_when, parents FROM preloaded_commit_stats GROUP BY hash;
`

// See comment above
const preloadCommitsWithoutFilePathPatternSQL = `
CREATE TABLE preloaded_commit_stats AS SELECT * FROM commits LEFT JOIN stats('', commits.hash) WHERE author_when > date($start, $start_mod) AND author_when < date($end, $end_mod);
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
	distinct_authors,
	distinct_files,
	PRINTF('%s (%s)',timediff(first_commit),first_commit) AS first_commit,
	PRINTF('%s (%s)',timediff(last_commit),last_commit) AS last_commit
	FROM preloaded_commits_summary;
	`

type CommitAuthorSummary struct {
	AuthorName    string         `db:"author_name"`
	AuthorEmail   string         `db:"author_email"`
	Commits       int            `db:"commit_count"`
	CommitPercent float32        `db:"commit_percent"`
	Additions     sql.NullInt64  `db:"additions"`
	Deletions     sql.NullInt64  `db:"deletions"`
	DistinctFiles int            `db:"distinct_files"`
	FirstCommit   sql.NullString `db:"first_commit"`
	LastCommit    sql.NullString `db:"last_commit"`
}
type CommitAuthorSummarySlice struct {
	casSlice []CommitAuthorSummary
}

// a function that turns a CommitAuthorSummary into a prettified string with a default delimiter of \t
func (cas *CommitAuthorSummarySlice) ToStringArr(delimiter ...string) []string {
	var stringifiedRow string
	fullStringArr := make([]string, len(cas.casSlice))
	delim := "\t"
	if len(delimiter) == 1 {
		delim = delimiter[0]
	}
	for i, v := range cas.casSlice {
		stringifiedRow = ""
		stringifiedRow += v.AuthorName + delim
		stringifiedRow += v.AuthorEmail + delim
		stringifiedRow += fmt.Sprint(v.Commits) + delim
		stringifiedRow += fmt.Sprintf("%.2f", v.CommitPercent) + delim
		stringifiedRow += fmt.Sprint(v.Additions.Int64) + delim
		stringifiedRow += fmt.Sprint(v.Deletions.Int64) + delim
		stringifiedRow += fmt.Sprint(v.DistinctFiles) + delim
		stringifiedRow += v.FirstCommit.String + delim
		stringifiedRow += v.LastCommit.String + delim
		fullStringArr[i] = stringifiedRow
	}
	//println(strings.Join(fullStringArr, " "))
	return fullStringArr
}

const commitAuthorSummarySQL = `
SELECT author_name,
    author_email,
    count(distinct hash) AS commit_count,
    ROUND(CAST(count(distinct hash) as REAL) / total, 4) * 100 AS commit_percent,
    sum(additions) AS additions,
    sum(deletions) AS deletions,
    count(distinct file_path) AS distinct_files,
	PRINTF('%s (%s)',timediff(min(author_when)),min(author_when)) AS first_commit,
	PRINTF('%s (%s)',timediff(max(author_when)),max(author_when)) AS last_commit
FROM preloaded_commit_stats,
    preloaded_commits_summary
GROUP BY author_name,
    author_email
ORDER BY commit_count DESC
`

type dateFilter struct {
	date string
	mod  string
}

type TermUI struct {
	db                    *sqlx.DB
	pathPattern           string
	dateFilterStart       dateFilter
	dateFilterEnd         dateFilter
	err                   error
	spinner               spinner.Model
	commitsPreloaded      bool
	commitSummary         *CommitSummary
	commitAuthorSummaries []CommitAuthorSummary
}

func NewTermUI(pathPattern, dateFilterStart, dateFilterEnd string) (*TermUI, error) {
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

	start := "now"
	startMod := "-1000 years"
	end := "now"
	endMod := "0 days"

	// if the start date cannot be parsed, assume it is a date modifier relative to 'now'
	if _, err := time.Parse("2006-01-02", dateFilterStart); err != nil {
		if dateFilterStart != "" {
			startMod = dateFilterStart
		}
	} else {
		start = dateFilterStart
		startMod = "0 days"
	}

	if _, err := time.Parse("2006-01-02", dateFilterEnd); err != nil {
		if dateFilterEnd != "" {
			endMod = dateFilterEnd
		}
	} else {
		end = dateFilterEnd
	}

	return &TermUI{
		db:              db,
		pathPattern:     pathPattern,
		spinner:         s,
		dateFilterStart: dateFilter{date: start, mod: startMod},
		dateFilterEnd:   dateFilter{date: end, mod: endMod},
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
	preloadCommitsSQL := preloadCommitsWithoutFilePathPatternSQL
	args := []interface{}{
		sql.Named("start", t.dateFilterStart.date),
		sql.Named("start_mod", t.dateFilterStart.mod),
		sql.Named("end", t.dateFilterEnd.date),
		sql.Named("end_mod", t.dateFilterEnd.mod),
	}

	if t.pathPattern != "" {
		preloadCommitsSQL = preloadCommitsWithFilePathPatternSQL
		args = append(args, sql.Named("file_path", t.pathPattern))
	}
	if _, err := t.db.Exec(preloadCommitsSQL, args...); err != nil {
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
	var b *bytes.Buffer
	var err error

	headingStyle := lipgloss.NewStyle().Bold(boldHeader)

	rows := []string{
		headingStyle.Render("Commits"),
		headingStyle.Render("Non-Merge Commits"),
		headingStyle.Render("Files"),
		headingStyle.Render("Unique Authors"),
		headingStyle.Render("First Commit"),
		headingStyle.Render("Latest Commit"),
	}
	if t.commitSummary != nil {
		b, err = dashboards.OneToOneOutputBuilder(rows, t.commitSummary)
		if err != nil {
			return err.Error()
		}
	} else {
		b, err = dashboards.LoadingSymbols(rows, t.spinner)
		if err != nil {
			return err.Error()
		}
	}

	return b.String()
}

func (t *TermUI) renderCommitAuthorSummary(limit int) string {
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
		var casSlice CommitAuthorSummarySlice
		casSlice.casSlice = t.commitAuthorSummaries
		formattedTable, err := dashboards.TableBuilder(headers, &casSlice)
		if err != nil {
			return err.Error()
		}
		b = *formattedTable

		d := t.commitSummary.DistinctAuthors - limit
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
	fmt.Fprint(&b, t.renderCommitAuthorSummary(25))

	return b.String()
}

// PrintNoTTY prints a version of output with no terminal styles
func (t *TermUI) PrintNoTTY() string {
	t.preloadCommits()
	t.loadCommitSummary()
	t.loadAuthorCommitSummary()

	if t.err != nil {
		return t.err.Error()
	}

	var b bytes.Buffer
	fmt.Fprint(&b, t.renderCommitSummaryTable(false))
	fmt.Fprint(&b, t.renderCommitAuthorSummary(0))

	return b.String()
}

// PrintJSON outputs summary results as a JSON object
func (t *TermUI) PrintJSON() string {
	t.preloadCommits()
	t.loadCommitSummary()
	t.loadAuthorCommitSummary()

	if t.err != nil {
		return t.err.Error()
	}

	output := map[string]interface{}{
		"commits":         t.commitSummary.Total,
		"nonMergeCommits": t.commitSummary.TotalNonMerges,
		"filesChanged":    t.commitSummary.DistinctFiles,
		"uniqueAuthors":   t.commitSummary.DistinctAuthors,
		"firstCommit":     t.commitSummary.FirstCommit.String,
		"lastCommit":      t.commitSummary.LastCommit.String,
	}

	authorSummaries := make([]map[string]interface{}, len(t.commitAuthorSummaries))

	for i, authorSummary := range t.commitAuthorSummaries {
		commitPercent := (float32(authorSummary.Commits) / float32(t.commitSummary.Total)) * 100.0
		authorSummaries[i] = map[string]interface{}{
			"name":          authorSummary.AuthorName,
			"email":         authorSummary.AuthorEmail,
			"commits":       authorSummary.Commits,
			"commitPercent": commitPercent,
			"filesModified": authorSummary.DistinctFiles,
			"additions":     authorSummary.Additions.Int64,
			"deletions":     authorSummary.Deletions.Int64,
		}
	}

	output["authors"] = authorSummaries

	if o, err := json.MarshalIndent(output, "", "  "); err != nil {
		return err.Error()
	} else {
		return string(o)
	}
}

func (t *TermUI) Close() error {
	return t.db.Close()
}
