package display

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"encoding/json"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestDisplayTable(t *testing.T) {
	db, mock, _ := sqlmock.New()

	mockRows := sqlmock.NewRows([]string{"id", "name", "value"}).
		AddRow("1", "name 1", "value 1").
		AddRow("2", "name 2", "value 2").
		AddRow("3", "name 3", "value 3")

	mock.ExpectQuery("select").WillReturnRows(mockRows)

	rows, _ := db.Query("select")

	var b bytes.Buffer
	err := WriteTo(rows, &b, "table", false)
	if err != nil {
		t.Fatal(err)
	}

	scanner := bufio.NewScanner(&b)
	lineCount := 0
	for scanner.Scan() {
		lineCount++
	}

	if lineCount < 3 {
		t.Fatalf("expected at least %d lines of output", lineCount)
	}

}

func TestDisplayCSV(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}

	mockRows := sqlmock.NewRows([]string{"id", "name", "value"}).
		AddRow("1", "name 1", "value 1").
		AddRow("2", "name 2", "value 2").
		AddRow("3", "name 3", "value 3")

	mock.ExpectQuery("select").WillReturnRows(mockRows)
	rows, err := db.Query("select")
	if err != nil {
		t.Fatal(err)
	}

	var b bytes.Buffer
	err = WriteTo(rows, &b, "csv", false)
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

func TestDisplayCSVNoHeader(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}

	mockRows := sqlmock.NewRows([]string{"id", "name", "value"}).
		AddRow("1", "name 1", "value 1").
		AddRow("2", "name 2", "value 2").
		AddRow("3", "name 3", "value 3")

	mock.ExpectQuery("select").WillReturnRows(mockRows)
	rows, err := db.Query("select")
	if err != nil {
		t.Fatal(err)
	}

	var b bytes.Buffer
	err = WriteTo(rows, &b, "csv-noheader", false)
	if err != nil {
		t.Fatal(err)
	}

	r := csv.NewReader(strings.NewReader(b.String()))

	records, err := r.ReadAll()
	if err != nil {
		t.Fatal(err)
	}

	lineCount := len(records)

	if lineCount != 3 {
		t.Fatalf("expected 3 lines of output, got: %d", lineCount)
	}

	if len(records[0]) != 3 {
		t.Fatalf("expected 3 columns of output, got: %d", len(records[0]))
	}
}

func TestDisplayTSV(t *testing.T) {
	db, mock, _ := sqlmock.New()

	mockRows := sqlmock.NewRows([]string{"id", "name", "value"}).
		AddRow("1", "name 1", "value 1").
		AddRow("2", "name 2", "value 2").
		AddRow("3", "name 3", "value 3")

	mock.ExpectQuery("select").WillReturnRows(mockRows)

	rows, _ := db.Query("select")

	var b bytes.Buffer
	err := WriteTo(rows, &b, "tsv", false)
	if err != nil {
		t.Fatal(err)
	}

	r := csv.NewReader(strings.NewReader(b.String()))
	r.Comma = '\t'

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

func TestDisplayTSVNoHeader(t *testing.T) {
	db, mock, _ := sqlmock.New()

	mockRows := sqlmock.NewRows([]string{"id", "name", "value"}).
		AddRow("1", "name 1", "value 1").
		AddRow("2", "name 2", "value 2").
		AddRow("3", "name 3", "value 3")

	mock.ExpectQuery("select").WillReturnRows(mockRows)

	rows, _ := db.Query("select")

	var b bytes.Buffer
	err := WriteTo(rows, &b, "tsv-noheader", false)
	if err != nil {
		t.Fatal(err)
	}

	r := csv.NewReader(strings.NewReader(b.String()))
	r.Comma = '\t'

	records, err := r.ReadAll()
	if err != nil {
		t.Fatal(err)
	}

	lineCount := len(records)

	if lineCount != 3 {
		t.Fatalf("expected 3 lines of output, got: %d", lineCount)
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
	err := WriteTo(rows, &b, "json", false)
	if err != nil {
		t.Fatal(err)
	}

	o := make([]map[string]interface{}, 0)
	err = json.Unmarshal(b.Bytes(), &o)
	if err != nil {
		t.Fatal(err)
	}

	if len(o) != 3 {
		t.Fatalf("expected 3 rows of output, got: %d", len(o))
	}

	if o[0]["name"] != "name 1" {
		t.Fatalf(`expected "name" in first row to be "name 1", got: %s`, o[0]["name"])
	}
}

func TestDisplayNDJSON(t *testing.T) {
	db, mock, _ := sqlmock.New()

	mockRows := sqlmock.NewRows([]string{"id", "name", "value"}).
		AddRow("1", "name 1", "value 1").
		AddRow("2", "name 2", "value 2").
		AddRow("3", "name 3", "value 3")

	mock.ExpectQuery("select").WillReturnRows(mockRows)

	rows, _ := db.Query("select")

	var b bytes.Buffer
	err := WriteTo(rows, &b, "ndjson", false)
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
	err := WriteTo(rows, &b, "single", false)
	if err != nil {
		t.Fatal(err)
	}

	if b.String() != "1" {
		t.Fatalf("expected output to be the only the first column of the first row: %s, got: %s", "1", b.String())
	}
}
