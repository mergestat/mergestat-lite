package git

import (
	"context"
	"github.com/augmentable-dev/askgit/tables/services"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/utils/merkletrie"
	"github.com/pkg/errors"
	"go.riyazali.net/sqlite"
	"os"
	"strings"
)

type StatsModule struct{ Locator services.RepoLocator }

func (mod *StatsModule) Connect(_ *sqlite.Conn, _ []string, declare func(string) error) (sqlite.VirtualTable, error) {
	const schema = `
		CREATE TABLE stats (
			file		TEXT,
			additions	INT,
			deletions	INT,
			action		TEXT,
			
			repository	HIDDEN,
			start		HIDDEN,
			end			HIDDEN
		);`

	return &gitStatsTable{Locator: mod.Locator}, declare(schema)
}

type gitStatsTable struct{ Locator services.RepoLocator }

func (tab *gitStatsTable) Disconnect() error { return nil }
func (tab *gitStatsTable) Destroy() error    { return nil }
func (tab *gitStatsTable) Open() (sqlite.VirtualCursor, error) {
	return &gitStatsCursor{Locator: tab.Locator}, nil
}

// BestIndex analyses the input constraint and generates the best possible query plan for sqlite3.
// Refer to gitLogTable.BestIndex for details on xFilter contract.
func (tab *gitStatsTable) BestIndex(input *sqlite.IndexInfoInput) (*sqlite.IndexInfoOutput, error) {
	var argv, seenFrom = 0, false
	var bitmap []byte
	var set = func(op, col int) { bitmap = append(bitmap, byte(op<<4|col)) }

	var out = &sqlite.IndexInfoOutput{}
	out.ConstraintUsage = make([]*sqlite.ConstraintUsage, len(input.Constraints))

	for i, constraint := range input.Constraints {
		idx := constraint.ColumnIndex

		if idx < 4 {
			continue // we do not use any constraints on output / non-hidden columns
		}

		if !constraint.Usable {
			// for other constraints, if provided they must be usable
			return nil, sqlite.SQLITE_CONSTRAINT
		}

		argv += 1
		if constraint.Op == sqlite.INDEX_CONSTRAINT_EQ {
			set(1, idx)
			out.ConstraintUsage[i] = &sqlite.ConstraintUsage{ArgvIndex: argv, Omit: true}
		}

		if idx == 5 {
			seenFrom = true
		}
	}

	if !seenFrom {
		// a constraint is required on from column
		return nil, sqlite.SQLITE_CONSTRAINT
	}

	if input.ColUsed != nil {
		bit := func(n, pos int) bool { return n&(1<<pos) > 0 }
		if cols := int(*input.ColUsed); bit(cols, 1) || bit(cols, 2) {
			out.IndexNumber = 1
		}
	} else {
		// for older versions there is no way of knowing
		// so we just assume that addition and/or deletion column will be used
		// and always compute patch
		out.IndexNumber = 1
	}

	out.IndexString = enc(bitmap)
	return out, nil
}

type gitStatsCursor struct {
	Locator services.RepoLocator

	repo         *git.Repository
	computePatch bool // true if addition and / or deletion columns are used and we need to compute patch

	current int              // current index into changes
	changes object.Changes   // list of changed files
	stats   object.FileStats // current diff stats
}

func (cur *gitStatsCursor) Filter(i int, s string, values ...sqlite.Value) (err error) {
	// values extracted from constraints
	var path string
	var fromHash, toHash plumbing.Hash

	var bitmap, _ = dec(s)
	for i, val := range values {
		switch b := bitmap[i]; b {
		case 0b0001_0100:
			path = val.Text()
		case 0b0001_0101:
			fromHash = plumbing.NewHash(val.Text())
		case 0b0001_0110:
			toHash = plumbing.NewHash(val.Text())
		}
	}

	var repo *git.Repository
	{ // open the git repository
		if path == "" {
			path, _ = os.Getwd()
		}

		if repo, err = cur.Locator.Open(context.Background(), path); err != nil {
			return errors.Wrapf(err, "failed to open %q", path)
		}
		cur.repo = repo
	}

	var from, to *object.Commit
	{ // find from and to commits; to would be nil for root commit
		if from, err = repo.CommitObject(fromHash); err != nil {
			return errors.Wrapf(err, "failed to find commit %q", fromHash)
		}

		if toHash == plumbing.ZeroHash && from.NumParents() > 0 { // no hash is provided
			if to, err = from.Parent(0); err != nil {
				return errors.Wrapf(err, "failed to fetch parent for %q", fromHash)
			}
		} else if toHash != plumbing.ZeroHash {
			if to, err = repo.CommitObject(toHash); err != nil {
				return errors.Wrapf(err, "failed to find commit %q", toHash)
			}
		}
	}

	var start, end *object.Tree
	{ // get trees from from and to
		if start, err = from.Tree(); err != nil {
			return errors.Wrapf(err, "failed to fetch tree for %q", fromHash)
		}

		if to != nil {
			if end, err = to.Tree(); err != nil {
				return errors.Wrapf(err, "failed to fetch tree for %q", toHash)
			}
		}
	}

	if cur.changes, err = start.Diff(end); err != nil {
		return errors.Wrapf(err, "failed to diff %q..%q", fromHash, toHash)
	}

	cur.current = -1 // cur.changes is 0-indexed and Next() increments the current counter
	cur.computePatch = i == 1 // 1 indicates either addition/deletion column is used and we must compute patch

	return cur.Next()
}

func (cur *gitStatsCursor) Column(c *sqlite.Context, col int) error {
	change, stats := cur.changes[cur.current], cur.stats

	switch col {
	case 0:
		{
			if action, _ := change.Action(); action == merkletrie.Insert {
				c.ResultText(change.To.Name)
			} else {
				c.ResultText(change.From.Name)
			}
		}
	case 1:
		var a = 0
		for i := range stats {
			a += stats[i].Addition
		}
		c.ResultInt(a)
	case 2:
		var d = 0
		for i := range stats {
			d += stats[i].Deletion
		}
		c.ResultInt(d)

	case 3:
		action, _ := change.Action()
		c.ResultText(strings.ToLower(action.String()))
	}

	return nil
}

func (cur *gitStatsCursor) Next() error {
	cur.current += 1
	if !cur.Eof() && cur.computePatch {
		patch, err := cur.changes[cur.current].Patch()
		if err != nil {
			return errors.Wrapf(err, "failed to compute patch")
		}
		cur.stats = patch.Stats()
	} else {
		cur.stats = nil
	}
	return nil
}

func (cur *gitStatsCursor) Rowid() (int64, error) { return int64(cur.current), nil }
func (cur *gitStatsCursor) Eof() bool             { return cur.current >= len(cur.changes) }
func (cur *gitStatsCursor) Close() error          { return nil }
