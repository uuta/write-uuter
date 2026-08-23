package app

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestPrivateRuntimeAuditBlocksRemovalWhileTrackedProcessRemains(t *testing.T) {
	runtime := &tmuxRuntime{privateRoot: t.TempDir()}
	command := exec.Command("sh", "-c", "exec -a '"+runtime.privateRoot+"' sleep 30")
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	if err := runtime.auditPrivateProcesses(); err == nil {
		t.Fatal("private runtime audit accepted a launched process")
	}
	if err := command.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	_, _ = command.Process.Wait()
	if err := runtime.auditPrivateProcesses(); err != nil {
		t.Fatalf("private runtime audit retained a dead process: %v", err)
	}
}

func TestProxyWithoutUserinfoAcceptsOnlyCredentialFreeOrigin(t *testing.T) {
	for _, value := range []string{
		"http://proxy.example:8080", "https://proxy.example",
	} {
		if !proxyWithoutUserinfo(value) {
			t.Errorf("valid proxy rejected: %q", value)
		}
	}
	for _, value := range []string{
		"http://proxy.example/path", "http://proxy.example/?secret=1",
		"http://proxy.example/#secret", "http:user:pass", "http://user:pass@proxy.example",
		"http://proxy.example/%73ecret", "http://proxy.example\nsecret",
	} {
		if proxyWithoutUserinfo(value) {
			t.Errorf("unsafe proxy accepted: %q", value)
		}
	}
}

func TestAgentEnvironmentOmitsCredentialBearingNoProxy(t *testing.T) {
	t.Setenv("NO_PROXY", "secret.invalid:token")
	for _, entry := range agentEnvironment(t.TempDir(), t.TempDir(), "role", "", 1, "rev", "inv") {
		if strings.HasPrefix(entry, "NO_PROXY=") || strings.Contains(entry, "secret.invalid") {
			t.Fatalf("credential-bearing NO_PROXY crossed agent boundary: %q", entry)
		}
	}
}

func TestPromptAndStyleReadsRejectSymlinkComponents(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "STYLE.md"), []byte("outside sentinel"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "docs")); err != nil {
		t.Fatal(err)
	}
	if _, err := findStyleGuide(root); err == nil {
		t.Fatal("style guide read followed a symlinked parent")
	}
	if _, err := loadPrompt(root, "docs/prompt.md"); err == nil {
		t.Fatal("prompt read followed a symlinked parent")
	}
	contentRoot, err := os.OpenRoot(root)
	if err != nil {
		t.Fatal(err)
	}
	defer contentRoot.Close()
	if err := validateRootParents(contentRoot, "docs/prompt.md"); err == nil {
		t.Fatal("root parent validation accepted a symlinked parent")
	}
}

func TestPMDecisionRequiresExactFenceLines(t *testing.T) {
	valid := []byte("```json\n{\"reviewed_revision\":\"r\",\"lenses\":{}}\n```\n")
	if _, err := parsePMDecisionDocument(valid); err != nil {
		t.Fatalf("valid fenced PM decision rejected: %v", err)
	}
	for _, malformed := range [][]byte{
		[]byte("```json {\"reviewed_revision\":\"r\",\"lenses\":{}} ```"),
		[]byte("```json suffix\n{}\n```"),
		[]byte("```json\n{}\n``` trailing"),
		[]byte("```json\n{}\n```\n```json\n{}\n```"),
	} {
		if _, err := parsePMDecisionDocument(malformed); err == nil {
			t.Fatalf("malformed PM fence accepted: %q", malformed)
		}
	}
}
