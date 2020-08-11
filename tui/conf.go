package tui

var (
	Queries = [][]string{

		[]string{"SELECT * FROM commits", "all entries from commits table"},
		[]string{"SELECT DISTINCT author_email FROM commits", "distinct author emails from commits table"},
		[]string{`SELECT 
		author_email, count(*) 
		FROM commits GROUP BY author_email 
		ORDER BY count(*) DESC`,
			`number of commits for each author by email`},
		[]string{`SELECT 
		count(*) AS commits, SUM(additions) AS additions, SUM(deletions) AS  deletions, author_email 
		FROM commits 
		GROUP BY author_email
		ORDER BY commits`,
			`number of additions and deletions for each author`},
		[]string{`SELECT
		count(*) AS commits,
		count(CASE WHEN strftime('%w',author_when)='0' THEN 1 END) AS sunday,
		count(CASE WHEN strftime('%w',author_when)='1' THEN 1 END) AS monday,
		count(CASE WHEN strftime('%w',author_when)='2' THEN 1 END) AS tuesday,
		count(CASE WHEN strftime('%w',author_when)='3' THEN 1 END) AS wednesday,
		count(CASE WHEN strftime('%w',author_when)='4' THEN 1 END) AS thursday,
		count(CASE WHEN strftime('%w',author_when)='5' THEN 1 END) AS friday,
		count(CASE WHEN strftime('%w',author_when)='6' THEN 1 END) AS saturday,
		author_email
		FROM commits GROUP BY author_email ORDER BY commits`,
			`number of commits for each author day of the week`},
	}
)
