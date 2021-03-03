package mailmap

import "testing"

func TestMailmap(t *testing.T) {
	m := NewMailmap("")
	users := []string{"53903050+Vialeon@users.noreply.github.com", "vialeon", "Vialeon"}
	answers := []string{"dnewberry21@amherst.edu", "Derrick Newberry", "Derrick Newberry"}
	for i, user := range users {
		if answers[i] != m.UseMailmap(user) {
			t.Fatalf("expected %s got %s", answers[i], m.UseMailmap(user))
		}
	}
}
