package blame

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jmoiron/sqlx"
	"github.com/mergestat/timediff"
	"github.com/mergestat/timediff/locale"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

const preloadBlameSQL = `
CREATE TABLE preloaded_blame AS
SELECT
    files.path,
    blame.line_no,
    commits.hash,
	commits.author_name,
	commits.author_email,
	commits.author_when,
	commits.committer_name,
	commits.committer_email,
	commits.committer_when
FROM files, blame('', '', files.path)
JOIN commits ON commits.hash = blame.commit_hash
WHERE path LIKE ?
`

const blameSummarySQL = `
SELECT
	count(*) AS loc,
	count(distinct path) AS files,
	count(distinct(author_email || author_name)) AS authors,
	MAX(author_when) AS latest,
	MIN(author_when) AS oldest,
	AVG(julianday('now') - julianday(author_when)) AS avg_age,
	count(distinct hash) AS commits
FROM preloaded_blame
`

type BlameSummary struct {
	Lines   int             `db:"loc"`
	Files   int             `db:"files"`
	Authors int             `db:"authors"`
	Latest  sql.NullString  `db:"latest"`
	Oldest  sql.NullString  `db:"oldest"`
	AvgAge  sql.NullFloat64 `db:"avg_age"`
	Commits int             `db:"commits"`
}

const blameAuthorSummarySQL = `
SELECT
	author_name, author_email,
	count(*) AS loc,
	MAX(author_when) AS latest,
	MIN(author_when) AS oldest,
	AVG(julianday('now') - julianday(author_when)) AS avg_age,
	count(distinct hash) AS commits,
	json_group_array(path) AS files
FROM preloaded_blame
GROUP BY author_name, author_email
ORDER BY loc DESC
`

type BlameAuthorSummary struct {
	AuthorName  string          `db:"author_name"`
	AuthorEmail string          `db:"author_email"`
	Lines       int             `db:"loc"`
	Latest      sql.NullString  `db:"latest"`
	Oldest      sql.NullString  `db:"oldest"`
	AvgAge      sql.NullFloat64 `db:"avg_age"`
	Commits     int             `db:"commits"`
	Files       string          `db:"files"`
}

type TermUI struct {
	db                   *sqlx.DB
	pathPattern          string
	err                  error
	spinner              spinner.Model
	blamePreloaded       bool
	blameSummary         *BlameSummary
	blameAuthorSummaries *[]*BlameAuthorSummary
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
		t.preloadBlame,
		t.loadBlameSummary,
		t.loadBlameAuthorSummary,
	)
}

func (t *TermUI) preloadBlame() tea.Msg {
	if _, err := t.db.Exec(preloadBlameSQL, t.pathPattern); err != nil {
		return err
	}

	t.blamePreloaded = true
	return nil
}

func (t *TermUI) loadBlameSummary() tea.Msg {
	for !t.blamePreloaded {
		time.Sleep(300 * time.Millisecond)
	}
	var blameSummary BlameSummary
	if err := t.db.QueryRowx(blameSummarySQL).StructScan(&blameSummary); err != nil {
		return err
	}

	t.blameSummary = &blameSummary
	return nil
}

func (t *TermUI) loadBlameAuthorSummary() tea.Msg {
	for !t.blamePreloaded {
		time.Sleep(300 * time.Millisecond)
	}
	var blameAuthorSummaries []*BlameAuthorSummary
	if err := t.db.Select(&blameAuthorSummaries, blameAuthorSummarySQL); err != nil {
		return err
	}

	t.blameAuthorSummaries = &blameAuthorSummaries
	return nil
}

func (t *TermUI) renderDurationString(d time.Duration) string {
	f := timediff.WithCustomFormatters(locale.Formatters{
		time.Second:           func(_ time.Duration) string { return "<none>" },
		44 * time.Second:      func(_ time.Duration) string { return "a few seconds" },
		89 * time.Second:      func(_ time.Duration) string { return "1 minute" },
		44 * time.Minute:      func(d time.Duration) string { return fmt.Sprintf("%.0f minutes", math.Ceil(d.Minutes())) },
		89 * time.Minute:      func(_ time.Duration) string { return "1 hour" },
		21 * time.Hour:        func(d time.Duration) string { return fmt.Sprintf("%.0f hours", math.Ceil(d.Hours())) },
		35 * time.Hour:        func(_ time.Duration) string { return "1 day" },
		25 * (24 * time.Hour): func(d time.Duration) string { return fmt.Sprintf("%.0f days", math.Ceil(d.Hours()/24.0)) },
		45 * (24 * time.Hour): func(_ time.Duration) string { return "1 month" },
		10 * (24 * time.Hour) * 30: func(d time.Duration) string {
			return fmt.Sprintf("%.0f months", math.Ceil(d.Hours()/(24.0*30)))
		},
		17 * (24 * time.Hour) * 30: func(d time.Duration) string {
			return fmt.Sprintf("1 year (%.0f months)", math.Round(d.Hours()/(24.0*30)))
		},
		1<<63 - 1: func(d time.Duration) string {
			return fmt.Sprintf("%.0f years (%.0f months)", math.Ceil(d.Hours()/(24.0*30*12)), math.Round(d.Hours()/(24.0*30)))
		},
	})

	return timediff.TimeDiff(time.Now().Add(-d), f)
}

