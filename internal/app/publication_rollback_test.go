package app

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func openTestStore(t *testing.T) (*artifactStore, string) {
	t.Helper()
	directory := t.TempDir()
	store, err := openArtifactStore(directory)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return store, directory
}

// takeOver replaces the artifact name with a file this controller never
// created, exactly as a competing publisher would.
func takeOver(t *testing.T, directory, name, content string) os.FileInfo {
	t.Helper()
	path := filepath.Join(directory, name)
	_ = os.Remove(path)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	info, err := os.Lstat(path)
	if err != nil {
		t.Fatal(err)
	}
	return info
}

func assertNoRollbackResidue(t *testing.T, directory string) {
	t.Helper()
	residue, err := filepath.Glob(filepath.Join(directory, ".write-uuter-*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(residue) != 0 {
		t.Fatalf("rollback left private residue: %v", residue)
	}
}

func TestRemoveOwnedRefusesCompetitorThatTookTheNameBeforeRollback(t *testing.T) {
	store, directory := openTestStore(t)
	owned, err := store.writeAtomicNoReplace("article.md", []byte("ours\n"), 0o644)
	if err != nil {
		t.Fatal(err)
	}
	competitor := takeOver(t, directory, "article.md", "competitor\n")

	err = store.removeOwned("article.md", owned)
	if !errors.Is(err, errCompetingArtifact) {
		t.Fatalf("rollback did not report a competing artifact: %v", err)
	}
	data, readErr := os.ReadFile(filepath.Join(directory, "article.md"))
	if readErr != nil || string(data) != "competitor\n" {
		t.Fatalf("competing article was removed or changed: %q, %v", data, readErr)
	}
	restored, statErr := os.Lstat(filepath.Join(directory, "article.md"))
	if statErr != nil || !os.SameFile(competitor, restored) {
		t.Fatalf("competing article identity changed: %v", statErr)
	}
	assertNoRollbackResidue(t, directory)
}

// The dangerous window is between deciding that the name still holds our file
// and unlinking that name. This drives a takeover into exactly that window.
func TestRemoveOwnedSurvivesCompetitorTakeoverDuringRollback(t *testing.T) {
	store, directory := openTestStore(t)
	owned, err := store.writeAtomicNoReplace("article.md", []byte("ours\n"), 0o644)
	if err != nil {
		t.Fatal(err)
	}
	var competitor os.FileInfo
	store.afterRollbackClaim = func() {
		store.afterRollbackClaim = nil
		competitor = takeOver(t, directory, "article.md", "competitor\n")
	}

	if err := store.removeOwned("article.md", owned); err != nil {
		t.Fatalf("rollback of our own article failed: %v", err)
	}
	if competitor == nil {
		t.Fatal("takeover seam never ran")
	}
	data, readErr := os.ReadFile(filepath.Join(directory, "article.md"))
	if readErr != nil || string(data) != "competitor\n" {
		t.Fatalf("rollback deleted the competitor that took the name: %q, %v", data, readErr)
	}
	survivor, statErr := os.Lstat(filepath.Join(directory, "article.md"))
	if statErr != nil || !os.SameFile(competitor, survivor) {
		t.Fatalf("competing article identity changed during rollback: %v", statErr)
	}
	assertNoRollbackResidue(t, directory)
}

func TestRemoveOwnedIsANoOpWhenTheArtifactIsAlreadyGone(t *testing.T) {
	store, directory := openTestStore(t)
	owned, err := store.writeAtomicNoReplace("article.md", []byte("ours\n"), 0o644)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(directory, "article.md")); err != nil {
		t.Fatal(err)
	}
	if err := store.removeOwned("article.md", owned); err != nil {
		t.Fatalf("rollback of an absent artifact failed: %v", err)
	}
	assertNoRollbackResidue(t, directory)
}

// A durability barrier failure after the rename still leaves the artifact on
// disk, so the identity must come back with the error or the caller can never
// roll it back.
func TestWriteAtomicNoReplaceReportsIdentityWhenDirectorySyncFails(t *testing.T) {
	store, directory := openTestStore(t)
	t.Setenv("WRITE_UUTER_TEST_FAIL_COMMIT_SYNC", "article.md")

	published, err := store.writeAtomicNoReplace("article.md", []byte("ours\n"), 0o644)
	if err == nil {
		t.Fatal("injected commit sync failure was not reported")
	}
	if published == nil {
		t.Fatal("committed article identity was lost with the sync error")
	}
	committed, statErr := os.Lstat(filepath.Join(directory, "article.md"))
	if statErr != nil {
		t.Fatalf("article was not committed despite a successful rename: %v", statErr)
	}
	if !os.SameFile(published, committed) {
		t.Fatal("reported identity does not name the committed article")
	}
	t.Setenv("WRITE_UUTER_TEST_FAIL_COMMIT_SYNC", "")
	if err := store.removeOwned("article.md", published); err != nil {
		t.Fatalf("committed-but-unsynced article could not be rolled back: %v", err)
	}
	if _, err := os.Lstat(filepath.Join(directory, "article.md")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("rollback left the unsynced article behind: %v", err)
	}
	assertNoRollbackResidue(t, directory)
}

func TestRenameNoReplacePathReportsCommitAcrossSyncFailure(t *testing.T) {
	parent := t.TempDir()
	source := filepath.Join(parent, "workspace.tmp")
	target := filepath.Join(parent, "run")
	if err := os.Mkdir(source, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("WRITE_UUTER_TEST_FAIL_COMMIT_SYNC", "run-workspace")

	committed, err := renameNoReplacePath(source, target)
	if err == nil {
		t.Fatal("injected run-workspace sync failure was not reported")
	}
	if !committed {
		t.Fatal("a successful rename was reported as uncommitted")
	}
	if info, statErr := os.Lstat(target); statErr != nil || !info.IsDir() {
		t.Fatalf("run workspace was not committed: %v", statErr)
	}
	if _, statErr := os.Lstat(source); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("temporary workspace name survived the commit: %v", statErr)
	}

	t.Setenv("WRITE_UUTER_TEST_FAIL_COMMIT_SYNC", "")
	blocked := filepath.Join(parent, "other.tmp")
	if err := os.Mkdir(blocked, 0o755); err != nil {
		t.Fatal(err)
	}
	committed, err = renameNoReplacePath(blocked, target)
	if err == nil || committed {
		t.Fatalf("a competing target was reported as committed: committed=%v err=%v", committed, err)
	}
}
