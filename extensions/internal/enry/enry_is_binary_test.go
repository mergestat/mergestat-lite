package enry

import (
	"os"
	"testing"

	"github.com/mergestat/mergestat-lite/extensions/internal/tools"
)

func TestEnryIsBinary(t *testing.T) {
	path := "./testdata/binary"
	fileContents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	rows, err := FixtureDatabase.Query("SELECT enry_is_binary(?)", fileContents)
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
