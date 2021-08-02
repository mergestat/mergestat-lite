package funcs

import (
	"testing"

	"github.com/askgitdev/askgit/tables/internal/tools"
)

func TestEnryIsTest(t *testing.T) {
	path := "./scripts_test.go"
	rows, err := FixtureDatabase.Query("SELECT enry_is_test(?)", path)
	if err != nil {
		t.Fatal(err)
	}

	rowNum, contents, err := tools.RowContent(rows)
	if err != nil {
		t.Fatalf("err %d at row %d", err, rowNum)
	}

	if contents[0][0] != "1" {
		t.Fatalf("expected string: %s, got %s", "1", contents[0][0])
	}
}
