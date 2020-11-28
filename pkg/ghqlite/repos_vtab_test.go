package ghqlite

import (
	"testing"
)

func TestReposTable(t *testing.T) {
	_, err := DB.Query("SELECT * FROM repos LIMIT 5")
	if err != nil {
		t.Fatal(err)
	}

}
