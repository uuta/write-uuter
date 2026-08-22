package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type tmuxRuntime struct {
	executable      string
	runner          string
	session         string
	codex           string
	codexRuntime    []string
	runDir          string
	privateRoot     string
	controlDir      string
	workspacesDir   string
	codexHomesDir   string
	sourceCodexHome string
	controlStore    *artifactStore
	fakeLogDir      string
	detachedPIDDir  string
	exitDelay       string
	readyDelay      string
	removeAudit     string
	auditRemoved    bool
	failRemoveOnce  bool
	failCleanup     bool
	testScenario    []byte
	commandTimeout  time.Duration
	sequence        int
	cleaned         bool
	closed          bool
	storeClosed     bool
	invocations     []invocation
}

type invocation struct {
	ID                string
	Role              string
	Lens              string
	Candidate         int
	Revision          string
	PromptRelative    string
	ExitRelative      string
	LogRelative       string
	ReadyRelative     string
	OwnershipRelative string
	ProfileRelative   string
	WorkspacePath     string
	CodexHomePath     string
	ClientPath        string
	Window            string
	launchAttempted   bool
	started           bool
	retired           bool
	archived          bool
}

func newTmuxRuntime(tmuxExecutable, codexExecutable string, agentTimeout time.Duration, runDir string) (*tmuxRuntime, error) {
	tmuxPath, err := exec.LookPath(tmuxExecutable)
	if err != nil {
		return nil, fmt.Errorf("locate tmux executable: %w", err)
	}
	codexPath, err := exec.LookPath(codexExecutable)
	if err != nil {
		return nil, fmt.Errorf("locate Codex executable: %w", err)
	}
	codexPath, err = filepath.EvalSymlinks(codexPath)
	if err != nil {
		return nil, fmt.Errorf("resolve Codex executable: %w", err)
	}
	runnerSource, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("locate write-uuter agent runner: %w", err)
	}
	runnerSource, err = filepath.EvalSymlinks(runnerSource)
	if err != nil {
		return nil, fmt.Errorf("resolve write-uuter agent runner: %w", err)
	}
	privateRoot, err := os.MkdirTemp(filepath.Dir(runDir), ".write-uuter-private-*")
	if err != nil {
		return nil, fmt.Errorf("create controller-private runtime: %w", err)
	}
	cleanupRoot := func() { _ = os.RemoveAll(privateRoot) }
	if err := os.Chmod(privateRoot, 0o700); err != nil {
		cleanupRoot()
		return nil, err
	}
	controlDir := filepath.Join(privateRoot, "control")
	workspacesDir := filepath.Join(privateRoot, "workspaces")
	codexHomesDir := filepath.Join(privateRoot, "codex-homes")
	for _, directory := range []string{controlDir, workspacesDir, codexHomesDir} {
		if err := os.Mkdir(directory, 0o700); err != nil {
			cleanupRoot()
			return nil, err
		}
	}
	runnerPath := filepath.Join(controlDir, "agent-runner")
	if err := installPrivateRunner(runnerSource, runnerPath); err != nil {
		cleanupRoot()
		return nil, fmt.Errorf("install controller-private agent runner: %w", err)
	}
	var testScenario []byte
	if os.Getenv("WRITE_UUTER_FAKE_LOG_DIR") != "" {
		testScenario, _ = os.ReadFile(filepath.Join(filepath.Dir(codexPath), "scenario"))
	}
	stagedCodexPath := filepath.Join(controlDir, "codex")
	if err := installPrivateRunner(codexPath, stagedCodexPath); err != nil {
		cleanupRoot()
		return nil, fmt.Errorf("stage Codex executable: %w", err)
	}
	codexRuntime := []string{stagedCodexPath}
	codeModeSource := filepath.Join(filepath.Dir(codexPath), "codex-code-mode-host")
	if info, statErr := os.Lstat(codeModeSource); statErr == nil {
		if !info.Mode().IsRegular() {
			cleanupRoot()
			return nil, fmt.Errorf("Codex code-mode host is not a regular file: %s", codeModeSource)
		}
		codeModeTarget := filepath.Join(controlDir, "codex-code-mode-host")
		if err := installPrivateRunner(codeModeSource, codeModeTarget); err != nil {
			cleanupRoot()
			return nil, fmt.Errorf("stage Codex code-mode host: %w", err)
		}
		codexRuntime = append(codexRuntime, codeModeTarget)
	} else if !errors.Is(statErr, os.ErrNotExist) {
		cleanupRoot()
		return nil, fmt.Errorf("inspect Codex code-mode host: %w", statErr)
	}
	controlStore, err := openArtifactStore(controlDir)
	if err != nil {
		cleanupRoot()
		return nil, err
	}
	for _, directory := range []string{"clients", "prompts", "logs", "exits", "ready", "ownership", "profiles"} {
		if err := controlStore.mkdirAll(directory, 0o700); err != nil {
			_ = controlStore.Close()
			cleanupRoot()
			return nil, err
		}
	}
	commandTimeout := agentTimeout
	if commandTimeout > 3*time.Second {
		commandTimeout = 3 * time.Second
	}
	if commandTimeout < 100*time.Millisecond {
		commandTimeout = 100 * time.Millisecond
	}
	seed := fmt.Sprintf("%s-%d-%d", privateRoot, os.Getpid(), time.Now().UnixNano())
	sourceCodexHome := os.Getenv("CODEX_HOME")
	if sourceCodexHome == "" {
		home, homeErr := os.UserHomeDir()
		if homeErr != nil {
			_ = controlStore.Close()
			cleanupRoot()
			return nil, homeErr
		}
		sourceCodexHome = filepath.Join(home, ".codex")
	}
	return &tmuxRuntime{
		executable: tmuxPath, runner: runnerPath, codex: stagedCodexPath, codexRuntime: codexRuntime, runDir: runDir,
		privateRoot: privateRoot, controlDir: controlDir, workspacesDir: workspacesDir,
		codexHomesDir: codexHomesDir, sourceCodexHome: sourceCodexHome,
		controlStore: controlStore, commandTimeout: commandTimeout,
		fakeLogDir: os.Getenv("WRITE_UUTER_FAKE_LOG_DIR"), detachedPIDDir: os.Getenv("WRITE_UUTER_TEST_DETACHED_PID_DIR"),
		exitDelay: os.Getenv("WRITE_UUTER_TEST_EXIT_MARKER_DELAY"), readyDelay: os.Getenv("WRITE_UUTER_TEST_READY_MARKER_DELAY"),
		removeAudit: os.Getenv("WRITE_UUTER_TEST_REMOVE_AUDIT"), failRemoveOnce: os.Getenv("WRITE_UUTER_TEST_FAIL_PRIVATE_REMOVE_ONCE") == "1",
		failCleanup:  os.Getenv("WRITE_UUTER_TEST_FAIL_CLEANUP_PERSISTENT") == "1",
		testScenario: testScenario,
		session:      "write-uuter-" + revisionFor([]byte(seed))[7:19],
	}, nil
}

