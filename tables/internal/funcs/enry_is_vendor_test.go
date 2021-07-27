package funcs

import (
	"testing"

	"github.com/askgitdev/askgit/tables/internal/tools"
)

func TestEneryIsVendor(t *testing.T) {
	path := "./testdata/node_modules/data"
	rows, err := FixtureDatabase.Query("SELECT enry_is_vendor(?)", path)
	if err != nil {
		t.Fatal(err)
	}

	rowNum, contents, err := tools.RowContent(rows)
	if err != nil {
		t.Fatalf("err %d at row Number %d", err, rowNum)
	}

	if contents[0][0] != "1" {
		t.Fatalf("expected string: %s, got %s", "1", contents[0][0])
	}
}
