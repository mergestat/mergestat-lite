// Package native provides virtual table implementations for git tables using libgit2
// via the git2go bindings (https://github.com/libgit2/git2go).
// Some operations are more performant using libgit2 vs go-git, namely, what's involved in
// the `stats`, `files` and `blame` tables, which are implemented in this package.
package native