func installPrivateRunner(source, target string) error {
	info, err := os.Lstat(source)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("runner source is not a regular file: %s", source)
	}
	input, err := os.Open(source)
	if err != nil {
		return err
	}
	defer input.Close()
	output, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o500)
	if err != nil {
		return err
	}
	keep := false
	defer func() {
		_ = output.Close()
		if !keep {
			_ = os.Remove(target)
		}
	}()
	if _, err := io.Copy(output, input); err != nil {
		return err
	}
	if err := output.Sync(); err != nil {
		return err
	}
	if err := output.Close(); err != nil {
		return err
	}
	keep = true
	return nil
}

func (runtime *tmuxRuntime) prepareInvocation(role, lens string, candidate int, revision, prompt string) (invocation, error) {
	runtime.sequence++
	id := fmt.Sprintf("%03d-%s", runtime.sequence, strings.ReplaceAll(role, "_", "-"))
	workspace, err := os.MkdirTemp(runtime.workspacesDir, id+"-*")
	if err != nil {
		return invocation{}, err
	}
	if err := os.Chmod(workspace, 0o700); err != nil {
		_ = os.RemoveAll(workspace)
		return invocation{}, err
	}
	if len(runtime.testScenario) != 0 {
		if err := os.WriteFile(filepath.Join(workspace, ".write-uuter-test-scenario"), runtime.testScenario, 0o600); err != nil {
			_ = os.RemoveAll(workspace)
			return invocation{}, err
		}
	}
	codexHome := filepath.Join(runtime.codexHomesDir, id)
	if err := os.Mkdir(codexHome, 0o700); err != nil {
		_ = os.RemoveAll(workspace)
		return invocation{}, err
	}
	for _, credential := range []string{"auth.json", "installation_id"} {
		if err := copyOptionalCredential(filepath.Join(runtime.sourceCodexHome, credential), filepath.Join(codexHome, credential)); err != nil {
			_ = os.RemoveAll(workspace)
			_ = os.RemoveAll(codexHome)
			return invocation{}, fmt.Errorf("stage Codex credential %s: %w", credential, err)
		}
	}
	clientDir := filepath.Join(runtime.controlDir, "clients", id)
	if err := os.Mkdir(clientDir, 0o700); err != nil {
		_ = os.RemoveAll(workspace)
		_ = os.RemoveAll(codexHome)
		return invocation{}, fmt.Errorf("create single-use Codex client directory: %w", err)
	}
	clientPath := filepath.Join(clientDir, "codex")
	if err := installPrivateRunner(runtime.codex, clientPath); err != nil {
		_ = os.RemoveAll(workspace)
		_ = os.RemoveAll(codexHome)
		_ = os.RemoveAll(clientDir)
		return invocation{}, fmt.Errorf("stage single-use Codex client: %w", err)
	}
	runtimeExecutables := []string{clientPath}
	for _, runtimeSource := range runtime.codexRuntime[1:] {
		runtimeTarget := filepath.Join(clientDir, filepath.Base(runtimeSource))
		if err := installPrivateRunner(runtimeSource, runtimeTarget); err != nil {
			_ = os.RemoveAll(workspace)
			_ = os.RemoveAll(codexHome)
			_ = os.RemoveAll(clientDir)
			return invocation{}, fmt.Errorf("stage single-use Codex runtime %s: %w", filepath.Base(runtimeSource), err)
		}
		runtimeExecutables = append(runtimeExecutables, runtimeTarget)
	}
	for index, executable := range runtimeExecutables {
		canonical, canonicalErr := filepath.EvalSymlinks(executable)
		if canonicalErr != nil {
			_ = os.RemoveAll(workspace)
			_ = os.RemoveAll(codexHome)
			_ = os.RemoveAll(clientDir)
			return invocation{}, fmt.Errorf("canonicalize single-use Codex runtime: %w", canonicalErr)
		}
		runtimeExecutables[index] = canonical
	}
	clientPath = runtimeExecutables[0]
	keepClient := false
	defer func() {
		if !keepClient {
			_ = os.RemoveAll(clientDir)
		}
	}()
	promptRelative := filepath.Join("prompts", id+".md")
	if err := runtime.controlStore.writeAtomic(promptRelative, []byte(prompt), 0o400); err != nil {
		_ = os.RemoveAll(workspace)
		return invocation{}, err
	}
	profile, err := isolationProfile(workspace, codexHome, runtimeExecutables)
	if err != nil {
		_ = os.RemoveAll(workspace)
		return invocation{}, err
	}
	profileRelative := filepath.Join("profiles", id+".sb")
	if err := runtime.controlStore.writeAtomic(profileRelative, []byte(profile), 0o400); err != nil {
		_ = os.RemoveAll(workspace)
		return invocation{}, err
	}
	inv := invocation{
		ID: id, Role: role, Lens: lens, Candidate: candidate, Revision: revision,
		PromptRelative: promptRelative, ExitRelative: filepath.Join("exits", id+".exit"),
		LogRelative: filepath.Join("logs", id+".log"), ReadyRelative: filepath.Join("ready", id+".ready"),
		OwnershipRelative: filepath.Join("ownership", id+".json"),
		ProfileRelative:   profileRelative, WorkspacePath: workspace, CodexHomePath: codexHome,
		ClientPath: clientPath, Window: id,
	}
	runtime.invocations = append(runtime.invocations, inv)
	keepClient = true
	return inv, nil
}

