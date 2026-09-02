package app

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/uuta/write-uuter/internal/captureprotocol"
)

const (
	captureRunnerEnv            = "WRITE_UUTER_CAPTURE_RUNNER"
	captureRunnerTimeoutTestEnv = "WRITE_UUTER_TEST_CAPTURE_RUNNER_TIMEOUT"
	captureTrackerDelayTestEnv  = "WRITE_UUTER_TEST_CAPTURE_TRACKER_DELAY"
	captureResultMaxBytes       = 1 << 20
	captureTraceReferenceMaxLen = 2048
)

var captureBackendPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$`)

func (control *controller) runCaptureRunner(requests []ScreenshotRequest) (results []validatedCapture, returnErr error) {
	executable, err := resolveCaptureRunner()
	if err != nil {
		return nil, err
	}
	privateRoot, err := os.MkdirTemp(filepath.Dir(control.runDir), ".write-uuter-capture-*")
	if err != nil {
		return nil, fmt.Errorf("create private capture-runner workspace: %w", err)
	}
	if err := os.Chmod(privateRoot, 0o700); err != nil {
		_ = os.RemoveAll(privateRoot)
		return nil, fmt.Errorf("protect private capture-runner workspace: %w", err)
	}
	workspace := filepath.Join(privateRoot, "workspace")
	if err := os.Mkdir(workspace, 0o700); err != nil {
		_ = os.RemoveAll(privateRoot)
		return nil, fmt.Errorf("create private capture-runner workspace: %w", err)
	}
	defer func() {
		if err := os.RemoveAll(privateRoot); err != nil {
			returnErr = errors.Join(returnErr, fmt.Errorf("remove private capture-runner workspace: %w", err))
		} else if _, err := os.Lstat(privateRoot); !errors.Is(err, os.ErrNotExist) {
			returnErr = errors.Join(returnErr, fmt.Errorf("private capture-runner workspace still exists after cleanup"))
		}
	}()

	requestDocument := captureprotocol.RequestDocument{SchemaVersion: captureprotocol.Version, Requests: make([]captureprotocol.Request, 0, len(requests))}
	for _, request := range requests {
		requestDocument.Requests = append(requestDocument.Requests, captureprotocol.Request{
			RequestID: request.ID, PublicURL: request.URL, Selector: request.Selector,
			Reason: request.Reason, SupportedClaimIDs: append([]string(nil), request.Supports...),
			PriorAttempt: request.PriorAttempt,
		})
	}
	requestData, err := marshalIndentedJSON(requestDocument)
	if err != nil {
		return nil, fmt.Errorf("encode capture-runner request: %w", err)
	}
	requestPath := filepath.Join(workspace, captureprotocol.RequestFile)
	if err := os.WriteFile(requestPath, requestData, 0o444); err != nil {
		return nil, fmt.Errorf("write capture-runner request: %w", err)
	}

	deadline := screenshotRequestTimeout * time.Duration(len(requests))
	if injected, parseErr := time.ParseDuration(os.Getenv(captureRunnerTimeoutTestEnv)); parseErr == nil && injected > 0 {
		deadline = injected
	}
	teardown := captureRunnerTeardown(deadline)
	startedAt := time.Now()
	controllerWork := deadline - teardown
	innerWork := deadline - 2*teardown
	// Extremely short injected/test budgets need proportionally more room for
	// the same process/workspace teardown. Production capture budgets retain
	// the wider adapter work window below.
	if deadline <= 2*time.Second {
		controllerWork = deadline - 2*teardown
		innerWork = deadline - 3*teardown
	}
	if controllerWork <= 0 {
		controllerWork = deadline / 2
	}
	ctx, cancel := context.WithDeadline(context.Background(), startedAt.Add(controllerWork))
	defer cancel()
	command := exec.CommandContext(ctx, executable, captureprotocol.VersionArgument)
	command.Dir = workspace
	// Give every runner a provider-neutral absolute budget strictly inside the
	// controller's own cancellation point. A runner can finish its backend
	// cleanup and exit normally before the outer process-tree sweep begins.
	if innerWork <= 0 {
		innerWork = controllerWork / 2
	}
	innerDeadline := startedAt.Add(innerWork)
	command.Env = append(os.Environ(), captureprotocol.RunnerDeadlineEnv+"="+strconv.FormatInt(innerDeadline.UnixMilli(), 10))
	command.Stdin = nil
	// Use a private standard-stream sink as a kernel-visible ownership marker.
	// It is not a log and is removed with the private workspace. Descendants
	// that detach from both ancestry and the runner's process group still inherit
	// this open file and remain attributable to this invocation on Darwin.
	boundaryPath := filepath.Join(privateRoot, "ownership-boundary")
	boundary, err := os.OpenFile(boundaryPath, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0o600)
	if err != nil {
		return nil, fmt.Errorf("create capture runner process ownership boundary: %w", err)
	}
	defer boundary.Close()
	command.Stdout = boundary
	command.Stderr = boundary
	configureTrackedProcessLaunch(command)
	command.Cancel = func() error { return terminateProcessGroup(command.Process.Pid) }
	command.WaitDelay = teardown / 2
	if err := command.Start(); err != nil {
		return nil, fmt.Errorf("capture runner could not start; verify %s names a healthy trusted executable", captureRunnerEnv)
	}
	if err := waitForTrackedProcessLaunch(command.Process.Pid, startedAt.Add(controllerWork)); err != nil {
		_ = command.Process.Kill()
		_ = command.Wait()
		return nil, fmt.Errorf("capture runner process ownership: %w", err)
	}
	if injected, parseErr := time.ParseDuration(os.Getenv(captureTrackerDelayTestEnv)); parseErr == nil && injected > 0 {
		time.Sleep(injected)
	}
	tracker, trackerErr := startProcessTracker(filepath.Join(privateRoot, "ownership.json"), command.Process.Pid, boundaryPath)
	if trackerErr != nil {
		_ = releaseTrackedProcessLaunch(command.Process.Pid)
		_ = terminateProcessGroup(command.Process.Pid)
		_ = command.Wait()
		return nil, fmt.Errorf("capture runner process ownership: %w", trackerErr)
	}
	var cleanupOnce sync.Once
	var cleanupErr error
	cleanup := func() error {
		cleanupOnce.Do(func() {
			cleanupDeadline := startedAt.Add(deadline)
			if ctx.Err() != nil {
				// Preserve headroom inside the advertised timeout for workspace
				// removal and the caller's durable blocked-state write.
				cleanupDeadline = startedAt.Add(deadline - teardown/2)
			}
			groupErr := terminateProcessGroup(command.Process.Pid)
			cleanupErr = errors.Join(groupErr, tracker.terminateUntil(cleanupDeadline))
		})
		return cleanupErr
	}
	// Register ownership cleanup immediately: every return after tracker
	// creation, including a future validation branch, sweeps the same tree once.
	defer func() {
		if err := cleanup(); err != nil && !errors.Is(returnErr, err) {
			returnErr = errors.Join(returnErr, err)
		}
	}()
	if err := releaseTrackedProcessLaunch(command.Process.Pid); err != nil {
		_ = command.Process.Kill()
		cleanupErr = cleanup()
		_ = command.Wait()
		return nil, errors.Join(fmt.Errorf("capture runner process ownership: %w", err), cleanupErr)
	}
	waitErr := command.Wait()
	cleanupErr = cleanup()
	if ctx.Err() == context.DeadlineExceeded {
		return nil, errors.Join(fmt.Errorf("capture runner timed out after %s; verify runner health and retry", deadline), cleanupErr)
	}
	if cleanupErr != nil {
		return nil, fmt.Errorf("capture runner process-tree cleanup failed: %w", cleanupErr)
	}
	if waitErr != nil && !errors.Is(waitErr, exec.ErrWaitDelay) {
		var exitError *exec.ExitError
		if errors.As(waitErr, &exitError) {
			return nil, fmt.Errorf("capture runner exited with status %d; verify its provider configuration and credentials outside write-uuter, then retry", exitError.ExitCode())
		}
		return nil, fmt.Errorf("capture runner failed; verify its provider configuration outside write-uuter, then retry")
	}
	return validateCaptureWorkspace(workspace, requestData, requests)
}

func captureRunnerTeardown(deadline time.Duration) time.Duration {
	teardown := deadline / 4
	if teardown > 2*time.Second {
		teardown = 2 * time.Second
	}
	if teardown < 4*time.Millisecond {
		teardown = 4 * time.Millisecond
	}
	if teardown >= deadline {
		teardown = deadline / 2
	}
	return teardown
}

func resolveCaptureRunner() (string, error) {
	configured := strings.TrimSpace(os.Getenv(captureRunnerEnv))
	if configured == "" {
		return "", fmt.Errorf("screenshot capture requires %s set to the trusted absolute capture-runner executable", captureRunnerEnv)
	}
	if !filepath.IsAbs(configured) || filepath.Clean(configured) != configured {
		return "", fmt.Errorf("%s must be a clean absolute executable path", captureRunnerEnv)
	}
	info, err := os.Lstat(configured)
	if err != nil {
		return "", fmt.Errorf("capture runner is unavailable; set %s to a healthy trusted absolute executable", captureRunnerEnv)
	}
	if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm()&0o111 == 0 {
		return "", fmt.Errorf("%s does not name an executable regular file", captureRunnerEnv)
	}
	return configured, nil
}

type validatedCapture struct {
	protocol captureprotocol.Result
	data     []byte
	width    int
	height   int
	time     time.Time
}

func validateCaptureWorkspace(workspace string, originalRequest []byte, requests []ScreenshotRequest) ([]validatedCapture, error) {
	requestData, err := os.ReadFile(filepath.Join(workspace, captureprotocol.RequestFile))
	if err != nil || !bytes.Equal(requestData, originalRequest) {
		return nil, fmt.Errorf("capture runner modified or removed read-only %s", captureprotocol.RequestFile)
	}
	requestInfo, err := os.Lstat(filepath.Join(workspace, captureprotocol.RequestFile))
	if err != nil || !requestInfo.Mode().IsRegular() || requestInfo.Mode().Perm() != 0o444 {
		return nil, fmt.Errorf("capture runner changed the read-only request artifact")
	}
	resultPath := filepath.Join(workspace, captureprotocol.ResultFile)
	resultInfo, err := os.Lstat(resultPath)
	if err != nil || !resultInfo.Mode().IsRegular() || resultInfo.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("capture runner did not produce a regular %s", captureprotocol.ResultFile)
	}
	if resultInfo.Size() <= 0 || resultInfo.Size() > captureResultMaxBytes {
		return nil, fmt.Errorf("capture runner %s exceeds the accepted size", captureprotocol.ResultFile)
	}
	resultData, err := os.ReadFile(resultPath)
	if err != nil || len(resultData) > captureResultMaxBytes {
		return nil, fmt.Errorf("read capture runner %s", captureprotocol.ResultFile)
	}
	var document captureprotocol.ResultDocument
	if err := decodeStrictJSONExactRequired(resultData, &document); err != nil {
		return nil, fmt.Errorf("invalid capture runner %s: %s", captureprotocol.ResultFile, strictCaptureJSONError(err))
	}
	if document.SchemaVersion != captureprotocol.Version {
		return nil, fmt.Errorf("capture runner result schema version %d does not match protocol version %d", document.SchemaVersion, captureprotocol.Version)
	}
	if len(document.Results) != len(requests) {
		return nil, fmt.Errorf("capture runner returned %d results for %d requests", len(document.Results), len(requests))
	}

	declaredExact := make(map[string]bool, len(document.Results))
	declaredFolded := make(map[string]bool, len(document.Results))
	validated := make([]validatedCapture, 0, len(document.Results))
	for index, result := range document.Results {
		request := requests[index]
		if result.RequestID != request.ID {
			return nil, fmt.Errorf("capture result %d request ID does not match the request", index)
		}
		if result.RequestedURL != request.URL {
			return nil, fmt.Errorf("capture result %q requested URL does not match the request", result.RequestID)
		}
		if _, err := validatePublicPageURL(result.FinalURL); err != nil {
			return nil, fmt.Errorf("capture result %q has an invalid final URL", result.RequestID)
		}
		capturedAt, err := time.Parse(time.RFC3339Nano, result.CapturedAt)
		if err != nil || capturedAt.IsZero() {
			return nil, fmt.Errorf("capture result %q has an invalid timestamp", result.RequestID)
		}
		if !captureBackendPattern.MatchString(result.Backend) {
			return nil, fmt.Errorf("capture result %q has an invalid backend identifier", result.RequestID)
		}
		if result.MediaType != screenshotMediaType {
			return nil, fmt.Errorf("capture result %q media type is not supported", result.RequestID)
		}
		if result.Viewport.Width < 1 || result.Viewport.Height < 1 || result.Viewport.Width > screenshotMaxDimension || result.Viewport.Height > screenshotMaxDimension {
			return nil, fmt.Errorf("capture result %q has invalid viewport dimensions", result.RequestID)
		}
		if !equalStrings(result.SupportedClaimIDs, request.Supports) || result.Rationale != request.Reason {
			return nil, fmt.Errorf("capture result %q changed the controller-owned claim binding or rationale", result.RequestID)
		}
		if err := validateCaptureTrace(result); err != nil {
			return nil, fmt.Errorf("capture result %q: %w", result.RequestID, err)
		}
		relative, err := validateRunnerAssetPath(result.ImagePath)
		if err != nil {
			return nil, fmt.Errorf("capture result %q: %w", result.RequestID, err)
		}
		folded := strings.ToLower(relative)
		if declaredFolded[folded] {
			return nil, fmt.Errorf("capture runner declared a duplicate asset path")
		}
		declaredFolded[folded] = true
		declaredExact[relative] = true
		data, err := readRunnerAsset(workspace, relative)
		if err != nil {
			return nil, fmt.Errorf("capture result %q asset is invalid: %s", result.RequestID, safeCaptureAssetError(err))
		}
		width, height, err := validatePNG(data)
		if err != nil {
			return nil, fmt.Errorf("capture result %q: %w", result.RequestID, err)
		}
		if result.ByteSize != int64(len(data)) || result.Width != width || result.Height != height {
			return nil, fmt.Errorf("capture result %q size or dimensions do not match its PNG", result.RequestID)
		}
		if result.SHA256 != revisionFor(data) {
			return nil, fmt.Errorf("capture result %q SHA-256 digest does not match its PNG", result.RequestID)
		}
		validated = append(validated, validatedCapture{protocol: result, data: data, width: width, height: height, time: capturedAt})
	}
	if err := validateCaptureWorkspaceEntries(workspace, declaredExact, declaredFolded); err != nil {
		return nil, err
	}
	return validated, nil
}

func validateRunnerAssetPath(name string) (string, error) {
	if name == "" || strings.Contains(name, "\\") || filepath.IsAbs(name) || filepath.ToSlash(filepath.Clean(name)) != name || name == "." || strings.ContainsRune(name, 0) {
		return "", fmt.Errorf("image path is not a clean relative path")
	}
	if !strings.HasPrefix(name, captureprotocol.AssetsDirectory+"/") {
		return "", fmt.Errorf("image path is outside the assets directory")
	}
	return name, nil
}

func readRunnerAsset(workspace, relative string) ([]byte, error) {
	root, err := os.OpenRoot(workspace)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	info, err := root.Lstat(filepath.FromSlash(relative))
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("declared asset is not a regular no-follow file")
	}
	if info.Size() <= 0 || info.Size() > screenshotMaxBytes {
		return nil, fmt.Errorf("declared asset exceeds the accepted byte size")
	}
	data, err := root.ReadFile(filepath.FromSlash(relative))
	if err != nil || len(data) > screenshotMaxBytes {
		return nil, fmt.Errorf("declared asset could not be read")
	}
	return data, nil
}

func validateCaptureWorkspaceEntries(workspace string, declaredExact, declaredFolded map[string]bool) error {
	seen := make(map[string]bool, len(declaredExact))
	err := filepath.WalkDir(workspace, func(name string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, _ := filepath.Rel(workspace, name)
		relative = filepath.ToSlash(relative)
		if relative == "." || relative == captureprotocol.RequestFile || relative == captureprotocol.ResultFile || relative == captureprotocol.AssetsDirectory {
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 || entry.Type()&os.ModeType != 0 && !entry.IsDir() {
			return fmt.Errorf("capture runner produced an undeclared symlink or special file")
		}
		if entry.IsDir() {
			prefix := strings.ToLower(relative) + "/"
			for path := range declaredFolded {
				if strings.HasPrefix(path, prefix) {
					return nil
				}
			}
			return fmt.Errorf("capture runner produced an undeclared directory")
		}
		// Folding declarations rejects portable duplicates, but membership is
		// exact: a second case variant is still an undeclared output on a
		// case-sensitive filesystem.
		if !declaredExact[relative] {
			return fmt.Errorf("capture runner produced an undeclared file")
		}
		seen[relative] = true
		return nil
	})
	if err != nil {
		return err
	}
	if len(seen) != len(declaredExact) {
		return fmt.Errorf("capture runner result is missing a declared asset")
	}
	return nil
}

func validateCaptureTrace(result captureprotocol.Result) error {
	if (len(result.ActionSummary) == 0) == (result.TraceReference == "") {
		return fmt.Errorf("must provide exactly one bounded action summary or trace reference")
	}
	if len(result.ActionSummary) > captureprotocol.MaxActionSteps {
		return fmt.Errorf("action summary exceeds %d steps", captureprotocol.MaxActionSteps)
	}
	for _, step := range result.ActionSummary {
		if strings.TrimSpace(step) != step || step == "" || len(step) > captureprotocol.MaxActionStepLen || containsControl(step) {
			return fmt.Errorf("action summary contains an invalid step")
		}
	}
	if result.TraceReference != "" && (strings.TrimSpace(result.TraceReference) != result.TraceReference || len(result.TraceReference) > captureTraceReferenceMaxLen || containsControl(result.TraceReference)) {
		return fmt.Errorf("trace reference is invalid")
	}
	return nil
}

func containsControl(value string) bool {
	for _, character := range value {
		if character < 0x20 || character == 0x7f {
			return true
		}
	}
	return false
}

func strictCaptureJSONError(err error) string {
	message := err.Error()
	switch {
	case strings.Contains(message, "duplicate JSON key"):
		return "duplicate JSON key"
	case strings.Contains(message, "unknown field"):
		return "unknown field"
	case strings.Contains(message, "missing required field"):
		return "missing required field"
	case strings.Contains(message, "must be a JSON boolean"):
		return "required full_page must be a JSON boolean"
	default:
		return "malformed JSON"
	}
}

func safeCaptureAssetError(err error) string {
	message := err.Error()
	for _, class := range []string{
		"not a regular no-follow file", "exceeds the accepted byte size", "could not be read",
	} {
		if strings.Contains(message, class) {
			return class
		}
	}
	return "asset validation failed"
}

func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func marshalIndentedJSON(value any) ([]byte, error) {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}
