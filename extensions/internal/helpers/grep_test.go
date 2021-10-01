package helpers

import (
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
}
