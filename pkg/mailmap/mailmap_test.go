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
