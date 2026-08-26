package app

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// RunAgent is an internal controller entry point. It is intentionally not part
// of the public CLI: tmux starts it so Go, rather than an agent-writable shell
// script, owns Codex IO, process groups, and completion publication.
func RunAgent(arguments []string) (returnErr error) {
	flags := flag.NewFlagSet("__agent", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	client := flags.String("client", "", "")
	provider := flags.String("provider", "", "")
	model := flags.String("model", "", "")
	effort := flags.String("effort", "", "")
	workspace := flags.String("workspace", "", "")
	promptPath := flags.String("prompt", "", "")
	logPath := flags.String("log", "", "")
	exitPath := flags.String("exit", "", "")
	readyPath := flags.String("ready", "", "")
	ownershipPath := flags.String("ownership", "", "")
	profilePath := flags.String("profile", "", "")
	role := flags.String("role", "", "")
	lens := flags.String("lens", "", "")
	candidate := flags.Int("candidate", 0, "")
	revision := flags.String("revision", "", "")
	invocation := flags.String("invocation", "", "")
	providerHome := flags.String("provider-home", "", "")
	exitMarkerDelay := flags.String("exit-marker-delay", "", "")
	readyMarkerDelay := flags.String("ready-marker-delay", "", "")
	if err := flags.Parse(arguments); err != nil || flags.NArg() != 0 {
		return fmt.Errorf("invalid internal agent runner arguments")
	}
	for name, value := range map[string]string{
		"client": *client, "provider": *provider, "model": *model, "effort": *effort,
		"workspace": *workspace, "prompt": *promptPath,
		"log": *logPath, "exit": *exitPath, "ready": *readyPath,
		"ownership": *ownershipPath,
		"profile":   *profilePath, "role": *role, "invocation": *invocation, "provider-home": *providerHome,
	} {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("internal agent runner is missing %s", name)
		}
	}
	if *provider != providerCodex && *provider != providerClaudeCode {
		return fmt.Errorf("internal agent runner received unsupported provider %q", *provider)
	}
	status := 1
	defer func() {
		if err := publishInteger(*exitPath, status, *exitMarkerDelay); err != nil {
			returnErr = errors.Join(returnErr, fmt.Errorf("publish agent exit status: %w", err))
		}
	}()

	prompt, err := os.Open(*promptPath)
	if err != nil {
		return fmt.Errorf("open agent prompt: %w", err)
	}
	defer prompt.Close()
	logFile, err := os.OpenFile(*logPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("open agent log: %w", err)
	}
	defer logFile.Close()
	defer func() {
		if returnErr != nil {
			_, _ = fmt.Fprintf(logFile, "\nwrite-uuter: agent runner failed: %v\n", returnErr)
		}
	}()
	if err := os.MkdirAll(filepath.Join(*workspace, ".tmp"), 0o700); err != nil {
		return fmt.Errorf("create agent temporary directory: %w", err)
	}
	tracker, err := startProcessTracker(*ownershipPath, os.Getpid())
	if err != nil {
		return fmt.Errorf("start process ownership tracker: %w", err)
	}
	trackerClosed := false
	defer func() {
		if !trackerClosed {
			returnErr = errors.Join(returnErr, tracker.terminate(2*time.Second))
		}
	}()
	if *readyMarkerDelay != "" {
		if delay, parseErr := time.ParseDuration(*readyMarkerDelay); parseErr == nil {
			time.Sleep(delay)
		}
	}
	clientArguments := providerArguments(*provider, *workspace, *model, *effort)
	executable, commandArguments, err := isolatedCommand(*profilePath, *client, clientArguments)
	if err != nil {
		return err
	}
	command := exec.Command(executable, commandArguments...)
	command.Dir = *workspace
	command.Stdin = prompt
	command.Stdout = logFile
	command.Stderr = logFile
	command.Env = agentEnvironment(*provider, *workspace, *providerHome, *role, *lens, *candidate, *revision, *invocation)
	configureProcessGroup(command)
	if err := command.Start(); err != nil {
		return fmt.Errorf("start %s client: %w", *provider, err)
	}
	identity, err := tracker.waitFor(command.Process.Pid, time.Now().Add(2*time.Second))
	if err != nil {
		_ = command.Process.Kill()
		_ = command.Wait()
		return fmt.Errorf("capture %s process ownership: %w", *provider, err)
	}
	if err := publishJSONAtomic(*readyPath, identity); err != nil {
		_ = command.Process.Kill()
		_ = command.Wait()
		return fmt.Errorf("publish live %s ready marker: %w", *provider, err)
	}
	waitErr := command.Wait()
	status = 0
	if waitErr != nil {
		var exitError *exec.ExitError
		if errors.As(waitErr, &exitError) {
			status = exitError.ExitCode()
		} else {
			status = 1
		}
	}
	if err := tracker.terminate(2 * time.Second); err != nil {
		status = 1
		_, _ = fmt.Fprintf(logFile, "\nwrite-uuter: owned-process cleanup failed: %v\n", err)
	}
	trackerClosed = true
	if status != 0 {
		return fmt.Errorf("%s exited with status %d", *provider, status)
	}
	return nil
}

