package mailmap_test

import (
	"testing"

	"github.com/mergestat/mergestat-lite/pkg/mailmap"
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

	t.Log(mm)

	if l := mm.Lookup(mailmap.NameAndEmail{Email: "jane@laptop.(none)"}); l.Name != "Jane Doe" && l.Email != "" {
		t.Fatalf("unexpected lookup result %s", l)
	}

	if l := mm.Lookup(mailmap.NameAndEmail{Name: "Pat Dev", Email: "some@email.com"}); l.Name != "Patrick DeVivo" && l.Email != "patrick@some-email.com" {
		t.Fatalf("unexpected lookup result %s", l)
	}
}

func TestLibgit2Mailmap(t *testing.T) {
	// https://github.com/libgit2/libgit2/blob/main/.mailmap
	m := `
Vicent Martí <vicent@github.com> Vicent Marti <tanoku@gmail.com>
Vicent Martí <vicent@github.com> Vicent Martí <tanoku@gmail.com>
Michael Schubert <schu@schu.io> schu <schu-github@schulog.org>
Ben Straub <bs@github.com> Ben Straub <ben@straubnet.net>
Ben Straub <bs@github.com> Ben Straub <bstraub@github.com>
Carlos Martín Nieto <cmn@dwim.me> <carlos@cmartin.tk>
Carlos Martín Nieto <cmn@dwim.me> <cmn@elego.de>
nulltoken <emeric.fermas@gmail.com> <emeric.fermas@gmail.com>
Scott J. Goldman <scottjg@github.com> <scottjgo@gmail.com>
Martin Woodward <martin.woodward@microsoft.com> <martinwo@microsoft.com>
Peter Drahoš <drahosp@gmail.com> <drahosp@gmail.com>
Adam Roben <adam@github.com> <adam@roben.org>
Adam Roben <adam@github.com> <adam@github.com>
Xavier L. <xavier.l@afrosoft.tk> <xavier.l@afrosoft.ca>
Xavier L. <xavier.l@afrosoft.tk> <xavier.l@afrosoft.tk>
Sascha Cunz <sascha@babbelbox.org> <Sascha@BabbelBox.org>
Authmillenon <authmillenon@googlemail.com> <martin@ucsmail.de>
Authmillenon <authmillenon@googlemail.com> <authmillenon@googlemail.com>
Edward Thomson <ethomson@edwardthomson.com> <ethomson@microsoft.com>
Edward Thomson <ethomson@edwardthomson.com> <ethomson@github.com>
J. David Ibáñez <jdavid.ibp@gmail.com> <jdavid@itaapy.com>
Russell Belfer <rb@github.com> <arrbee@arrbee.com>
`

	mm, err := mailmap.Parse(m)
	if err != nil {
		t.Fatal(err)
	}

	t.Log(mm)

	if l := mm.Lookup(mailmap.NameAndEmail{Name: "Some Name", Email: "ethomson@github.com"}); l.Name != "Edward Thomson" && l.Email != "ethomson@edwardthomson.com" {
		t.Fatalf("unexpected lookup result %s", l)
	}
}

func TestMailmapSyntaxForms(t *testing.T) {
	m := `
# Form 1: Proper Name <commit@email>
Canonical Name <old-email@example.com>

# Form 2: <proper@email> <commit@email>
<canonical-email@example.com> <alias-email@example.com>

# Form 3: Proper Name <proper@email> <commit@email>
Both Mapped <both-new@example.com> <both-old@example.com>

# Form 4: Proper Name <proper@email> Commit Name <commit@email>
Specific Person <spec-new@example.com> Old Alias <spec-old@example.com>

# Form 5: <proper@email> Commit Name <commit@email>
<email-only-spec@example.com> Specific Alias <email-only-old@example.com>
`

	mm, err := mailmap.Parse(m)
	if err != nil {
		t.Fatalf("unexpected error parsing mailmap: %v", err)
	}

	t.Run("Form 1 preserves commit email and updates name", func(t *testing.T) {
		res := mm.Lookup(mailmap.NameAndEmail{Name: "Original Name", Email: "old-email@example.com"})
		if res.Name != "Canonical Name" {
			t.Errorf("expected Canonical Name, got %s", res.Name)
		}
		if res.Email != "old-email@example.com" {
			t.Errorf("expected original email old-email@example.com preserved, got %s", res.Email)
		}
	})

	t.Run("Form 2 preserves commit name and updates email", func(t *testing.T) {
		res := mm.Lookup(mailmap.NameAndEmail{Name: "Author One", Email: "alias-email@example.com"})
		if res.Name != "Author One" {
			t.Errorf("expected Author One preserved, got %s", res.Name)
		}
		if res.Email != "canonical-email@example.com" {
			t.Errorf("expected canonical-email@example.com, got %s", res.Email)
		}
	})

	t.Run("Form 3 updates both name and email", func(t *testing.T) {
		res := mm.Lookup(mailmap.NameAndEmail{Name: "Any Name", Email: "both-old@example.com"})
		if res.Name != "Both Mapped" {
			t.Errorf("expected Both Mapped, got %s", res.Name)
		}
		if res.Email != "both-new@example.com" {
			t.Errorf("expected both-new@example.com, got %s", res.Email)
		}
	})

	t.Run("Form 4 updates both when name and email match", func(t *testing.T) {
		res := mm.Lookup(mailmap.NameAndEmail{Name: "Old Alias", Email: "spec-old@example.com"})
		if res.Name != "Specific Person" {
			t.Errorf("expected Specific Person, got %s", res.Name)
		}
		if res.Email != "spec-new@example.com" {
			t.Errorf("expected spec-new@example.com, got %s", res.Email)
		}

		// When name does not match, does not map
		unmatched := mm.Lookup(mailmap.NameAndEmail{Name: "Other Alias", Email: "spec-old@example.com"})
		if unmatched.Name != "Other Alias" || unmatched.Email != "spec-old@example.com" {
			t.Errorf("expected unchanged author, got %v", unmatched)
		}
	})

	t.Run("Form 5 updates email when name and email match and preserves name", func(t *testing.T) {
		res := mm.Lookup(mailmap.NameAndEmail{Name: "Specific Alias", Email: "email-only-old@example.com"})
		if res.Name != "Specific Alias" {
			t.Errorf("expected Specific Alias, got %s", res.Name)
		}
		if res.Email != "email-only-spec@example.com" {
			t.Errorf("expected email-only-spec@example.com, got %s", res.Email)
		}
	})
}

