package services

// Context is a key-value store that holds general configuration parameters for the "environment"
// in which a module/function is executed in. For instance, it can be used to hold API tokens or configuration.
type Context map[string]string
