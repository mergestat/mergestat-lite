package funcs

import (
	"io/ioutil"
	"testing"

	"github.com/askgitdev/askgit/tables/internal/tools"
)

func TestEnryDetectLanguage(t *testing.T) {
	path := "./testdata/configuration.json"
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
		t.Fatalf("err %d at row Number %d", err, rowNum)
	}

	if contents[0][0] != "JSON" {
		t.Fatalf("expected string: %s, got %s", "JSON", contents[0][0])
	}
}
