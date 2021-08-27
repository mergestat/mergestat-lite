package npm_test

import (
	"testing"

	"github.com/askgitdev/askgit/extensions/internal/tools"
)

func TestGetPackage(t *testing.T) {
	cleanup := newRecorder(t)
	defer cleanup()

	rows, err := FixtureDatabase.Query("SELECT npm_get_package(?)", "jquery")
	if err != nil {
		t.Fatal(err)
	}

	rowNum, contents, err := tools.RowContent(rows)
	if err != nil {
		t.Fatalf("err %d at row %d", err, rowNum)
	}

	if len(contents[0][0]) < 10 {
		t.Fatalf("expected string with length greater than 10")
	}
}

func TestGetPackageVersion(t *testing.T) {
	cleanup := newRecorder(t)
	defer cleanup()

	rows, err := FixtureDatabase.Query("SELECT npm_get_package(?, ?)", "jquery", "latest")
	if err != nil {
		t.Fatal(err)
	}

	rowNum, contents, err := tools.RowContent(rows)
	if err != nil {
		t.Fatalf("err %d at row %d", err, rowNum)
	}

	if len(contents[0][0]) < 10 {
		t.Fatalf("expected string with length greater than 10")
	}
}
