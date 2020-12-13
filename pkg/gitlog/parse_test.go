package gitlog

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/user"
	"path"
	"testing"

	"github.com/gitsight/go-vcsurl"
	git "github.com/libgit2/git2go/v31"
)

var (
	fixtureRepoCloneURL = "https://github.com/augmentable-dev/gitqlite"
	fixtureRepo         *git.Repository
	fixtureRepoDir      string
)

func TestMain(m *testing.M) {
	close, err := initFixtureRepo()
	if err != nil {
		panic(err)
	}
	code := m.Run()
	close()
	os.Exit(code)
}

func initFixtureRepo() (func() error, error) {
	dir, err := ioutil.TempDir("", "repo")
	if err != nil {
		return nil, err
	}

	remote, err := vcsurl.Parse(fixtureRepoCloneURL)
	if err != nil {
		return nil, err
	}
	cloneOptions := CreateAuthenticationCallback(remote)
	fixtureRepo, err = git.Clone(fixtureRepoCloneURL, dir, cloneOptions)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	fixtureRepoDir = dir

	return func() error {
		err := os.RemoveAll(dir)
		if err != nil {
			return err
		}
		return nil
	}, nil
}

func TestParse(t *testing.T) {
	iter, err := Execute(fixtureRepoDir)
	if err != nil {
		t.Fatal(err)
	}

	count := 0
	for {
		commit, err := iter.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			t.Fatal(err)
		}
		if commit.SHA == "" {
			t.Fatal("expected SHA, got <empty string>")
		}
		count++
	}

	if err != nil {
		t.Fatal(err)
	}

	revWalk, err := fixtureRepo.Walk()
	if err != nil {
		t.Fatal(err)
	}
	defer revWalk.Free()

	err = revWalk.PushHead()
	if err != nil {
		t.Fatal(err)
	}

	shouldBeCount := 0
	err = revWalk.Iterate(func(*git.Commit) bool {
		shouldBeCount++
		return true
	})
	if err != nil {
		t.Fatal(err)
	}

	if count != shouldBeCount {
		t.Fatalf("incorrect number of commits, expected: %d got: %d", shouldBeCount, count)
	}
}

func CreateAuthenticationCallback(remote *vcsurl.VCS) *git.CloneOptions {
	cloneOptions := &git.CloneOptions{}

	if _, err := remote.Remote(vcsurl.SSH); err == nil { // if SSH, use "default" credentials
		// use FetchOptions instead of directly RemoteCallbacks
		// https://github.com/libgit2/git2go/commit/36e0a256fe79f87447bb730fda53e5cbc90eb47c
		cloneOptions.FetchOptions = &git.FetchOptions{
			RemoteCallbacks: git.RemoteCallbacks{
				CredentialsCallback: func(url string, username string, allowedTypes git.CredType) (*git.Cred, error) {
					usr, _ := user.Current()
					publicSSH := path.Join(usr.HomeDir, ".ssh/id_rsa.pub")
					privateSSH := path.Join(usr.HomeDir, ".ssh/id_rsa")

					cred, ret := git.NewCredSshKey("git", publicSSH, privateSSH, "")
					return cred, ret
				},
				CertificateCheckCallback: func(cert *git.Certificate, valid bool, hostname string) git.ErrorCode {
					return git.ErrOk
				},
			}}
	}
	return cloneOptions
}
