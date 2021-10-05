package helpers

import (
	"testing"

	"github.com/askgitdev/askgit/extensions/internal/tools"
)

func TestStrSplitTbl(t *testing.T) {
	rows, err := FixtureDatabase.Query("SELECT * from str_split('hello,my,name,is,what,my,name,is,who', ',')")
	if err != nil {
		t.Fatal(err)
	}

	rowNum, contents, err := tools.RowContent(rows)
	if err != nil {
		t.Fatalf("err %d at row Number %d", err, rowNum)
	}

	if contents[0][1] != "hello" {
		t.Fatalf("expected string: %s, got %s", "hello", contents[0][0])
	}

	if len(contents) != 9 {
		t.Fatalf("expected 9 rows instead got %d", len(contents[0]))
	}
}
