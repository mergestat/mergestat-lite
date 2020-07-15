package gitqlite

import (
	"database/sql"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"testing"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

var (
	fixtureRepoCloneURL = "https://github.com/augmentable-dev/tickgit"
	fixtureRepo         *git.Repository
	instance            *GitQLite
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

	fixtureRepo, err = git.PlainClone(dir, false, &git.CloneOptions{
		URL: fixtureRepoCloneURL,
	})
	if err != nil {
		return nil, err
	}
	instance, err = New(dir, &Options{
		UseCli:  true,
		Testing: true,
	})
	if err != nil {
		return nil, err
	}

	return func() error {
		err := os.RemoveAll(dir)
		if err != nil {
			return err
		}
		return nil
	}, nil
}

func TestModuleInitialization(t *testing.T) {
	if instance.DB == nil {
		t.Fatal("expected non-nil DB, got nil")
	}
}

func TestCommitCounts(t *testing.T) {
	headRef, err := fixtureRepo.Head()
	if err != nil {
		t.Fatal(err)
	}
	commitChecker, err := fixtureRepo.Log(&git.LogOptions{
		From:  headRef.Hash(),
		Order: git.LogOrderCommitterTime,
	})
	if err != nil {
		t.Fatal(err)
	}

	commitCount := 0
	err = commitChecker.ForEach(func(c *object.Commit) error {
		commitCount++
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	//checks commits
	rows, err := instance.DB.Query("SELECT * FROM commits")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	columns, err := rows.Columns()
	if err != nil {
		t.Fatal(err)
	}
	expected := 14
	if len(columns) != expected {
		t.Fatalf("expected %d columns, got: %d", expected, len(columns))
	}
	numRows := getRowsCount(rows)

	expected = commitCount
	if numRows != expected {
		t.Fatalf("expected %d rows got: %d", expected, numRows)
	}

	rows, err = instance.DB.Query("SELECT id, author_name FROM commits")
	if err != nil {
		t.Fatal(err)
	}
	rowNum, contents, err := getContents(rows)
	if err != nil {
		t.Fatalf("err %d at row Number %d", err, rowNum)
	}
	for i, c := range contents {
		commit, err := commitChecker.Next()
		if err != nil {
			if err == io.EOF {
				break
			} else {
				t.Fatal(err)
			}
		}
		if commit.ID().String() != c[0] {
			t.Fatalf("expected %s at row %d got %s", commit.ID().String(), i, c[0])
		}
		if commit.Author.Name != c[1] {
			t.Fatalf("expected %s at row %d got %s", commit.Author.Name, i, c[1])
		}

	}

}
func TestGoGitCommit(t *testing.T) {
	// only use this if sys doesn't have git CLI as name of vtable will be commits
	_, err := exec.LookPath("git")
	if err == nil {
		headRef, err := fixtureRepo.Head()
		if err != nil {
			t.Fatal(err)
		}
		commitChecker, err := fixtureRepo.Log(&git.LogOptions{
			From:  headRef.Hash(),
			Order: git.LogOrderCommitterTime,
		})
		if err != nil {
			t.Fatal(err)
		}

		commitCount := 0
		err = commitChecker.ForEach(func(c *object.Commit) error {
			commitCount++
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}

		//checks commits
		rows, err := instance.DB.Query("SELECT * FROM go_git_commits")
		if err != nil {
			t.Fatal(err)
		}
		defer rows.Close()
		columns, err := rows.Columns()
		if err != nil {
			t.Fatal(err)
		}
		expected := 14
		if len(columns) != expected {
			t.Fatalf("expected %d columns, got: %d", expected, len(columns))
		}
		numRows := getRowsCount(rows)

		expected = commitCount
		if numRows != expected {
			t.Fatalf("expected %d rows got: %d", expected, numRows)
		}

		rows, err = instance.DB.Query("SELECT id, author_name FROM go_git_commits")
		if err != nil {
			t.Fatal(err)
		}
		rowNum, contents, err := getContents(rows)
		if err != nil {
			t.Fatalf("err %d at row Number %d", err, rowNum)
		}
		for i, c := range contents {
			commit, err := commitChecker.Next()
			if err != nil {
				if err == io.EOF {
					break
				} else {
					t.Fatal(err)
				}
			}
			if commit.ID().String() != c[0] {
				t.Fatalf("expected %s at row %d got %s", commit.ID().String(), i, c[0])
			}
			if commit.Author.Name != c[1] {
				t.Fatalf("expected %s at row %d got %s", commit.Author.Name, i, c[1])
			}

		}
	}

}

func TestFileCounts(t *testing.T) {
	headRef, err := fixtureRepo.Head()
	if err != nil {
		t.Fatal(err)
	}
	commitChecker, err := fixtureRepo.Log(&git.LogOptions{
		From:  headRef.Hash(),
		Order: git.LogOrderCommitterTime,
	})
	if err != nil {
		t.Fatal(err)
	}
	numFiles := 0
	err = commitChecker.ForEach(func(c *object.Commit) error {
		numFiles++
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	fileRows, err := instance.DB.Query("SELECT DISTINCT commit_id FROM files")
	if err != nil {
		t.Fatal(err)
	}
	defer fileRows.Close()
	numFileRows := getRowsCount(fileRows)
	if numFileRows != numFiles {
		t.Fatalf("expected %d rows got : %d", numFiles, numFileRows)
	}
	columnQuery, err := instance.DB.Query("SELECT * FROM files LIMIT 1")
	if err != nil {
		t.Fatal(err)
	}
	columns, err := columnQuery.Columns()
	if err != nil {
		t.Fatal(err)
	}
	if len(columns) != 6 {
		t.Fatalf("expected %d columns got : %d", 6, len(columns))
	}
}

func TestRefCounts(t *testing.T) {
	refChecker, err := fixtureRepo.References()
	if err != nil {
		t.Fatal(err)
	}
	refCount := 0
	err = refChecker.ForEach(func(r *plumbing.Reference) error {
		refCount++
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	//check refs
	refRows, err := instance.DB.Query("SELECT * FROM refs")
	if err != nil {
		t.Fatal(err)
	}

	columns, err := refRows.Columns()
	if err != nil {
		t.Fatal(err)
	}
	expected := 3
	if len(columns) != expected {
		t.Fatalf("expected %d columns, got: %d", expected, len(columns))
	}
	numRows := getRowsCount(refRows)
	if numRows != refCount {
		t.Fatalf("expected %d rows got : %d", refCount, numRows)
	}
	refRows, err = instance.DB.Query("SELECT * FROM refs")
	if err != nil {
		t.Fatal(err)
	}
	rowNum, contents, err := getContents(refRows)
	if err != nil {
		t.Fatalf("err %d at row Number %d", err, rowNum)
	}
	for i, c := range contents {
		ref, err := refChecker.Next()
		if err != nil {
			if err == io.EOF {
				break
			} else {
				t.Fatal(err)
			}
		}
		if ref.Name().String() != c[0] {
			t.Fatalf("expected %s at row %d got %s", ref.Name().String(), i, c[0])
		}
		if ref.Type().String() != c[1] {
			t.Fatalf("expected %s at row %d got %s", ref.Type().String(), i, c[1])
		}
		if ref.Hash().String() != c[2] {
			t.Fatalf("expected %s at row %d got %s", ref.Hash().String(), i, c[2])
		}

	}
}

func TestTags(t *testing.T) {

	tagIterator, err := fixtureRepo.Tags()
	if err != nil {
		t.Fatal(err)
	}
	tagRows, err := instance.DB.Query("SELECT * FROM tags")
	if err != nil {
		t.Fatal(err)
	}
	rowNum, contents, err := getContents(tagRows)
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
func TestBranches(t *testing.T) {
	localBranchIterator, err := fixtureRepo.Branches()
	if err != nil {
		t.Fatal(err)
	}
	remoteBranchIterator, err := remoteBranches(fixtureRepo.Storer)
	if err != nil {
		t.Fatal(err)
	}
	branchRows, err := instance.DB.Query("SELECT * FROM branches")
	if err != nil {
		t.Fatal(err)
	}
	rowNum, contents, err := getContents(branchRows)
	if err != nil {
		t.Fatalf("err %d at row Number %d", err, rowNum)
	}
	for i, c := range contents {
		branch, err := localBranchIterator.Next()
		if err != nil {
			if err == io.EOF {
				branch, err = remoteBranchIterator.Next()
				if err != nil {
					if err == io.EOF {
						break
					} else {
						t.Fatal(err)
					}
				}
			} else {
				t.Fatal(err)
			}
		}
		if branch.Name().Short() != c[0] || branch.Hash().String() != c[4] {
			t.Fatalf("expected %s at row %d got %s", branch.Name().String(), i, c[0])
		}
		if branch.Hash().String() != c[4] {
			t.Fatalf("expected %s at row %d got %s", branch.Hash().String(), i, c[4])
		}

	}
}
func getRowsCount(rows *sql.Rows) int {
	count := 0
	for rows.Next() {
		count++
	}

	return count
}
func getContents(rows *sql.Rows) (int, [][]string, error) {
	var (
		count int = 0
	)
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
