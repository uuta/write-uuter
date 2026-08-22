//go:build darwin

package app

// Darwin has no Linux-style child-subreaper primitive. The stable process
// handle/identity implementation remains the containment boundary there.
func enableProcessBoundary() error { return nil }
