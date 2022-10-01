package golang

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/mergestat/mergestat-lite/extensions/internal/tools"
)

func TestGoModToJSONOK(t *testing.T) {
	goMod, err := os.ReadFile("testdata/GoModOK")
	if err != nil {
		t.Fatal(err)
	}

	rows, err := FixtureDatabase.Query("SELECT go_mod_to_json(?)", goMod)
	if err != nil {
		t.Fatal(err)
	}

	rowNum, contents, err := tools.RowContent(rows)
	if err != nil {
		t.Fatalf("err %d at row Number %d", err, rowNum)
	}

	var parsed map[string]interface{}
	err = json.Unmarshal([]byte(contents[0][0]), &parsed)
	if err != nil {
		t.Fatal(err)
	}

	goVersion := parsed["go"].(string)
	if goVersion != "1.13" {
		t.Fatalf("expected go version 1.13, got %s", goVersion)
	}

	req := parsed["require"].([]interface{})
	if len(req) != 33 {
		t.Fatalf("expected 33 required modules, got %d", len(req))
	}
}

func TestGoModToJSONEmpty(t *testing.T) {
	rows, err := FixtureDatabase.Query("SELECT go_mod_to_json('')")
	if err != nil {
		t.Fatal(err)
	}

	rowNum, contents, err := tools.RowContent(rows)
	if err != nil {
		t.Fatalf("err %d at row Number %d", err, rowNum)
	}

	if contents[0][0] != "NULL" {
		t.Fatalf("expected NULL, got %s", contents[0][0])
	}
}

func TestGoModToJSONMissingVals(t *testing.T) {
	goMod, err := os.ReadFile("testdata/GoModMissingVals")
	if err != nil {
		t.Fatal(err)
	}

	rows, err := FixtureDatabase.Query("SELECT go_mod_to_json(?)", goMod)
	if err != nil {
		t.Fatal(err)
	}

	rowNum, contents, err := tools.RowContent(rows)
	if err != nil {
		t.Fatalf("err %d at row Number %d", err, rowNum)
	}

	var parsed map[string]interface{}
	err = json.Unmarshal([]byte(contents[0][0]), &parsed)
	if err != nil {
		t.Fatal(err)
	}

	goVersion := parsed["go"].(string)
	if goVersion != "" {
		t.Fatalf("expected no go version, got %s", goVersion)
	}
}
