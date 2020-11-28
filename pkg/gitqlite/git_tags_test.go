package gitqlite

import (
	"testing"
)

func TestTags(t *testing.T) {
	tags, err := fixtureRepo.Tags.List()
	if err != nil {
		t.Fatal(err)
	}

	tagRows, err := fixtureDB.Query("SELECT * FROM tags")
	if err != nil {
		t.Fatal(err)
	}

	rowNum, contents, err := GetRowContents(tagRows)
	if err != nil {
		t.Fatalf("err %d at row Number %d", err, rowNum)
	}

	if len(contents) != len(tags) {
		t.Fatalf("mismatched count of tags, expected %d got %d", len(tags), len(contents))
	}

	// for i, c := range contents {
	// 	tag, err := tagIterator.Next()
	// 	if err != nil {
	// 		if err == io.EOF {
	// 			break
	// 		} else {
	// 			t.Fatal(err)
	// 		}
	// 	}
	// 	if tag.Hash().String() != c[0] {
	// 		t.Fatalf("expected %s at row %d got %s", tag.Hash().String(), i, c[0])
	// 	}
	// 	if tag.Name().String() != c[1] {
	// 		t.Fatalf("expected %s at row %d got %s", tag.Name(), i, c[1])
	// 	}

	// }
}
func BenchmarkTagsCounts(b *testing.B) {
	for i := 0; i < b.N; i++ {
		rows, err := fixtureDB.Query("SELECT * FROM tags")
		if err != nil {
			b.Fatal(err)
		}
		rowNum, _, err := GetRowContents(rows)
		if err != nil {
			b.Fatalf("err %d at row Number %d", err, rowNum)
		}
	}
}
