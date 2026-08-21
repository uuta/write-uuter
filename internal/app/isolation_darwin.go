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
	profile.WriteString("(version 1)\n(deny default)\n(import \"system.sb\")\n")
	profile.WriteString("(allow process-exec)\n")
	profile.WriteString("(allow mach-lookup)\n")
	profile.WriteString("(allow sysctl-read)\n")
	profile.WriteString("(deny file-read* file-write* (subpath \"/usr/local\"))\n")
	profile.WriteString("(deny file-read* file-write* (literal \"/dev/zero\"))\n")
	for _, readable := range []string{"/System", "/usr/bin", "/usr/lib", "/bin", "/Library/Apple", "/private/etc/ssl"} {
		fmt.Fprintf(&profile, "(allow file-read* (subpath %s))\n", strconv.Quote(readable))
	}
	for _, readable := range []string{"/dev/null", "/dev/random", "/dev/urandom", "/dev/tty"} {
		fmt.Fprintf(&profile, "(allow file-read* file-write* (literal %s))\n", strconv.Quote(readable))
	}
	metadataPaths := append(pathAncestors(workspace), pathAncestors(codexHome)...)
	for _, executable := range runtimeExecutables {
		metadataPaths = append(metadataPaths, pathAncestors(executable)...)
	}
	seenMetadata := make(map[string]bool)
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
			// Only the staged Codex client may fork, read authentication, or open
			// the network. Model-invoked runtimes and tools inherit CODEX_HOME for
			// compatibility, but the kernel denies them those capabilities. A tool
			// can replace itself with one executable; it cannot double-fork.
			fmt.Fprintf(&profile, "(with-filter (process-path %s) (allow process-fork))\n", strconv.Quote(executable))
			fmt.Fprintf(&profile, "(with-filter (process-path %s) (allow file-read* file-write* (subpath %s)))\n", strconv.Quote(executable), strconv.Quote(codexHome))
			fmt.Fprintf(&profile, "(with-filter (process-path %s) (allow network*))\n", strconv.Quote(executable))
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
