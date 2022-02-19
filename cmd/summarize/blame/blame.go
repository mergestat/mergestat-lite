package blame

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
	"github.com/mergestat/mergestat/cmd/dashboards"
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
WHERE path LIKE '%.go'
`

const blameSummarySQL = `
SELECT
	count(*) AS loc,
	count(distinct path) AS files,
	count(distinct(author_email || author_name)) AS authors,
	PRINTF('%s (%s)',timediff(MAX(author_when)),STRFTIME('%Y-%m-%d',MAX(author_when))) AS latest,
	PRINTF('%s (%s)',timediff(MIN(author_when)),STRFTIME('%Y-%m-%d',MIN(author_when))) AS oldest,
	printduration(AVG(julianday('now') - julianday(author_when))) AS avg_age,
	count(distinct hash) AS commits
FROM preloaded_blame
`

type BlameSummary struct {
	Lines   int            `db:"loc"`
	Files   int            `db:"files"`
	Authors int            `db:"authors"`
	Latest  sql.NullString `db:"latest"`
	Oldest  sql.NullString `db:"oldest"`
	AvgAge  sql.NullString `db:"avg_age"`
	Commits int            `db:"commits"`
}

func (bs *BlameSummary) ToStringArr() []string {
	stringArr := make([]string, 7)
	stringArr[0] = fmt.Sprint(bs.Files)
	stringArr[1] = fmt.Sprint(bs.Lines)
	stringArr[2] = fmt.Sprint(bs.Authors)
	stringArr[3] = fmt.Sprint(bs.Commits)
	stringArr[4] = bs.AvgAge.String
	stringArr[5] = bs.Oldest.String
	stringArr[6] = bs.Latest.String

	return stringArr
}

const blameAuthorSummarySQL = `
SELECT
	author_name, author_email,
	count(*) AS loc,
	(CAST(count(*) AS FLOAT)/(SELECT count(*) FROM preloaded_blame))*100 AS percent_loc,
	PRINTF('%s (%s)',timediff(MAX(author_when)),STRFTIME('%Y-%m-%d',MAX(author_when))) AS latest,
	PRINTF('%s (%s)',timediff(MIN(author_when)),STRFTIME('%Y-%m-%d',MIN(author_when))) AS oldest,
	printduration(AVG(julianday('now') - julianday(author_when))) AS avg_age,
	count(distinct hash) AS commits
FROM preloaded_blame
GROUP BY author_name, author_email
ORDER BY loc DESC
`

type BlameAuthorSummary struct {
	AuthorName   string          `db:"author_name"`
	AuthorEmail  string          `db:"author_email"`
	Lines        int             `db:"loc"`
	LinesPercent sql.NullFloat64 `db:"percent_loc"`
	Latest       sql.NullString  `db:"latest"`
	Oldest       sql.NullString  `db:"oldest"`
	AvgAge       sql.NullString  `db:"avg_age"`
	Commits      int             `db:"commits"`
}
type BlameAuthorSummarySlice struct {
	bass []BlameAuthorSummary
}

func (bass *BlameAuthorSummarySlice) ToStringArr(delimiter ...string) []string {
	var stringifiedRow string
	fullStringArr := make([]string, len(bass.bass))
	delim := "\t"
	if len(delimiter) == 1 {
		delim = delimiter[0]
	}
	for i, v := range bass.bass {
		stringifiedRow = ""
		stringifiedRow += v.AuthorName + delim
		stringifiedRow += fmt.Sprint(v.Lines) + delim
		stringifiedRow += fmt.Sprintf("%.2f", v.LinesPercent.Float64) + delim
		stringifiedRow += fmt.Sprintf("%d", v.Commits) + delim
		stringifiedRow += v.AvgAge.String + delim
		stringifiedRow += v.Oldest.String + delim
		stringifiedRow += v.Latest.String
		fullStringArr[i] = stringifiedRow
	}
	return fullStringArr
}

type TermUI struct {
	db                   *sqlx.DB
	pathPattern          string
	err                  error
	spinner              spinner.Model
	blamePreloaded       bool
	blameSummary         *BlameSummary
	blameAuthorSummaries []BlameAuthorSummary
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
	var blameAuthorSummaries []BlameAuthorSummary
	if err := t.db.Select(&blameAuthorSummaries, blameAuthorSummarySQL); err != nil {
		return err
	}
	t.blameAuthorSummaries = blameAuthorSummaries
	return nil
}

func (t *TermUI) renderBlameSummaryTable(boldHeader bool) string {
	var b *bytes.Buffer
	var err error
	var headingStyle = lipgloss.NewStyle().Bold(boldHeader)

	rows := []string{
		headingStyle.Render("Matched Files"),
		headingStyle.Render("Total Lines"),
		headingStyle.Render("Distinct Authors"),
		headingStyle.Render("Commits"),
		headingStyle.Render("Avg. Age of Lines"),
		headingStyle.Render("Oldest Line"),
		headingStyle.Render("Newest Line"),
	}
	if t.blameSummary != nil {
		b, err = dashboards.OneToOneOutputBuilder(rows, t.blameSummary)
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

func (t *TermUI) renderBlameAuthorSummary(limit int) string {
	var b bytes.Buffer
	p := message.NewPrinter(language.English)

	if t.blameAuthorSummaries != nil && t.blameSummary != nil {

		if len(t.blameAuthorSummaries) == 0 {
			return "<no authors>"
		}

		headers := []string{
			"Author",
			"Blameable Lines",
			"Line %%",
			"Commits",
			"Avg. Age",
			"First Commit",
			"Latest Commit",
		}
		var bass BlameAuthorSummarySlice
		bass.bass = t.blameAuthorSummaries
		formattedTable, err := dashboards.TableBuilder(headers, &bass)
		if err != nil {
			return err.Error()
		}
		b = *formattedTable

		if limit != 0 {
			d := t.blameSummary.Authors - limit
			if d == 1 {
				p.Fprintf(&b, "...1 more author\n")
			} else if d > 1 {
				p.Fprintf(&b, "...%d more authors\n", d)
			}
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
		output["avgAgeLines"] = t.blameSummary.AvgAge
	}

	if t.blameSummary.Oldest.Valid {
		output["oldestLine"] = t.blameSummary.Oldest.String
	}

	if t.blameSummary.AvgAge.Valid {
		output["newestLine"] = t.blameSummary.Latest.String
	}

	authorSummaries := make([]map[string]interface{}, len(t.blameAuthorSummaries))

	for i, authorSummary := range t.blameAuthorSummaries {
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

		authorSummaries[i] = map[string]interface{}{
			"name":           authorSummary.AuthorName,
			"email":          authorSummary.AuthorEmail,
			"blameableLines": authorSummary.Lines,
			"linePercent":    linesPercent,
			"commits":        authorSummary.Commits,
			"avgAge":         authorSummary.AvgAge.String,
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
	defer t.db.Close()
	if t.err != nil {
		return t.err
	}
	return nil
}
