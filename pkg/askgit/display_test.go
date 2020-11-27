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

	if len(records[0]) != 3 {
		t.Fatalf("expected 3 columns of output, got: %d", len(records[0]))
	}

	// TODO perhaps test the actual content of the lines?
}

func TestDisplayJSON(t *testing.T) {
	db, mock, _ := sqlmock.New()

	mockRows := sqlmock.NewRows([]string{"id", "name", "value"}).
		AddRow("1", "name 1", "value 1").
		AddRow("2", "name 2", "value 2").
		AddRow("3", "name 3", "value 3")

	mock.ExpectQuery("select").WillReturnRows(mockRows)

	rows, _ := db.Query("select")

	var b bytes.Buffer
	err := DisplayDB(rows, &b, "json")
	if err != nil {
		t.Fatal(err)
	}

	lineCount := strings.Count(b.String(), "\n")
	if lineCount != 3 {
		t.Fatalf("expected 3 lines of output, got: %d", lineCount)
	}
}

func TestDisplaySingle(t *testing.T) {
	db, mock, _ := sqlmock.New()

	mockRows := sqlmock.NewRows([]string{"id", "name", "value"}).
		AddRow("1", "name 1", "value 1").
		AddRow("2", "name 2", "value 2").
		AddRow("3", "name 3", "value 3")

	mock.ExpectQuery("select").WillReturnRows(mockRows)

	rows, _ := db.Query("select")

	var b bytes.Buffer
	err := DisplayDB(rows, &b, "single")
	if err != nil {
		t.Fatal(err)
	}

	if b.String() != "1" {
		t.Fatalf("expected output to be the only the first column of the first row: %s, got: %s", "1", b.String())
	}
}
