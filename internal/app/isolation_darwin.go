//go:build darwin

package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func isolationProfile(workspace, codexHome, runDir, privateRoot, codexPath, testLogDir, detachedPIDDir string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	paths := []*string{&home, &workspace, &codexHome, &runDir, &privateRoot, &codexPath}
	if testLogDir != "" {
		paths = append(paths, &testLogDir)
	}
	if detachedPIDDir != "" {
		paths = append(paths, &detachedPIDDir)
	}
	for _, path := range paths {
		canonical, canonicalErr := canonicalIsolationPath(*path)
		if canonicalErr != nil {
			return "", canonicalErr
		}
		*path = canonical
	}
	var profile strings.Builder
	profile.WriteString("(version 1)\n(allow default)\n")
	for _, denied := range []string{home, filepath.Dir(runDir), privateRoot} {
		fmt.Fprintf(&profile, "(deny file-read* file-write* (subpath %s))\n", strconv.Quote(denied))
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
	for _, writable := range []string{testLogDir, detachedPIDDir} {
		if writable != "" {
			fmt.Fprintf(&profile, "(allow file-read* file-write* (subpath %s))\n", strconv.Quote(writable))
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