func TestPrecedenceAndOverrides(t *testing.T) {
	m := `
# Generic rule
Default Author <default@example.com> <shared@example.com>

# Specific rule for same email
Special Author <special@example.com> Custom Alias <shared@example.com>

# Rule to be overridden
Override Target <wrong@example.com> <override@example.com>

# Overriding rule
Override Target <correct@example.com> <override@example.com>
`

	mm, err := mailmap.Parse(m)
	if err != nil {
		t.Fatalf("unexpected error parsing mailmap: %v", err)
	}

	t.Run("specific commit name takes precedence over generic match", func(t *testing.T) {
		res := mm.Lookup(mailmap.NameAndEmail{Name: "Custom Alias", Email: "shared@example.com"})
		if res.Name != "Special Author" || res.Email != "special@example.com" {
			t.Errorf("expected Special Author <special@example.com>, got %v", res)
		}

		fallback := mm.Lookup(mailmap.NameAndEmail{Name: "Unrelated Name", Email: "shared@example.com"})
		if fallback.Name != "Default Author" || fallback.Email != "default@example.com" {
			t.Errorf("expected Default Author <default@example.com>, got %v", fallback)
		}
	})

	t.Run("later lines override earlier lines", func(t *testing.T) {
		res := mm.Lookup(mailmap.NameAndEmail{Name: "Any", Email: "override@example.com"})
		if res.Email != "correct@example.com" {
			t.Errorf("expected correct@example.com, got %s", res.Email)
		}
	})

	t.Run("case insensitivity in email and name lookup", func(t *testing.T) {
		res := mm.Lookup(mailmap.NameAndEmail{Name: "custom alias", Email: "SHARED@example.com"})
		if res.Name != "Special Author" || res.Email != "special@example.com" {
			t.Errorf("expected case-insensitive match Special Author, got %v", res)
		}
	})

	t.Run("unmatched authors remain completely unchanged", func(t *testing.T) {
		orig := mailmap.NameAndEmail{Name: "Unknown Dev", Email: "unknown@example.com"}
		res := mm.Lookup(orig)
		if res.Name != orig.Name || res.Email != orig.Email {
			t.Errorf("expected %v, got %v", orig, res)
		}
	})
}

func TestCommentsAndWhitespace(t *testing.T) {
	m := `
   # Leading spaces before comment
# Another comment line

   Whitespace Name   <ws@example.com>   # Trailing comment
<email-only@example.com>    <ws-old@example.com>  # Trailing comment with <tags>
`

	mm, err := mailmap.Parse(m)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	res1 := mm.Lookup(mailmap.NameAndEmail{Name: "Some Name", Email: "ws@example.com"})
	if res1.Name != "Whitespace Name" || res1.Email != "ws@example.com" {
		t.Errorf("unexpected lookup %v", res1)
	}

	res2 := mm.Lookup(mailmap.NameAndEmail{Name: "Existing Name", Email: "ws-old@example.com"})
	if res2.Name != "Existing Name" || res2.Email != "email-only@example.com" {
		t.Errorf("unexpected lookup %v", res2)
	}
}

