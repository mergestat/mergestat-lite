package services

import (
	"strconv"
	"strings"
)

// Context is a key-value store that holds general configuration parameters for the "environment"
// in which a module/function is executed in. For instance, it can be used to hold API tokens or configuration.
type Context map[string]string

// GetInt retrieves a key from the context and tries to parse it as an int.
// The second return val (bool) is false if the key could not be parsed or is unset
func (ctx Context) GetInt(key string) (int, bool) {
	if val, ok := ctx[key]; ok && val != "" {
		if i, err := strconv.Atoi(val); err != nil {
			return 0, false
		} else {
			return i, true
		}
	} else {
		return 0, false
	}
}

// GetBool retrieves a key from the context and returns true if it matches the string "true" (case insensitive),
// otherwise it returns false if the key is unset or does not match "true". The second return indicates whether
// a value was set in the key at all.
func (ctx Context) GetBool(key string) (bool, bool) {
	if val, ok := ctx[key]; ok && val != "" {
		if strings.EqualFold(val, "true") {
			return true, true
		} else {
			return false, true
		}
	} else {
		return false, false
	}
}
