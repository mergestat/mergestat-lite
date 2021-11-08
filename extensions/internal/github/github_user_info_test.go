package github_test

import (
	"strings"
	"testing"
)

func TestUserInfo(t *testing.T) {
	cleanup := newRecorder(t)
	defer cleanup()

	db := Connect(t, Memory)

	row := db.QueryRow("SELECT github_user('patrickdevivo')")
	if err := row.Err(); err != nil {
		t.Fatalf("failed to execute query: %v", err.Error())
	}

	var output string
	err := row.Scan(&output)
	if err != nil {
		t.Fatalf("could not scan row: %v", err.Error())
	}

	if !(strings.Contains(output, "2009-02-23T21:42:03Z")) {
		t.Fatalf("did not receive expected date in result: %s", output)
	}
}
