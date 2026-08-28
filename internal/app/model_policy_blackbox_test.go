package app_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
)

// policyRoleForInvocation maps a controller invocation identifier to the policy
// role key that must have produced its profile.
func policyRoleForInvocation(record invocationRecord) string {
	if record.Lens != "" {
		return "reviewer_" + record.Lens
	}
	return record.Role
}

func readPolicyArtifact(t *testing.T, runDir string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(runDir, "model-policy.json"))
	if err != nil {
		t.Fatalf("run does not retain the validated policy: %v", err)
	}
	return data
}

func readAuditRecords(t *testing.T, runDir string) map[string]auditRecord {
	t.Helper()
	paths, err := filepath.Glob(filepath.Join(runDir, ".control", "invocations", "*.json"))
	if err != nil {
		t.Fatal(err)
	}
	records := make(map[string]auditRecord, len(paths))
	for _, path := range paths {
		info, statErr := os.Lstat(path)
		if statErr != nil {
			t.Fatal(statErr)
		}
		if info.Mode().Perm()&0o222 != 0 {
			t.Errorf("invocation audit %s is writable: %v", filepath.Base(path), info.Mode())
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			t.Fatal(readErr)
		}
		var record auditRecord
		if err := json.Unmarshal(data, &record); err != nil {
			t.Fatalf("invalid invocation audit %s: %v", filepath.Base(path), err)
		}
		if strings.Contains(string(data), "prompt") || strings.Contains(strings.ToUpper(string(data)), "ANTHROPIC") || strings.Contains(string(data), "auth") {
			t.Errorf("invocation audit %s recorded prompt/authentication material: %s", filepath.Base(path), data)
		}
		records[record.Invocation] = record
	}
	return records
}

func argumentValue(arguments []string, name string) (string, bool) {
	for index, argument := range arguments {
		if argument == name && index+1 < len(arguments) {
			return arguments[index+1], true
		}
	}
	return "", false
}

func hasArgument(arguments []string, name string) bool {
	for _, argument := range arguments {
		if argument == name {
			return true
		}
	}
	return false
}

// assertLaunchedProfile checks that one launched process received exactly the
// declared provider executable, model, and reasoning effort, and that its audit
// record agrees with the arguments the process actually observed.
func assertLaunchedProfile(t *testing.T, record invocationRecord, audit auditRecord, digest string) {
	t.Helper()
	role := policyRoleForInvocation(record)
	want, declared := checkedInPolicy[role]
	if !declared {
		t.Fatalf("invocation %s used undeclared role %q", record.Invocation, role)
	}
	wantTag := "codex"
	if want.Provider == "claude_code" {
		wantTag = "claude"
	}
	if record.ExecutableTag != wantTag {
		t.Errorf("%s ran the %q fixture executable, want %q (executable %s)", role, record.ExecutableTag, wantTag, record.Executable)
	}
	model, hasModel := argumentValue(record.Args, "--model")
	if !hasModel || model != want.Model {
		t.Errorf("%s launched with model %q (present=%v), want %q: %v", role, model, hasModel, want.Model, record.Args)
	}
	switch want.Provider {
	case "claude_code":
		effort, hasEffort := argumentValue(record.Args, "--effort")
		if !hasEffort || effort != want.ReasoningEffort {
			t.Errorf("%s launched with Claude effort %q (present=%v), want %q: %v", role, effort, hasEffort, want.ReasoningEffort, record.Args)
		}
		for _, required := range []string{"--print", "--safe-mode", "--dangerously-skip-permissions", "--no-session-persistence"} {
			if !hasArgument(record.Args, required) {
				t.Errorf("%s Claude invocation is missing %s: %v", role, required, record.Args)
			}
		}
		if hasArgument(record.Args, "--bare") {
			t.Errorf("%s Claude invocation used the forbidden --bare mode: %v", role, record.Args)
		}
		for _, forbidden := range []string{"--fallback-model", "--settings", "--resume", "--continue"} {
			if hasArgument(record.Args, forbidden) {
				t.Errorf("%s Claude invocation used %s: %v", role, forbidden, record.Args)
			}
		}
	default:
		effort, hasEffort := argumentValue(record.Args, "--config")
		want := "model_reasoning_effort=" + fmt.Sprintf("%q", want.ReasoningEffort)
		if !hasEffort || effort != want {
			t.Errorf("%s launched with Codex config %q (present=%v), want %q: %v", role, effort, hasEffort, want, record.Args)
		}
		for _, required := range []string{"exec", "--ephemeral", "--ignore-user-config"} {
			if !hasArgument(record.Args, required) {
				t.Errorf("%s Codex invocation is missing %s: %v", role, required, record.Args)
			}
		}
	}
	if audit.Provider != want.Provider || audit.Model != want.Model || audit.ReasoningEffort != want.ReasoningEffort {
		t.Errorf("%s audit record %+v does not match the declared profile %+v", role, audit, want)
	}
	if audit.ModelPolicyDigest != digest {
		t.Errorf("%s audit record digest %q does not match the copied policy digest %q", role, audit.ModelPolicyDigest, digest)
	}
	if audit.Lens != record.Lens || audit.Candidate != record.Candidate {
		t.Errorf("%s audit record identity %+v does not match the launched invocation %s/%s/%d", role, audit, record.Role, record.Lens, record.Candidate)
	}
}

