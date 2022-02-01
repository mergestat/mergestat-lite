package summary

import (
	"bytes"
	"database/sql"
	"encoding/json"
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
CREATE TABLE preloaded_commit_stats AS SELECT * FROM commits, stats('', commits.hash) WHERE file_path LIKE $file_path AND author_when > date($start, $start_mod) AND author_when < date($end, $end_mod);
CREATE TABLE preloaded_commits AS SELECT hash, author_name, author_email, author_when, parents FROM preloaded_commit_stats GROUP BY hash;
`

const commitSummarySQL = `
SELECT
	(SELECT count(*) FROM preloaded_commits) AS total,
	(SELECT count(*) FROM preloaded_commits WHERE parents < 2) AS total_non_merges,
	(SELECT author_when FROM preloaded_commits ORDER BY author_when ASC LIMIT 1) AS first_commit,
	(SELECT author_when FROM preloaded_commits ORDER BY author_when DESC LIMIT 1) AS last_commit,
	(SELECT count(distinct(author_email || author_name)) FROM preloaded_commits) AS distinct_authors,
	(SELECT count(distinct(file_path)) FROM preloaded_commit_stats WHERE file_path LIKE ?) AS distinct_files
`

type CommitAuthorSummary struct {
	AuthorName    string `db:"author_name"`
	AuthorEmail   string `db:"author_email"`
	Commits       int    `db:"commit_count"`
	Additions     int    `db:"additions"`
	Deletions     int    `db:"deletions"`
	DistinctFiles int    `db:"distinct_files"`
	FirstCommit   string `db:"first_commit"`
	LastCommit    string `db:"last_commit"`
}

const commitAuthorSummarySQL = `
SELECT
	author_name, author_email,
	count(distinct hash) AS commit_count,
	sum(additions) AS additions,
	sum(deletions) AS deletions,
	count(distinct file_path) AS distinct_files,
	min(author_when) AS first_commit,
	max(author_when) AS last_commit
FROM preloaded_commit_stats
GROUP BY author_name, author_email
ORDER BY commit_count DESC
LIMIT 25
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
	commitAuthorSummaries *[]*CommitAuthorSummary
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

	if pathPattern == "" {
		pathPattern = "%"
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
	if _, err := t.db.Exec(preloadCommitsSQL,
		sql.Named("file_path", t.pathPattern),
		sql.Named("start", t.dateFilterStart.date),
		sql.Named("start_mod", t.dateFilterStart.mod),
		sql.Named("end", t.dateFilterEnd.date),
		sql.Named("end_mod", t.dateFilterEnd.mod),
	); err != nil {
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
	var commitAuthorSummaries []*CommitAuthorSummary
	if err := t.db.Select(&commitAuthorSummaries, commitAuthorSummarySQL); err != nil {
		return err
	}

	t.commitAuthorSummaries = &commitAuthorSummaries
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
		strings.Join([]string{headingStyle.Render("Files Δ"), distinctFiles}, "\t"),
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
	w := tabwriter.NewWriter(&b, 0, 0, 3, ' ', tabwriter.TabIndent)

	if t.commitAuthorSummaries != nil && t.commitSummary != nil {

		if len(*t.commitAuthorSummaries) == 0 {
			return "<no authors>"
		}

		r := strings.Join([]string{
			"Author",
			"Commits",
			"Commit %",
			"Files Δ",
			"Additions",
			"Deletions",
			"First Commit",
			"Latest Commit",
		}, "\t")

		p.Fprintln(w, r)

		for _, authorRow := range *t.commitAuthorSummaries {
			commitPercent := (float32(authorRow.Commits) / float32(t.commitSummary.Total)) * 100.0

			var firstCommit, lastCommit time.Time
			var err error
			if firstCommit, err = time.Parse(time.RFC3339, authorRow.FirstCommit); err != nil {
				return err.Error()
			}
			if lastCommit, err = time.Parse(time.RFC3339, authorRow.LastCommit); err != nil {
				return err.Error()
			}

			r := strings.Join([]string{
				authorRow.AuthorName,
				p.Sprintf("%d", authorRow.Commits),
				p.Sprintf("%.2f%%", commitPercent),
				p.Sprintf("%d", authorRow.DistinctFiles),
				p.Sprintf("%d", authorRow.Additions),
				p.Sprintf("%d", authorRow.Deletions),
				p.Sprintf("%s (%s)", timediff.TimeDiff(firstCommit), firstCommit.Format("2006-01-02")),
				p.Sprintf("%s (%s)", timediff.TimeDiff(lastCommit), lastCommit.Format("2006-01-02")),
			}, "\t")

			p.Fprintln(w, r)
		}

		if err := w.Flush(); err != nil {
			return err.Error()
		}

		d := t.commitSummary.DistinctAuthors - len(*t.commitAuthorSummaries)
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

	authorSummaries := make([]map[string]interface{}, len(*t.commitAuthorSummaries))

	for i, authorSummary := range *t.commitAuthorSummaries {
		commitPercent := (float32(authorSummary.Commits) / float32(t.commitSummary.Total)) * 100.0
		authorSummaries[i] = map[string]interface{}{
			"name":          authorSummary.AuthorName,
			"email":         authorSummary.AuthorEmail,
			"commits":       authorSummary.Commits,
			"commitPercent": commitPercent,
			"filesModified": authorSummary.DistinctFiles,
			"additions":     authorSummary.Additions,
			"deletions":     authorSummary.Deletions,
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
