package helpers

import (
	"testing"

	"github.com/mergestat/mergestat-lite/extensions/internal/tools"
)

func TestYmlToJson(t *testing.T) {
	rows, err := FixtureDatabase.Query(`SELECT yml_to_json('doe: "a deer, a female deer"')`)
	if err != nil {
		t.Fatal(err)
	}
	rowNum, contents, err := tools.RowContent(rows)
	if err != nil {
		t.Fatalf("err %d at row Number %d", err, rowNum)
	}
	if contents[0][0] != `{"doe":"a deer, a female deer"}` {
		t.Fatalf("expected string: %s, got %s", "", contents[0][0])
	}
}
