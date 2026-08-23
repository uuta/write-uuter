package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writePromptBundleFixture(t *testing.T, directory, marker string) {
	t.Helper()
	if err := os.MkdirAll(directory, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range requiredPromptFiles {
		if err := os.WriteFile(filepath.Join(directory, name), []byte(marker+" "+name+"\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
}

func TestPromptBundleContentSurvivesRootAncestorAndFileMutation(t *testing.T) {
	ancestor := t.TempDir()
	parent := filepath.Join(ancestor, "parent")
	directory := filepath.Join(parent, "prompts")
	writePromptBundleFixture(t, directory, "ORIGINAL")

	bundle, err := openPromptBundle(directory)
	if err != nil {
		t.Fatalf("validated prompt bundle rejected: %v", err)
	}
	defer bundle.Close()

	// Final-component mutation: the validated file is unlinked and replaced.
	if err := os.Remove(filepath.Join(directory, "pm.md")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "pm.md"), []byte("HOSTILE pm.md\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	// Final-component mutation via symlink to a file outside the root.
	outside := filepath.Join(ancestor, "outside.md")
	if err := os.WriteFile(outside, []byte("HOSTILE outside\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(directory, "writer.md")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(directory, "writer.md")); err != nil {
		t.Fatal(err)
	}
	// Root mutation: the validated directory is moved away and a hostile
	// directory takes its exact path.
	if err := os.Rename(directory, filepath.Join(parent, "prompts-moved")); err != nil {
		t.Fatal(err)
	}
	writePromptBundleFixture(t, directory, "HOSTILE")
	// Ancestor mutation: the parent of the validated root is replaced too.
	if err := os.Rename(parent, filepath.Join(ancestor, "parent-moved")); err != nil {
		t.Fatal(err)
	}
	writePromptBundleFixture(t, directory, "HOSTILE")

	for _, name := range requiredPromptFiles {
		content, loadErr := bundle.load(name)
		if loadErr != nil {
			t.Fatalf("bound prompt %s became unreadable after mutation: %v", name, loadErr)
		}
		if strings.TrimSpace(content) != "ORIGINAL "+name {
			t.Fatalf("prompt %s was retargeted after validation: %q", name, content)
		}
	}
}

func TestPromptBundleRejectsIncompleteAndNonRegularBundles(t *testing.T) {
	incomplete := t.TempDir()
	writePromptBundleFixture(t, incomplete, "ORIGINAL")
	if err := os.Remove(filepath.Join(incomplete, "reviewer-copy.md")); err != nil {
		t.Fatal(err)
	}
	if bundle, err := openPromptBundle(incomplete); err == nil {
		_ = bundle.Close()
		t.Fatal("incomplete prompt bundle accepted")
	}

	symlinked := t.TempDir()
	writePromptBundleFixture(t, symlinked, "ORIGINAL")
	outside := filepath.Join(t.TempDir(), "pm.md")
	if err := os.WriteFile(outside, []byte("outside\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(symlinked, "pm.md")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(symlinked, "pm.md")); err != nil {
		t.Fatal(err)
	}
	if bundle, err := openPromptBundle(symlinked); err == nil {
		_ = bundle.Close()
		t.Fatal("symlinked prompt accepted into the validated bundle")
	}

	linkedRoot := filepath.Join(t.TempDir(), "prompts")
	real := t.TempDir()
	writePromptBundleFixture(t, real, "ORIGINAL")
	if err := os.Symlink(real, linkedRoot); err != nil {
		t.Fatal(err)
	}
	if bundle, err := openPromptBundle(linkedRoot); err == nil {
		_ = bundle.Close()
		t.Fatal("symlinked prompt root accepted")
	}
}

func TestOpenPromptsBundleExplicitOverrideNeverFallsBack(t *testing.T) {
	ambient := t.TempDir()
	writePromptBundleFixture(t, filepath.Join(ambient, "prompts"), "AMBIENT")
	t.Setenv("WRITE_UUTER_PROMPTS_DIR", filepath.Join(ambient, "prompts"))

	if bundle, err := openPromptsBundle("", true); err == nil {
		_ = bundle.Close()
		t.Fatal("explicit empty prompt directory fell back to the ambient bundle")
	} else if !strings.Contains(err.Error(), "explicit prompt directory is empty") {
		t.Fatalf("unexpected explicit empty error: %v", err)
	}

	missing := filepath.Join(t.TempDir(), "missing")
	if bundle, err := openPromptsBundle(missing, true); err == nil {
		_ = bundle.Close()
		t.Fatal("explicit missing prompt directory fell back to the ambient bundle")
	} else if !strings.Contains(err.Error(), "explicit prompt directory") {
		t.Fatalf("unexpected explicit missing error: %v", err)
	}

	bundle, err := openPromptsBundle("", false)
	if err != nil {
		t.Fatalf("documented WRITE_UUTER_PROMPTS_DIR precedence rejected: %v", err)
	}
	defer bundle.Close()
	content, err := bundle.load("pm.md")
	if err != nil || strings.TrimSpace(content) != "AMBIENT pm.md" {
		t.Fatalf("ambient override did not outrank the working-directory bundle: %q, %v", content, err)
	}
}
