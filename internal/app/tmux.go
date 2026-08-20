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
	executable     string
	session        string
	codex          string
	controlDir     string
	controlStore   *artifactStore
	commandTimeout time.Duration
	sequence       int
	cleaned        bool
	invocations    []invocation
}

type invocation struct {
	ID             string
	Role           string
	Lens           string
	Candidate      int
	Revision       string
	PromptRelative string
	ExitRelative   string
	LogRelative    string
	WorkspacePath  string
	Window         string
	archived       bool
}

func newTmuxRuntime(tmuxExecutable, codexExecutable string, agentTimeout time.Duration) (*tmuxRuntime, error) {
	tmuxPath, err := exec.LookPath(tmuxExecutable)
	if err != nil {
		return nil, fmt.Errorf("locate tmux executable: %w", err)
	}
	codexPath, err := exec.LookPath(codexExecutable)
	if err != nil {
		return nil, fmt.Errorf("locate Codex executable: %w", err)
	}
	controlDir, err := os.MkdirTemp("", "write-uuter-controller-*")
	if err != nil {
		return nil, fmt.Errorf("create controller-private runtime: %w", err)
	}
	if err := os.Chmod(controlDir, 0o700); err != nil {
		_ = os.RemoveAll(controlDir)
		return nil, err
	}
	controlStore, err := openArtifactStore(controlDir)
	if err != nil {
		_ = os.RemoveAll(controlDir)
		return nil, err
	}
	for _, directory := range []string{"prompts", "logs", "exits"} {
		if err := controlStore.mkdirAll(directory, 0o700); err != nil {
			controlStore.Close()
			_ = os.RemoveAll(controlDir)
			return nil, err
		}
	}
	launcher := `#!/bin/sh
set +e
"$WRITE_UUTER_CODEX" -s workspace-write -a never -C "$WRITE_UUTER_WORK_DIR" exec --ephemeral --skip-git-repo-check - < "$WRITE_UUTER_PROMPT" > "$WRITE_UUTER_LOG_FILE" 2>&1
status=$?
printf '%s\n' "$status" > "$WRITE_UUTER_EXIT_FILE"
exit "$status"
`
	if err := controlStore.writeAtomic("launch-agent.sh", []byte(launcher), 0o500); err != nil {
		controlStore.Close()
		_ = os.RemoveAll(controlDir)
		return nil, err
	}
	commandTimeout := agentTimeout
	if commandTimeout > 3*time.Second {
		commandTimeout = 3 * time.Second
	}
	if commandTimeout < 100*time.Millisecond {
		commandTimeout = 100 * time.Millisecond
	}
	seed := fmt.Sprintf("%s-%d-%d", controlDir, os.Getpid(), time.Now().UnixNano())
	return &tmuxRuntime{
		executable: tmuxPath, codex: codexPath, controlDir: controlDir,
		controlStore: controlStore, commandTimeout: commandTimeout,
		session: "write-uuter-" + revisionFor([]byte(seed))[7:19],
	}, nil
}

func (runtime *tmuxRuntime) prepareInvocation(role, lens string, candidate int, revision, prompt string) (invocation, error) {
	runtime.sequence++
	id := fmt.Sprintf("%03d-%s", runtime.sequence, strings.ReplaceAll(role, "_", "-"))
	workspace, err := os.MkdirTemp("", "write-uuter-agent-"+id+"-*")
	if err != nil {
		return invocation{}, err
	}
	if err := os.Chmod(workspace, 0o700); err != nil {
		_ = os.RemoveAll(workspace)
		return invocation{}, err
	}
	promptRelative := filepath.Join("prompts", id+".md")
	if err := runtime.controlStore.writeAtomic(promptRelative, []byte(prompt), 0o400); err != nil {
		_ = os.RemoveAll(workspace)
		return invocation{}, err
	}
	inv := invocation{
		ID: id, Role: role, Lens: lens, Candidate: candidate, Revision: revision,
		PromptRelative: promptRelative,
		ExitRelative:   filepath.Join("exits", id+".exit"),
		LogRelative:    filepath.Join("logs", id+".log"),
		WorkspacePath:  workspace,
		Window:         id,
	}
	runtime.invocations = append(runtime.invocations, inv)
	return inv, nil
}