func (runtime *tmuxRuntime) startPM(inv invocation, deadlineUnixNano int64) error {
	runtime.markLaunchAttempted(inv.ID)
	output, err := runtime.runCommand("new-session", "-d", "-s", runtime.session, "-n", inv.Window, runtime.command(inv))
	if err != nil {
		return errors.Join(fmt.Errorf("start PM tmux session: %w: %s", err, strings.TrimSpace(string(output))), runtime.stopInvocation(inv))
	}
	runtime.markStarted(inv.ID)
	if err := runtime.waitInvocationReady(inv, deadlineUnixNano, true); err != nil {
		return errors.Join(err, runtime.stopInvocation(inv))
	}
	if err := runtime.waitPMProtocolReady(inv, deadlineUnixNano); err != nil {
		return errors.Join(err, runtime.stopInvocation(inv))
	}
	return nil
}

func (runtime *tmuxRuntime) startWorker(pm, inv invocation, deadlineUnixNano int64) error {
	live, err := runtime.invocationLive(pm)
	if err != nil {
		return fmt.Errorf("verify PM before %s launch: %w", inv.Role, err)
	}
	if !live {
		return fmt.Errorf("PM exited before %s launch", inv.Role)
	}
	runtime.markLaunchAttempted(inv.ID)
	output, err := runtime.runCommand("new-window", "-d", "-t", runtime.session, "-n", inv.Window, runtime.command(inv))
	if err != nil {
		return errors.Join(fmt.Errorf("start %s worker: %w: %s", inv.Role, err, strings.TrimSpace(string(output))), runtime.stopInvocation(inv))
	}
	runtime.markStarted(inv.ID)
	if err := runtime.waitInvocationReady(inv, deadlineUnixNano, false); err != nil {
		return errors.Join(err, runtime.stopInvocation(inv))
	}
	live, err = runtime.invocationLive(pm)
	if err != nil {
		return errors.Join(fmt.Errorf("verify PM after %s launch: %w", inv.Role, err), runtime.stopInvocation(inv))
	}
	if !live {
		return errors.Join(fmt.Errorf("PM exited during %s launch", inv.Role), runtime.stopInvocation(inv))
	}
	return nil
}

