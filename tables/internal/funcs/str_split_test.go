package funcs

import (
	"github.com/augmentable-dev/askgit/tables/internal/tools"
	"testing"
)

func TestStrSplit(t *testing.T) {
	rows, err := FixtureDatabase.Query("SELECT str_split('hello world', ' ', 0)")
	if err != nil {
		t.Fatal(err)
	}

	rowNum, contents, err := tools.RowContent(rows)
	if err != nil {
		t.Fatalf("err %d at row Number %d", err, rowNum)
	}

	if contents[0][0] != "hello" {
		t.Fatalf("expected string: %s, got %s", "hello", contents[0][0])
	}

	rows, err = FixtureDatabase.Query("SELECT str_split('hello world', ' ', 10)")
	if err != nil {
		t.Fatal(err)
	}

	rowNum, contents, err = tools.RowContent(rows)
	if err != nil {
		t.Fatalf("err %d at row Number %d", err, rowNum)
	}

	if contents[0][0] != "NULL" {
		t.Fatalf("expected: %s, got %s", "NULL", contents[0][0])
	}
}