func (runtime *tmuxRuntime) startPM(inv invocation) error {
	output, err := runtime.runCommand("new-session", "-d", "-s", runtime.session, "-n", inv.Window, runtime.command(inv))
	if err != nil {
		return fmt.Errorf("start PM tmux session: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

func (runtime *tmuxRuntime) startWorker(inv invocation) error {
	output, err := runtime.runCommand("new-window", "-d", "-t", runtime.session, "-n", inv.Window, runtime.command(inv))
	if err != nil {
		return fmt.Errorf("start %s worker: %w: %s", inv.Role, err, strings.TrimSpace(string(output)))
	}
	return nil
}

func (runtime *tmuxRuntime) command(inv invocation) string {
	values := map[string]string{
		"WRITE_UUTER_CODEX":      runtime.codex,
		"WRITE_UUTER_WORK_DIR":   inv.WorkspacePath,
		"WRITE_UUTER_PROMPT":     filepath.Join(runtime.controlDir, inv.PromptRelative),
		"WRITE_UUTER_EXIT_FILE":  filepath.Join(runtime.controlDir, inv.ExitRelative),
		"WRITE_UUTER_LOG_FILE":   filepath.Join(runtime.controlDir, inv.LogRelative),
		"WRITE_UUTER_ROLE":       inv.Role,
		"WRITE_UUTER_LENS":       inv.Lens,
		"WRITE_UUTER_CANDIDATE":  strconv.Itoa(inv.Candidate),
		"WRITE_UUTER_REVISION":   inv.Revision,
		"WRITE_UUTER_INVOCATION": inv.ID,
	}
	parts := []string{"env"}
	keys := []string{"WRITE_UUTER_CODEX", "WRITE_UUTER_WORK_DIR", "WRITE_UUTER_PROMPT", "WRITE_UUTER_EXIT_FILE", "WRITE_UUTER_LOG_FILE", "WRITE_UUTER_ROLE", "WRITE_UUTER_LENS", "WRITE_UUTER_CANDIDATE", "WRITE_UUTER_REVISION", "WRITE_UUTER_INVOCATION"}
	for _, key := range keys {
		parts = append(parts, shellQuote(key+"="+values[key]))
	}
	parts = append(parts, shellQuote(filepath.Join(runtime.controlDir, "launch-agent.sh")))
	return strings.Join(parts, " ")
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}

func (runtime *tmuxRuntime) waitForWorker(ctx context.Context, pm, worker invocation) error {
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()
	for {
		if status, exited := runtime.exitStatus(worker); exited {
			if status != 0 {
				return fmt.Errorf("%s worker exited with status %d", worker.Role, status)
			}
			if err := runtime.waitWindowAbsent(ctx, worker.Window); err != nil {
				return fmt.Errorf("verify %s worker termination: %w", worker.Role, err)
			}
			return nil
		}
		if status, exited := runtime.exitStatus(pm); exited {
			stopErr := runtime.stopWindow(worker.Window)
			if stopErr != nil {
				return fmt.Errorf("PM exited unexpectedly with status %d; worker cleanup failed: %v", status, stopErr)
			}
			return fmt.Errorf("PM exited unexpectedly with status %d", status)
		}
		select {
		case <-ctx.Done():
			if err := runtime.stopWindow(worker.Window); err != nil {
				return fmt.Errorf("%s timed out and termination failed: %v", worker.Role, err)
			}
			return fmt.Errorf("%s timed out waiting for process completion: %w", worker.Role, ctx.Err())
		case <-ticker.C:
		}
	}
}

func (runtime *tmuxRuntime) stopWindow(window string) error {
	exists, err := runtime.windowExists(window)
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}
	output, killErr := runtime.runCommand("kill-window", "-t", runtime.session+":"+window)
	if killErr != nil {
		existsAfter, checkErr := runtime.windowExists(window)
		if checkErr == nil && !existsAfter {
			return nil
		}
		return fmt.Errorf("kill tmux window: %w: %s", killErr, strings.TrimSpace(string(output)))
	}
	ctx, cancel := context.WithTimeout(context.Background(), runtime.commandTimeout)
	defer cancel()
	return runtime.waitWindowAbsent(ctx, window)
}

func (runtime *tmuxRuntime) cleanup() error {
	if runtime == nil || runtime.cleaned {
		return nil
	}
	exists, err := runtime.sessionExists()
	if err != nil {
		return err
	}
	if exists {
		output, killErr := runtime.runCommand("kill-session", "-t", runtime.session)
		if killErr != nil {
			existsAfter, checkErr := runtime.sessionExists()
			if checkErr != nil || existsAfter {
				return fmt.Errorf("kill tmux session: %w: %s", killErr, strings.TrimSpace(string(output)))
			}
		}
	}
	deadline := time.Now().Add(runtime.commandTimeout)
	for time.Now().Before(deadline) {
		exists, checkErr := runtime.sessionExists()
		if checkErr != nil {
			return checkErr
		}
		if !exists {
			runtime.cleaned = true
			return nil
		}
		time.Sleep(25 * time.Millisecond)
	}
	return fmt.Errorf("tmux session %s still exists after cleanup", runtime.session)
}

func (runtime *tmuxRuntime) sessionExists() (bool, error) {
	_, err := runtime.runCommand("has-session", "-t", runtime.session)
	if err == nil {
		return true, nil
	}
	var exitError *exec.ExitError
	if errors.As(err, &exitError) {
		return false, nil
	}
	return false, err
}

func (runtime *tmuxRuntime) windowExists(window string) (bool, error) {
	exists, err := runtime.sessionExists()
	if err != nil || !exists {
		return false, err
	}
	output, err := runtime.runCommand("list-windows", "-t", runtime.session, "-F", "#{window_name}")
	if err != nil {
		return false, err
	}
	for _, name := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		if name == window {
			return true, nil
		}
	}
	return false, nil
}

func (runtime *tmuxRuntime) waitWindowAbsent(ctx context.Context, window string) error {
	ticker := time.NewTicker(25 * time.Millisecond)
	defer ticker.Stop()
	for {
		exists, err := runtime.windowExists(window)
		if err != nil {
			return err
		}
		if !exists {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func (runtime *tmuxRuntime) exitStatus(inv invocation) (int, bool) {
	data, err := runtime.controlStore.readRegular(inv.ExitRelative)
	if err != nil {
		return 0, false
	}
	status, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return -1, true
	}
	return status, true
}

func (runtime *tmuxRuntime) runCommand(arguments ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), runtime.commandTimeout)
	defer cancel()
	output, err := exec.CommandContext(ctx, runtime.executable, arguments...).CombinedOutput()
	if ctx.Err() != nil {
		return output, fmt.Errorf("tmux command timed out: %w", ctx.Err())
	}
	return output, err
}

func (runtime *tmuxRuntime) workspaceStore(inv invocation) (*artifactStore, error) {
	return openArtifactStore(inv.WorkspacePath)
}

func (runtime *tmuxRuntime) archive(inv *invocation, destination *artifactStore) error {
	if inv.archived {
		return nil
	}
	for _, item := range []struct {
		source string
		target string
		mode   os.FileMode
	}{
		{inv.PromptRelative, filepath.Join(".control", "prompts", filepath.Base(inv.PromptRelative)), 0o600},
		{inv.LogRelative, filepath.Join(".control", "logs", filepath.Base(inv.LogRelative)), 0o600},
		{inv.ExitRelative, filepath.Join(".control", "exits", filepath.Base(inv.ExitRelative)), 0o600},
	} {
		if _, err := destination.copyRegularFrom(runtime.controlStore, item.source, item.target, item.mode); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	inv.archived = true
	return nil
}

func (runtime *tmuxRuntime) archiveAll(destination *artifactStore) error {
	for index := range runtime.invocations {
		if err := runtime.archive(&runtime.invocations[index], destination); err != nil {
			return err
		}
	}
	return nil
}

func (runtime *tmuxRuntime) closePrivate() error {
	for _, inv := range runtime.invocations {
		if err := os.RemoveAll(inv.WorkspacePath); err != nil {
			return err
		}
	}
	if err := runtime.controlStore.Close(); err != nil {
		return err
	}
	return os.RemoveAll(runtime.controlDir)
}
