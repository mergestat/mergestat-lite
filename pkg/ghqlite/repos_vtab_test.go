package ghqlite

import (
	"testing"
)

func TestReposTable(t *testing.T) {
	_, err := DB.Query("SELECT count(*) FROM repos")
	if err != nil {
		t.Fatal(err)
	}

}
