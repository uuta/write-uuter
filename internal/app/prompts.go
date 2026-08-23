package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var requiredPromptFiles = []string{
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

func resolvePromptsDir(configured string, explicit bool) (string, error) {
	if explicit {
		if configured == "" {
			return "", fmt.Errorf("explicit prompt directory is empty")
		}
		absolute, err := filepath.Abs(configured)
		if err != nil {
			return "", fmt.Errorf("resolve explicit prompt directory: %w", err)
		}
		if info, statErr := os.Lstat(absolute); statErr != nil {
			return "", fmt.Errorf("inspect explicit prompt directory: %w", statErr)
		} else if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return "", fmt.Errorf("explicit prompt directory is not a real directory: %s", absolute)
		}
		if err := validatePromptsDir(absolute); err != nil {
			return "", fmt.Errorf("explicit prompt directory %s: %w", absolute, err)
		}
		return absolute, nil
	}
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
		if validatePromptsDir(absolute) == nil {
			return absolute, nil
		}
	}
	return "", fmt.Errorf("prompt directory not found or incomplete; use --prompts-dir")
}

func validatePromptsDir(directory string) error {
	root, err := os.OpenRoot(directory)
	if err != nil {
		return err
	}
	defer root.Close()
	for _, name := range requiredPromptFiles {
		if err := validateRootParents(root, name); err != nil {
			return fmt.Errorf("required prompt %s: %w", name, err)
		}
		info, err := root.Lstat(name)
		if err != nil {
			return fmt.Errorf("required prompt %s: %w", name, err)
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
			return fmt.Errorf("required prompt %s is not a regular file", name)
		}
	}
	return nil
}

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