// assertPolicyBinding checks the run-level policy artifacts and every launched
// invocation against the checked-in table.
func assertPolicyBinding(t *testing.T, runDir, fixtureDir string, wantInvocations int) []invocationRecord {
	t.Helper()
	bundled, err := os.ReadFile(filepath.Join(repositoryRoot(t), "prompts", "models.json"))
	if err != nil {
		t.Fatal(err)
	}
	copied := readPolicyArtifact(t, runDir)
	if string(copied) != string(bundled) {
		t.Fatalf("run policy copy differs from the validated bundle policy")
	}
	digest := revision(copied)
	var workflow struct {
		ModelPolicyDigest string            `json:"model_policy_digest"`
		ArtifactPaths     map[string]string `json:"artifact_paths"`
	}
	data, err := os.ReadFile(filepath.Join(runDir, "workflow.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &workflow); err != nil {
		t.Fatal(err)
	}
	if workflow.ModelPolicyDigest != digest {
		t.Fatalf("workflow digest %q does not match the copied policy %q", workflow.ModelPolicyDigest, digest)
	}
	if workflow.ArtifactPaths["model_policy"] != "model-policy.json" {
		t.Errorf("workflow does not publish the model policy path: %v", workflow.ArtifactPaths)
	}
	audits := readAuditRecords(t, runDir)
	records := readInvocationRecords(t, fixtureDir)
	if len(records) != wantInvocations {
		t.Fatalf("got %d launched invocations, want %d", len(records), wantInvocations)
	}
	if len(audits) != wantInvocations {
		t.Fatalf("got %d invocation audit records, want %d", len(audits), wantInvocations)
	}
	for _, record := range records {
		audit, found := audits[record.Invocation]
		if !found {
			t.Fatalf("invocation %s launched without an audit record", record.Invocation)
		}
		assertLaunchedProfile(t, record, audit, digest)
	}
	return records
}

func TestBlackBoxEveryRoleLaunchesItsDeclaredProfile(t *testing.T) {
	run := executeScenario(t, "happy")
	if run.err != nil {
		t.Fatalf("CLI failed: %v\n%s", run.err, run.output)
	}
	// PM, researcher, story editor, Writer prose draft, Visual Editor, Writer
	// assembly, and four review lenses.
	records := assertPolicyBinding(t, run.runDir, run.fixtureDir, 10)
	seen := map[string]bool{}
	for _, record := range records {
		seen[policyRoleForInvocation(record)] = true
	}
	for role := range checkedInPolicy {
		if !seen[role] {
			t.Errorf("role %s never launched", role)
		}
	}
}

// A revision must reuse the same declared profiles: reviewer lenses keep their
// own identities and the Writer does not drift to another model.
func TestBlackBoxRevisionsPreserveDeclaredProfiles(t *testing.T) {
	run := executeScenario(t, "mustfix_once")
	if run.err != nil {
		t.Fatalf("CLI failed: %v\n%s", run.err, run.output)
	}
	// PM, researcher, story editor, the three-pass candidate sequence twice,
	// the evidence review of candidate 1, and four reviews of candidate 2.
	records := assertPolicyBinding(t, run.runDir, run.fixtureDir, 14)
	writers := 0
	visualEditors := 0
	for _, record := range records {
		switch record.Role {
		case "writer":
			writers++
		case "visual_editor":
			visualEditors++
		}
	}
	if writers != 4 || visualEditors != 2 {
		t.Fatalf("got %d writer and %d visual editor invocations, want 4 and 2", writers, visualEditors)
	}
}

// Reviewer lenses select distinct profiles: two Codex lenses and two Claude
// lenses, with different models and efforts inside each provider.
func TestBlackBoxReviewerLensesSelectDistinctProfiles(t *testing.T) {
	run := executeScenario(t, "happy")
	if run.err != nil {
		t.Fatalf("CLI failed: %v\n%s", run.err, run.output)
	}
	audits := readAuditRecords(t, run.runDir)
	found := map[string]auditRecord{}
	for _, audit := range audits {
		if audit.Role == "reviewer" {
			found[audit.Lens] = audit
		}
	}
	for lens, want := range map[string]auditRecord{
		"evidence": checkedInPolicy["reviewer_evidence"],
		"story":    checkedInPolicy["reviewer_story"],
		"clarity":  checkedInPolicy["reviewer_clarity"],
		"copy":     checkedInPolicy["reviewer_copy"],
	} {
		got, present := found[lens]
		if !present {
			t.Fatalf("no audit record for reviewer lens %s", lens)
		}
		if got.Provider != want.Provider || got.Model != want.Model || got.ReasoningEffort != want.ReasoningEffort {
			t.Errorf("reviewer lens %s used %s/%s/%s, want %s/%s/%s", lens,
				got.Provider, got.Model, got.ReasoningEffort, want.Provider, want.Model, want.ReasoningEffort)
		}
	}
}

// No ambient credential or provider-selection variable may reach a provider
// process, and no ambient model variable may change what was launched.
func TestBlackBoxProviderProcessesNeverReceiveExternalCredentials(t *testing.T) {
	binary, fake, runDir, fixtureDir := prepareScenario(t, "happy")
	command := newRunCommand(t, binary, fake, runDir, "5s", filepath.Join(repositoryRoot(t), "examples", "brief.md"))
	command.Env = append(command.Env,
		"ANTHROPIC_API_KEY=test-anthropic-key",
		"ANTHROPIC_AUTH_TOKEN=test-anthropic-token",
		"ANTHROPIC_BASE_URL=https://proxy.invalid",
		"ANTHROPIC_MODEL=claude-fable-5",
		"CLAUDE_CODE_USE_BEDROCK=1",
		"CLAUDE_CODE_USE_VERTEX=1",
		"AWS_BEARER_TOKEN_BEDROCK=test-bedrock-token",
		"AWS_ACCESS_KEY_ID=test-aws-key",
		"GOOGLE_APPLICATION_CREDENTIALS=/tmp/does-not-exist.json",
		"OPENAI_API_KEY=test-openai-key",
	)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("CLI failed with hostile ambient provider environment: %v\n%s", err, output)
	}
	records := assertPolicyBinding(t, runDir, fixtureDir, 10)
	realHome, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	for _, record := range records {
		for name, value := range record.Environment {
			if value == "PRESENT" {
				t.Errorf("%s received external provider credential %s", record.Role, name)
			}
		}
		if checkedInPolicy[policyRoleForInvocation(record)].Provider == "claude_code" {
			// The Max session is resolved from the user's account record, so
			// HOME stays real; the sandbox, not the environment, is what keeps
			// the rest of the user's Claude state out of reach. Everything the
			// invocation writes must land in run-owned scratch.
			if record.Environment["HOME"] != realHome {
				t.Errorf("%s Claude invocation did not receive the authenticated user home: %q", record.Role, record.Environment["HOME"])
			}
			if !strings.Contains(record.Environment["CLAUDE_CODE_TMPDIR"], "provider-homes") {
				t.Errorf("%s Claude invocation kept a shared scratch root: %q", record.Role, record.Environment["CLAUDE_CODE_TMPDIR"])
			}
			if record.Environment["CODEX_HOME"] != "ABSENT" {
				t.Errorf("%s Claude invocation received a Codex home: %q", record.Role, record.Environment["CODEX_HOME"])
			}
		} else if !strings.Contains(record.Environment["CODEX_HOME"], "provider-homes") {
			t.Errorf("%s Codex invocation lost its private Codex home: %q", record.Role, record.Environment["CODEX_HOME"])
		}
	}
}

