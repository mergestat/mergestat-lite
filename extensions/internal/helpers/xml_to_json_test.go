package helpers

import (
	"testing"

	"github.com/mergestat/mergestat/extensions/internal/tools"
)

func TestXmlToJson(t *testing.T) {
	rows, err := FixtureDatabase.Query(`SELECT xml_to_json('
	<?xml version ="1.0" encoding="UTF-8"?>
	<employee>
		<fname>john</fname>
		<lname>doe</lname>
		<home>neverland</home>
	</employee>')`)
	if err != nil {
		t.Fatal(err)
	}
	rowNum, contents, err := tools.RowContent(rows)
	if err != nil {
		t.Fatalf("err %d at row Number %d", err, rowNum)
	}
	if contents[0][0] != `{"employee":{"fname":"john","home":"neverland","lname":"doe"}}` {
		t.Fatalf("expected string: %s, got %s", "", contents[0][0])
	}
}
