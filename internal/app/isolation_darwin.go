//go:build darwin

package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// keychainClientPath is the system keychain client Claude Code runs to read the
// stored OAuth credential.
const keychainClientPath = "/usr/bin/security"

func isolationProfile(request isolationRequest) (string, error) {
	workspace := request.Workspace
	providerHome := request.ProviderHome
	runtimeExecutables := append([]string(nil), request.RuntimeExecutables...)
	paths := []*string{&workspace, &providerHome}
	for _, path := range paths {
		canonical, canonicalErr := canonicalIsolationPath(*path)
		if canonicalErr != nil {
			return "", canonicalErr
		}
		*path = canonical
	}
	for index := range runtimeExecutables {
		canonical, err := canonicalIsolationPath(runtimeExecutables[index])
		if err != nil {
			return "", err
		}
		runtimeExecutables[index] = canonical
	}
	// Claude Code resolves the logged-in Max account from two places: the
	// account record in the user's top-level Claude configuration file, which
	// the client reads itself, and the OAuth credential in the login keychain,
	// which it reads by running the system keychain client. Both are scoped as
	// narrowly as the operating system allows below. Everything else the user
	// owns - the home directory, ~/.claude and its settings, history, plugins,
	// skills, hooks, MCP configuration, sessions, and projects - stays outside
	// the sandbox for the client as well, and --safe-mode additionally stops
	// the client loading any non-managed customization it could otherwise read.
	var claudeClientReads []string
	var keychainReads []string
	if request.Provider == providerClaudeCode {
		home, homeErr := os.UserHomeDir()
		if homeErr != nil {
			return "", fmt.Errorf("resolve Claude authentication location: %w", homeErr)
		}
		claudeClientReads = existingIsolationPaths(claudeAuthenticationPaths(home)...)
		keychainReads = existingIsolationPaths(
			filepath.Join(home, "Library", "Keychains"),
			"/Library/Keychains",
			"/private/var/db/mds",
			"/usr/share",
		)
	}
	keychainClient := ""
	var profile strings.Builder
	// Import only the dynamic-loader rules, not system.sb: its version-1 XPC
	// compatibility grant would expose every user-session service to tools.
	// Runtime services needed by the client are granted explicitly below.
	profile.WriteString("(version 1)\n(deny default)\n(import \"dyld-support.sb\")\n")
	profile.WriteString("(allow sysctl-read)\n")
	// Never expose interactive user secrets to either the client or a spawned
	// tool. Codex authentication is file-scoped to its staged CODEX_HOME and
	// does not require the login Keychain or pasteboard services.
	for _, service := range []string{
		"com.apple.pasteboard", "com.apple.pasteboard.1",
		"com.apple.securityd", "com.apple.securityd.xpc", "com.apple.securityd.systemkeychain",
	} {
		fmt.Fprintf(&profile, "(deny mach-lookup (global-name %s))\n", strconv.Quote(service))
		fmt.Fprintf(&profile, "(deny mach-lookup (xpc-service-name %s))\n", strconv.Quote(service))
	}
	profile.WriteString("(deny file-read* file-write* (subpath \"/usr/local\"))\n")
	profile.WriteString("(deny file-read* file-write* (literal \"/dev/zero\"))\n")
	systemReads := []string{"/System", "/usr/bin", "/usr/lib", "/bin", "/Library/Apple", "/private/etc/ssl"}
	if request.Provider == providerClaudeCode {
		// The Claude client loads localization and timezone data from
		// /usr/share; without it the runtime aborts before it can start.
		systemReads = append(systemReads, "/usr/share")
	}
	for _, readable := range systemReads {
		fmt.Fprintf(&profile, "(allow file-read* (subpath %s))\n", strconv.Quote(readable))
	}
	// The installed Codex client has a machine policy at this fixed location.
	// Permit that file only, not the containing configuration directory.
	profile.WriteString("(allow file-read* (literal \"/private/etc/codex/requirements.toml\"))\n")
	for _, readable := range []string{"/dev/null", "/dev/random", "/dev/urandom", "/dev/tty"} {
		fmt.Fprintf(&profile, "(allow file-read* file-write* (literal %s))\n", strconv.Quote(readable))
	}
	metadataPaths := append(pathAncestors(workspace), pathAncestors(providerHome)...)
	for _, path := range append(append([]string(nil), claudeClientReads...), keychainReads...) {
		metadataPaths = append(metadataPaths, pathAncestors(path)...)
	}
	for _, executable := range runtimeExecutables {
		metadataPaths = append(metadataPaths, pathAncestors(executable)...)
	}
	seenMetadata := make(map[string]bool)
	// macOS presents /var as a symlink to /private/var. Controller paths are
	// canonicalized for policy rules, while Go may still traverse the spelling
	// inherited through TMPDIR before reaching the canonical subtree.
	metadataPaths = append(metadataPaths, "/var", "/etc", "/private/etc/codex")
	for _, parent := range metadataPaths {
		if seenMetadata[parent] {
			continue
		}
		seenMetadata[parent] = true
		fmt.Fprintf(&profile, "(allow file-read-metadata (literal %s))\n", strconv.Quote(parent))
	}
	fmt.Fprintf(&profile, "(allow file-read* file-write* (subpath %s))\n", strconv.Quote(workspace))
	for index, executable := range runtimeExecutables {
		fmt.Fprintf(&profile, "(allow file-read* (literal %s))\n", strconv.Quote(executable))
		if index == 0 {
			// The initial sandbox-exec may enter this invocation's client exactly
			// once. Sandboxed descendants may execute other tools, but cannot enter
			// either sandbox-exec or the privileged client target, so path-filtered
			// authority cannot be reacquired by exec.
			fmt.Fprintf(&profile, "(allow process-exec (require-all (require-not (literal %s)) (require-not (literal \"/usr/bin/sandbox-exec\"))))\n", strconv.Quote(executable))
			fmt.Fprintf(&profile, "(with-filter (process-path \"/usr/bin/sandbox-exec\") (allow process-exec (literal %s)))\n", strconv.Quote(executable))
			fmt.Fprintf(&profile, "(with-filter (process-path %s) (allow process-fork))\n", strconv.Quote(executable))
			// The authenticated Codex process and its in-process tools share the
			// initial local trust boundary for this CLI slice. The private staged
			// CODEX_HOME remains run-owned and is removed only after process cleanup.
			fmt.Fprintf(&profile, "(with-filter (process-path %s) (allow file-read* file-write* (subpath %s)))\n", strconv.Quote(executable), strconv.Quote(providerHome))
			// The account record is readable, never writable, and only by the
			// exact client path. A model-invoked tool is a different process
			// path and keeps the default denial.
			for _, clientOnly := range claudeClientReads {
				fmt.Fprintf(&profile, "(with-filter (process-path %s) (allow file-read* (subpath %s) (literal %s)))\n",
					strconv.Quote(executable), strconv.Quote(clientOnly), strconv.Quote(clientOnly))
			}
			keychainClient = executable
			fmt.Fprintf(&profile, "(with-filter (process-path %s) (allow network*))\n", strconv.Quote(executable))
			// Revoke every named bootstrap/XPC lookup for model-invoked tools so
			// Keychain, pasteboard, and other user-session services stay outside
			// the privileged client boundary.
			for _, serviceFilter := range []string{"global-name-regex", "local-name-regex", "xpc-service-name-regex"} {
				fmt.Fprintf(&profile, "(with-filter (require-not (process-path %s)) (deny mach-lookup (%s #\".*\")))\n", strconv.Quote(executable), serviceFilter)
			}
			fmt.Fprintf(&profile, "(with-filter (process-path %s) (allow mach-lookup))\n", strconv.Quote(executable))
		}
	}
	// Keychain rules come last on purpose: a Seatbelt profile resolves to its
	// last matching rule, and the blanket service denial above would otherwise
	// win over the narrow grant below.
	if len(keychainReads) != 0 && keychainClient != "" {
		// The OAuth credential itself is never handed to the client, to its
		// tools, or to a run artifact. Only the system keychain client may read
		// the keychain stores, and only the exact staged Claude client may start
		// it: a shell, an interpreter, or any copied binary a model invokes is a
		// different process path, so it can neither run the keychain client nor
		// impersonate it.
		fmt.Fprintf(&profile, "(with-filter (process-path %s) (allow mach-lookup))\n", strconv.Quote(keychainClientPath))
		for _, readable := range keychainReads {
			fmt.Fprintf(&profile, "(with-filter (process-path %s) (allow file-read* (subpath %s)))\n",
				strconv.Quote(keychainClientPath), strconv.Quote(readable))
		}
		fmt.Fprintf(&profile, "(with-filter (require-not (process-path %s)) (deny process-exec (literal %s)))\n",
			strconv.Quote(keychainClient), strconv.Quote(keychainClientPath))
	}
	// The admin-managed policy trees are denied last, so the denial survives
	// rule resolution no matter what an earlier rule grants. A Seatbelt profile
	// resolves to its last matching rule, which means re-granting these trees
	// would require deleting this block rather than merely shadowing it.
	denied := append([]string(nil), claudeManagedPolicyPaths...)
	denied = append(denied, existingIsolationPaths(claudeManagedPolicyPaths...)...)
	seenDenied := make(map[string]bool, len(denied))
	for _, path := range denied {
		if seenDenied[path] {
			continue
		}
		seenDenied[path] = true
		fmt.Fprintf(&profile, "(deny file-read* file-write* (subpath %s))\n", strconv.Quote(path))
	}
	return profile.String(), nil
}