// --codex and --claude select separate executables, and a provider the policy
// never references is not required to exist.
func TestBlackBoxProviderExecutableSelectionIsIndependent(t *testing.T) {
	t.Run("separate executables", func(t *testing.T) {
		run := executeScenario(t, "happy")
		if run.err != nil {
			t.Fatalf("CLI failed: %v\n%s", run.err, run.output)
		}
		tags := map[string]bool{}
		for _, record := range readInvocationRecords(t, run.fixtureDir) {
			tags[record.ExecutableTag] = true
			if !strings.Contains(record.Executable, "/clients/") {
				t.Errorf("%s did not run a single-use staged client: %s", record.Role, record.Executable)
			}
		}
		if !tags["codex"] || !tags["claude"] {
			t.Fatalf("mixed-provider run did not use both fixtures: %v", tags)
		}
	})

	t.Run("unused provider is not required", func(t *testing.T) {
		promptsDir := promptsDirWithPolicy(t, codexOnlyPolicy)
		binary, fake, runDir, fixtureDir := prepareScenario(t, "happy")
		command := newRunCommandWithPrompts(t, binary, fake, runDir, "5s", filepath.Join(repositoryRoot(t), "examples", "brief.md"), promptsDir)
		command.Args = replaceArgument(command.Args, "--claude", filepath.Join(t.TempDir(), "no-such-claude"))
		if output, err := command.CombinedOutput(); err != nil {
			t.Fatalf("codex-only policy required an unused Claude executable: %v\n%s", err, output)
		}
		for _, record := range readInvocationRecords(t, fixtureDir) {
			if record.ExecutableTag != "codex" {
				t.Errorf("codex-only policy launched the %q fixture for %s", record.ExecutableTag, record.Role)
			}
		}
		for _, audit := range readAuditRecords(t, runDir) {
			if audit.Provider != "codex" {
				t.Errorf("codex-only policy recorded provider %q", audit.Provider)
			}
		}
	})
}

