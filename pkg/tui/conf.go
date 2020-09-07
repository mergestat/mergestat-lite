package tui

// type ExampleQuery struct
// 	Query       string
// 	Description string
//

var (
	Queries = map[string]string{

		"commit-info":      "SELECT * FROM commits",
		"distinct-authors": "SELECT DISTINCT author_email FROM commits",
		"commits-per-author": `SELECT 
		author_email, count(*) 
		FROM commits GROUP BY author_email 
		ORDER BY count(*) DESC`,
		"author-stats": `SELECT 
		count(*) AS commits, SUM(additions) AS additions, SUM(deletions) AS  deletions, author_email 
		FROM commits 
		GROUP BY author_email
		ORDER BY commits`,
		"author-commits-dow": `SELECT
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
	}
)
