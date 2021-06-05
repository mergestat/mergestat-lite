// Package services provide types and interfaces that defines the various
// abstract services that different virtual module implementations depend upon.
// The reason its extracted out like this is to prevent a circular dependency
// from virtual modules to tables package and vice-versa.
package services