func (runtime *tmuxRuntime) command(inv invocation) string {
	readyDelay := runtime.readyDelay
	if inv.Role == "pm" {
		readyDelay = ""
	}
	arguments := []string{
		runtime.runner, "__agent", "--codex", inv.ClientPath, "--workspace", inv.WorkspacePath,
		"--prompt", filepath.Join(runtime.controlDir, inv.PromptRelative), "--log", filepath.Join(runtime.controlDir, inv.LogRelative),
		"--exit", filepath.Join(runtime.controlDir, inv.ExitRelative), "--ready", filepath.Join(runtime.controlDir, inv.ReadyRelative),
		"--ownership", filepath.Join(runtime.controlDir, inv.OwnershipRelative),
		"--profile", filepath.Join(runtime.controlDir, inv.ProfileRelative), "--role", inv.Role, "--lens", inv.Lens,
		"--candidate", strconv.Itoa(inv.Candidate), "--revision", inv.Revision, "--invocation", inv.ID,
		"--codex-home", inv.CodexHomePath,
		"--exit-marker-delay", runtime.exitDelay, "--ready-marker-delay", readyDelay,
	}
	parts := []string{"exec"}
	for _, argument := range arguments {
		parts = append(parts, shellQuote(argument))
	}
	return strings.Join(parts, " ")
}

func (runtime *tmuxRuntime) markLaunchAttempted(id string) {
	if record := runtime.invocationRecord(id); record != nil {
		record.launchAttempted = true
		record.started = true // Conservative: tmux may apply a request before its client times out.
	}
}

func (runtime *tmuxRuntime) markStarted(id string) {
	if record := runtime.invocationRecord(id); record != nil {
		record.started = true
	}
}

func (runtime *tmuxRuntime) invocationRecord(id string) *invocation {
	for index := range runtime.invocations {
		if runtime.invocations[index].ID == id {
			return &runtime.invocations[index]
		}
	}
	return nil
}

func (runtime *tmuxRuntime) waitInvocationReady(inv invocation, deadlineUnixNano int64, requireLive bool) error {
	ticker := time.NewTicker(20 * time.Millisecond)
	defer ticker.Stop()
	for time.Now().UnixNano() < deadlineUnixNano {
		if _, ready, err := runtime.readyStatus(inv); err != nil {
			return fmt.Errorf("read %s launch marker: %w", inv.Role, err)
		} else if ready {
			if !requireLive {
				return nil
			}
			live, err := runtime.invocationLive(inv)
			if err != nil {
				return err
			}
			if !live {
				return fmt.Errorf("%s exited before its ready handshake was accepted", inv.Role)
			}
			return nil
		}
		if status, exited, err := runtime.exitStatus(inv); err != nil {
			return err
		} else if exited {
			return fmt.Errorf("%s exited with status %d before publishing its ready marker", inv.Role, status)
		}
		<-ticker.C
	}
	return fmt.Errorf("%s timed out before publishing its ready marker", inv.Role)
}