// The sanitized Max preflight decides before anything is created.
func TestBlackBoxClaudeMaxPreflightRunsBeforeRunCreation(t *testing.T) {
	for _, testCase := range []struct {
		scenario string
		reason   string
	}{
		{"auth_logged_out", "logged out"},
		{"auth_api_key", "authMethod"},
		{"auth_not_max", "subscriptionType"},
		{"auth_malformed", "Claude Max preflight"},
		{"auth_missing_field", "omitted loggedIn"},
		{"auth_duplicate_key", "duplicate JSON key"},
		{"auth_two_documents", "more than one JSON document"},
		{"auth_nonzero", "auth status` failed"},
		{"auth_oversized", "produced more than"},
	} {
		t.Run(testCase.scenario, func(t *testing.T) {
			run := executeScenario(t, testCase.scenario)
			if run.err == nil {
				t.Fatalf("CLI accepted an unusable Claude session: %s", run.output)
			}
			if !strings.Contains(run.output, testCase.reason) {
				t.Errorf("preflight failure is not actionable: %s", run.output)
			}
			if _, err := os.Stat(run.runDir); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("failed Claude preflight created a run directory: %v", err)
			}
			if records := readInvocationRecords(t, run.fixtureDir); len(records) != 0 {
				t.Fatalf("failed Claude preflight launched %d agents", len(records))
			}
			for _, secret := range []string{"fixture@example.invalid", "00000000-0000-0000-0000-000000000000"} {
				if strings.Contains(run.output, secret) {
					t.Errorf("preflight disclosed account identity: %s", run.output)
				}
			}
		})
	}

	t.Run("missing executable", func(t *testing.T) {
		binary, fake, runDir, _ := prepareScenario(t, "happy")
		command := newRunCommand(t, binary, fake, runDir, "5s", filepath.Join(repositoryRoot(t), "examples", "brief.md"))
		command.Args = replaceArgument(command.Args, "--claude", filepath.Join(t.TempDir(), "no-such-claude"))
		output, err := command.CombinedOutput()
		if err == nil || !strings.Contains(string(output), "locate Claude Code executable") {
			t.Fatalf("missing Claude executable was not reported before the run: %v\n%s", err, output)
		}
		if _, err := os.Stat(runDir); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("missing Claude executable created a run directory: %v", err)
		}
	})
}

