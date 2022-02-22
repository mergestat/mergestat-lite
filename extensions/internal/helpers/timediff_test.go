package helpers

import (
	"database/sql"
	"testing"
	"time"

	"github.com/mergestat/mergestat/extensions/internal/tools"
)

func TestTimediff(t *testing.T) {
	type test struct {
		query      string
		queryInput []interface{}
		expected   string
	}
	tests := []test{
		// 3 inputs end date, start date, format
		{query: `SELECT time_diff(DATE('now','-4 year','-6 month'),DATE('now'),'2006-01-02')`, expected: "5 years ago"},
		{query: `SELECT time_diff(DATE('now','-1 year'),DATE('now'),'2006-01-02')`, expected: "a year ago"},
		{query: `SELECT time_diff(DATE('now','-6 month'),DATE('now'),'2006-01-02')`, expected: "7 months ago"},
		{query: `SELECT time_diff(DATE('now','-1 month'),DATE('now'),'2006-01-02')`, expected: "a month ago"},
		{query: `SELECT time_diff(DATE('now','-5 day'),DATE('now'),'2006-01-02')`, expected: "5 days ago"},
		{query: `SELECT time_diff(DATE('now','-1 day'),DATE('now'),'2006-01-02')`, expected: "a day ago"},
		// one input end date
		{query: `SELECT time_diff(?)`, queryInput: []interface{}{time.Now().Add(-time.Hour * 7).Add(time.Minute * 1).Format(time.RFC3339)}, expected: "7 hours ago"},
		{query: `SELECT time_diff(?)`, queryInput: []interface{}{time.Now().Add(-time.Hour * 1).Format(time.RFC3339)}, expected: "an hour ago"},
		{query: `SELECT time_diff(?)`, queryInput: []interface{}{time.Now().Add(-time.Minute * 32).Add(time.Second * 2).Format(time.RFC3339)}, expected: "32 minutes ago"},
		{query: `SELECT time_diff(?)`, queryInput: []interface{}{time.Now().Add(-time.Minute * 1).Format(time.RFC3339)}, expected: "a minute ago"},
		{query: `SELECT time_diff(?)`, queryInput: []interface{}{time.Now().Add(-time.Second * 15).Format(time.RFC3339)}, expected: "a few seconds ago"},
		// two inputs end date, start date
		{query: `SELECT time_diff(?,?)`, queryInput: []interface{}{time.Now().Add(-time.Second * 15).Format(time.RFC3339), time.Now().Format(time.RFC3339)}, expected: "a few seconds ago"},
		{query: `SELECT time_diff(?,?)`, queryInput: []interface{}{time.Now().Add(-time.Minute * 1).Format(time.RFC3339), time.Now().Format(time.RFC3339)}, expected: "a minute ago"},
		{query: `SELECT time_diff(?,?)`, queryInput: []interface{}{time.Now().Add(-time.Minute * 32).Format(time.RFC3339), time.Now().Format(time.RFC3339)}, expected: "32 minutes ago"},
		{query: `SELECT time_diff(?,?)`, queryInput: []interface{}{time.Now().Add(-time.Hour * 1).Format(time.RFC3339), time.Now().Format(time.RFC3339)}, expected: "an hour ago"},
	}

	var rows *sql.Rows
	var err error
	var rowNum int
	var contents [][]string
	for _, testCase := range tests {
		if testCase.queryInput != nil {
			rows, err = FixtureDatabase.Query(testCase.query, testCase.queryInput...)
			if err != nil {
				t.Fatal(err)
			}
		} else {
			rows, err = FixtureDatabase.Query(testCase.query)
			if err != nil {
				t.Fatal(err)
			}
		}

		rowNum, contents, err = tools.RowContent(rows)
		if err != nil {
			t.Fatalf("err %d at row Number %d", err, rowNum)
		}
		if contents[0][0] != testCase.expected {
			t.Fatalf("expected string: %s, got %s", testCase.expected, contents[0][0])
		}
	}

}
