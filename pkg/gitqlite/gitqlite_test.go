package gitqlite

import (
	"database/sql"
	"io/ioutil"
	"os"
	"os/user"
	"testing"

	git "github.com/libgit2/git2go/v30"
	//"github.com/augmentable-dev/askgit/cmd"
)

var (
	fixtureRepoCloneURL = "https://github.com/augmentable-dev/tickgit"
	fixtureRepo         *git.Repository
	fixtureRepoDir      string
)

func CredentialsCallback(url string, username string, allowedTypes git.CredType) (*git.Cred, error) {
	usr, _ := user.Current()
	publicSSH := usr.HomeDir + "/.ssh/id_rsa.pub"
	privateSSH := usr.HomeDir + "/.ssh/id_rsa"
	cred, ret := git.NewCredSshKey("git", publicSSH, privateSSH, "")
	return cred, ret
}

// Made this one just return 0 during troubleshooting...
func CertificateCheckCallback(cert *git.Certificate, valid bool, hostname string) git.ErrorCode {
	return 0
}
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
	cloneOptions := &git.CloneOptions{}
	// use FetchOptions instead of directly RemoteCallbacks
	// https://github.com/libgit2/git2go/commit/36e0a256fe79f87447bb730fda53e5cbc90eb47c
	cloneOptions.FetchOptions = &git.FetchOptions{
		RemoteCallbacks: git.RemoteCallbacks{
			CredentialsCallback:      CredentialsCallback,
			CertificateCheckCallback: CertificateCheckCallback,
		}}
	fixtureRepo, err = git.Clone(fixtureRepoCloneURL, dir, cloneOptions)
	if err != nil {
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

func TestModuleInitialization(t *testing.T) {
	instance, err := New(fixtureRepoDir, &Options{})
	if err != nil {
		t.Fatal(err)
	}

	if instance.DB == nil {
		t.Fatal("expected non-nil DB, got nil")
	}
}

func GetRowsCount(rows *sql.Rows) int {
	count := 0
	for rows.Next() {
		count++
	}

	return count
}
func GetContents(rows *sql.Rows) (int, [][]string, error) {
	count := 0
	columns, err := rows.Columns()
	if err != nil {
		return count, nil, err
	}

	pointers := make([]interface{}, len(columns))
	container := make([]sql.NullString, len(columns))
	var ret [][]string

	for i := range pointers {
		pointers[i] = &container[i]
	}

	for rows.Next() {
		err = rows.Scan(pointers...)
		if err != nil {
			return count, nil, err
		}

		r := make([]string, len(columns))
		for i, c := range container {
			if c.Valid {
				r[i] = c.String
			} else {
				r[i] = "NULL"
			}
		}
		ret = append(ret, r)
	}
	return count, ret, err

}
