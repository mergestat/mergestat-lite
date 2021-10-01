package helpers

import (
	"strings"
	"testing"

	"github.com/askgitdev/askgit/extensions/internal/tools"
)

func TestGrepSimple(t *testing.T) {
	dummy := `
line1
line2
abc
line3
line4
	`

	// no preview
	rows, err := FixtureDatabase.Query("SELECT * FROM grep(?, 'abc')", dummy)
	if err != nil {
		t.Fatal(err)
	}

	rowNum, contents, err := tools.RowContent(rows)
	if err != nil {
		t.Fatalf("err %v at row: %d", err, rowNum)
	}

	if len(contents) != 1 {
		t.Fatalf("expected 1 row")
	}

	if contents[0][1] != "abc" {
		t.Fatalf("expected string: %s, got %s", "hello", contents[0][1])
	}

	// with before lines
	rows, err = FixtureDatabase.Query("SELECT * FROM grep(?, 'abc', 2)", dummy)
	if err != nil {
		t.Fatal(err)
	}

	rowNum, contents, err = tools.RowContent(rows)
	if err != nil {
		t.Fatalf("err %v at row: %d", err, rowNum)
	}

	if len(contents) != 1 {
		t.Fatalf("expected 1 row")
	}

	preview := strings.Split(contents[0][1], "\n")

	if len(preview) != 3 {
		t.Fatalf("expected 3 lines of output")
	}

	if preview[0] != "line1" || preview[1] != "line2" {
		t.Fatalf("unexpected lines in before")
	}

	// with after lines
	rows, err = FixtureDatabase.Query("SELECT * FROM grep(?, 'abc', 0, 2)", dummy)
	if err != nil {
		t.Fatal(err)
	}

	rowNum, contents, err = tools.RowContent(rows)
	if err != nil {
		t.Fatalf("err %v at row: %d", err, rowNum)
	}

	if len(contents) != 1 {
		t.Fatalf("expected 1 row")
	}

	preview = strings.Split(contents[0][1], "\n")

	if len(preview) != 3 {
		t.Fatalf("expected 3 lines of output")
	}

	if preview[1] != "line3" || preview[2] != "line4" {
		t.Fatalf("unexpected lines in before")
	}
}
