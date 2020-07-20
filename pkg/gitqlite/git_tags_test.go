package gitqlite

import (
	"io"
	"testing"
)

func TestTags(t *testing.T) {
	instance, err := New(fixtureRepoDir, &Options{})
	if err != nil {
		t.Fatal(err)
	}

	tagIterator, err := fixtureRepo.Tags()
	if err != nil {
		t.Fatal(err)
	}
	tagRows, err := instance.DB.Query("SELECT * FROM tags")
	if err != nil {
		t.Fatal(err)
	}
	rowNum, contents, err := GetContents(tagRows)
	if err != nil {
		t.Fatalf("err %d at row Number %d", err, rowNum)
	}
	for i, c := range contents {
		tag, err := tagIterator.Next()
		if err != nil {
			if err == io.EOF {
				break
			} else {
				t.Fatal(err)
			}
		}
		if tag.Hash().String() != c[0] {
			t.Fatalf("expected %s at row %d got %s", tag.Hash().String(), i, c[0])
		}
		if tag.Name().String() != c[1] {
			t.Fatalf("expected %s at row %d got %s", tag.Name(), i, c[1])
		}

	}
}
