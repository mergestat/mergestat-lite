package mailmap

import (
	"io/ioutil"
	"os/user"
	"path"
	"testing"

	"github.com/gitsight/go-vcsurl"
	git "github.com/libgit2/git2go/v31"
)

var (
	fixtureRepoCloneURL = "https://github.com/sympy/sympy"
	fixtureRepo         *git.Repository
	// fixtureRepoDir      string
	// fixtureDB           *sql.DB
)

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

// use the mailmap thing with git and compare it to what I get by running the SQL. If that works
func TestMailmap(t *testing.T) {
	dir, err := ioutil.TempDir("", "repo")
	if err != nil {
		t.Fatal(err)
	}
	vcurl, err := vcsurl.Parse(fixtureRepoCloneURL)
	if err != nil {
		t.Fatal(err)
	}
	cloneOpts := CreateAuthenticationCallback(vcurl)
	fixtureRepo, err = git.Clone(fixtureRepoCloneURL, dir, cloneOpts)
	if err != nil {
		t.Fatal(err)
	}
	m, err := NewMailmap(dir)
	if err != nil {
		t.Fatal(err) 
	}
	// for i, v := range m.userMap {
	// 	fmt.Println(i, v)
	// }
	// gitPath, err := exec.LookPath("git")
	// if err != nil {
	// 	t.Fatal(err)
	// }
	/*
	* TODO: Check based off of check-mailmap. The caveat to this is it uses whole 'users'
	* So if you have a line as follows
	* john doe <john.doe@gmail.com> jane doe <jane.doe@gmail.com>
	* The call to git check-mailmap has to be the entire line
	* e.g $ git check-mailmap "jane doe <jane.doe@gmail.com>"
	* Which puts us in a bit of a tizzy since our use case requires us to be able to use just
	* an authors name or email address without being associated with the other necessarily
	 */
	// for i, user := range m.userMap {
	// 	if strings.Contains(user, "@") {
	// 		fmt.Println(i, user)
	// 		args := []string{"check-mailmap"}
	// 		args = append(args, user)
	// 		fmt.Printf("%s %s", gitPath, strings.Join(args, " "))

	// 		cmd := exec.Command(gitPath, args...)
	// 		cmd.Dir = dir
	// 		reader, err := cmd.StdoutPipe()
	// 		if err != nil {
	// 			t.Fatal(err)
	// 		}
	// 		if err := cmd.Start(); err != nil {
	// 			t.Fatal(err)
	// 		}
	// 		scn := bufio.NewScanner(reader)
	// 		scn.Scan()
	// 		txt := scn.Text()
	// 		if txt != m.UseMailmap(user) {
	// 			t.Fatalf("expected %s got %s", txt, m.UseMailmap(user))
	// 		}
	// 	}
	// }
}