// providerArguments builds the exact non-interactive argument vector for one
// provider. Model and reasoning effort are always explicit: neither CLI is
// allowed to pick a default, and neither is given a fallback model.
func providerArguments(provider, workspace, model, effort string) []string {
	switch provider {
	case providerClaudeCode:
		// --safe-mode disables user customizations while leaving Max OAuth
		// usable; --bare is forbidden because it disables OAuth and keychain
		// reads and accepts only an API key. --no-session-persistence keeps
		// conversation state out of the user's Claude directories. The prompt
		// arrives on stdin.
		return []string{
			"--print",
			"--safe-mode",
			"--dangerously-skip-permissions",
			"--no-session-persistence",
			"--model", model,
			"--effort", effort,
		}
	default:
		return []string{
			"--model", model,
			"--config", "model_reasoning_effort=" + strconv.Quote(effort),
			"--dangerously-bypass-approvals-and-sandbox", "-C", workspace,
			"exec", "--ephemeral", "--ignore-user-config", "--ignore-rules",
			"--skip-git-repo-check", "-",
		}
	}
}

// providerBaseEnvironment is the allowlisted host environment every provider
// process and the Claude authentication preflight receive. ANTHROPIC_API_KEY
// and every alternative API, Bedrock, Vertex, Foundry, or provider-selection
// credential is absent by construction, so an ambient credential can never
// move a run to API billing or another provider.
func providerBaseEnvironment() []string {
	allowed := []string{
		"HOME", "USER", "LOGNAME", "PATH", "SHELL", "LANG", "LC_ALL", "TERM",
		"SSL_CERT_FILE", "SSL_CERT_DIR",
	}
	environment := make([]string, 0, len(allowed)+8)
	for _, key := range allowed {
		if value, found := os.LookupEnv(key); found {
			environment = append(environment, key+"="+value)
		}
	}
	for _, key := range []string{"HTTP_PROXY", "HTTPS_PROXY"} {
		if value, found := os.LookupEnv(key); found && proxyWithoutUserinfo(value) {
			environment = append(environment, key+"="+value)
		}
	}
	return environment
}

// controllerCommandEnvironment is the environment for a controller-owned
// helper process such as the tmux client. Screenshot credentials are removed:
// the controller performs the authenticated capture in-process, so no child -
// not even the tmux server that later starts the agent runner - has a reason
// to inherit them.
func controllerCommandEnvironment() []string {
	environment := os.Environ()
	result := make([]string, 0, len(environment))
	for _, entry := range environment {
		name, _, found := strings.Cut(entry, "=")
		if found && (name == screenshotAccountEnv || name == screenshotTokenEnv) {
			continue
		}
		result = append(result, entry)
	}
	return result
}

func agentEnvironment(provider, workspace, providerHome, role, lens string, candidate int, revision, invocation string) []string {
	environment := providerBaseEnvironment()
	switch provider {
	case providerClaudeCode:
		// Claude Code keeps a scratch root at <CLAUDE_CODE_TMPDIR>/claude-<uid>
		// and defaults it to /tmp, which is a shared user path outside this run.
		// Point it at the run-owned provider home instead, so nothing this
		// invocation writes outlives the run. HOME is deliberately left alone:
		// the Max session is resolved from the user's top-level Claude
		// configuration file and the login keychain, and the sandbox - not the
		// environment - is what keeps every other part of the user's Claude
		// state out of reach.
		environment = append(environment, "CLAUDE_CODE_TMPDIR="+providerHome)
	default:
		environment = append(environment, "CODEX_HOME="+providerHome)
	}
	environment = append(environment,
		"WRITE_UUTER_WORK_DIR="+workspace,
		"WRITE_UUTER_ROLE="+role,
		"WRITE_UUTER_LENS="+lens,
		"WRITE_UUTER_CANDIDATE="+strconv.Itoa(candidate),
		"WRITE_UUTER_REVISION="+revision,
		"WRITE_UUTER_INVOCATION="+invocation,
		"TMPDIR="+filepath.Join(workspace, ".tmp"),
	)
	return environment
}

func replaceEnvironmentValue(environment []string, key, value string) []string {
	prefix := key + "="
	result := make([]string, 0, len(environment)+1)
	for _, entry := range environment {
		if !strings.HasPrefix(entry, prefix) {
			result = append(result, entry)
		}
	}
	return append(result, prefix+value)
}

func proxyWithoutUserinfo(value string) bool {
	if value == "" || strings.IndexFunc(value, func(r rune) bool { return r < 0x20 || r == 0x7f }) >= 0 {
		return false
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.User != nil || parsed.Opaque != "" || parsed.Host == "" || parsed.Hostname() == "" {
		return false
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" || parsed.Path != "" || parsed.RawPath != "" || parsed.RawQuery != "" || parsed.Fragment != "" || parsed.RawFragment != "" || parsed.ForceQuery {
		return false
	}
	return !strings.ContainsAny(value, "@%")
}

func publishJSONAtomic(path string, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+"-*")
	if err != nil {
		return err
	}
	name := temporary.Name()
	keep := false
	defer func() {
		_ = temporary.Close()
		if !keep {
			_ = os.Remove(name)
		}
	}()
	if err := temporary.Chmod(0o600); err != nil {
		return err
	}
	if _, err := temporary.Write(data); err != nil {
		return err
	}
	if err := temporary.Sync(); err != nil {
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	if err := os.Rename(name, path); err != nil {
		return err
	}
	keep = true
	return nil
}

func publishInteger(path string, value int, delayText string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+"-*")
	if err != nil {
		return err
	}
	name := temporary.Name()
	keep := false
	defer func() {
		_ = temporary.Close()
		if !keep {
			_ = os.Remove(name)
		}
	}()
	if delayText != "" && strings.Contains(filepath.Base(path), ".exit") {
		if delay, parseErr := time.ParseDuration(delayText); parseErr == nil {
			time.Sleep(delay)
		}
	}
	if _, err := fmt.Fprintf(temporary, "%d\n", value); err != nil {
		return err
	}
	if err := temporary.Sync(); err != nil {
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	if err := os.Rename(name, path); err != nil {
		return err
	}
	keep = true
	return nil
}
