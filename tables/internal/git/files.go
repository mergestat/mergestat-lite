package git

import (
	"fmt"
	"go.riyazali.net/sqlite"
	"io"
	"path"

	git "github.com/libgit2/git2go/v31"
)

type FilesModule struct{}

func (m *FilesModule) Connect(_ *sqlite.Conn, args []string, declare func(string) error) (sqlite.VirtualTable, error) {
	// TODO(@riyaz): parse args to extract repo
	var repo = "."

	var schema = fmt.Sprintf(`CREATE TABLE %q(commit_id TEXT, path TEXT, contents TEXT, executable BOOL)`, args[0])
	return &gitFilesTable{repoPath: repo}, declare(schema)
}

type gitFilesTable struct {
	repoPath string
	repo     *git.Repository
}

func (v *gitFilesTable) BestIndex(input *sqlite.IndexInfoInput) (*sqlite.IndexInfoOutput, error) {
	var usage = make([]*sqlite.ConstraintUsage, len(input.Constraints))

	// TODO implement an index for file name glob patterns?
	// TODO this loop construct won't work well for multiple constraints...
	for c, constraint := range input.Constraints {
		// check if filtered by "WHERE commit_id = 'xxx'" ..
		if constraint.Usable && constraint.ColumnIndex == 0 && constraint.Op == sqlite.INDEX_CONSTRAINT_EQ {
			usage[c] = &sqlite.ConstraintUsage{ArgvIndex: 1, Omit: false}
			return &sqlite.IndexInfoOutput{ConstraintUsage: usage, IndexNumber: 1, IndexString: "files-by-commit-id", EstimatedCost: 1.0, EstimatedRows: 1}, nil
		}
	}

	return &sqlite.IndexInfoOutput{ConstraintUsage: usage, EstimatedCost: 100}, nil
}

func (v *gitFilesTable) Open() (sqlite.VirtualCursor, error) {
	repo, err := git.OpenRepository(v.repoPath)
	if err != nil {
		return nil, err
	}
	v.repo = repo

	return &gitFilesCursor{repo: v.repo}, nil
}

func (v *gitFilesTable) Disconnect() error { return nil }
func (v *gitFilesTable) Destroy() error    { return nil }

type gitFilesCursor struct {
	repo     *git.Repository
	iterator *commitFileIter
	current  *commitFile
}

func (vc *gitFilesCursor) Filter(i int, _ string, values ...sqlite.Value) error {
	var opt *commitFileIterOptions

	switch i {
	case 0:
		opt = &commitFileIterOptions{}
	case 1:
		opt = &commitFileIterOptions{commitID: values[0].Text()}
	}

	iter, err := NewCommitFileIter(vc.repo, opt)
	if err != nil {
		return err
	}

	vc.iterator = iter

	file, err := vc.iterator.Next()
	if err != nil {
		if err == io.EOF {
			vc.current = nil
			return nil
		}
		return err
	}

	vc.current = file
	return nil
}

func (vc *gitFilesCursor) Next() error {
	// iterates to next file
	file, err := vc.iterator.Next()
	if err != nil {
		if err == io.EOF {
			vc.current = nil
			return nil
		}
		return err
	}
	vc.current = file
	return nil
}

func (vc *gitFilesCursor) Column(c *sqlite.Context, col int) error {
	file := vc.current

	switch col {
	case 0:
		//commit id
		c.ResultText(file.commitID)
	case 1:
		//tree name
		c.ResultText(path.Join(file.path, file.Name))
	case 2:
		c.ResultText(string(file.Contents()))
	case 3:
		c.ResultInt(boolToInt(file.Filemode == git.FilemodeBlobExecutable))
	}

	return nil
}

func (vc *gitFilesCursor) Eof() bool             { return vc.current == nil }
func (vc *gitFilesCursor) Rowid() (int64, error) { return int64(0), nil }
func (vc *gitFilesCursor) Close() error          { vc.iterator.Close(); return nil }
