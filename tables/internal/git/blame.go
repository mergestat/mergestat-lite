package git

import (
	"fmt"
	"go.riyazali.net/sqlite"
	"io"

	git "github.com/libgit2/git2go/v31"
)

// BlameModule implements sqlite.Module that provides the "blame" virtual table
type BlameModule struct{}

func (m *BlameModule) Connect(_ *sqlite.Conn, args []string, declare func(string) error) (sqlite.VirtualTable, error) {
	// TODO(@riyaz): parse args to extract repo
	var repo = "."

	var schema = fmt.Sprintf(`CREATE TABLE %q (line_no INT, file_path TEXT, commit_id TEXT, line_content TEXT)`, args[0])
	return &gitBlameTable{repoPath: repo}, declare(schema)
}

type gitBlameTable struct{ repoPath string }

func (v *gitBlameTable) BestIndex(_ *sqlite.IndexInfoInput) (*sqlite.IndexInfoOutput, error) {
	// TODO: this should actually be implemented!
	return &sqlite.IndexInfoOutput{}, nil
}

func (v *gitBlameTable) Open() (sqlite.VirtualCursor, error) {
	repo, err := git.OpenRepository(v.repoPath)
	if err != nil {
		return nil, err
	}

	return &blameCursor{repo: repo}, nil
}

func (v *gitBlameTable) Disconnect() error { return nil }
func (v *gitBlameTable) Destroy() error    { return nil }

type blameCursor struct {
	repo    *git.Repository
	current *BlamedLine
	iter    *BlameIterator
}

func (vc *blameCursor) Filter(_ int, _ string, _ ...sqlite.Value) error {
	iterator, err := NewBlameIterator(vc.repo)
	if err != nil {
		return err
	}

	blamedLine, err := iterator.Next()
	if err != nil {
		if err == io.EOF {
			vc.current = nil
			return nil
		}
		return err
	}

	vc.iter = iterator
	vc.current = blamedLine
	return nil
}


func (vc *blameCursor) Next() error {
	blamedLine, err := vc.iter.Next()
	if err != nil {
		if err == io.EOF {
			vc.current = nil
			return nil
		}
		return err
	}
	vc.current = blamedLine
	return nil
}

func (vc *blameCursor) Column(c *sqlite.Context, col int) error {
	blamedLine := vc.current
	switch col {
	case 0:
		c.ResultInt(blamedLine.Line)
	case 1:
		c.ResultText(blamedLine.File)
	case 2:
		c.ResultText(blamedLine.CommitID)
	case 3:
		c.ResultText(blamedLine.Content)
	}

	return nil
}

func (vc *blameCursor) Eof() bool             { return vc.current == nil }
func (vc *blameCursor) Rowid() (int64, error) { return int64(0), nil }
func (vc *blameCursor) Close() error          { return nil }