func (runtime *tmuxRuntime) waitPMProtocolReady(inv invocation, deadlineUnixNano int64) error {
	workspace, err := runtime.workspaceStore(inv)
	if err != nil {
		return err
	}
	defer workspace.Close()
	ticker := time.NewTicker(20 * time.Millisecond)
	defer ticker.Stop()
	for time.Now().UnixNano() < deadlineUnixNano {
		data, readErr := workspace.readRegular("pm-ready")
		switch {
		case readErr == nil && string(data) == "ready\n":
			live, liveErr := runtime.invocationLive(inv)
			if liveErr != nil {
				return fmt.Errorf("verify PM protocol handshake: %w", liveErr)
			}
			if !live {
				return fmt.Errorf("PM exited before its protocol-ready handshake was accepted")
			}
			return nil
		case readErr == nil:
			return fmt.Errorf("invalid PM protocol-ready marker")
		case !errors.Is(readErr, os.ErrNotExist):
			return fmt.Errorf("read PM protocol-ready marker: %w", readErr)
		}
		if status, exited, exitErr := runtime.exitStatus(inv); exitErr != nil {
			return exitErr
		} else if exited {
			return fmt.Errorf("PM exited with status %d before publishing its protocol-ready marker", status)
		}
		<-ticker.C
	}
	return fmt.Errorf("PM timed out before publishing its protocol-ready marker")
}

func copyOptionalCredential(source, target string) error {
	info, err := os.Lstat(source)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("credential is not a regular file: %s", source)
	}
	input, err := os.Open(source)
	if err != nil {
		return err
	}
	defer input.Close()
	output, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	keep := false
	defer func() {
		_ = output.Close()
		if !keep {
			_ = os.Remove(target)
		}
	}()
	if _, err := io.Copy(output, input); err != nil {
		return err
	}
	if err := output.Sync(); err != nil {
		return err
	}
	if err := output.Close(); err != nil {
		return err
	}
	keep = true
	return nil
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}

func (runtime *tmuxRuntime) waitForWorker(ctx context.Context, deadlineUnixNano int64, pm, worker invocation) error {
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()
	for {
		if time.Now().UnixNano() >= deadlineUnixNano {
			if err := runtime.stopInvocation(worker); err != nil {
				return fmt.Errorf("%s exceeded its wall-clock deadline and termination failed: %v", worker.Role, err)
			}
			return fmt.Errorf("%s timed out waiting for process completion: wall-clock deadline exceeded", worker.Role)
		}
		pmStatus, pmExited, pmErr := runtime.exitStatus(pm)
		if pmErr != nil {
			return fmt.Errorf("read PM completion: %w", pmErr)
		}
		if pmExited {
			stopErr := runtime.stopInvocation(worker)
			if stopErr != nil {
				return fmt.Errorf("PM exited unexpectedly with status %d; worker cleanup failed: %v", pmStatus, stopErr)
			}
			return fmt.Errorf("PM exited unexpectedly with status %d", pmStatus)
		}
		pmLive, pmLiveErr := runtime.invocationLive(pm)
		if pmLiveErr != nil {
			return fmt.Errorf("verify PM while %s is active: %w", worker.Role, pmLiveErr)
		}
		if !pmLive {
			_ = runtime.stopInvocation(worker)
			return fmt.Errorf("PM exited unexpectedly while %s was active", worker.Role)
		}
		status, exited, statusErr := runtime.exitStatus(worker)
		if statusErr != nil {
			return fmt.Errorf("read %s completion: %w", worker.Role, statusErr)
		}
		if exited {
			if status != 0 {
				return fmt.Errorf("%s worker exited with status %d", worker.Role, status)
			}
			if err := runtime.waitWindowAbsent(ctx, worker.Window); err != nil {
				return fmt.Errorf("verify %s worker termination: %w", worker.Role, err)
			}
			if err := runtime.ensureInvocationStopped(worker); err != nil {
				return fmt.Errorf("verify %s owned processes terminated: %w", worker.Role, err)
			}
			runtime.markNaturalExit(worker.ID)
			return nil
		}
		select {
		case <-ctx.Done():
			if err := runtime.stopInvocation(worker); err != nil {
				return fmt.Errorf("%s timed out and termination failed: %v", worker.Role, err)
			}
			return fmt.Errorf("%s timed out waiting for process completion: %w", worker.Role, ctx.Err())
		case <-ticker.C:
		}
	}
}

