//go:build darwin

package app

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// A Claude invocation may read exactly the account record that identifies its
// Max session. Admin-managed policy must never become readable through this
// boundary: `--safe-mode` still applies admin-managed policy, and managed
// settings can inject environment values, API keys, base URLs, provider
// routing, models, helper commands, and hooks, so a readable managed tree
// would let a run diverge from the recorded model profile without that
// divergence appearing in argv or in the run audit.
func TestClaudeIsolationNeverGrantsAdminManagedPolicy(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	// The account record is the only authentication artifact the client is
	// allowed to read. This list is asserted directly so a re-added managed
	// path fails here even on a host where that path does not exist.
	authentication := claudeAuthenticationPaths(home)
	if len(authentication) != 1 || authentication[0] != filepath.Join(home, ".claude.json") {
		t.Fatalf("Claude client authentication reads are no longer only the account record: %v", authentication)
	}

	workspace := filepath.Join(t.TempDir(), "workspace")
	providerHome := filepath.Join(t.TempDir(), "provider-home")
	client := filepath.Join(t.TempDir(), "claude")
	for _, directory := range []string{workspace, providerHome} {
		if err := os.MkdirAll(directory, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(client, []byte("#!/bin/sh\nexit 0\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	profile, err := isolationProfile(isolationRequest{
		Provider: providerClaudeCode, Workspace: workspace, ProviderHome: providerHome,
		RuntimeExecutables: []string{client},
	})
	if err != nil {
		t.Fatal(err)
	}

	lines := strings.Split(profile, "\n")
	for _, managed := range claudeManagedPolicyPaths {
		quoted := strconv.Quote(managed)
		denial := "(deny file-read* file-write* (subpath " + quoted + "))"
		if !strings.Contains(profile, denial) {
			t.Errorf("profile does not deny the admin-managed policy tree %s", managed)
		}
		lastAllow, lastDeny := -1, -1
		for index, line := range lines {
			if !strings.Contains(line, quoted) {
				continue
			}
			if strings.Contains(line, "(allow") {
				lastAllow = index
			}
			if strings.Contains(line, "(deny") {
				lastDeny = index
			}
		}
		// A Seatbelt profile resolves to its last matching rule, so an allow
		// after the denial would silently re-open the tree.
		if lastAllow > lastDeny {
			t.Errorf("admin-managed policy tree %s is granted after its denial: %s", managed, lines[lastAllow])
		}
	}

	// No rule may grant any admin-managed Claude policy location under any
	// spelling, including one reached through a parent grant.
	for _, line := range lines {
		if !strings.HasPrefix(line, "(allow") && !strings.Contains(line, "(allow file-read") {
			continue
		}
		for _, fragment := range []string{"ClaudeCode", "Managed Preferences", "/Library/Application Support"} {
			if strings.Contains(line, fragment) {
				t.Errorf("profile grants an admin-managed policy location: %s", line)
			}
		}
	}
}