// Every invalid policy fails before the run directory exists, so an invalid
// configuration never leaves partial state.
func TestBlackBoxInvalidPolicyFailsBeforeRunCreation(t *testing.T) {
	valid := func() map[string]any {
		var document map[string]any
		data, err := os.ReadFile(filepath.Join(repositoryRoot(t), "prompts", "models.json"))
		if err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(data, &document); err != nil {
			t.Fatal(err)
		}
		return document
	}
	mutate := func(change func(roles map[string]any)) string {
		document := valid()
		change(document["roles"].(map[string]any))
		data, err := json.MarshalIndent(document, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		return string(data) + "\n"
	}

	for _, testCase := range []struct {
		name   string
		policy string
		reason string
	}{
		{"missing file", "", "required prompt models.json"},
		{"empty file", "   \n", "prompt is empty"},
		{"unsupported schema", `{"schema_version": 2, "roles": {}}`, "unsupported models.json schema_version 2"},
		{"duplicate key", `{"schema_version": 1, "schema_version": 1, "roles": {}}`, "duplicate JSON key"},
		{"unknown top-level field", `{"schema_version": 1, "roles": {}, "default": {"provider": "codex"}}`, "unknown field"},
		{"missing role", mutate(func(roles map[string]any) { delete(roles, "reviewer_copy") }), "missing required role profile(s) reviewer_copy"},
		{"unknown role", mutate(func(roles map[string]any) {
			roles["layout_editor"] = map[string]any{"provider": "claude_code", "model": "claude-opus-5", "reasoning_effort": "medium"}
		}), "unknown role(s) layout_editor"},
		{"unknown role field", mutate(func(roles map[string]any) {
			roles["writer"] = map[string]any{"provider": "claude_code", "model": "claude-opus-5", "reasoning_effort": "medium", "fallback": "claude-fable-5"}
		}), "unknown field"},
		{"unsupported provider", mutate(func(roles map[string]any) {
			roles["writer"] = map[string]any{"provider": "bedrock", "model": "claude-opus-5", "reasoning_effort": "medium"}
		}), "unsupported provider"},
		{"empty model", mutate(func(roles map[string]any) {
			roles["writer"] = map[string]any{"provider": "claude_code", "model": "", "reasoning_effort": "medium"}
		}), "empty model"},
		{"invalid effort for provider", mutate(func(roles map[string]any) {
			roles["pm"] = map[string]any{"provider": "codex", "model": "gpt-5.6-sol", "reasoning_effort": "xhigh"}
		}), "does not accept"},
		{"unknown effort", mutate(func(roles map[string]any) {
			roles["writer"] = map[string]any{"provider": "claude_code", "model": "claude-opus-5", "reasoning_effort": "extreme"}
		}), "does not accept"},
		{"claude model on codex", mutate(func(roles map[string]any) {
			roles["pm"] = map[string]any{"provider": "codex", "model": "claude-opus-5", "reasoning_effort": "high"}
		}), "assigns claude model"},
		{"gpt model on claude", mutate(func(roles map[string]any) {
			roles["writer"] = map[string]any{"provider": "claude_code", "model": "gpt-5.6-sol", "reasoning_effort": "medium"}
		}), "assigns gpt model"},
		{"missing roles object", `{"schema_version": 1}`, "missing the roles object"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			promptsDir := promptsDirWithPolicy(t, testCase.policy)
			binary, fake, runDir, fixtureDir := prepareScenario(t, "happy")
			command := newRunCommandWithPrompts(t, binary, fake, runDir, "5s", filepath.Join(repositoryRoot(t), "examples", "brief.md"), promptsDir)
			output, err := command.CombinedOutput()
			if err == nil {
				t.Fatalf("CLI accepted an invalid policy: %s", output)
			}
			if !strings.Contains(string(output), testCase.reason) {
				t.Fatalf("policy rejection is not actionable: want %q, got:\n%s", testCase.reason, output)
			}
			if _, err := os.Stat(runDir); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("invalid policy created a run directory: %v", err)
			}
			if records := readInvocationRecords(t, fixtureDir); len(records) != 0 {
				t.Fatalf("invalid policy launched %d agents", len(records))
			}
		})
	}
}

