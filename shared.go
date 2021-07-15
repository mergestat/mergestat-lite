// +build shared

// This file provides a build target while building the dynamically loadable shared object library.
// It imports github.com/askgitdev/askgit/tables which provides the actual extension implementation.
package main

import (
	"github.com/askgitdev/askgit/pkg/locator"
	"github.com/askgitdev/askgit/tables"
	"go.riyazali.net/sqlite"
)

func init() {
	sqlite.Register(tables.RegisterFn(
		tables.WithExtraFunctions(),
		tables.WithRepoLocator(locator.CachedLocator(locator.MultiLocator())),
	))
}

func main() { /* noting here fellas */ }
