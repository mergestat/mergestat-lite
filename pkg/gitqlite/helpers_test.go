package gitqlite

import (
	"testing"
)

func TestStrSplit(t *testing.T) {
	instance, err := New(fixtureRepoDir, &Options{})
	if err != nil {
		t.Fatal(err)
	}

	rows, err := instance.DB.Query("SELECT str_split('hello world', ' ', 0)")
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


	rows, err = instance.DB.Query("SELECT str_split('hello world', ' ', 10)")
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
