package gitqlite

import (
	"bytes"
	"encoding/csv"
	"strings"
	"testing"
)

func TestDisplayCSV(t *testing.T) {
	instance, err := New(fixtureRepoDir, &Options{})
	if err != nil {
		t.Fatal(err)
	}

	rows, err := instance.DB.Query("select * from commits limit 10")
	if err != nil {
		t.Fatal(err)
	}

	var b bytes.Buffer
	err = DisplayDB(rows, &b, "csv")
	if err != nil {
		t.Fatal(err)
	}

	r := csv.NewReader(strings.NewReader(b.String()))

	records, err := r.ReadAll()
	if err != nil {
		t.Fatal(err)
	}

	lineCount := len(records)

	if lineCount != 11 {
		t.Fatalf("expected 11 lines of output, got: %d", lineCount)
	}

	// TODO perhaps test the actual content of the lines?
}
