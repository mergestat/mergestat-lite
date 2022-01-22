package cmd

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/jmoiron/sqlx"
	"github.com/mergestat/timediff"
	"github.com/spf13/cobra"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

var headingStyle = lipgloss.NewStyle().
	Bold(true)

// var underlineStyle = lipgloss.NewStyle().Underline(true)

// var textStyle = lipgloss.NewStyle()

type CommitSummary struct {
	Total           int       `db:"total"`
	TotalNonMerges  int       `db:"total_non_merges"`
	FirstCommit     time.Time `db:"first_commit"`
	LastCommit      time.Time `db:"last_commit"`
	DistinctAuthors int       `db:"distinct_authors"`
	DistinctFiles   int       `db:"distinct_files"`
}

const commitSummarySQL = `
SELECT
	(SELECT count(*) FROM commits) AS total,
	(SELECT count(*) FROM commits WHERE parents < 2) AS total_non_merges,
	(SELECT author_when FROM commits ORDER BY author_when ASC LIMIT 1) AS first_commit,
	(SELECT author_when FROM commits ORDER BY author_when DESC LIMIT 1) AS last_commit,
	(SELECT count(distinct(author_email || author_name)) FROM commits) AS distinct_authors,
	(SELECT count(distinct(path)) FROM files) AS distinct_files
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
FROM commits, stats('', commits.hash)
GROUP BY author_name, author_email
ORDER BY commit_count DESC
LIMIT 25
`

var summaryCmd = &cobra.Command{
	Use:  "summary",
	Long: "prints a summary of commit activity in the default repository.",
	Args: cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		var db *sqlx.DB
		var err error
		if db, err = sqlx.Open("sqlite3", ":memory:"); err != nil {
			handleExitError(fmt.Errorf("failed to initialize database connection: %v", err))
		}
		defer func() {
			if err := db.Close(); err != nil {
				handleExitError(err)
			}
		}()

		p := message.NewPrinter(language.English)

		var commitSummary CommitSummary
		if err := db.QueryRowx(commitSummarySQL).StructScan(&commitSummary); err != nil {
			handleExitError(err)
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', tabwriter.TabIndent)

		rows := []string{
			strings.Join([]string{headingStyle.Render("Commits"), p.Sprintf("%d", commitSummary.Total)}, "\t"),
			strings.Join([]string{headingStyle.Render("Non-Merge Commits"), p.Sprintf("%d", commitSummary.TotalNonMerges)}, "\t"),
			strings.Join([]string{headingStyle.Render("Files"), p.Sprintf("%d", commitSummary.DistinctFiles)}, "\t"),
			strings.Join([]string{headingStyle.Render("Unique Authors"), p.Sprintf("%d", commitSummary.DistinctAuthors)}, "\t"),
			strings.Join([]string{headingStyle.Render("First Commit"), fmt.Sprintf("%s (%s)", timediff.TimeDiff(commitSummary.FirstCommit), commitSummary.FirstCommit.Format("2006-01-02"))}, "\t"),
			strings.Join([]string{headingStyle.Render("Latest Commit"), fmt.Sprintf("%s (%s)", timediff.TimeDiff(commitSummary.LastCommit), commitSummary.FirstCommit.Format("2006-01-02"))}, "\t"),
		}
		p.Fprintln(w, strings.Join(rows, "\n"))

		if err := w.Flush(); err != nil {
			handleExitError(err)
		}

		p.Println()
		p.Println()

		var commitAuthorSummaries []*CommitAuthorSummary
		if err := db.Select(&commitAuthorSummaries, commitAuthorSummarySQL); err != nil {
			handleExitError(err)
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

		for _, authorRow := range commitAuthorSummaries {
			commitPercent := (float32(authorRow.Commits) / float32(commitSummary.Total)) * 100.0

			var firstCommit, lastCommit time.Time
			if firstCommit, err = time.Parse(time.RFC3339, authorRow.FirstCommit); err != nil {
				handleExitError(err)
			}
			if lastCommit, err = time.Parse(time.RFC3339, authorRow.LastCommit); err != nil {
				handleExitError(err)
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

		d := commitSummary.DistinctAuthors - len(commitAuthorSummaries)
		if d == 1 {
			p.Fprintf(w, "...1 more author\n")
		} else if d > 1 {
			p.Fprintf(w, "...%d more authors\n", d)
		}

		if err := w.Flush(); err != nil {
			handleExitError(err)
		}
	},
}
