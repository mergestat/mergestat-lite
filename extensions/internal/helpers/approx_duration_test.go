package helpers

import (
	"database/sql"
	"testing"

	"github.com/mergestat/mergestat/extensions/internal/tools"
)

func TestApproxDurationPrintDuration(t *testing.T) {
	type test struct {
		query    string
		expected string
	}
	tests := []test{
		// multiple years
		{query: `SELECT approx_dur(1000)`, expected: "3 years (33 months)"},
		// single year
		{query: `SELECT approx_dur(365)`, expected: "1 year (12 months)"},
		// multiple months
		{query: `SELECT approx_dur(185)`, expected: "7 months"},
		// single month
		{query: `SELECT approx_dur(31)`, expected: "1 month"},
		// multiple days
		{query: `SELECT approx_dur(5)`, expected: "5 days"},
		// single day
		{query: `SELECT approx_dur(1)`, expected: "1 day"},
		// multiple hours
		{query: `SELECT approx_dur(.5)`, expected: "12 hours"},
		// single hour
		{query: `SELECT approx_dur(.04)`, expected: "1 hour"},
		// multiple minutes
		{query: `SELECT approx_dur(.022)`, expected: "32 minutes"},
		// single minute
		{query: `SELECT approx_dur(.001)`, expected: "1 minute"},
		// seconds
		{query: `SELECT approx_dur(.0001)`, expected: "a few seconds"},
		// null
		{query: `SELECT approx_dur(0)`, expected: "<none>"},
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
