package helpers

import (
	"testing"

	"github.com/mergestat/mergestat-lite/extensions/internal/tools"
)

func TestTomlToJson(t *testing.T) {
	rows, err := FixtureDatabase.Query(`SELECT toml_to_json('[package] 
	name = "hog"')`)
	if err != nil {
		t.Fatal(err)
	}

	rowNum, contents, err := tools.RowContent(rows)
	if err != nil {
		t.Fatalf("err %d at row Number %d", err, rowNum)
	}
	if contents[0][0] != "{\"package\":{\"name\":\"hog\"}}" {
		t.Fatalf("expected string: %s, got %s", "", contents[0][0])
	}
}
