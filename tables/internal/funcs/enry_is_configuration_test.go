package funcs

import (
	"testing"

	"github.com/askgitdev/askgit/tables/internal/tools"
)

func TestEneryIsConfiguration(t *testing.T) {
	path := "./testdata/configuration.json"
	//TODO: isGenerated from here -> https://github.com/go-enry/go-enry/tree/master/_testdata
	rows, err := FixtureDatabase.Query("SELECT enry_is_configuration(?)", path)
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
