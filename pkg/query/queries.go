package query

var queries = map[string]string{
	// show all commit information from repository, in the current directory
	"commit-info": "SELECT * FROM commits",

	// list all distinct author information from commits in current directory
	"distinct-author-emails": "SELECT DISTINCT( author_email ) FROM commits",

	// list all count of commits, grouping by authors, in descending author
	"commits-per-author": `SELECT 
		author_email, count(*) 
		FROM commits GROUP BY author_email 
		ORDER BY count(*) DESC`,

	"author-stats": `SELECT count(DISTINCT commits.hash) AS commits, SUM(additions) AS additions, SUM(deletions) AS deletions, author_email
		FROM commits, stats('', commits.hash)
		WHERE commits.parents < 2
		GROUP BY author_email ORDER BY commits`,

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

// Find finds and return the named query
func Find(name string) (string, bool) { q, ok := queries[name]; return q, ok }
