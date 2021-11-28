package enry

import (
	"testing"

	"github.com/mergestat/mergestat/extensions/internal/tools"
)

func TestEnryIsGenerated(t *testing.T) {
	path := ".xcworkspacedata"
	rows, err := FixtureDatabase.Query("SELECT enry_is_generated(?, ?)", path, "")
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
