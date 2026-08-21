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

func resolvePromptsDir(configured string) (string, error) {
	if configured != "" {
		absolute, err := filepath.Abs(configured)
		if err != nil {
			return "", fmt.Errorf("resolve explicit prompt directory: %w", err)
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
	for _, name := range requiredPromptFiles {
		info, err := os.Stat(filepath.Join(directory, name))
		if err != nil {
			return fmt.Errorf("required prompt %s: %w", name, err)
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("required prompt %s is not a regular file", name)
		}
	}
	return nil
}

func loadPrompt(dir, name string) (string, error) {
	data, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(string(data)) == "" {
		return "", fmt.Errorf("prompt is empty: %s", name)
	}
	return string(data), nil
}

func contextBlock(label string, content []byte) string {
	return fmt.Sprintf("\n\n## Provided context: %s\n\n<write-uuter-context name=%q>\n%s\n</write-uuter-context>", label, label, content)
}