func (runtime *tmuxRuntime) invocationLive(inv invocation) (bool, error) {
	_, exited, err := runtime.exitStatus(inv)
	if err != nil || exited {
		return false, err
	}
	exists, err := runtime.windowExists(inv.Window)
	if err != nil || !exists {
		return false, err
	}
	identity, ready, err := runtime.readyStatus(inv)
	if err != nil || !ready {
		return false, err
	}
	return identityMatches(identity)
}

func (runtime *tmuxRuntime) stopInvocation(inv invocation) error {
	var failures []error
	exists, probeErr := runtime.windowExists(inv.Window)
	if probeErr != nil {
		failures = append(failures, probeErr)
	}
	if exists || probeErr != nil {
		output, killErr := runtime.runCommand("kill-window", "-t", runtime.session+":"+inv.Window)
		if killErr != nil && !recognizedTmuxAbsent(killErr, output) {
			failures = append(failures, fmt.Errorf("kill tmux window: %w: %s", killErr, strings.TrimSpace(string(output))))
		}
	}
	if err := runtime.terminateInvocation(inv); err != nil {
		failures = append(failures, err)
	}
	if len(failures) == 0 {
		ctx, cancel := context.WithTimeout(context.Background(), runtime.commandTimeout)
		defer cancel()
		if err := runtime.waitWindowAbsent(ctx, inv.Window); err != nil {
			failures = append(failures, err)
		}
	}
	result := errors.Join(failures...)
	if result == nil {
		if record := runtime.invocationRecord(inv.ID); record != nil {
			record.retired = true
		}
	}
	return result
}

func (runtime *tmuxRuntime) cleanup(requirePMLive bool, pm invocation) error {
	if runtime == nil || runtime.cleaned {
		return nil
	}
	var failures []error
	if requirePMLive {
		live, err := runtime.invocationLive(pm)
		if err != nil {
			failures = append(failures, fmt.Errorf("verify persistent PM before terminal cleanup: %w", err))
		} else if !live {
			failures = append(failures, fmt.Errorf("persistent PM exited before controller-initiated terminal cleanup"))
		}
	}
	exists, probeErr := runtime.sessionExists()
	if probeErr != nil {
		failures = append(failures, fmt.Errorf("probe tmux session before cleanup: %w", probeErr))
	}
	if exists || probeErr != nil {
		output, killErr := runtime.runCommand("kill-session", "-t", runtime.session)
		if killErr != nil && !recognizedTmuxAbsent(killErr, output) {
			failures = append(failures, fmt.Errorf("kill tmux session: %w: %s", killErr, strings.TrimSpace(string(output))))
		}
	}
	for index := range runtime.invocations {
		inv := &runtime.invocations[index]
		if !inv.started || inv.retired {
			continue
		}
		if err := runtime.terminateInvocation(*inv); err != nil {
			failures = append(failures, fmt.Errorf("terminate %s owned processes: %w", inv.Role, err))
		} else {
			inv.retired = true
		}
	}
	deadline := time.Now().Add(runtime.commandTimeout)
	verifiedAbsent := false
	for time.Now().Before(deadline) {
		exists, err := runtime.sessionExists()
		if err != nil {
			failures = append(failures, fmt.Errorf("verify tmux session cleanup: %w", err))
			break
		}
		if !exists {
			verifiedAbsent = true
			break
		}
		time.Sleep(25 * time.Millisecond)
	}
	if !verifiedAbsent && probeErr == nil {
		failures = append(failures, fmt.Errorf("tmux session %s still exists after cleanup", runtime.session))
	}
	if runtime.failCleanup {
		failures = append(failures, fmt.Errorf("injected persistent runtime cleanup verification failure"))
	}
	if len(failures) == 0 {
		runtime.cleaned = true
	}
	return errors.Join(failures...)
}

