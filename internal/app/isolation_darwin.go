//go:build darwin

package app

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
)

func isolationProfile(workspace, codexHome string, runtimeExecutables []string) (string, error) {
	paths := []*string{&workspace, &codexHome}
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
	for _, readable := range []string{"/System", "/usr/bin", "/usr/lib", "/bin", "/Library/Apple", "/private/etc/ssl"} {
		fmt.Fprintf(&profile, "(allow file-read* (subpath %s))\n", strconv.Quote(readable))
	}
	// The installed Codex client has a machine policy at this fixed location.
	// Permit that file only, not the containing configuration directory.
	profile.WriteString("(allow file-read* (literal \"/private/etc/codex/requirements.toml\"))\n")
	for _, readable := range []string{"/dev/null", "/dev/random", "/dev/urandom", "/dev/tty"} {
		fmt.Fprintf(&profile, "(allow file-read* file-write* (literal %s))\n", strconv.Quote(readable))
	}
	metadataPaths := append(pathAncestors(workspace), pathAncestors(codexHome)...)
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
			fmt.Fprintf(&profile, "(with-filter (process-path %s) (allow file-read* file-write* (subpath %s)))\n", strconv.Quote(executable), strconv.Quote(codexHome))
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
	return profile.String(), nil
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

func isolatedCommand(profilePath, codexPath string, codexArguments []string) (string, []string, error) {
	arguments := []string{"-f", profilePath, codexPath}
	arguments = append(arguments, codexArguments...)
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