// claudeAuthenticationPaths are the concrete artifacts the Claude client must
// read to resolve the logged-in Max session: the account record, and nothing
// else. Admin-managed policy locations are deliberately excluded; see
// claudeManagedPolicyPaths.
func claudeAuthenticationPaths(home string) []string {
	return []string{filepath.Join(home, ".claude.json")}
}

// claudeManagedPolicyPaths are the admin-managed policy trees Claude Code
// applies. `--safe-mode` disables user customization but still applies
// admin-managed policy, and managed settings can inject environment values,
// API keys, base URLs, provider routing, models, permission decisions, helper
// commands, and hooks. A readable managed tree could therefore move a run off
// the model profile the controller validated and recorded, or off Max OAuth
// billing entirely, without any of it appearing in argv or in the run audit.
// The client is never granted these paths, and they are denied explicitly.
var claudeManagedPolicyPaths = []string{
	"/Library/Application Support/ClaudeCode",
	"/Library/Managed Preferences",
}

// existingIsolationPaths canonicalizes the paths that exist. A host without one
// of them simply does not receive that rule.
func existingIsolationPaths(paths ...string) []string {
	var resolved []string
	for _, path := range paths {
		canonical, err := filepath.EvalSymlinks(path)
		if err != nil {
			continue
		}
		resolved = append(resolved, canonical)
	}
	return resolved
}

func canonicalIsolationPath(path string) (string, error) {
	if !filepath.IsAbs(path) {
		return "", fmt.Errorf("isolation path is not absolute: %s", path)
	}
	canonical, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", fmt.Errorf("canonicalize isolation path %s: %w", path, err)
	}
	return canonical, nil
}

func isolatedCommand(profilePath, clientPath string, clientArguments []string) (string, []string, error) {
	arguments := []string{"-f", profilePath, clientPath}
	arguments = append(arguments, clientArguments...)
	return "/usr/bin/sandbox-exec", arguments, nil
}

func pathAncestors(path string) []string {
	var reversed []string
	for current := filepath.Clean(path); ; current = filepath.Dir(current) {
		reversed = append(reversed, current)
		parent := filepath.Dir(current)
		if parent == current {
			break
		}
	}
	ancestors := make([]string, 0, len(reversed))
	for index := len(reversed) - 1; index >= 0; index-- {
		ancestors = append(ancestors, reversed[index])
	}
	return ancestors
}
