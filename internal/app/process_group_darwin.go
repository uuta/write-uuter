//go:build darwin

package app

// macOS uses stable libproc-backed identities plus the capture runner's private
// inherited open-file boundary. The latter keeps a detached, reparented child
// attributable after ancestry and process-group membership no longer do.
func enableProcessBoundary() error { return nil }
