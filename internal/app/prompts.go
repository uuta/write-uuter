package app

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// modelPolicyFile is the required per-bundle model policy. It is bound through
// the same stable no-follow boundary as the role prompts, so an explicit
// --prompts-dir bundle and the checked-in bundle are each a complete policy.
const modelPolicyFile = "models.json"

// modelPolicyArtifact is the durable copy of the validated policy inside a run.
const modelPolicyArtifact = "model-policy.json"

var requiredPromptFiles = []string{
	modelPolicyFile,
	"pm.md",
	"pm-runtime.md",
	"researcher.md",
	"story-editor.md",
	"writer.md",
	"reviewer-output.md",
	"reviewer-evidence.md",
	"reviewer-story.md",
	"reviewer-clarity.md",
	"reviewer-copy.md",
}

// promptBundle binds prompt content to the exact directory and file identities
// observed during validation. The validated root is held as an open directory
// handle and every required prompt is held as an open no-follow descriptor, so
// replacing the root, an ancestor of the root, or a final prompt component
// after validation cannot change what the controller later reads.
type promptBundle struct {
	directory string
	root      *os.Root
	files     map[string]*os.File
}

// openPromptsBundle resolves and binds the prompt bundle. An explicit
// directory is binding; otherwise the ambient precedence documented in
// README.md applies. The returned bundle owns open descriptors and must be
// closed by the caller.
func openPromptsBundle(configured string, explicit bool) (*promptBundle, error) {
	if explicit {
		if strings.TrimSpace(configured) == "" {
			return nil, fmt.Errorf("explicit prompt directory is empty")
		}
		absolute, err := filepath.Abs(configured)
		if err != nil {
			return nil, fmt.Errorf("resolve explicit prompt directory: %w", err)
		}
		bundle, err := openPromptBundle(absolute)
		if err != nil {
			return nil, fmt.Errorf("explicit prompt directory %s: %w", absolute, err)
		}
		return bundle, nil
	}
	// Ambient precedence, highest first. Keep this list and README.md in step.
	var candidates []string
	if fromEnvironment := os.Getenv("WRITE_UUTER_PROMPTS_DIR"); fromEnvironment != "" {
		candidates = append(candidates, fromEnvironment)
	}
	if workingDirectory, err := os.Getwd(); err == nil {
		candidates = append(candidates, filepath.Join(workingDirectory, "prompts"))
	}
	if executable, err := os.Executable(); err == nil {
		candidates = append(candidates,
			filepath.Join(filepath.Dir(executable), "prompts"),
			filepath.Join(filepath.Dir(executable), "..", "prompts"),
			filepath.Join(filepath.Dir(executable), "..", "share", "write-uuter", "prompts"),
		)
	}
	for _, candidate := range candidates {
		absolute, err := filepath.Abs(candidate)
		if err != nil {
			continue
		}
		if bundle, err := openPromptBundle(absolute); err == nil {
			return bundle, nil
		}
	}
	return nil, fmt.Errorf("prompt directory not found or incomplete; use --prompts-dir")
}

func openPromptBundle(directory string) (*promptBundle, error) {
	expected, err := os.Lstat(directory)
	if err != nil {
		return nil, err
	}
	if expected.Mode()&os.ModeSymlink != 0 || !expected.IsDir() {
		return nil, fmt.Errorf("prompt directory is not a real directory: %s", directory)
	}
	root, err := os.OpenRoot(directory)
	if err != nil {
		return nil, err
	}
	bundle := &promptBundle{directory: directory, root: root, files: make(map[string]*os.File, len(requiredPromptFiles))}
	// The validated directory and the opened root must be the same object, so
	// a swap between the check and the open cannot retarget the bundle.
	opened, err := root.Stat(".")
	if err == nil && !os.SameFile(expected, opened) {
		err = fmt.Errorf("prompt directory changed while it was opened: %s", directory)
	}
	if err != nil {
		_ = bundle.Close()
		return nil, err
	}
	for _, name := range requiredPromptFiles {
		file, openErr := bundle.openValidated(name)
		if openErr != nil {
			_ = bundle.Close()
			return nil, fmt.Errorf("required prompt %s: %w", name, openErr)
		}
		bundle.files[name] = file
	}
	return bundle, nil
}