func (runtime *tmuxRuntime) closeCredentials() error {
	if runtime == nil {
		return nil
	}
	if err := runtime.auditPrivateProcesses(); err != nil {
		return err
	}
	deadline := time.Now().Add(runtime.commandTimeout)
	var lastErr error
	for {
		lastErr = os.RemoveAll(runtime.codexHomesDir)
		if lastErr == nil {
			if _, err := os.Lstat(runtime.codexHomesDir); errors.Is(err, os.ErrNotExist) {
				return nil
			} else if err != nil {
				lastErr = err
			} else {
				lastErr = fmt.Errorf("private Codex credential directory still exists after removal")
			}
		}
		if time.Now().After(deadline) {
			return lastErr
		}
		time.Sleep(25 * time.Millisecond)
	}
}

// auditPrivateProcesses is the final controller-owned boundary before private
// state is removed. A runner can disappear while a launched client remains
// reparented; in that case the manifest is no longer sufficient, but the
// private executable/workspace path is still an exact identity anchor.
func (runtime *tmuxRuntime) auditPrivateProcesses() error {
	if runtime == nil || runtime.privateRoot == "" {
		return nil
	}
	output, err := exec.Command("ps", "-axo", "pid=,command=").Output()
	if err != nil {
		return fmt.Errorf("audit private runtime processes: %w", err)
	}
	self := strconv.Itoa(os.Getpid())
	var matches []string
	for _, line := range strings.Split(string(output), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 || fields[0] == self || !strings.Contains(line, runtime.privateRoot) {
			continue
		}
		matches = append(matches, strings.TrimSpace(line))
	}
	if len(matches) != 0 {
		return fmt.Errorf("private runtime processes remain: %s", strings.Join(matches, "; "))
	}
	return nil
}

func (runtime *tmuxRuntime) sessionExists() (bool, error) {
	output, err := runtime.runCommand("has-session", "-t", runtime.session)
	if err == nil {
		return true, nil
	}
	if recognizedTmuxAbsent(err, output) {
		return false, nil
	}
	return false, fmt.Errorf("tmux has-session failed: %w: %s", err, strings.TrimSpace(string(output)))
}

func recognizedTmuxAbsent(err error, output []byte) bool {
	var exitError *exec.ExitError
	if !errors.As(err, &exitError) || exitError.ExitCode() != 1 {
		return false
	}
	message := strings.ToLower(strings.TrimSpace(string(output)))
	return strings.Contains(message, "can't find session:") || strings.Contains(message, "no server running on")
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

func (runtime *tmuxRuntime) exitStatus(inv invocation) (int, bool, error) {
	data, err := runtime.controlStore.readRegular(inv.ExitRelative)
	if errors.Is(err, os.ErrNotExist) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	status, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0, false, fmt.Errorf("invalid atomically published exit marker for %s", inv.ID)
	}
	return status, true, nil
}

func (runtime *tmuxRuntime) readyStatus(inv invocation) (processIdentity, bool, error) {
	data, err := runtime.controlStore.readRegular(inv.ReadyRelative)
	if errors.Is(err, os.ErrNotExist) {
		return processIdentity{}, false, nil
	}
	if err != nil {
		return processIdentity{}, false, err
	}
	var identity processIdentity
	if err := decodeStrictJSON(data, &identity); err != nil || identity.PID <= 0 || identity.Started == "" {
		return processIdentity{}, false, fmt.Errorf("invalid ready marker for %s", inv.ID)
	}
	return identity, true, nil
}

func (runtime *tmuxRuntime) terminateInvocation(inv invocation) error {
	record := runtime.invocationRecord(inv.ID)
	if record == nil || !record.started {
		return nil
	}
	manifestPath := filepath.Join(runtime.controlDir, inv.OwnershipRelative)
	terminationErr := terminateOwnedProcesses(manifestPath, runtime.commandTimeout, 0)
	if terminationErr == nil {
		if _, exited, err := runtime.exitStatus(inv); err != nil {
			terminationErr = err
		} else if !exited {
			terminationErr = runtime.controlStore.writeAtomic(inv.ExitRelative, []byte("143\n"), 0o600)
		}
	}
	return terminationErr
}

