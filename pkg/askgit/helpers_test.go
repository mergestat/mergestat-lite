package askgit

import (
	"testing"
)

func TestStrSplit(t *testing.T) {
	ag, err := New(&Options{RepoPath: fixtureRepoDir})
	if err != nil {
		t.Fatal(err)
	}

	rows, err := ag.DB().Query("SELECT str_split('hello world', ' ', 0)")
	if err != nil {
		t.Fatal(err)
	}
	rowNum, contents, err := GetContents(rows)
	if err != nil {
		t.Fatalf("err %d at row Number %d", err, rowNum)
	}

	if contents[0][0] != "hello" {
		t.Fatalf("expected string: %s, got %s", "hello", contents[0][0])
	}

	rows, err = ag.DB().Query("SELECT str_split('hello world', ' ', 10)")
	if err != nil {
		t.Fatal(err)
	}
	rowNum, contents, err = GetContents(rows)
	if err != nil {
		t.Fatalf("err %d at row Number %d", err, rowNum)
	}

	if contents[0][0] != "" {
		t.Fatalf("expected string: %s, got %s", "", contents[0][0])
	}
}
func TestTomlToJson(t *testing.T) {
	ag, err := New(&Options{RepoPath: fixtureRepoDir})
	if err != nil {
		t.Fatal(err)
	}
	rows, err := ag.DB().Query(`SELECT toml_to_json('[package] 
	name = "hog"')`)
	if err != nil {
		t.Fatal(err)
	}
	rowNum, contents, err := GetContents(rows)
	if err != nil {
		t.Fatalf("err %d at row Number %d", err, rowNum)
	}
	if contents[0][0] != "{\"package\":{\"name\":\"hog\"}}" {
		t.Fatalf("expected string: %s, got %s", "", contents[0][0])
	}
}
func TestYmlToJson(t *testing.T) {
	ag, err := New(&Options{RepoPath: fixtureRepoDir})
	if err != nil {
		t.Fatal(err)
	}
	rows, err := ag.DB().Query(`SELECT yml_to_json('doe: "a deer, a female deer"')`)
	if err != nil {
		t.Fatal(err)
	}
	rowNum, contents, err := GetContents(rows)
	if err != nil {
		t.Fatalf("err %d at row Number %d", err, rowNum)
	}
	if contents[0][0] != `{"doe":"a deer, a female deer"}` {
		t.Fatalf("expected string: %s, got %s", "", contents[0][0])
	}

}
func TestXmlToJson(t *testing.T) {
	ag, err := New(&Options{RepoPath: fixtureRepoDir})
	if err != nil {
		t.Fatal(err)
	}
	rows, err := ag.DB().Query(`SELECT xml_to_json('
	<?xml version ="1.0" encoding="UTF-8"?>
	<employee>
		<fname>john</fname>
		<lname>doe</lname>
		<home>neverland</home>
	</employee>')`)
	if err != nil {
		t.Fatal(err)
	}
	rowNum, contents, err := GetContents(rows)
	if err != nil {
		t.Fatalf("err %d at row Number %d", err, rowNum)
	}
	if contents[0][0] != `{"employee":{"fname":"john","home":"neverland","lname":"doe"}}` {
		t.Fatalf("expected string: %s, got %s", "", contents[0][0])
	}
}
