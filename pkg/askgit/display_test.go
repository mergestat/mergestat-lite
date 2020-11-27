package askgit

import (
	"bytes"
	"encoding/csv"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestDisplayCSV(t *testing.T) {
	db, mock, _ := sqlmock.New()

	mockRows := sqlmock.NewRows([]string{"id", "name", "value"}).
		AddRow("1", "name 1", "value 1").
		AddRow("2", "name 2", "value 2").
		AddRow("3", "name 3", "value 3")

	mock.ExpectQuery("select").WillReturnRows(mockRows)

	rows, _ := db.Query("select")

	var b bytes.Buffer
	err := DisplayDB(rows, &b, "csv")
	if err != nil {
		t.Fatal(err)
	}

	r := csv.NewReader(strings.NewReader(b.String()))

	records, err := r.ReadAll()
	if err != nil {
		t.Fatal(err)
	}

	lineCount := len(records)

	if lineCount != 4 {
		t.Fatalf("expected 4 lines of output, got: %d", lineCount)
	}

	// TODO perhaps test the actual content of the lines?
}
