package mailmap_test

import (
	"testing"

	"github.com/mergestat/mergestat/pkg/mailmap"
)

func TestBasicOK(t *testing.T) {
	m := `
Joe Developer <joe@example.com>
Joe R. Developer <joe@example.com>
# Comment
Jane Doe <jane@example.com>
Jane Doe <jane@laptop.(none)>
Jane D. <jane@desktop.(none)>
# Another Comment
Patrick D <patrick@some-email.com> <some-other@email.com>
Patrick DeVivo <patrick@some-email.com> Pat Dev <some@email.com>
`

	mm, err := mailmap.Parse(m)
	if err != nil {
		t.Fatal(err)
	}

	if l := mm.Lookup(mailmap.NameAndEmail{Email: "jane@laptop.(none)"}); l.Name != "Jane Doe" && l.Email != "" {
		t.Fatalf("unexpected lookup result for %s", "jane@laptop.(none)")
	}

	if l := mm.Lookup(mailmap.NameAndEmail{Name: "Pat Dev", Email: "some@email.com"}); l.Name != "Patrick DeVivo" && l.Email != "patrick@some-email.com" {
		t.Fatalf("unexpected lookup result for %s and %s", "Pat Dev", "some@email.com")
	}

	t.Log(mm)
}