func (runtime *tmuxRuntime) ensureInvocationStopped(inv invocation) error {
	pids, err := ownedProcessIDs(filepath.Join(runtime.controlDir, inv.OwnershipRelative))
	if err != nil {
		return err
	}
	if len(pids) != 0 {
		return fmt.Errorf("owned processes remain for %s: %v", inv.ID, pids)
	}
	return nil
}

func (runtime *tmuxRuntime) markNaturalExit(id string) {
	if record := runtime.invocationRecord(id); record != nil {
		record.retired = true
	}
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
	if !inv.started {
		inv.archived = true
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
		if _, err := destination.copyRegularFrom(runtime.controlStore, item.source, item.target, item.mode); err != nil {
			return fmt.Errorf("archive required %s audit file for %s: %w", filepath.Base(item.source), inv.ID, err)
		}
	}
	if err := runtime.exportTestArtifacts(*inv); err != nil {
		return err
	}
	inv.archived = true
	return nil
}

func (runtime *tmuxRuntime) exportTestArtifacts(inv invocation) error {
	for _, item := range []struct {
		source    string
		directory string
		target    string
	}{
		{".write-uuter-test-log.json", runtime.fakeLogDir, inv.ID + ".json"},
		{".write-uuter-isolation.probe", runtime.fakeLogDir, "isolation-" + inv.ID + ".probe"},
		{".write-uuter-detached.pid", runtime.detachedPIDDir, inv.ID + ".pid"},
		{".write-uuter-detached.pgid", runtime.detachedPIDDir, inv.ID + ".pgid"},
	} {
		if item.directory == "" {
			continue
		}
		info, err := os.Lstat(filepath.Join(inv.WorkspacePath, item.source))
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("test artifact %s is not a regular file", item.source)
		}
		data, err := os.ReadFile(filepath.Join(inv.WorkspacePath, item.source))
		if err != nil {
			return err
		}
		if err := os.MkdirAll(item.directory, 0o700); err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(item.directory, item.target), data, 0o600); err != nil {
			return err
		}
	}
	return nil
}

func (runtime *tmuxRuntime) archiveAll(destination *artifactStore) error {
	if runtime.removeAudit != "" && !runtime.auditRemoved {
		runtime.auditRemoved = true
		for index := len(runtime.invocations) - 1; index >= 0; index-- {
			inv := runtime.invocations[index]
			if !inv.started {
				continue
			}
			relative := map[string]string{"prompt": inv.PromptRelative, "log": inv.LogRelative, "exit": inv.ExitRelative}[runtime.removeAudit]
			if relative != "" {
				if err := runtime.controlStore.remove(relative); err != nil {
					return fmt.Errorf("inject missing audit source: %w", err)
				}
			}
			break
		}
	}
	for index := range runtime.invocations {
		if err := runtime.archive(&runtime.invocations[index], destination); err != nil {
			return err
		}
	}
	return nil
}

func (runtime *tmuxRuntime) closePrivate() error {
	if runtime == nil || runtime.closed {
		return nil
	}
	deadline := time.Now().Add(runtime.commandTimeout)
	var closeErr error
	var lastRemoveErr error
	for {
		if auditErr := runtime.auditPrivateProcesses(); auditErr != nil {
			return errors.Join(closeErr, auditErr)
		}
		if !runtime.storeClosed {
			if err := runtime.controlStore.Close(); err != nil {
				closeErr = errors.Join(closeErr, err)
			} else {
				runtime.storeClosed = true
			}
		}
		var removeErr error
		if runtime.failRemoveOnce {
			runtime.failRemoveOnce = false
			removeErr = fmt.Errorf("injected private runtime removal failure")
		} else {
			removeErr = os.RemoveAll(runtime.privateRoot)
			if removeErr == nil {
				if _, err := os.Lstat(runtime.privateRoot); errors.Is(err, os.ErrNotExist) {
					runtime.closed = true
					return closeErr
				} else if err != nil {
					removeErr = err
				} else {
					removeErr = fmt.Errorf("private runtime still exists after removal")
				}
			}
		}
		lastRemoveErr = removeErr
		if time.Now().After(deadline) {
			return errors.Join(closeErr, lastRemoveErr)
		}
		time.Sleep(25 * time.Millisecond)
	}
}
