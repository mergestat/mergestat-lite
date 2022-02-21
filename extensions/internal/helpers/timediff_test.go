package helpers

import (
	"testing"
	"time"

	"github.com/mergestat/mergestat/extensions/internal/tools"
)

func TestTimediff(t *testing.T) {
	// test multiple years
	expected := "5 years ago"
	rows, err := FixtureDatabase.Query(`SELECT timediff(DATE('now','-4 year','-6 month'),DATE('now'),'2006-01-02')`)
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
	expected = "a year ago"
	rows, err = FixtureDatabase.Query(`SELECT timediff(DATE('now','-1 year'),DATE('now'),'2006-01-02')`)
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
	expected = "7 months ago"
	rows, err = FixtureDatabase.Query(`SELECT timediff(DATE('now','-6 month'),DATE('now'),'2006-01-02')`)
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
	expected = "a month ago"
	rows, err = FixtureDatabase.Query(`SELECT timediff(DATE('now','-1 month'),DATE('now'),'2006-01-02')`)
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
	expected = "5 days ago"
	rows, err = FixtureDatabase.Query(`SELECT timediff(DATE('now','-5 day'),DATE('now'),'2006-01-02')`)
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
	expected = "a day ago"
	rows, err = FixtureDatabase.Query(`SELECT timediff(DATE('now','-1 day'),DATE('now'),'2006-01-02')`)
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
	testTime := time.Now().Add(-time.Hour * 7).Add(time.Minute * 1)

	expected = "7 hours ago"
	rows, err = FixtureDatabase.Query(`SELECT timediff(?)`, testTime.Format(time.RFC3339))
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
	testTime = time.Now().Add(-time.Hour * 1)

	expected = "an hour ago"
	rows, err = FixtureDatabase.Query(`SELECT timediff(?)`, testTime.Format(time.RFC3339))
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
	testTime = time.Now().Add(-time.Minute * 32).Add(time.Second * 2)
	expected = "32 minutes ago"
	rows, err = FixtureDatabase.Query(`SELECT timediff(?)`, testTime.Format(time.RFC3339))
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
	testTime = time.Now().Add(-time.Minute * 1)

	expected = "a minute ago"
	rows, err = FixtureDatabase.Query(`SELECT timediff(?)`, testTime.Format(time.RFC3339))
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
	testTime = time.Now().Add(-time.Second * 15)

	expected = "a few seconds ago"
	rows, err = FixtureDatabase.Query(`SELECT timediff(?)`, testTime.Format(time.RFC3339))
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
	// testing for 2 inputs as well
	// test hours
	testTime = time.Now().Add(-time.Hour * 7)
	currentTime := time.Now()
	expected = "7 hours ago"
	rows, err = FixtureDatabase.Query(`SELECT timediff(?,?)`, testTime.Format(time.RFC3339), currentTime.Format(time.RFC3339))
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
	testTime = time.Now().Add(-time.Hour * 1)
	currentTime = time.Now()
	expected = "an hour ago"
	rows, err = FixtureDatabase.Query(`SELECT timediff(?,?)`, testTime.Format(time.RFC3339), currentTime.Format(time.RFC3339))
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
	testTime = time.Now().Add(-time.Minute * 32)
	currentTime = time.Now()
	expected = "32 minutes ago"
	rows, err = FixtureDatabase.Query(`SELECT timediff(?,?)`, testTime.Format(time.RFC3339), currentTime.Format(time.RFC3339))
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
	testTime = time.Now().Add(-time.Minute * 1)
	currentTime = time.Now()
	expected = "a minute ago"
	rows, err = FixtureDatabase.Query(`SELECT timediff(?,?)`, testTime.Format(time.RFC3339), currentTime.Format(time.RFC3339))
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
	testTime = time.Now().Add(-time.Second * 15)
	currentTime = time.Now()

	expected = "a few seconds ago"
	rows, err = FixtureDatabase.Query(`SELECT timediff(?,?)`, testTime.Format(time.RFC3339), currentTime.Format(time.RFC3339))
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
