// +build shared

// This file provides a build target while building the dynamically loadable shared object library.
// It imports github.com/augmentable-dev/askgit/tables which provides the actual extension implementation.
package main

import _ "github.com/augmentable-dev/askgit/tables"

func main() { /* noting here fellas */ }
