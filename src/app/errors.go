package app

import "errors"

// ErrNamespaceNotFound indicates there are no processes in the given namespace
var ErrNamespaceNotFound = errors.New("namespace not found")