// The policy is bound at validation time. Replacing the bundle, its parent, or
// the policy file afterwards cannot change a launched profile or the recorded
// digest.
func TestBlackBoxPolicyMutationAfterValidationCannotChangeLaunchedProfiles(t *testing.T) {
	parent := t.TempDir()
	promptsDir := filepath.Join(parent, "prompts")
	copyPromptBundle(t, filepath.Join(repositoryRoot(t), "prompts"), promptsDir)
	original, err := os.ReadFile(filepath.Join(promptsDir, "models.json"))
	if err != nil {
		t.Fatal(err)
	}

	binary, fake, runDir, fixtureDir := prepareScenario(t, "happy")
	barrier := filepath.Join(t.TempDir(), "commit")
	command := newRunCommandWithPrompts(t, binary, fake, runDir, "20s", filepath.Join(repositoryRoot(t), "examples", "brief.md"), promptsDir)
	command.Env = append(command.Env, "WRITE_UUTER_TEST_COMMIT_BARRIER="+barrier)
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	waitForPath(t, barrier+".ready")
	// Replace the validated policy file, then the whole validated bundle root.
	if err := os.Remove(filepath.Join(promptsDir, "models.json")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(promptsDir, "models.json"), []byte(hostilePolicy), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(promptsDir, filepath.Join(parent, "prompts-moved")); err != nil {
		t.Fatal(err)
	}
	copyPromptBundle(t, filepath.Join(repositoryRoot(t), "prompts"), promptsDir)
	if err := os.WriteFile(filepath.Join(promptsDir, "models.json"), []byte(hostilePolicy), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(barrier+".continue", []byte("continue\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := command.Wait(); err != nil {
		t.Fatalf("CLI failed after prompt-root mutation: %v", err)
	}

	copied := readPolicyArtifact(t, runDir)
	if string(copied) != string(original) {
		t.Fatalf("mutated policy reached the durable run copy")
	}
	digest := revision(original)
	for _, record := range readInvocationRecords(t, fixtureDir) {
		assertLaunchedProfile(t, record, readAuditRecords(t, runDir)[record.Invocation], digest)
	}
}

// An unavailable model or provider blocks the run with the effective profile
// and never retries with another model, provider, or account.
func TestBlackBoxUnavailableModelBlocksWithoutFallback(t *testing.T) {
	run := executeScenario(t, "model_unavailable")
	if run.err == nil {
		t.Fatalf("CLI succeeded despite an unavailable model: %s", run.output)
	}
	state := readWorkflow(t, run.runDir)
	if state.Status != "blocked" {
		t.Fatalf("unexpected workflow: %+v", state)
	}
	for _, expected := range []string{"provider=claude_code", "model=claude-opus-5", "reasoning_effort=medium"} {
		if !strings.Contains(state.BlockReason, expected) {
			t.Errorf("blocked run does not record the effective profile %q: %s", expected, state.BlockReason)
		}
	}
	assertNoArticle(t, run.runDir)
	records := readInvocationRecords(t, run.fixtureDir)
	audits := readAuditRecords(t, run.runDir)
	if len(audits) != len(records) {
		t.Fatalf("blocked run retained %d audit records for %d launched invocations", len(audits), len(records))
	}
	digest := revision(readPolicyArtifact(t, run.runDir))
	writers := 0
	for _, record := range records {
		if record.Role == "writer" {
			writers++
		}
		assertLaunchedProfile(t, record, audits[record.Invocation], digest)
	}
	if writers != 1 {
		t.Fatalf("blocked run retried the writer %d times", writers)
	}
	assertProcessesGone(t, records)
}

// The staged Claude client may use the keychain store that backs the Max
// session, while its model-invoked tools may not, and neither may read the
// user's Claude configuration, history, or session state.
func TestBlackBoxClaudeKeychainAccessIsProcessScoped(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("OS-enforced isolation is only implemented on darwin")
	}
	run := executeScenario(t, "filesystem_isolation")
	if run.err != nil {
		t.Fatalf("CLI failed: %v\n%s", run.err, run.output)
	}
	// 009-reviewer-clarity is a Claude Code invocation in the checked-in policy.
	data, err := os.ReadFile(filepath.Join(run.fixtureDir, "logs", "isolation-009-reviewer-clarity.probe"))
	if err != nil {
		t.Fatalf("Claude reviewer did not publish an isolation probe: %v", err)
	}
	var probes map[string]string
	if err := json.Unmarshal(data, &probes); err != nil {
		t.Fatal(err)
	}
	// The account record identifies the logged-in Max session, so the exact
	// staged client - and nothing else - may read it.
	if !strings.Contains(probes["client_user_claude_config"], "READ_SUCCEEDED") {
		t.Errorf("staged Claude client cannot read the account record that identifies its session: %q", probes["client_user_claude_config"])
	}
	for _, label := range []string{
		"client_user_claude_dir", "client_user_home", "client_keychain_file",
		"tool_user_claude_config", "tool_keychain_file", "tool_keychain_client",
	} {
		result := strings.ToLower(probes[label])
		if result == "" || strings.Contains(result, "succeeded") {
			t.Errorf("Claude boundary leaked %s: %q", label, probes[label])
		}
	}
}

func replaceArgument(arguments []string, name, value string) []string {
	result := append([]string(nil), arguments...)
	for index, argument := range result {
		if argument == name && index+1 < len(result) {
			result[index+1] = value
		}
	}
	return result
}

// promptsDirWithPolicy copies the checked-in bundle and replaces its policy.
// An empty policy removes the file, which is the missing-policy case.
func promptsDirWithPolicy(t *testing.T, policy string) string {
	t.Helper()
	directory := filepath.Join(t.TempDir(), "prompts")
	copyPromptBundle(t, filepath.Join(repositoryRoot(t), "prompts"), directory)
	target := filepath.Join(directory, "models.json")
	if policy == "" {
		if err := os.Remove(target); err != nil {
			t.Fatal(err)
		}
		return directory
	}
	if err := os.WriteFile(target, []byte(policy), 0o600); err != nil {
		t.Fatal(err)
	}
	return directory
}

func copyPromptBundle(t *testing.T, source, target string) {
	t.Helper()
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(source)
	if err != nil {
		t.Fatal(err)
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.Type().IsRegular() {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)
	for _, name := range names {
		copyFile(t, filepath.Join(source, name), filepath.Join(target, name), 0o644)
	}
}

const codexOnlyPolicy = `{
  "schema_version": 1,
  "roles": {
    "pm": {"provider": "codex", "model": "gpt-5.6-sol", "reasoning_effort": "high"},
    "researcher": {"provider": "codex", "model": "gpt-5.6-luna", "reasoning_effort": "medium"},
    "story_editor": {"provider": "codex", "model": "gpt-5.6-sol", "reasoning_effort": "high"},
    "visual_editor": {"provider": "codex", "model": "gpt-5.6-sol", "reasoning_effort": "high"},
    "writer": {"provider": "codex", "model": "gpt-5.6-sol", "reasoning_effort": "medium"},
    "reviewer_evidence": {"provider": "codex", "model": "gpt-5.6-sol", "reasoning_effort": "medium"},
    "reviewer_story": {"provider": "codex", "model": "gpt-5.6-luna", "reasoning_effort": "medium"},
    "reviewer_clarity": {"provider": "codex", "model": "gpt-5.6-luna", "reasoning_effort": "medium"},
    "reviewer_copy": {"provider": "codex", "model": "gpt-5.6-luna", "reasoning_effort": "low"}
  }
}
`

const hostilePolicy = `{
  "schema_version": 1,
  "roles": {
    "pm": {"provider": "codex", "model": "gpt-hostile", "reasoning_effort": "low"},
    "researcher": {"provider": "codex", "model": "gpt-hostile", "reasoning_effort": "low"},
    "story_editor": {"provider": "codex", "model": "gpt-hostile", "reasoning_effort": "low"},
    "visual_editor": {"provider": "codex", "model": "gpt-hostile", "reasoning_effort": "low"},
    "writer": {"provider": "codex", "model": "gpt-hostile", "reasoning_effort": "low"},
    "reviewer_evidence": {"provider": "codex", "model": "gpt-hostile", "reasoning_effort": "low"},
    "reviewer_story": {"provider": "codex", "model": "gpt-hostile", "reasoning_effort": "low"},
    "reviewer_clarity": {"provider": "codex", "model": "gpt-hostile", "reasoning_effort": "low"},
    "reviewer_copy": {"provider": "codex", "model": "gpt-hostile", "reasoning_effort": "low"}
  }
}
`

// Admin-managed Claude policy must not become readable or effective through
// this sandbox boundary. `--safe-mode` disables user customization but still
// applies admin-managed policy, and managed settings can inject environment
// values, API keys, base URLs, provider routing, models, helper commands, and
// hooks - a run could otherwise diverge from the profile the controller
// validated and recorded without that divergence appearing in argv or audit.
func TestBlackBoxAdminManagedClaudeSettingsCannotCrossSandboxBoundary(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("OS-enforced isolation is only implemented on darwin")
	}
	run := executeScenario(t, "filesystem_isolation")
	if run.err != nil {
		t.Fatalf("CLI failed: %v\n%s", run.err, run.output)
	}
	// 009-reviewer-clarity is a Claude Code invocation in the checked-in policy.
	data, err := os.ReadFile(filepath.Join(run.fixtureDir, "logs", "isolation-009-reviewer-clarity.probe"))
	if err != nil {
		t.Fatalf("Claude reviewer did not publish an isolation probe: %v", err)
	}
	var probes map[string]string
	if err := json.Unmarshal(data, &probes); err != nil {
		t.Fatal(err)
	}
	for _, label := range []string{
		"client_managed_settings_root", "client_managed_settings",
		"tool_managed_settings", "tool_managed_settings_db",
	} {
		result := strings.ToLower(probes[label])
		if result == "" || strings.Contains(result, "succeeded") {
			t.Errorf("admin-managed Claude policy is reachable through %s: %q", label, probes[label])
		}
	}
	// The containing directory exists on every macOS host, so its denial has to
	// be a sandbox refusal. Without this the whole check could pass merely
	// because no managed tree happens to be installed.
	root := strings.ToLower(probes["client_managed_settings_root"])
	if !strings.Contains(root, "not permitted") && !strings.Contains(root, "permission denied") {
		t.Errorf("managed-policy denial is absence, not a sandbox refusal: %q", probes["client_managed_settings_root"])
	}
	// No invocation may diverge from the recorded profile: the run policy copy,
	// the workflow digest, every audit record, and every launched argv still
	// agree after the boundary probes have run.
	assertPolicyBinding(t, run.runDir, run.fixtureDir, 10)
}
