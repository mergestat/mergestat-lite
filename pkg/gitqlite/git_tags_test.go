package gitqlite

import (
	"fmt"
	"testing"
)

func TestTags(t *testing.T) {
	testCases := []test{
		{"checkTags", "SELECT COUNT(*) FROM tags", numTags(t)},
	}
	for _, tc := range testCases {
		expected := tc.want
		results := runQuery(t, tc.query)
		if len(expected) != len(results) {
			t.Fatalf("expected %d entries got %d, test: %s, %s, %s", len(expected), len(results), tc.name, expected, results)
		}
		for x := 0; x < len(expected); x++ {
			if results[x] != expected[x] {
				t.Fatalf("expected %s, got %s, test %s", expected[x], results[x], tc.name)
			}
		}
	}
}

func numTags(t *testing.T) []string {
	tags, err := fixtureRepo.Tags.List()
	if err != nil {
		t.Fatal(err)
	}
	return []string{fmt.Sprintf("%d", len(tags))}

}
func BenchmarkTagsCounts(b *testing.B) {
	for i := 0; i < b.N; i++ {
		rows, err := fixtureDB.Query("SELECT * FROM tags")
		if err != nil {
			b.Fatal(err)
		}
		rowNum, _, err := GetContents(rows)
		if err != nil {
			b.Fatalf("err %d at row Number %d", err, rowNum)
		}
	}
}
