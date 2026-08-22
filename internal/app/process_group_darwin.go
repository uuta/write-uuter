//go:build darwin

package app

// macOS uses stable libproc-backed identities for the controller-launched
// processes it can track. Intentional ancestry escapes are outside this slice's
// guarantee and are audited after termination.
func enableProcessBoundary() error { return nil }
