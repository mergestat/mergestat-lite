// Package sqlite provides a golang target that depends on sqlite's C amalgamation file.
// This way we only need to build the C file once and the Golang toolchain would cache
// the generated artifact, thus, speeding up future builds.
package sqlite

// #cgo CFLAGS: -DSQLITE_CORE
// #cgo CFLAGS: -DSQLITE_ENABLE_EXPLAIN_COMMENTS
// #cgo CFLAGS: -DSQLITE_DQS=0
// #cgo CFLAGS: -DSQLITE_LIKE_DOESNT_MATCH_BLOBS
// #cgo CFLAGS: -DSQLITE_THREADSAFE=2
// #cgo CFLAGS: -DSQLITE_DEFAULT_MEMSTATUS=0
// #cgo CFLAGS: -DSQLITE_OMIT_DEPRECATED
// #cgo CFLAGS: -DSQLITE_OMIT_GET_TABLE
// #cgo CFLAGS: -DSQLITE_OMIT_TCL_VARIABLE
// #cgo CFLAGS: -DSQLITE_OMIT_PROGRESS_CALLBACK
// #cgo CFLAGS: -DSQLITE_OMIT_SHARED_CACHE
// #cgo CFLAGS: -DSQLITE_TRACE_SIZE_LIMIT=32
// #cgo CFLAGS: -DSQLITE_DEFAULT_FOREIGN_KEYS=1
// #cgo CFLAGS: -DSQLITE_DEFAULT_WAL_SYNCHRONOUS=1
// #cgo CFLAGS: -DSQLITE_ENABLE_COLUMN_METADATA
// #cgo CFLAGS: -DSQLITE_INTROSPECTION_PRAGMAS
// #cgo CFLAGS: -DSQLITE_USE_URI=1
// #cgo CFLAGS: -DSQLITE_ENABLE_RTREE
// #cgo CFLAGS: -DHAVE_USLEEP=1
// #cgo CFLAGS: -DSQLITE_ENABLE_FTS3
// #cgo CFLAGS: -DSQLITE_ENABLE_FTS3_PARENTHESIS
// #cgo CFLAGS: -DSQLITE_ENABLE_UPDATE_DELETE_LIMIT
//
// #include "../../.build/sqlite3/sqlite3.c"
//
// // extension function defined in the archive from go.riyazali.net/sqlite
// // the symbol is only available during the final linkage when compiling the binary
// extern int sqlite3_extension_init(sqlite3*, char**, const sqlite3_api_routines*);
import "C"

// register sqlite3_extension_init with sqlite3_auto_extension so that
// the extension is registered with all the database connections
// opened with the sqlite3 library
func init() { C.sqlite3_auto_extension((*[0]byte)(C.sqlite3_extension_init)) }
