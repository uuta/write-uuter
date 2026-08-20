//go:build darwin

package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRenameNoReplaceSupportsRelativeDarwinPaths(t *testing.T) {
	originalWorkingDirectory, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	temporary := t.TempDir()
	if err := os.Chdir(temporary); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(originalWorkingDirectory) })
	if err := os.Mkdir("source", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := renameNoReplace("source", "target"); err != nil {
		t.Fatalf("relative no-replace rename failed: %v", err)
	}
	if info, err := os.Stat(filepath.Join(temporary, "target")); err != nil || !info.IsDir() {
		t.Fatalf("relative target was not committed: %v", err)
	}
}
