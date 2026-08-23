//go:build darwin

package app

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestRenameNoReplaceAtCommitsRelativeToDirectoryDescriptor(t *testing.T) {
	temporary := t.TempDir()
	if err := os.WriteFile(filepath.Join(temporary, "source"), []byte("payload\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	directory, err := os.Open(temporary)
	if err != nil {
		t.Fatal(err)
	}
	defer directory.Close()
	if err := renameNoReplaceIn(directory, "source", "target"); err != nil {
		t.Fatalf("descriptor-relative no-replace rename failed: %v", err)
	}
	if data, err := os.ReadFile(filepath.Join(temporary, "target")); err != nil || string(data) != "payload\n" {
		t.Fatalf("relative target was not committed: %v %q", err, data)
	}
	if err := os.WriteFile(filepath.Join(temporary, "source"), []byte("second\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	err = renameNoReplaceIn(directory, "source", "target")
	if !errors.Is(err, os.ErrExist) {
		t.Fatalf("no-replace rename replaced an existing target: %v", err)
	}
	if data, readErr := os.ReadFile(filepath.Join(temporary, "target")); readErr != nil || string(data) != "payload\n" {
		t.Fatalf("existing target was disturbed: %v %q", readErr, data)
	}
}