func (t *TermUI) renderBlameSummaryTable(boldHeader bool) string {
	var b bytes.Buffer
	p := message.NewPrinter(language.English)
	w := tabwriter.NewWriter(&b, 0, 0, 3, ' ', tabwriter.TabIndent)

	var files, authors, commits, avgAge, lines string
	firstCommit, lastCommit := "<none>", "<none>"

	if t.blameSummary != nil {
		files = p.Sprintf("%d", t.blameSummary.Files)
		authors = p.Sprintf("%d", t.blameSummary.Authors)
		commits = p.Sprintf("%d", t.blameSummary.Commits)
		var avgAgeDur time.Duration
		if t.blameSummary.AvgAge.Valid {
			avgAgeDur = time.Duration((t.blameSummary.AvgAge.Float64 * 24 * float64(time.Hour.Nanoseconds())))
		}

		avgAge = p.Sprintf("%s", t.renderDurationString(avgAgeDur))
		lines = p.Sprintf("%d", t.blameSummary.Lines)

		if t.blameSummary.Oldest.Valid {
			when, _ := time.Parse(time.RFC3339, t.blameSummary.Oldest.String)
			firstCommit = fmt.Sprintf("%s (%s)", timediff.TimeDiff(when), when.Format("2006-01-02"))
		}

		if t.blameSummary.Latest.Valid {
			when, _ := time.Parse(time.RFC3339, t.blameSummary.Latest.String)
			lastCommit = fmt.Sprintf("%s (%s)", timediff.TimeDiff(when), when.Format("2006-01-02"))
		}

	} else {
		files = t.spinner.View()
		authors = t.spinner.View()
		commits = t.spinner.View()
		avgAge = t.spinner.View()
		lines = t.spinner.View()
		firstCommit = t.spinner.View()
		lastCommit = t.spinner.View()
	}

	var headingStyle = lipgloss.NewStyle().Bold(boldHeader)

	rows := []string{
		strings.Join([]string{headingStyle.Render("Matched Files"), files}, "\t"),
		strings.Join([]string{headingStyle.Render("Total Lines"), lines}, "\t"),
		strings.Join([]string{headingStyle.Render("Distinct Authors"), authors}, "\t"),
		strings.Join([]string{headingStyle.Render("Commits"), commits}, "\t"),
		strings.Join([]string{headingStyle.Render("Avg. Age of Lines"), avgAge}, "\t"),
		strings.Join([]string{headingStyle.Render("Oldest Line"), firstCommit}, "\t"),
		strings.Join([]string{headingStyle.Render("Newest Line"), lastCommit}, "\t"),
	}

	p.Fprintln(w, strings.Join(rows, "\n"))
	if err := w.Flush(); err != nil {
		return err.Error()
	}

	p.Fprintln(&b)
	p.Fprintln(&b)

	return b.String()
}