func (bundle *promptBundle) openValidated(name string) (*os.File, error) {
	if err := validateRootParents(bundle.root, name); err != nil {
		return nil, err
	}
	expected, err := bundle.root.Lstat(name)
	if err != nil {
		return nil, err
	}
	if expected.Mode()&os.ModeSymlink != 0 || !expected.Mode().IsRegular() {
		return nil, fmt.Errorf("prompt is not a regular no-follow file")
	}
	file, err := bundle.root.OpenFile(name, os.O_RDONLY, 0)
	if err != nil {
		return nil, err
	}
	opened, err := file.Stat()
	if err == nil && !os.SameFile(expected, opened) {
		err = fmt.Errorf("prompt identity changed while it was opened")
	}
	if err != nil {
		_ = file.Close()
		return nil, err
	}
	return file, nil
}

// load reads a bound prompt from its held descriptor. No path is resolved
// again, so the content is the validated content even under a mutation race.
func (bundle *promptBundle) load(name string) (string, error) {
	if bundle == nil || bundle.files == nil {
		return "", fmt.Errorf("prompt bundle is not open")
	}
	file, found := bundle.files[name]
	if !found {
		return "", fmt.Errorf("prompt is not part of the validated bundle: %s", name)
	}
	info, err := file.Stat()
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("prompt is no longer a regular file: %s", name)
	}
	data, err := io.ReadAll(io.NewSectionReader(file, 0, info.Size()))
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(string(data)) == "" {
		return "", fmt.Errorf("prompt is empty: %s", name)
	}
	return string(data), nil
}

func (bundle *promptBundle) Close() error {
	if bundle == nil {
		return nil
	}
	var failures []error
	for _, file := range bundle.files {
		if err := file.Close(); err != nil {
			failures = append(failures, err)
		}
	}
	bundle.files = nil
	if bundle.root != nil {
		if err := bundle.root.Close(); err != nil {
			failures = append(failures, err)
		}
		bundle.root = nil
	}
	return errors.Join(failures...)
}

// loadPrompt reads a single no-follow regular file below dir. It serves
// content-root reads, which are resolved per use rather than bound like the
// prompt bundle.
func loadPrompt(dir, name string) (string, error) {
	root, err := os.OpenRoot(dir)
	if err != nil {
		return "", err
	}
	defer root.Close()
	if err := validateRootParents(root, name); err != nil {
		return "", err
	}
	info, err := root.Lstat(name)
	if err != nil {
		return "", err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return "", fmt.Errorf("prompt is not a regular no-follow file: %s", name)
	}
	data, err := root.ReadFile(name)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(string(data)) == "" {
		return "", fmt.Errorf("prompt is empty: %s", name)
	}
	return string(data), nil
}

func validateRootParents(root *os.Root, name string) error {
	clean := filepath.Clean(name)
	if filepath.IsAbs(name) || clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return fmt.Errorf("path escapes root: %s", name)
	}
	parent := filepath.Dir(clean)
	if parent == "." {
		return nil
	}
	current := ""
	for _, component := range strings.Split(parent, string(filepath.Separator)) {
		if current == "" {
			current = component
		} else {
			current = filepath.Join(current, component)
		}
		info, err := root.Lstat(current)
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return fmt.Errorf("path component is not a directory: %s", current)
		}
	}
	return nil
}

func contextBlock(label string, content []byte) string {
	return fmt.Sprintf("\n\n## Provided context: %s\n\n<write-uuter-context name=%q>\n%s\n</write-uuter-context>", label, label, content)
}
