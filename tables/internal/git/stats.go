package git

import (
	"fmt"
	"go.riyazali.net/sqlite"
	"io"

	git "github.com/libgit2/git2go/v31"
)

// StatsModule implements sqlite.Module that provides the "stats" virtual table
type StatsModule struct{}

func (m *StatsModule) Connect(_ *sqlite.Conn, args []string, declare func(string) error) (sqlite.VirtualTable, error) {
	// TODO(@riyaz): parse args to extract repo
	var repo = "."

	var schema = fmt.Sprintf(`CREATE TABLE %q (commit_id TEXT, file_path TEXT, additions INT, deletions INT)`, args[0])
	return &gitStatsTable{path: repo}, declare(schema)
}

type gitStatsTable struct{ path string }

func (v *gitStatsTable) BestIndex(input *sqlite.IndexInfoInput) (*sqlite.IndexInfoOutput, error) {
	var usage = make([]*sqlite.ConstraintUsage, len(input.Constraints))

	// TODO implement an index for file name glob patterns?
	// TODO this loop construct won't work well for multiple constraints...
	for c, constraint := range input.Constraints {
		switch {
		case constraint.Usable && constraint.ColumnIndex == 0 && constraint.Op == sqlite.INDEX_CONSTRAINT_EQ:
			usage[c] = &sqlite.ConstraintUsage{ArgvIndex: 1, Omit: false}
			return &sqlite.IndexInfoOutput{ConstraintUsage: usage, IndexNumber: 1, IndexString: "stats-by-commit-id", EstimatedCost: 1.0, EstimatedRows: 1}, nil
		}
	}

	return &sqlite.IndexInfoOutput{ConstraintUsage: usage, EstimatedCost: 100}, nil
}

func (v *gitStatsTable) Open() (sqlite.VirtualCursor, error) {
	repo, err := git.OpenRepository(v.path)
	if err != nil {
		return nil, err
	}
	return &statsCursor{repo: repo}, nil
}

func (v *gitStatsTable) Disconnect() error { return nil }
func (v *gitStatsTable) Destroy() error    { return nil }

type statsCursor struct {
	repo     *git.Repository
	iterator *commitStatsIter
	current  *commitStat
}

func (vc *statsCursor) Filter(i int, _ string, values ...sqlite.Value) error {
	var opt *commitStatsIterOptions
	switch i {
	case 0:
		opt = &commitStatsIterOptions{}
	case 1:
		opt = &commitStatsIterOptions{commitID: values[0].Text()}
	}

	iter, err := NewCommitStatsIter(vc.repo, opt)
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

func (vc *statsCursor) Column(context *sqlite.Context, i int) error {
	switch stat := vc.current; i {
	case 0:
		context.ResultText(stat.commitID)
	case 1:
		context.ResultText(stat.file)
	case 2:
		context.ResultInt(stat.additions)
	case 3:
		context.ResultInt(stat.deletions)
	}
	return nil
}

func (vc *statsCursor) Next() error {
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

func (vc *statsCursor) Eof() bool             { return vc.current == nil }
func (vc *statsCursor) Rowid() (int64, error) { return int64(0), nil }
func (vc *statsCursor) Close() error          { vc.iterator.Close(); return nil }