func (t *TermUI) renderBlameAuthorSummary(limit int) string {
	var b bytes.Buffer
	p := message.NewPrinter(language.English)
	w := tabwriter.NewWriter(&b, 0, 0, 3, ' ', tabwriter.TabIndent)

	if t.blameAuthorSummaries != nil && t.blameSummary != nil {

		if len(*t.blameAuthorSummaries) == 0 {
			return "<no authors>"
		}

		r := strings.Join([]string{
			"Author",
			"Blameable Lines",
			"Line %",
			"Commits",
			"Avg. Age",
			"First Commit",
			"Latest Commit",
		}, "\t")

		p.Fprintln(w, r)

		for i, authorRow := range *t.blameAuthorSummaries {
			if i > limit-1 && limit != 0 {
				break
			}

			linesPercent := (float32(authorRow.Lines) / float32(t.blameSummary.Lines)) * 100.0

			var firstCommit, lastCommit time.Time
			var err error

			if authorRow.Oldest.Valid {
				if firstCommit, err = time.Parse(time.RFC3339, authorRow.Oldest.String); err != nil {
					return err.Error()
				}
			}

			if authorRow.Latest.Valid {
				if lastCommit, err = time.Parse(time.RFC3339, authorRow.Latest.String); err != nil {
					return err.Error()
				}
			}

			var avgAgeDur time.Duration
			if authorRow.AvgAge.Valid {
				avgAgeDur = time.Duration((authorRow.AvgAge.Float64 * 24 * float64(time.Hour.Nanoseconds())))
			}

			r := strings.Join([]string{
				authorRow.AuthorName,
				p.Sprintf("%d", authorRow.Lines),
				p.Sprintf("%.2f%%", linesPercent),
				p.Sprintf("%d", authorRow.Commits),
				p.Sprintf("%s", t.renderDurationString(avgAgeDur)),
				p.Sprintf("%s (%s)", timediff.TimeDiff(firstCommit), firstCommit.Format("2006-01-02")),
				p.Sprintf("%s (%s)", timediff.TimeDiff(lastCommit), lastCommit.Format("2006-01-02")),
			}, "\t")

			p.Fprintln(w, r)
		}

		if err := w.Flush(); err != nil {
			return err.Error()
		}

		d := t.blameSummary.Authors - limit
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
		if t.blameSummary != nil && t.blameAuthorSummaries != nil && t.blamePreloaded {
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
	fmt.Fprint(&b, t.renderBlameSummaryTable(true))
	fmt.Fprint(&b, t.renderBlameAuthorSummary(25))

	return b.String()
}

// PrintNoTTY prints a version of output with no terminal styles
func (t *TermUI) PrintNoTTY() string {
	t.preloadBlame()
	t.loadBlameSummary()
	t.loadBlameAuthorSummary()

	if t.err != nil {
		return t.err.Error()
	}

	var b bytes.Buffer
	fmt.Fprint(&b, t.renderBlameSummaryTable(false))
	fmt.Fprint(&b, t.renderBlameAuthorSummary(0))

	return b.String()
}

// PrintJSON outputs summary results as a JSON object
func (t *TermUI) PrintJSON() string {
	t.preloadBlame()
	t.loadBlameSummary()
	t.loadBlameAuthorSummary()

	if t.err != nil {
		return t.err.Error()
	}

	output := map[string]interface{}{
		"matchedFiles":    t.blameSummary.Files,
		"totalLines":      t.blameSummary.Lines,
		"distinctAuthors": t.blameSummary.Authors,
		"commits":         t.blameSummary.Commits,
		"avgAgeLines":     nil,
		"oldestLine":      nil,
		"newestLine":      nil,
	}

	if t.blameSummary.AvgAge.Valid {
		output["avgAgeLines"] = t.blameSummary.AvgAge.Float64
	}

	if t.blameSummary.Oldest.Valid {
		output["oldestLine"] = t.blameSummary.Oldest.String
	}

	if t.blameSummary.AvgAge.Valid {
		output["newestLine"] = t.blameSummary.Latest.String
	}

	authorSummaries := make([]map[string]interface{}, len(*t.blameAuthorSummaries))

	for i, authorSummary := range *t.blameAuthorSummaries {
		linesPercent := (float32(authorSummary.Lines) / float32(t.blameSummary.Lines)) * 100.0

		var firstCommit, lastCommit time.Time
		var err error

		if authorSummary.Oldest.Valid {
			if firstCommit, err = time.Parse(time.RFC3339, authorSummary.Oldest.String); err != nil {
				return err.Error()
			}
		}

		if authorSummary.Latest.Valid {
			if lastCommit, err = time.Parse(time.RFC3339, authorSummary.Latest.String); err != nil {
				return err.Error()
			}
		}

		var avgAgeDur time.Duration
		if authorSummary.AvgAge.Valid {
			avgAgeDur = time.Duration((authorSummary.AvgAge.Float64 * 24 * float64(time.Hour.Nanoseconds())))
		}

		authorSummaries[i] = map[string]interface{}{
			"name":           authorSummary.AuthorName,
			"email":          authorSummary.AuthorEmail,
			"blameableLines": authorSummary.Lines,
			"linePercent":    linesPercent,
			"commits":        authorSummary.Commits,
			"avgAge":         t.renderDurationString(avgAgeDur),
			"avgAgeSeconds":  avgAgeDur.Seconds(),
			"oldestLine":     firstCommit.Format(time.RFC3339),
			"newestLine":     lastCommit.Format(time.RFC3339),
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
