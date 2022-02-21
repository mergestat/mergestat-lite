package helpers

import (
	"testing"

	"github.com/mergestat/mergestat/extensions/internal/tools"
)

func TestPrintDuration(t *testing.T) {
	// test multiple years
	expected := "3 years (33 months)"
	rows, err := FixtureDatabase.Query(`SELECT printduration('1000')`)
	if err != nil {
		t.Fatal(err)
	}

	rowNum, contents, err := tools.RowContent(rows)
	if err != nil {
		t.Fatalf("err %d at row Number %d", err, rowNum)
	}
	if contents[0][0] != expected {
		t.Fatalf("expected string: %s, got %s", expected, contents[0][0])
	}
	// test year
	expected = "1 year (12 months)"
	rows, err = FixtureDatabase.Query(`SELECT printduration('365')`)
	if err != nil {
		t.Fatal(err)
	}

	rowNum, contents, err = tools.RowContent(rows)
	if err != nil {
		t.Fatalf("err %d at row Number %d", err, rowNum)
	}
	if contents[0][0] != expected {
		t.Fatalf("expected string: %s, got %s", expected, contents[0][0])
	}
	// test months
	expected = "7 months"
	rows, err = FixtureDatabase.Query(`SELECT printduration('185')`)
	if err != nil {
		t.Fatal(err)
	}

	rowNum, contents, err = tools.RowContent(rows)
	if err != nil {
		t.Fatalf("err %d at row Number %d", err, rowNum)
	}
	if contents[0][0] != expected {
		t.Fatalf("expected string: %s, got %s", expected, contents[0][0])
	}

	// test month
	expected = "1 month"
	rows, err = FixtureDatabase.Query(`SELECT printduration('31')`)
	if err != nil {
		t.Fatal(err)
	}

	rowNum, contents, err = tools.RowContent(rows)
	if err != nil {
		t.Fatalf("err %d at row Number %d", err, rowNum)
	}
	if contents[0][0] != expected {
		t.Fatalf("expected string: %s, got %s", expected, contents[0][0])
	}
	// test days
	expected = "5 days"
	rows, err = FixtureDatabase.Query(`SELECT printduration('5')`)
	if err != nil {
		t.Fatal(err)
	}

	rowNum, contents, err = tools.RowContent(rows)
	if err != nil {
		t.Fatalf("err %d at row Number %d", err, rowNum)
	}
	if contents[0][0] != expected {
		t.Fatalf("expected string: %s, got %s", expected, contents[0][0])
	}
	// test day
	expected = "1 day"
	rows, err = FixtureDatabase.Query(`SELECT printduration('1')`)
	if err != nil {
		t.Fatal(err)
	}

	rowNum, contents, err = tools.RowContent(rows)
	if err != nil {
		t.Fatalf("err %d at row Number %d", err, rowNum)
	}
	if contents[0][0] != expected {
		t.Fatalf("expected string: %s, got %s", expected, contents[0][0])
	}
	// test hours
	expected = "12 hours"
	rows, err = FixtureDatabase.Query(`SELECT printduration('.5')`)
	if err != nil {
		t.Fatal(err)
	}

	rowNum, contents, err = tools.RowContent(rows)
	if err != nil {
		t.Fatalf("err %d at row Number %d", err, rowNum)
	}
	if contents[0][0] != expected {
		t.Fatalf("expected string: %s, got %s", expected, contents[0][0])
	}
	// test hour
	expected = "1 hour"
	rows, err = FixtureDatabase.Query(`SELECT printduration('.04')`)
	if err != nil {
		t.Fatal(err)
	}
	rowNum, contents, err = tools.RowContent(rows)
	if err != nil {
		t.Fatalf("err %d at row Number %d", err, rowNum)
	}
	if contents[0][0] != expected {
		t.Fatalf("expected string: %s, got %s", expected, contents[0][0])
	}
	// test minutes
	expected = "32 minutes"
	rows, err = FixtureDatabase.Query(`SELECT printduration('.022')`)
	if err != nil {
		t.Fatal(err)
	}
	rowNum, contents, err = tools.RowContent(rows)
	if err != nil {
		t.Fatalf("err %d at row Number %d", err, rowNum)
	}
	if contents[0][0] != expected {
		t.Fatalf("expected string: %s, got %s", expected, contents[0][0])
	}
	// test minute
	expected = "1 minute"
	rows, err = FixtureDatabase.Query(`SELECT printduration('.001')`)
	if err != nil {
		t.Fatal(err)
	}
	rowNum, contents, err = tools.RowContent(rows)
	if err != nil {
		t.Fatalf("err %d at row Number %d", err, rowNum)
	}
	if contents[0][0] != expected {
		t.Fatalf("expected string: %s, got %s", expected, contents[0][0])
	}
	// test seconds
	expected = "a few seconds"
	rows, err = FixtureDatabase.Query(`SELECT printduration('.0001')`)
	if err != nil {
		t.Fatal(err)
	}
	rowNum, contents, err = tools.RowContent(rows)
	if err != nil {
		t.Fatalf("err %d at row Number %d", err, rowNum)
	}
	if contents[0][0] != expected {
		t.Fatalf("expected string: %s, got %s", expected, contents[0][0])
	}
	// test none
	expected = "<none>"
	rows, err = FixtureDatabase.Query(`SELECT printduration('0')`)
	if err != nil {
		t.Fatal(err)
	}

	rowNum, contents, err = tools.RowContent(rows)
	if err != nil {
		t.Fatalf("err %d at row Number %d", err, rowNum)
	}
	if contents[0][0] != expected {
		t.Fatalf("expected string: %s, got %s", expected, contents[0][0])
	}

}
