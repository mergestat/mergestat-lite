package enry

import (
	"io/ioutil"
	"testing"

	"github.com/mergestat/mergestat-lite/extensions/internal/tools"
)

func TestEnryDetectLanguage(t *testing.T) {
	path := "./testdata/main.go"
	fileContents, err := ioutil.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	rows, err := FixtureDatabase.Query("SELECT enry_detect_language(?,?)", path, fileContents)
	if err != nil {
		t.Fatal(err)
	}

	rowNum, contents, err := tools.RowContent(rows)
	if err != nil {
		t.Fatalf("err %d at row %d", err, rowNum)
	}

	if contents[0][0] != "Go" {
		t.Fatalf("expected string: %s, got %s", "Go", contents[0][0])
	}
}
