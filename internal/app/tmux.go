package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type tmuxRuntime struct {
	executable string
	session    string
	runDir     string
	codex      string
	sequence   int
}

type invocation struct {
	ID         string
	Role       string
	Lens       string
	Candidate  int
	Revision   string
	PromptPath string
	ExitPath   string
	LogPath    string
	Window     string
}

func newTmuxRuntime(tmuxExecutable, codexExecutable, runDir string) (*tmuxRuntime, error) {
	tmuxPath, err := exec.LookPath(tmuxExecutable)
	if err != nil {
		return nil, fmt.Errorf("locate tmux executable: %w", err)
	}
	codexPath, err := exec.LookPath(codexExecutable)
	if err != nil {
		return nil, fmt.Errorf("locate Codex executable: %w", err)
	}
	absoluteRunDir, err := filepath.Abs(runDir)
	if err != nil {
		return nil, err
	}
	seed := fmt.Sprintf("%s-%d-%d", absoluteRunDir, os.Getpid(), time.Now().UnixNano())
	return &tmuxRuntime{
		executable: tmuxPath,
		session:    "write-uuter-" + revisionFor([]byte(seed))[7:19],
		runDir:     absoluteRunDir,
		codex:      codexPath,
	}, nil
}

func (runtime *tmuxRuntime) startPM(promptPath string) (invocation, error) {
	inv := runtime.makeInvocation("pm", "", 0, "", promptPath)
	command := runtime.command(inv)
	output, err := exec.Command(runtime.executable, "new-session", "-d", "-s", runtime.session, "-n", inv.Window, command).CombinedOutput()
	if err != nil {
		return invocation{}, fmt.Errorf("start PM tmux session: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return inv, nil
}

func (runtime *tmuxRuntime) startWorker(role, lens string, candidate int, revision, promptPath string) (invocation, error) {
	inv := runtime.makeInvocation(role, lens, candidate, revision, promptPath)
	command := runtime.command(inv)
	output, err := exec.Command(runtime.executable, "new-window", "-d", "-t", runtime.session, "-n", inv.Window, command).CombinedOutput()
	if err != nil {
		return invocation{}, fmt.Errorf("start %s worker: %w: %s", role, err, strings.TrimSpace(string(output)))
	}
	return inv, nil
}

func (runtime *tmuxRuntime) makeInvocation(role, lens string, candidate int, revision, promptPath string) invocation {
	runtime.sequence++
	id := fmt.Sprintf("%03d-%s", runtime.sequence, strings.ReplaceAll(role, "_", "-"))
	return invocation{
		ID:         id,
		Role:       role,
		Lens:       lens,
		Candidate:  candidate,
		Revision:   revision,
		PromptPath: promptPath,
		ExitPath:   filepath.Join(runtime.runDir, ".control", "exits", id+".exit"),
		LogPath:    filepath.Join(runtime.runDir, ".control", "logs", id+".log"),
		Window:     strings.ReplaceAll(id, "_", "-"),
	}
}

func (runtime *tmuxRuntime) command(inv invocation) string {
	values := map[string]string{
		"WRITE_UUTER_CODEX":      runtime.codex,
		"WRITE_UUTER_RUN_DIR":    runtime.runDir,
		"WRITE_UUTER_PROMPT":     inv.PromptPath,
		"WRITE_UUTER_EXIT_FILE":  inv.ExitPath,
		"WRITE_UUTER_LOG_FILE":   inv.LogPath,
		"WRITE_UUTER_ROLE":       inv.Role,
		"WRITE_UUTER_LENS":       inv.Lens,
		"WRITE_UUTER_CANDIDATE":  strconv.Itoa(inv.Candidate),
		"WRITE_UUTER_REVISION":   inv.Revision,
		"WRITE_UUTER_INVOCATION": inv.ID,
	}
	parts := []string{"env"}
	keys := []string{"WRITE_UUTER_CODEX", "WRITE_UUTER_RUN_DIR", "WRITE_UUTER_PROMPT", "WRITE_UUTER_EXIT_FILE", "WRITE_UUTER_LOG_FILE", "WRITE_UUTER_ROLE", "WRITE_UUTER_LENS", "WRITE_UUTER_CANDIDATE", "WRITE_UUTER_REVISION", "WRITE_UUTER_INVOCATION"}
	for _, key := range keys {
		parts = append(parts, shellQuote(key+"="+values[key]))
	}
	parts = append(parts, shellQuote(filepath.Join(runtime.runDir, ".control", "launch-agent.sh")))
	return strings.Join(parts, " ")
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}

func (runtime *tmuxRuntime) stopWindow(window string) {
	_ = exec.Command(runtime.executable, "kill-window", "-t", runtime.session+":"+window).Run()
}

func (runtime *tmuxRuntime) cleanup() error {
	command := exec.Command(runtime.executable, "kill-session", "-t", runtime.session)
	output, err := command.CombinedOutput()
	if err != nil && !strings.Contains(string(output), "can't find session") {
		return fmt.Errorf("kill tmux session: %w: %s", err, strings.TrimSpace(string(output)))
	}
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if !runtime.sessionExists() {
			return nil
		}
		time.Sleep(25 * time.Millisecond)
	}
	return fmt.Errorf("tmux session %s still exists after cleanup", runtime.session)
}

func (runtime *tmuxRuntime) sessionExists() bool {
	return exec.Command(runtime.executable, "has-session", "-t", runtime.session).Run() == nil
}

func (runtime *tmuxRuntime) waitForContract(ctx context.Context, pm invocation, worker invocation, validate func() error) error {
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()
	for {
		validationError := validate()
		if validationError == nil {
			runtime.stopWindow(worker.Window)
			return nil
		}
		if !errors.Is(validationError, errNotReady) {
			runtime.stopWindow(worker.Window)
			return validationError
		}
		if status, exited := exitStatus(worker.ExitPath); exited {
			return fmt.Errorf("%s worker exited with status %d before satisfying its artifact contract: %w", worker.Role, status, validationError)
		}
		if status, exited := exitStatus(pm.ExitPath); exited {
			runtime.stopWindow(worker.Window)
			return fmt.Errorf("PM exited unexpectedly with status %d", status)
		}
		select {
		case <-ctx.Done():
			runtime.stopWindow(worker.Window)
			return fmt.Errorf("%s timed out waiting for artifact contract: %w", worker.Role, ctx.Err())
		case <-ticker.C:
		}
	}
}

func exitStatus(path string) (int, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, false
	}
	status, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return -1, true
	}
	return status, true
}
