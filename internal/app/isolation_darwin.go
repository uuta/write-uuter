//go:build darwin

package app

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
)

func isolationProfile(workspace, codexHome, codexPath string) (string, error) {
	paths := []*string{&workspace, &codexHome, &codexPath}
	for _, path := range paths {
		canonical, canonicalErr := canonicalIsolationPath(*path)
		if canonicalErr != nil {
			return "", canonicalErr
		}
		*path = canonical
	}
	var profile strings.Builder
	profile.WriteString("(version 1)\n(deny default)\n(import \"system.sb\")\n")
	profile.WriteString("(allow process-exec process-fork)\n")
	profile.WriteString("(allow network*)\n")
	profile.WriteString("(allow mach-lookup)\n")
	profile.WriteString("(allow sysctl-read)\n")
	for _, readable := range []string{"/System", "/usr", "/bin", "/sbin", "/dev", "/Library/Apple", "/private/etc/ssl"} {
		fmt.Fprintf(&profile, "(allow file-read* (subpath %s))\n", strconv.Quote(readable))
	}
	metadataPaths := append(pathAncestors(workspace), pathAncestors(codexHome)...)
	seenMetadata := make(map[string]bool)
	for _, parent := range metadataPaths {
		if seenMetadata[parent] {
			continue
		}
		seenMetadata[parent] = true
		fmt.Fprintf(&profile, "(allow file-read-metadata (literal %s))\n", strconv.Quote(parent))
	}
	fmt.Fprintf(&profile, "(allow file-read* file-write* (subpath %s))\n", strconv.Quote(workspace))
	fmt.Fprintf(&profile, "(allow file-read* file-write* (subpath %s))\n", strconv.Quote(codexHome))
	fmt.Fprintf(&profile, "(allow file-read* (subpath %s))\n", strconv.Quote(filepath.Dir(codexPath)))
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
