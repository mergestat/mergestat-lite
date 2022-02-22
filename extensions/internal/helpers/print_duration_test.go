package helpers

import (
	"database/sql"
	"testing"

	"github.com/mergestat/mergestat/extensions/internal/tools"
)

func TestPrintDuration(t *testing.T) {
	type test struct {
		query    string
		expected string
	}
	tests := []test{
		// multiple years
		{query: `SELECT printduration('1000')`, expected: "3 years (33 months)"},
		// single year
		{query: `SELECT printduration('365')`, expected: "1 year (12 months)"},
		// multiple months
		{query: `SELECT printduration('185')`, expected: "7 months"},
		// single month
		{query: `SELECT printduration('31')`, expected: "1 month"},
		// multiple days
		{query: `SELECT printduration('5')`, expected: "5 days"},
		// single day
		{query: `SELECT printduration('1')`, expected: "1 day"},
		// multiple hours
		{query: `SELECT printduration('.5')`, expected: "12 hours"},
		// single hour
		{query: `SELECT printduration('.04')`, expected: "1 hour"},
		// multiple minutes
		{query: `SELECT printduration('.022')`, expected: "32 minutes"},
		// single minute
		{query: `SELECT printduration('.001')`, expected: "1 minute"},
		// seconds
		{query: `SELECT printduration('.0001')`, expected: "a few seconds"},
		// null
		{query: `SELECT printduration('0')`, expected: "<none>"},
	}
	var rows *sql.Rows
	var err error
	var rowNum int
	var contents [][]string
	for _, testCase := range tests {
		rows, err = FixtureDatabase.Query(testCase.query)
		if err != nil {
			t.Fatal(err)
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
