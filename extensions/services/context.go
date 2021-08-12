package services

import "strconv"

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
