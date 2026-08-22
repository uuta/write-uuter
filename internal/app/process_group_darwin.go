//go:build darwin

package app

import "fmt"

// Darwin does not expose a child-subreaper or equivalent lossless descendant
// boundary to this stdlib-only controller. Refuse to launch Codex rather than
// claiming that ancestry sampling is a complete containment mechanism.
func enableProcessBoundary() error {
	return fmt.Errorf("lossless process boundary unavailable on darwin; refusing Codex launch")
}
