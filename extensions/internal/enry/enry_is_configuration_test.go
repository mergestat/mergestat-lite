package enry

import (
	"testing"

	"github.com/mergestat/mergestat/extensions/internal/tools"
)

func TestEnryIsConfiguration(t *testing.T) {
	path := "./testdata/configuration.json" // from here -> https://github.com/go-enry/go-enry/tree/master/_testdata
	rows, err := FixtureDatabase.Query("SELECT enry_is_configuration(?)", path)
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
