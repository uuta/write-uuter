package app_test

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
)

type workflowState struct {
	Status             string `json:"status"`
	Phase              string `json:"phase"`
	CurrentCandidate   int    `json:"current_candidate"`
	CurrentRevision    string `json:"current_revision"`
	ReviewAttemptCount int    `json:"review_attempt_count"`
	BlockReason        string `json:"block_reason"`
}

type invocationRecord struct {
	PID            int               `json:"pid"`
	Role           string            `json:"role"`
	Lens           string            `json:"lens"`
	Candidate      int               `json:"candidate"`
	Revision       string            `json:"revision"`
	Invocation     string            `json:"invocation"`
	Prompt         string            `json:"prompt"`
	Workspace      string            `json:"workspace"`
	WorkspaceFiles []string          `json:"workspace_files"`
	Isolation      map[string]string `json:"isolation"`
}

var (
	buildOnce sync.Once
	buildDir  string
	buildErr  error
)

func TestBlackBoxHappyPathAndReviewerIsolation(t *testing.T) {
	run := executeScenario(t, "happy")
	if run.err != nil {
		t.Fatalf("CLI failed: %v\n%s", run.err, run.output)
	}
	state := readWorkflow(t, run.runDir)
	if state.Status != "succeeded" || state.Phase != "complete" {
		t.Fatalf("unexpected workflow: %+v", state)
	}
	if state.CurrentCandidate != 1 || state.ReviewAttemptCount != 4 {
		t.Fatalf("unexpected candidate/review count: %+v", state)
	}
	requiredFiles := []string{
		"brief.md", "workflow.json", "evidence/sources.md", "claim-ledger.md", "outline.md",
		"drafts/article-001.md", "pm-decisions/article-001.md", "article.md",
	}
	for _, lens := range []string{"evidence", "story", "clarity", "copy"} {
		requiredFiles = append(requiredFiles,
			filepath.Join("reviews", "article-001", lens, "result.json"),
			filepath.Join("reviews", "article-001", lens, "report.md"),
		)
	}
	for _, relative := range requiredFiles {
		data, err := os.ReadFile(filepath.Join(run.runDir, relative))
		if err != nil || len(data) == 0 {
			t.Errorf("required artifact %s is missing/empty: %v", relative, err)
		}
	}
	assertExactFiles(t, filepath.Join(run.runDir, "drafts", "article-001.md"), filepath.Join(run.runDir, "article.md"))
	article, _ := os.ReadFile(filepath.Join(run.runDir, "article.md"))
	for _, lens := range []string{"evidence", "story", "clarity", "copy"} {
		data, err := os.ReadFile(filepath.Join(run.runDir, "reviews", "article-001", lens, "result.json"))
		if err != nil {
			t.Fatal(err)
		}
		var result struct {
			ReviewedRevision string `json:"reviewed_revision"`
		}
		if err := json.Unmarshal(data, &result); err != nil {
			t.Fatal(err)
		}
		if result.ReviewedRevision != revision(article) {
			t.Errorf("%s reviewed %q, final article is %q", lens, result.ReviewedRevision, revision(article))
		}
	}

	records := readInvocationRecords(t, run.fixtureDir)
	var reviewers []invocationRecord
	var researcher *invocationRecord
	var pm *invocationRecord
	for _, record := range records {
		if record.Role == "pm" {
			copy := record
			pm = &copy
		}
		if record.Role == "researcher" {
			copy := record
			researcher = &copy
		}
		if strings.HasPrefix(record.Role, "reviewer_") {
			reviewers = append(reviewers, record)
		}
	}
	hasSourceHint := false
	if researcher != nil {
		for _, name := range researcher.WorkspaceFiles {
			hasSourceHint = hasSourceHint || name == "context/source-hints/001-README.md"
		}
	}
	if !hasSourceHint {
		t.Errorf("researcher did not receive resolved local source hints: %+v", researcher)
	}
	if pm == nil || !strings.Contains(pm.Prompt, `"decision": "valid_must_fix"`) || strings.Contains(pm.Prompt, `"classification":`) {
		t.Errorf("PM prompt does not define the durable decision field exactly: %+v", pm)
	}
	sort.Slice(reviewers, func(i, j int) bool { return reviewers[i].Invocation < reviewers[j].Invocation })
	if len(reviewers) != 4 {
		t.Fatalf("got %d reviewer invocations, want 4", len(reviewers))
	}
	wantLenses := []string{"evidence", "story", "clarity", "copy"}
	seenPIDs := map[int]bool{}
	for index, reviewer := range reviewers {
		if reviewer.Lens != wantLenses[index] {
			t.Errorf("reviewer %d lens = %q, want %q", index, reviewer.Lens, wantLenses[index])
		}
		if seenPIDs[reviewer.PID] {
			t.Errorf("reviewer process PID %d was reused", reviewer.PID)
		}
		seenPIDs[reviewer.PID] = true
		if strings.HasPrefix(reviewer.Workspace, run.runDir) {
			t.Errorf("%s reviewer workspace is inside durable run: %s", reviewer.Lens, reviewer.Workspace)
		}
		assertReviewerFilesystem(t, reviewer)
		for _, required := range []string{"Provided context: brief.md", "CANDIDATE_ONLY_MARKER", reviewer.Revision} {
			if !strings.Contains(reviewer.Prompt, required) {
				t.Errorf("%s prompt missing %q", reviewer.Lens, required)
			}
		}
		switch reviewer.Lens {
		case "evidence":
			assertPromptMarkers(t, reviewer, []string{"EVIDENCE_ONLY_MARKER", "claim-ledger.md"}, []string{"STORY_ONLY_MARKER", "clarity-fields"})
		case "story":
			assertPromptMarkers(t, reviewer, []string{"STORY_ONLY_MARKER"}, []string{"EVIDENCE_ONLY_MARKER", "claim-ledger.md", "clarity-fields"})
		case "clarity":
			assertPromptMarkers(t, reviewer, []string{"clarity-fields"}, []string{"EVIDENCE_ONLY_MARKER", "STORY_ONLY_MARKER", "claim-ledger.md"})
		case "copy":
			assertPromptMarkers(t, reviewer, nil, []string{"EVIDENCE_ONLY_MARKER", "STORY_ONLY_MARKER", "claim-ledger.md", "clarity-fields"})
		}
	}
	assertProcessesGone(t, records)
}

func TestBlackBoxMustFixCreatesRevisionAndRestartsEvidence(t *testing.T) {
	run := executeScenario(t, "mustfix_once")
	if run.err != nil {
		t.Fatalf("CLI failed: %v\n%s", run.err, run.output)
	}
	state := readWorkflow(t, run.runDir)
	if state.Status != "succeeded" || state.CurrentCandidate != 2 || state.ReviewAttemptCount != 5 {
		t.Fatalf("unexpected workflow: %+v", state)
	}
	for _, relative := range []string{
		"drafts/article-001.md", "drafts/article-002.md",
		"reviews/article-001/evidence/result.json", "pm-decisions/article-001.md",
		"reviews/article-002/evidence/result.json", "reviews/article-002/story/result.json",
		"reviews/article-002/clarity/result.json", "reviews/article-002/copy/result.json",
	} {
		if _, err := os.Stat(filepath.Join(run.runDir, relative)); err != nil {
			t.Errorf("missing preserved artifact %s: %v", relative, err)
		}
	}
	if _, err := os.Stat(filepath.Join(run.runDir, "reviews", "article-001", "story")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("remaining candidate-001 lenses were not stopped")
	}
	articleTwo, _ := os.ReadFile(filepath.Join(run.runDir, "drafts", "article-002.md"))
	if got, want := state.CurrentRevision, revision(articleTwo); got != want {
		t.Errorf("revision = %q, want %q", got, want)
	}
	assertExactFiles(t, filepath.Join(run.runDir, "drafts", "article-002.md"), filepath.Join(run.runDir, "article.md"))
	records := readInvocationRecords(t, run.fixtureDir)
	foundRevisionContext := false
	for _, record := range records {
		if record.Role == "writer" && record.Candidate == 2 {
			foundRevisionContext = strings.Contains(record.Prompt, "The opening needs a verified detail.") &&
				strings.Contains(record.Prompt, "Add the supported workflow detail.")
		}
	}
	if !foundRevisionContext {
		t.Fatal("revision writer did not receive the validated finding and suggested direction")
	}
	assertProcessesGone(t, records)
}

func TestBlackBoxStaleRevisionBlocks(t *testing.T) {
	run := executeScenario(t, "stale")
	if run.err == nil {
		t.Fatal("CLI unexpectedly succeeded")
	}
	state := readWorkflow(t, run.runDir)
	if state.Status != "blocked" || !strings.Contains(state.BlockReason, "stale review revision") {
		t.Fatalf("unexpected blocked workflow: %+v", state)
	}
	if _, err := os.Stat(filepath.Join(run.runDir, "article.md")); !errors.Is(err, os.ErrNotExist) {
		t.Error("stale review advanced to article.md")
	}
	assertProcessesGone(t, readInvocationRecords(t, run.fixtureDir))
}

func TestBlackBoxThirdCandidateMustFixBlocksAndPreserves(t *testing.T) {
	run := executeScenario(t, "budget")
	if run.err == nil {
		t.Fatal("CLI unexpectedly succeeded")
	}
	state := readWorkflow(t, run.runDir)
	if state.Status != "blocked" || state.CurrentCandidate != 3 || state.ReviewAttemptCount != 3 || !strings.Contains(state.BlockReason, "review budget exhausted") {
		t.Fatalf("unexpected blocked workflow: %+v", state)
	}
	for candidate := 1; candidate <= 3; candidate++ {
		for _, relative := range []string{
			fmt.Sprintf("drafts/article-%03d.md", candidate),
			fmt.Sprintf("reviews/article-%03d/evidence/result.json", candidate),
			fmt.Sprintf("pm-decisions/article-%03d.md", candidate),
		} {
			if _, err := os.Stat(filepath.Join(run.runDir, relative)); err != nil {
				t.Errorf("missing preserved artifact %s: %v", relative, err)
			}
		}
	}
	assertProcessesGone(t, readInvocationRecords(t, run.fixtureDir))
}

func TestBlackBoxHumanJudgmentBlocksAndPreserves(t *testing.T) {
	run := executeScenario(t, "human")
	if run.err == nil {
		t.Fatal("CLI unexpectedly succeeded")
	}
	state := readWorkflow(t, run.runDir)
	if state.Status != "blocked" || !strings.Contains(state.BlockReason, "human judgment required") {
		t.Fatalf("unexpected blocked workflow: %+v", state)
	}
	for _, relative := range []string{"drafts/article-001.md", "reviews/article-001/evidence/result.json", "pm-decisions/article-001.md"} {
		if _, err := os.Stat(filepath.Join(run.runDir, relative)); err != nil {
			t.Errorf("missing preserved artifact %s: %v", relative, err)
		}
	}
	assertProcessesGone(t, readInvocationRecords(t, run.fixtureDir))
}

func TestBlackBoxTimeoutBlocksAndCleansProcesses(t *testing.T) {
	run := executeScenarioWithTimeout(t, "timeout", "200ms")
	if run.err == nil {
		t.Fatal("CLI unexpectedly succeeded")
	}
	state := readWorkflow(t, run.runDir)
	if state.Status != "blocked" || !strings.Contains(state.BlockReason, "timed out") {
		t.Fatalf("unexpected blocked workflow: %+v", state)
	}
	assertProcessesGone(t, readInvocationRecords(t, run.fixtureDir))
}

func TestBlackBoxTimeoutBoundsPrivateRunnerAndDetachedGroup(t *testing.T) {
	tmuxDirectory, err := os.MkdirTemp("/tmp", "wu-tmux-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(tmuxDirectory) })
	t.Setenv("TMUX", "")
	t.Setenv("TMUX_TMPDIR", tmuxDirectory)
	binary, fake, runDir, fixtureDir := prepareScenario(t, "timeout_detached")
	pidDirectory := t.TempDir()
	command := newRunCommand(t, binary, fake, runDir, "5s", filepath.Join(repositoryRoot(t), "examples", "brief.md"))
	command.Env = append(command.Env,
		"WRITE_UUTER_TEST_DETACHED_PID_DIR="+pidDirectory,
		"WRITE_UUTER_TEST_EXIT_MARKER_DELAY=2s",
		"WRITE_UUTER_TEST_WORKER_READY_FILE=.write-uuter-detached.ready",
		"WRITE_UUTER_TEST_WORKER_ACTIVE_TIMEOUT=250ms",
	)
	started := time.Now()
	output, err := command.CombinedOutput()
	if err == nil {
		t.Fatalf("CLI unexpectedly succeeded: %s", output)
	}
	if elapsed := time.Since(started); elapsed > 8*time.Second {
		t.Fatalf("private runner timeout took %s: %s", elapsed, output)
	}
	state := readWorkflow(t, runDir)
	if state.Status != "blocked" || !strings.Contains(state.BlockReason, "timed out") && !strings.Contains(state.BlockReason, "wall-clock deadline") {
		t.Fatalf("unexpected blocked workflow: %+v", state)
	}
	paths, _ := filepath.Glob(filepath.Join(pidDirectory, "*.pid"))
	if len(paths) == 0 {
		t.Fatal("fake Codex did not start a detached descendant")
	}
	for _, path := range paths {
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			t.Fatal(readErr)
		}
		pid, parseErr := strconv.Atoi(strings.TrimSpace(string(data)))
		if parseErr != nil {
			t.Fatal(parseErr)
		}
		pgidData, readErr := os.ReadFile(strings.TrimSuffix(path, ".pid") + ".pgid")
		if readErr != nil {
			t.Fatal(readErr)
		}
		pgid, parseErr := strconv.Atoi(strings.TrimSpace(string(pgidData)))
		if parseErr != nil || pgid != pid {
			t.Fatalf("child did not create a new session/process group: pid=%d pgid=%d err=%v", pid, pgid, parseErr)
		}
		assertPIDGone(t, pid)
	}
	assertProcessesGone(t, readInvocationRecords(t, fixtureDir))
	privatePaths, _ := filepath.Glob(filepath.Join(filepath.Dir(runDir), ".write-uuter-private-*"))
	if len(privatePaths) != 0 {
		t.Fatalf("private runtime survived timeout cleanup: %v", privatePaths)
	}
	sessions := exec.Command("tmux", "list-sessions", "-F", "#{session_name}")
	sessions.Env = append(os.Environ(), "TMUX=", "TMUX_TMPDIR="+tmuxDirectory)
	listing, _ := sessions.CombinedOutput()
	if strings.Contains(string(listing), "write-uuter-") {
		t.Fatalf("timeout left run tmux session: %s", listing)
	}
}

func TestBlackBoxOptionalAndInvalidFindingsDoNotConsumeCandidate(t *testing.T) {
	run := executeScenario(t, "optional_invalid")
	if run.err != nil {
		t.Fatalf("CLI failed: %v\n%s", run.err, run.output)
	}
	state := readWorkflow(t, run.runDir)
	if state.Status != "succeeded" || state.CurrentCandidate != 1 || state.ReviewAttemptCount != 4 {
		t.Fatalf("unexpected workflow: %+v", state)
	}
	if _, err := os.Stat(filepath.Join(run.runDir, "drafts", "article-002.md")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("optional/invalid decisions consumed a candidate")
	}
	decisionData, err := os.ReadFile(filepath.Join(run.runDir, "pm-decisions", "article-001.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"valid_optional", "invalid", "The claimed problem is not present", "request_id", "review_digest"} {
		if !strings.Contains(string(decisionData), expected) {
			t.Errorf("decision audit missing %q", expected)
		}
	}
}

func TestBlackBoxInvalidDecisionWithoutReasonBlocks(t *testing.T) {
	run := executeScenario(t, "invalid_no_reason")
	if run.err == nil {
		t.Fatal("CLI unexpectedly succeeded")
	}
	state := readWorkflow(t, run.runDir)
	if state.Status != "blocked" || !strings.Contains(state.BlockReason, "requires a reason") {
		t.Fatalf("unexpected workflow: %+v", state)
	}
	assertNoArticle(t, run.runDir)
}

func TestBlackBoxMixedHumanAndMustFixAlwaysBlocks(t *testing.T) {
	run := executeScenario(t, "mixed")
	if run.err == nil {
		t.Fatal("CLI unexpectedly succeeded")
	}
	state := readWorkflow(t, run.runDir)
	if state.Status != "blocked" || state.CurrentCandidate != 1 || !strings.Contains(state.BlockReason, "human judgment required") {
		t.Fatalf("unexpected workflow: %+v", state)
	}
	if _, err := os.Stat(filepath.Join(run.runDir, "drafts", "article-002.md")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("mixed decision incorrectly created a revision")
	}
	assertNoArticle(t, run.runDir)
}

func TestBlackBoxInvalidReviewerAndMutationAndSymlinkBlock(t *testing.T) {
	for _, testCase := range []struct {
		scenario string
		reason   string
	}{
		{"invalid_review", "review lens mismatch"},
		{"mutate", "reviewer edited the candidate input"},
		{"symlink_output", "no-follow"},
	} {
		t.Run(testCase.scenario, func(t *testing.T) {
			run := executeScenario(t, testCase.scenario)
			if run.err == nil {
				t.Fatal("CLI unexpectedly succeeded")
			}
			state := readWorkflow(t, run.runDir)
			if state.Status != "blocked" || !strings.Contains(state.BlockReason, testCase.reason) {
				t.Fatalf("unexpected workflow: %+v", state)
			}
			assertNoArticle(t, run.runDir)
			assertProcessesGone(t, readInvocationRecords(t, run.fixtureDir))
		})
	}
}

func TestBlackBoxWorkerMustExitBeforeArtifactsAdvance(t *testing.T) {
	start := time.Now()
	run := executeScenario(t, "partial")
	if run.err != nil {
		t.Fatalf("CLI failed: %v\n%s", run.err, run.output)
	}
	if time.Since(start) < 300*time.Millisecond {
		t.Fatal("controller advanced from transient worker output before process exit")
	}
	article, err := os.ReadFile(filepath.Join(run.runDir, "article.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(article), "COMPLETE_MARKER") {
		t.Fatal("published truncated writer output")
	}
}

func TestBlackBoxPMHistoryCannotBePrepopulatedOrDropped(t *testing.T) {
	for _, scenario := range []string{"prepopulate", "drop_history"} {
		t.Run(scenario, func(t *testing.T) {
			run := executeScenario(t, scenario)
			if run.err == nil {
				t.Fatal("CLI unexpectedly succeeded")
			}
			state := readWorkflow(t, run.runDir)
			if state.Status != "blocked" || !strings.Contains(state.BlockReason, "lens history") && !strings.Contains(state.BlockReason, "missing reached lens") {
				t.Fatalf("unexpected workflow: %+v", state)
			}
			assertNoArticle(t, run.runDir)
		})
	}
}

func TestBlackBoxFinalCandidateMutationCannotPublish(t *testing.T) {
	binary, fake, runDir, fixtureDir := prepareScenario(t, "slow_final")
	command := newRunCommand(t, binary, fake, runDir, "5s", filepath.Join(repositoryRoot(t), "examples", "brief.md"))
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	copyResult := filepath.Join(runDir, "reviews", "article-001", "copy", "result.json")
	deadline := time.Now().Add(5 * time.Second)
	for {
		if _, err := os.Stat(copyResult); err == nil {
			break
		}
		if time.Now().After(deadline) {
			_ = command.Process.Kill()
			t.Fatal("copy review did not appear")
		}
		time.Sleep(20 * time.Millisecond)
	}
	if err := os.WriteFile(filepath.Join(runDir, "drafts", "article-001.md"), []byte("late mutation\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := command.Wait()
	if err == nil {
		t.Fatal("CLI unexpectedly published mutated candidate")
	}
	state := readWorkflow(t, runDir)
	if state.Status != "blocked" || !strings.Contains(state.BlockReason, "changed before publication") {
		t.Fatalf("unexpected workflow: %+v", state)
	}
	assertNoArticle(t, runDir)
	assertProcessesGone(t, readInvocationRecords(t, fixtureDir))
}

func TestBlackBoxFinalCommitBoundaryRechecksDurableDraft(t *testing.T) {
	binary, fake, runDir, fixtureDir := prepareScenario(t, "happy")
	barrier := filepath.Join(t.TempDir(), "final-boundary")
	command := newRunCommand(t, binary, fake, runDir, "5s", filepath.Join(repositoryRoot(t), "examples", "brief.md"))
	command.Env = append(command.Env, "WRITE_UUTER_TEST_FINAL_BOUNDARY_BARRIER="+barrier)
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	waitForPath(t, barrier+".ready")
	if err := os.WriteFile(filepath.Join(runDir, "drafts", "article-001.md"), []byte("mutation after initial final audit\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(barrier+".continue", []byte("continue\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := command.Wait(); err == nil {
		t.Fatal("CLI published bytes read before the final commit boundary")
	}
	state := readWorkflow(t, runDir)
	if state.Status != "blocked" || !strings.Contains(state.BlockReason, "publication boundary") {
		t.Fatalf("unexpected workflow: %+v", state)
	}
	assertNoArticle(t, runDir)
	assertProcessesGone(t, readInvocationRecords(t, fixtureDir))
}

func TestBlackBoxFinalCommitBoundaryRechecksCompleteAudit(t *testing.T) {
	binary, fake, runDir, fixtureDir := prepareScenario(t, "happy")
	barrier := filepath.Join(t.TempDir(), "final-audit-boundary")
	command := newRunCommand(t, binary, fake, runDir, "5s", filepath.Join(repositoryRoot(t), "examples", "brief.md"))
	command.Env = append(command.Env, "WRITE_UUTER_TEST_FINAL_BOUNDARY_BARRIER="+barrier)
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	waitForPath(t, barrier+".ready")
	if err := os.WriteFile(filepath.Join(runDir, "pm-decisions", "article-001.md"), []byte("```json\n{}\n```\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(barrier+".continue", []byte("continue\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := command.Wait(); err == nil {
		t.Fatal("CLI published after final PM audit mutation")
	}
	state := readWorkflow(t, runDir)
	if state.Status != "blocked" || !strings.Contains(state.BlockReason, "audit changed at publication boundary") {
		t.Fatalf("unexpected workflow: %+v", state)
	}
	assertNoArticle(t, runDir)
	assertProcessesGone(t, readInvocationRecords(t, fixtureDir))
}

func TestBlackBoxAtomicCommitDoesNotReplaceCompetingTarget(t *testing.T) {
	binary, fake, runDir, _ := prepareScenario(t, "happy")
	barrier := filepath.Join(t.TempDir(), "commit")
	command := newRunCommand(t, binary, fake, runDir, "5s", filepath.Join(repositoryRoot(t), "examples", "brief.md"))
	command.Env = append(command.Env, "WRITE_UUTER_TEST_COMMIT_BARRIER="+barrier)
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	waitForPath(t, barrier+".ready")
	if err := os.Mkdir(runDir, 0o755); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(runDir, "competitor.txt")
	if err := os.WriteFile(marker, []byte("preserve"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(barrier+".continue", []byte("continue\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := command.Wait(); err == nil {
		t.Fatal("CLI unexpectedly replaced competing target")
	}
	data, err := os.ReadFile(marker)
	if err != nil || string(data) != "preserve" {
		t.Fatalf("competing target changed: %q, %v", data, err)
	}
	entries, _ := os.ReadDir(runDir)
	if len(entries) != 1 || entries[0].Name() != "competitor.txt" {
		t.Fatalf("competing target contents changed: %v", entries)
	}
}

func TestBlackBoxHappyPathWithIsolatedTmuxServer(t *testing.T) {
	tmuxDirectory, err := os.MkdirTemp("/tmp", "wu-tmux-*")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(tmuxDirectory) })
	t.Setenv("TMUX", "")
	t.Setenv("TMUX_TMPDIR", tmuxDirectory)
	run := executeScenario(t, "happy")
	if run.err != nil {
		t.Fatalf("isolated tmux CLI failed: %v\n%s", run.err, run.output)
	}
	if state := readWorkflow(t, run.runDir); state.Status != "succeeded" {
		t.Fatalf("unexpected workflow: %+v", state)
	}
	assertProcessesGone(t, readInvocationRecords(t, run.fixtureDir))
}

func TestBlackBoxAgentWorkspaceCannotReplaceHostLauncher(t *testing.T) {
	run := executeScenario(t, "launcher_attack")
	if run.err != nil {
		t.Fatalf("workspace launcher attack affected later invocations: %v\n%s", run.err, run.output)
	}
	if _, err := os.Stat(filepath.Join(run.runDir, ".control", "agent-runner")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("launch-critical executable was exposed in durable agent-visible run: %v", err)
	}
	if state := readWorkflow(t, run.runDir); state.Status != "succeeded" {
		t.Fatalf("unexpected workflow: %+v", state)
	}
	foundDeniedAttack := false
	for _, record := range readInvocationRecords(t, run.fixtureDir) {
		if record.Role == "researcher" && record.Isolation["actual_launcher_write"] != "" {
			foundDeniedAttack = true
		}
	}
	if !foundDeniedAttack {
		t.Fatal("fake researcher did not attempt and record denial of the actual controller launcher path")
	}
	privatePaths, _ := filepath.Glob(filepath.Join(filepath.Dir(run.runDir), ".write-uuter-private-*"))
	if len(privatePaths) != 0 {
		t.Fatalf("controller-private runtime survived terminal cleanup: %v", privatePaths)
	}
}

func TestBlackBoxReviewerHostFilesystemReadsAreDenied(t *testing.T) {
	for name, arguments := range map[string][]string{
		"/usr/bin/security":  {"list-keychains"},
		"/usr/bin/osascript": {"-e", "use scripting additions", "-e", "clipboard info"},
	} {
		probe := exec.Command(name, arguments...)
		probe.Stdout = nil
		probe.Stderr = nil
		if err := probe.Run(); err != nil {
			t.Fatalf("host service control probe %s is unavailable: %v", name, err)
		}
	}
	binary, fake, runDir, fixtureDir := prepareScenario(t, "filesystem_isolation")
	command := newRunCommand(t, binary, fake, runDir, "5s", filepath.Join(repositoryRoot(t), "examples", "brief.md"))
	protectedAncestor := filepath.Dir(runDir)
	credentialHome := t.TempDir()
	const credentialSecret = "WRITE_UUTER_TOOL_BOUNDARY_SECRET"
	const proxySecret = "WRITE_UUTER_PROXY_USERINFO_SECRET"
	if err := os.WriteFile(filepath.Join(credentialHome, "auth.json"), []byte(credentialSecret+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(credentialHome, "installation_id"), []byte("test-installation\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	command.Env = replaceCommandEnv(command.Env, "CODEX_HOME", credentialHome)
	command.Env = replaceCommandEnv(command.Env, "HTTP_PROXY", "http://probe:"+proxySecret+"@127.0.0.1:9")
	command.Env = replaceCommandEnv(command.Env, "HTTPS_PROXY", "http://probe:"+proxySecret+"@127.0.0.1:9")
	command.Env = replaceCommandEnv(command.Env, "WRITE_UUTER_FAKE_LOG_DIR", protectedAncestor)
	command.Env = replaceCommandEnv(command.Env, "WRITE_UUTER_TEST_DETACHED_PID_DIR", protectedAncestor)
	output, runErr := command.CombinedOutput()
	if runErr != nil {
		pmLog, _ := os.ReadFile(filepath.Join(runDir, ".control", "logs", "001-pm.log"))
		t.Fatalf("CLI failed: %v\n%s\nPM log:\n%s", runErr, output, pmLog)
	}
	probePath := filepath.Join(protectedAncestor, "isolation-008-reviewer-copy.probe")
	data, err := os.ReadFile(probePath)
	if err != nil {
		t.Fatal(err)
	}
	var probes map[string]string
	if err := json.Unmarshal(data, &probes); err != nil {
		t.Fatal(err)
	}
	for _, label := range []string{"durable", "prior_lens", "pm_workspace_parent", "controller", "host", "usr_sibling", "dev_sibling", "codex_sibling", "tool_auth", "tool_network", "double_fork", "unrelated_process"} {
		result := strings.ToLower(probes[label])
		if result == "" || strings.Contains(result, "succeeded") || !strings.Contains(result, "not permitted") && !strings.Contains(result, "permission denied") && !strings.Contains(result, "operation not permitted") {
			t.Errorf("copy reviewer was not OS-denied %s read: %q", label, probes[label])
		}
	}
	for _, label := range []string{"tool_keychain", "tool_pasteboard"} {
		if result := strings.ToLower(probes[label]); result == "" || strings.Contains(result, "succeeded") {
			t.Errorf("host control probe succeeded across model-tool boundary: %s=%q", label, probes[label])
		}
	}
	if result := strings.ToLower(probes["exact_client_reexec"]); strings.Contains(result, "succeeded") || !strings.Contains(result, "not permitted") && !strings.Contains(result, "permission denied") && !strings.Contains(result, "operation not permitted") {
		t.Errorf("single-use privileged client path was reusable: %q", probes["exact_client_reexec"])
	}
	if strings.Contains(string(data), credentialSecret) || strings.Contains(string(data), proxySecret) {
		t.Fatal("model-invoked probe disclosed staged authentication or proxy userinfo")
	}
	for _, name := range []string{"proxy_http_proxy", "proxy_https_proxy"} {
		if probes[name] != "ABSENT" {
			t.Errorf("credential-bearing ambient proxy survived sanitization: %s=%q", name, probes[name])
		}
	}
	for _, name := range []string{"WRITE_UUTER_FAKE_LOG_DIR", "WRITE_UUTER_TEST_DETACHED_PID_DIR"} {
		if probes[name] != "ABSENT" {
			t.Errorf("controller-only test path leaked into agent environment %s=%q", name, probes[name])
		}
	}
	_ = fixtureDir
}

func TestBlackBoxPMDecisionRequiresPresentNonNullDecisionArrays(t *testing.T) {
	for _, scenario := range []string{"missing_pm_decisions", "null_pm_decisions"} {
		t.Run(scenario, func(t *testing.T) {
			run := executeScenario(t, scenario)
			if run.err == nil {
				t.Fatal("CLI accepted a PM lens record without a decisions array")
			}
			state := readWorkflow(t, run.runDir)
			if state.Status != "blocked" || !strings.Contains(state.BlockReason, "non-null decisions array") {
				t.Fatalf("unexpected workflow: %+v", state)
			}
			assertNoArticle(t, run.runDir)
		})
	}
}

func TestBlackBoxCopyStyleGuideUsesContentRootNotPromptBundle(t *testing.T) {
	binary, fake, runDir, fixtureDir := prepareScenario(t, "happy")
	contentRoot := t.TempDir()
	const styleMarker = "CONTENT_ROOT_COPY_STYLE_MARKER"
	if err := os.WriteFile(filepath.Join(contentRoot, "STYLE.md"), []byte("# Style\n\n"+styleMarker+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	command := newRunCommand(t, binary, fake, runDir, "5s", filepath.Join(repositoryRoot(t), "examples", "brief.md"))
	command.Dir = contentRoot
	if output, err := command.CombinedOutput(); err != nil {
		pmLog, _ := os.ReadFile(filepath.Join(runDir, ".control", "logs", "001-pm.log"))
		t.Fatalf("CLI failed with external prompt bundle and content-root style guide: %v\n%s\nPM log:\n%s", err, output, pmLog)
	}
	seenCopy := false
	for _, record := range readInvocationRecords(t, fixtureDir) {
		if !strings.HasPrefix(record.Role, "reviewer_") {
			continue
		}
		hasStyleFile := false
		for _, relative := range record.WorkspaceFiles {
			if relative == "context/style-guide.md" {
				hasStyleFile = true
			}
		}
		if record.Lens == "copy" {
			seenCopy = true
			if !strings.Contains(record.Prompt, styleMarker) || !hasStyleFile {
				t.Errorf("Copy reviewer did not receive content-root style guide: files=%v", record.WorkspaceFiles)
			}
		} else if strings.Contains(record.Prompt, styleMarker) || hasStyleFile {
			t.Errorf("%s reviewer received Copy-only style guide", record.Lens)
		}
	}
	if !seenCopy {
		t.Fatal("Copy reviewer did not run")
	}
}

func TestBlackBoxPMExitDuringWorkerCommitRollsBackArtifacts(t *testing.T) {
	binary, fake, runDir, fixtureDir := prepareScenario(t, "happy")
	barrier := filepath.Join(t.TempDir(), "worker-commit")
	command := newRunCommand(t, binary, fake, runDir, "5s", filepath.Join(repositoryRoot(t), "examples", "brief.md"))
	command.Env = append(command.Env, "WRITE_UUTER_TEST_PM_EXIT_WORKER_COMMIT_BARRIER="+barrier)
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	waitForPath(t, barrier+".ready")
	killPersistentPM(t, runDir)
	if err := os.WriteFile(barrier+".continue", []byte("continue\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := command.Wait(); err == nil {
		t.Fatal("CLI committed a worker artifact after PM exit")
	}
	state := readWorkflow(t, runDir)
	if state.Status != "blocked" || !strings.Contains(state.BlockReason, "PM exited during researcher artifact commit") {
		t.Fatalf("unexpected workflow: %+v", state)
	}
	for _, relative := range []string{"evidence/sources.md", "claim-ledger.md"} {
		if _, err := os.Lstat(filepath.Join(runDir, relative)); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("rolled-back worker artifact remains at %s: %v", relative, err)
		}
	}
	assertProcessesGone(t, readInvocationRecords(t, fixtureDir))
}

func TestBlackBoxPMExitDuringDecisionCommitRollsBackDecision(t *testing.T) {
	binary, fake, runDir, fixtureDir := prepareScenario(t, "happy")
	barrier := filepath.Join(t.TempDir(), "decision-commit")
	command := newRunCommand(t, binary, fake, runDir, "5s", filepath.Join(repositoryRoot(t), "examples", "brief.md"))
	command.Env = append(command.Env, "WRITE_UUTER_TEST_PM_EXIT_DECISION_COMMIT_BARRIER="+barrier)
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	waitForPath(t, barrier+".ready")
	killPersistentPM(t, runDir)
	if err := os.WriteFile(barrier+".continue", []byte("continue\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := command.Wait(); err == nil {
		t.Fatal("CLI committed a PM decision after PM exit")
	}
	state := readWorkflow(t, runDir)
	if state.Status != "blocked" || !strings.Contains(state.BlockReason, "PM exited during evidence decision commit") {
		t.Fatalf("unexpected workflow: %+v", state)
	}
	if _, err := os.Lstat(filepath.Join(runDir, "pm-decisions", "article-001.md")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("rolled-back PM decision remains: %v", err)
	}
	assertProcessesGone(t, readInvocationRecords(t, fixtureDir))
}

func TestBlackBoxStrictRoutingJSONRejectsDuplicatesAndUnknownFields(t *testing.T) {
	for _, testCase := range []struct {
		scenario string
		reason   string
	}{
		{"duplicate_review", "duplicate JSON key"},
		{"unknown_review", "unknown field"},
		{"duplicate_pm", "duplicate JSON key"},
		{"unknown_pm", "unknown field"},
		{"duplicate_nested_finding", "duplicate JSON key"},
		{"unknown_nested_finding", "unknown field"},
		{"duplicate_pm_decision", "duplicate JSON key"},
		{"unknown_pm_lens", "unknown field"},
	} {
		t.Run(testCase.scenario, func(t *testing.T) {
			run := executeScenario(t, testCase.scenario)
			if run.err == nil {
				t.Fatal("CLI unexpectedly accepted ambiguous routing JSON")
			}
			state := readWorkflow(t, run.runDir)
			if state.Status != "blocked" || !strings.Contains(state.BlockReason, testCase.reason) {
				t.Fatalf("unexpected workflow: %+v", state)
			}
			assertNoArticle(t, run.runDir)
		})
	}
}

func TestBlackBoxPMCannotRewriteAcceptedHistory(t *testing.T) {
	run := executeScenario(t, "rewrite_history")
	if run.err == nil {
		t.Fatal("CLI unexpectedly accepted rewritten PM history")
	}
	state := readWorkflow(t, run.runDir)
	if state.Status != "blocked" || !strings.Contains(state.BlockReason, "changed accepted classifications") {
		t.Fatalf("unexpected workflow: %+v", state)
	}
	assertNoArticle(t, run.runDir)
}

func TestBlackBoxUnexpectedTmuxProbeFailureIsNotAbsence(t *testing.T) {
	realTmux, err := exec.LookPath("tmux")
	if err != nil {
		t.Skip("tmux unavailable")
	}
	tmuxDirectory := filepath.Join(t.TempDir(), "tmux")
	if err := os.Mkdir(tmuxDirectory, 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("TMUX", "")
	t.Setenv("TMUX_TMPDIR", tmuxDirectory)
	binary, fake, runDir, fixtureDir := prepareScenario(t, "happy")
	wrapper := filepath.Join(t.TempDir(), "tmux-wrapper")
	script := fmt.Sprintf("#!/bin/sh\nif [ \"$1\" = \"has-session\" ]; then echo 'injected probe failure' >&2; exit 42; fi\nexec %q \"$@\"\n", realTmux)
	if err := os.WriteFile(wrapper, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	command := newRunCommand(t, binary, fake, runDir, "1s", filepath.Join(repositoryRoot(t), "examples", "brief.md"))
	command.Args = append(command.Args, "--tmux", wrapper)
	output, err := command.CombinedOutput()
	if err == nil {
		t.Fatalf("CLI unexpectedly succeeded: %s", output)
	}
	state := readWorkflow(t, runDir)
	if state.Status != "blocked" || !strings.Contains(state.BlockReason, "has-session failed") {
		t.Fatalf("unexpected workflow: %+v", state)
	}
	assertNoArticle(t, runDir)
	assertProcessesGone(t, readInvocationRecords(t, fixtureDir))
	sessions := exec.Command(realTmux, "list-sessions")
	sessions.Env = append(os.Environ(), "TMUX=", "TMUX_TMPDIR="+tmuxDirectory)
	listing, _ := sessions.CombinedOutput()
	if strings.Contains(string(listing), "write-uuter-") {
		t.Fatalf("probe failure left run tmux session: %s", listing)
	}
	privatePaths, _ := filepath.Glob(filepath.Join(filepath.Dir(runDir), ".write-uuter-private-*"))
	if len(privatePaths) != 1 {
		t.Fatalf("probe failure did not preserve one recoverable ownership root: %v", privatePaths)
	}
	t.Cleanup(func() { _ = os.RemoveAll(privatePaths[0]) })
	if _, err := os.Lstat(filepath.Join(privatePaths[0], "codex-homes")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("probe failure left copied credentials behind: %v", err)
	}
}

func TestBlackBoxFinalPMExitBlocksPublication(t *testing.T) {
	run := executeScenario(t, "final_pm_exit")
	if run.err == nil {
		t.Fatal("CLI unexpectedly accepted a final response from an exited PM")
	}
	state := readWorkflow(t, run.runDir)
	if state.Status != "blocked" || !strings.Contains(state.BlockReason, "PM exited") && !strings.Contains(state.BlockReason, "persistent PM") {
		t.Fatalf("unexpected workflow: %+v", state)
	}
	assertNoArticle(t, run.runDir)
	assertProcessesGone(t, readInvocationRecords(t, run.fixtureDir))
}

func TestBlackBoxPMMustEnterProtocolBeforeAnyWorkerStarts(t *testing.T) {
	run := executeScenario(t, "pm_exit_before_ready")
	if run.err == nil {
		t.Fatal("CLI unexpectedly started without a protocol-ready PM")
	}
	state := readWorkflow(t, run.runDir)
	if state.Status != "blocked" || !strings.Contains(state.BlockReason, "protocol-ready marker") {
		t.Fatalf("unexpected workflow: %+v", state)
	}
	for _, record := range readInvocationRecords(t, run.fixtureDir) {
		if record.Role != "pm" {
			t.Fatalf("worker %s started before PM protocol readiness", record.Role)
		}
	}
	assertProcessesGone(t, readInvocationRecords(t, run.fixtureDir))
}

func TestBlackBoxWorkerCannotCommitAfterPMExit(t *testing.T) {
	run := executeScenario(t, "pm_exit_during_worker")
	if run.err == nil {
		t.Fatal("CLI unexpectedly accepted work after PM exit")
	}
	state := readWorkflow(t, run.runDir)
	if state.Status != "blocked" || !strings.Contains(state.BlockReason, "PM exited") {
		t.Fatalf("unexpected workflow: %+v", state)
	}
	if _, err := os.Stat(filepath.Join(run.runDir, "evidence", "sources.md")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("research artifact committed after PM exit: %v", err)
	}
	assertProcessesGone(t, readInvocationRecords(t, run.fixtureDir))
}

func TestBlackBoxDetachedDescendantsAreKilledOnTerminalPaths(t *testing.T) {
	for _, testCase := range []struct {
		scenario string
		success  bool
	}{{"detached_child_success", true}, {"detached_child_block", false}} {
		t.Run(testCase.scenario, func(t *testing.T) {
			binary, fake, runDir, _ := prepareScenario(t, testCase.scenario)
			pidDirectory := t.TempDir()
			command := newRunCommand(t, binary, fake, runDir, "5s", filepath.Join(repositoryRoot(t), "examples", "brief.md"))
			command.Env = append(command.Env, "WRITE_UUTER_TEST_DETACHED_PID_DIR="+pidDirectory)
			output, err := command.CombinedOutput()
			if (err == nil) != testCase.success {
				t.Fatalf("unexpected terminal result: %v\n%s", err, output)
			}
			paths, _ := filepath.Glob(filepath.Join(pidDirectory, "*.pid"))
			if len(paths) == 0 {
				t.Fatal("fake agent did not create a detached child")
			}
			for _, path := range paths {
				data, readErr := os.ReadFile(path)
				if readErr != nil {
					t.Fatal(readErr)
				}
				pid, _ := strconv.Atoi(strings.TrimSpace(string(data)))
				pgidData, readErr := os.ReadFile(strings.TrimSuffix(path, ".pid") + ".pgid")
				if readErr != nil {
					t.Fatal(readErr)
				}
				pgid, parseErr := strconv.Atoi(strings.TrimSpace(string(pgidData)))
				if parseErr != nil || pgid != pid {
					t.Fatalf("child did not detach into a new process group: pid=%d pgid=%d err=%v", pid, pgid, parseErr)
				}
				assertPIDGone(t, pid)
			}
		})
	}
}

func TestBlackBoxExitMarkerIsAtomicallyPublished(t *testing.T) {
	binary, fake, runDir, _ := prepareScenario(t, "happy")
	command := newRunCommand(t, binary, fake, runDir, "5s", filepath.Join(repositoryRoot(t), "examples", "brief.md"))
	command.Env = append(command.Env, "WRITE_UUTER_TEST_EXIT_MARKER_DELAY=300ms")
	var output strings.Builder
	command.Stdout = &output
	command.Stderr = &output
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	finished := false
	defer func() {
		if !finished {
			_ = command.Process.Kill()
			_ = command.Wait()
		}
	}()
	deadline := time.Now().Add(5 * time.Second)
	observedTemporary := false
	for time.Now().Before(deadline) {
		temporary, _ := filepath.Glob(filepath.Join(filepath.Dir(runDir), ".write-uuter-private-*", "control", "exits", ".*.exit-*"))
		if len(temporary) != 0 {
			base := filepath.Base(temporary[0])
			separator := strings.Index(base, ".exit-")
			if separator < 1 {
				t.Fatalf("unexpected temporary marker name %s", base)
			}
			final := filepath.Join(filepath.Dir(temporary[0]), strings.TrimPrefix(base[:separator+len(".exit")], "."))
			if _, err := os.Lstat(final); err == nil {
				t.Fatalf("final exit marker appeared while its temporary marker was partial: %s", final)
			}
			observedTemporary = true
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if !observedTemporary {
		_ = command.Process.Kill()
		t.Fatal("did not observe delayed same-directory temporary exit marker")
	}
	if err := command.Wait(); err != nil {
		t.Fatalf("CLI failed after atomic marker publication: %v\n%s", err, output.String())
	}
	finished = true
}

func TestBlackBoxBlockedPersistenceFailureStillRemovesPrivateRuntime(t *testing.T) {
	binary, fake, runDir, fixtureDir := prepareScenario(t, "stale")
	command := newRunCommand(t, binary, fake, runDir, "5s", filepath.Join(repositoryRoot(t), "examples", "brief.md"))
	command.Env = append(command.Env, "WRITE_UUTER_TEST_FAIL_BLOCK_SAVE=1")
	output, err := command.CombinedOutput()
	if err == nil || !strings.Contains(string(output), "persist blocked workflow") {
		t.Fatalf("missing blocked persistence error: %v\n%s", err, output)
	}
	privatePaths, _ := filepath.Glob(filepath.Join(filepath.Dir(runDir), ".write-uuter-private-*"))
	if len(privatePaths) != 0 {
		t.Fatalf("private runtime survived blocked persistence failure: %v", privatePaths)
	}
	assertProcessesGone(t, readInvocationRecords(t, fixtureDir))
}

func TestBlackBoxPrivateCleanupRetriesAfterRemovalFailure(t *testing.T) {
	binary, fake, runDir, fixtureDir := prepareScenario(t, "stale")
	command := newRunCommand(t, binary, fake, runDir, "5s", filepath.Join(repositoryRoot(t), "examples", "brief.md"))
	command.Env = append(command.Env, "WRITE_UUTER_TEST_FAIL_PRIVATE_REMOVE_ONCE=1")
	output, err := command.CombinedOutput()
	if err == nil || !strings.Contains(string(output), "stale review revision") {
		t.Fatalf("missing original blocked reason after cleanup retry: %v\n%s", err, output)
	}
	if strings.Contains(string(output), "private runtime removal failure") {
		t.Fatalf("transient cleanup failure escaped retry: %s", output)
	}
	privatePaths, _ := filepath.Glob(filepath.Join(filepath.Dir(runDir), ".write-uuter-private-*"))
	if len(privatePaths) != 0 {
		t.Fatalf("retry left controller credentials behind: %v", privatePaths)
	}
	assertProcessesGone(t, readInvocationRecords(t, fixtureDir))
}

func TestBlackBoxPersistentCleanupFailurePreservesOwnershipButDeletesCredentials(t *testing.T) {
	binary, fake, runDir, fixtureDir := prepareScenario(t, "stale")
	command := newRunCommand(t, binary, fake, runDir, "5s", filepath.Join(repositoryRoot(t), "examples", "brief.md"))
	command.Env = append(command.Env, "WRITE_UUTER_TEST_FAIL_CLEANUP_PERSISTENT=1")
	output, err := command.CombinedOutput()
	if err == nil || !strings.Contains(string(output), "cleanup failed") {
		t.Fatalf("missing persistent cleanup diagnostic: %v\n%s", err, output)
	}
	privatePaths, globErr := filepath.Glob(filepath.Join(filepath.Dir(runDir), ".write-uuter-private-*"))
	if globErr != nil || len(privatePaths) != 1 {
		t.Fatalf("recoverable controller state was not preserved: %v, %v", privatePaths, globErr)
	}
	privateRoot := privatePaths[0]
	t.Cleanup(func() { _ = os.RemoveAll(privateRoot) })
	if _, err := os.Lstat(filepath.Join(privateRoot, "codex-homes")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("staged credentials survived cleanup failure: %v", err)
	}
	manifests, err := filepath.Glob(filepath.Join(privateRoot, "control", "ownership", "*.json"))
	if err != nil || len(manifests) == 0 {
		t.Fatalf("recoverable ownership identities were deleted: %v, %v", manifests, err)
	}
	assertProcessesGone(t, readInvocationRecords(t, fixtureDir))
}

func TestBlackBoxMissingAuditSourcesBlockSuccess(t *testing.T) {
	for _, source := range []string{"prompt", "log", "exit"} {
		t.Run(source, func(t *testing.T) {
			binary, fake, runDir, _ := prepareScenario(t, "happy")
			command := newRunCommand(t, binary, fake, runDir, "5s", filepath.Join(repositoryRoot(t), "examples", "brief.md"))
			command.Env = append(command.Env, "WRITE_UUTER_TEST_REMOVE_AUDIT="+source)
			output, err := command.CombinedOutput()
			if err == nil {
				t.Fatalf("CLI succeeded without required %s audit: %s", source, output)
			}
			state := readWorkflow(t, runDir)
			if state.Status != "blocked" || !strings.Contains(state.BlockReason, "archive required") {
				t.Fatalf("unexpected workflow: %+v", state)
			}
			assertNoArticle(t, runDir)
		})
	}
}

func TestBlackBoxInvocationDeadlineStartsBeforePublicationAndLaunch(t *testing.T) {
	t.Run("worker prelaunch", func(t *testing.T) {
		binary, fake, runDir, fixtureDir := prepareScenario(t, "happy")
		command := newRunCommand(t, binary, fake, runDir, "500ms", filepath.Join(repositoryRoot(t), "examples", "brief.md"))
		command.Env = append(command.Env, "WRITE_UUTER_TEST_BEFORE_WORKER_START_DELAY=700ms")
		if output, err := command.CombinedOutput(); err == nil {
			t.Fatalf("CLI accepted a late worker launch: %s", output)
		}
		for _, record := range readInvocationRecords(t, fixtureDir) {
			if record.Role == "researcher" {
				t.Fatal("researcher started after its absolute deadline")
			}
		}
	})
	t.Run("PM post-publication", func(t *testing.T) {
		binary, fake, runDir, _ := prepareScenario(t, "optional_invalid")
		command := newRunCommand(t, binary, fake, runDir, "5s", filepath.Join(repositoryRoot(t), "examples", "brief.md"))
		command.Env = append(command.Env,
			"WRITE_UUTER_TEST_PM_DECISION_TIMEOUT=100ms",
			"WRITE_UUTER_TEST_AFTER_PM_REQUEST_DELAY=200ms",
		)
		if output, err := command.CombinedOutput(); err == nil {
			t.Fatalf("CLI accepted a late PM decision: %s", output)
		}
		state := readWorkflow(t, runDir)
		if state.Status != "blocked" || !strings.Contains(state.BlockReason, "PM timed out") {
			t.Fatalf("unexpected workflow: %+v", state)
		}
	})
}

func TestBlackBoxAmbiguousTmuxLaunchIsReconciledAndCleaned(t *testing.T) {
	realTmux, err := exec.LookPath("tmux")
	if err != nil {
		t.Skip("tmux unavailable")
	}
	for _, operation := range []string{"new-session", "new-window"} {
		t.Run(operation, func(t *testing.T) {
			binary, fake, runDir, fixtureDir := prepareScenario(t, "timeout")
			wrapper := filepath.Join(t.TempDir(), "tmux-wrapper")
			script := fmt.Sprintf("#!/bin/sh\nif [ \"$1\" = %q ]; then\n  %q \"$@\"\n  sleep 2\n  exit 0\nfi\nexec %q \"$@\"\n", operation, realTmux, realTmux)
			if err := os.WriteFile(wrapper, []byte(script), 0o755); err != nil {
				t.Fatal(err)
			}
			command := newRunCommand(t, binary, fake, runDir, "400ms", filepath.Join(repositoryRoot(t), "examples", "brief.md"))
			command.Args = append(command.Args, "--tmux", wrapper)
			output, err := command.CombinedOutput()
			if err == nil || (!strings.Contains(string(output), "tmux command timed out") && !strings.Contains(string(output), "ready marker")) {
				t.Fatalf("ambiguous launch was not reconciled as an indeterminate start: %v\n%s", err, output)
			}
			assertProcessesGone(t, readInvocationRecords(t, fixtureDir))
			privatePaths, _ := filepath.Glob(filepath.Join(filepath.Dir(runDir), ".write-uuter-private-*"))
			if len(privatePaths) != 0 {
				t.Fatalf("ambiguous launch left private state: %v", privatePaths)
			}
		})
	}
}

func TestBlackBoxReadyPublicationTimeoutCleansOwnedRunner(t *testing.T) {
	binary, fake, runDir, fixtureDir := prepareScenario(t, "happy")
	command := newRunCommand(t, binary, fake, runDir, "400ms", filepath.Join(repositoryRoot(t), "examples", "brief.md"))
	command.Env = append(command.Env, "WRITE_UUTER_TEST_READY_MARKER_DELAY=2s")
	output, err := command.CombinedOutput()
	if err == nil || !strings.Contains(string(output), "ready marker") {
		t.Fatalf("missing ready-marker timeout: %v\n%s", err, output)
	}
	assertProcessesGone(t, readInvocationRecords(t, fixtureDir))
	privatePaths, _ := filepath.Glob(filepath.Join(filepath.Dir(runDir), ".write-uuter-private-*"))
	if len(privatePaths) != 0 {
		t.Fatalf("unpublished owned runner survived cleanup: %v", privatePaths)
	}
}

func TestBlackBoxAssetTreeSymlinksAreRejected(t *testing.T) {
	for _, scenario := range []string{"asset_root_symlink", "asset_nested_symlink"} {
		t.Run(scenario, func(t *testing.T) {
			run := executeScenario(t, scenario)
			if run.err == nil {
				t.Fatal("CLI unexpectedly accepted symlinked asset tree")
			}
			state := readWorkflow(t, run.runDir)
			if state.Status != "blocked" || !strings.Contains(state.BlockReason, "symlink") {
				t.Fatalf("unexpected workflow: %+v", state)
			}
			assertNoArticle(t, run.runDir)
		})
	}
}

func TestBlackBoxMalformedReviewReportsAndWhitespaceAreRejected(t *testing.T) {
	for _, testCase := range []struct{ scenario, reason string }{
		{"whitespace_finding", "fields must be non-empty"},
		{"incomplete_report", "finding entry"},
	} {
		t.Run(testCase.scenario, func(t *testing.T) {
			run := executeScenario(t, testCase.scenario)
			if run.err == nil {
				t.Fatal("CLI unexpectedly accepted malformed review")
			}
			state := readWorkflow(t, run.runDir)
			if state.Status != "blocked" || !strings.Contains(state.BlockReason, testCase.reason) {
				t.Fatalf("unexpected workflow: %+v", state)
			}
			assertNoArticle(t, run.runDir)
		})
	}
}

func TestBlackBoxHumanReportAcceptsEquivalentFieldLayout(t *testing.T) {
	run := executeScenario(t, "unbulleted_report")
	if run.err != nil {
		t.Fatalf("CLI rejected complete unbulleted report entries: %v\n%s", run.err, run.output)
	}
	if state := readWorkflow(t, run.runDir); state.Status != "succeeded" {
		t.Fatalf("unexpected workflow: %+v", state)
	}
}

func TestBlackBoxMultiplePMDocumentsAreRejected(t *testing.T) {
	run := executeScenario(t, "multiple_pm_documents")
	if run.err == nil {
		t.Fatal("CLI unexpectedly accepted multiple PM documents")
	}
	state := readWorkflow(t, run.runDir)
	if state.Status != "blocked" || !strings.Contains(state.BlockReason, "exactly one complete fenced JSON document") {
		t.Fatalf("unexpected workflow: %+v", state)
	}
	assertNoArticle(t, run.runDir)
}

func TestBlackBoxHungTmuxCommandIsBoundedAndPersistsBlock(t *testing.T) {
	binary, fake, runDir, _ := prepareScenario(t, "happy")
	hangingTmux := filepath.Join(t.TempDir(), "hanging-tmux")
	if err := os.WriteFile(hangingTmux, []byte("#!/bin/sh\nexec sleep 60\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	command := newRunCommand(t, binary, fake, runDir, "150ms", filepath.Join(repositoryRoot(t), "examples", "brief.md"))
	command.Args = append(command.Args, "--tmux", hangingTmux)
	start := time.Now()
	output, err := command.CombinedOutput()
	if err == nil {
		t.Fatalf("CLI unexpectedly succeeded: %s", output)
	}
	if time.Since(start) > 2*time.Second {
		t.Fatalf("hung tmux command exceeded bounded lifecycle timeout: %s", time.Since(start))
	}
	state := readWorkflow(t, runDir)
	if state.Status != "blocked" || !strings.Contains(state.BlockReason, "tmux command timed out") {
		t.Fatalf("unexpected workflow: %+v", state)
	}
	assertNoArticle(t, runDir)
}

func TestBlackBoxWorkerTerminationErrorStopsWorkflowAndSession(t *testing.T) {
	realTmux, err := exec.LookPath("tmux")
	if err != nil {
		t.Skip("tmux unavailable")
	}
	binary, fake, runDir, fixtureDir := prepareScenario(t, "timeout")
	wrapper := filepath.Join(t.TempDir(), "tmux-wrapper")
	script := fmt.Sprintf("#!/bin/sh\nif [ \"$1\" = \"kill-window\" ]; then exit 42; fi\nexec %q \"$@\"\n", realTmux)
	if err := os.WriteFile(wrapper, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	command := newRunCommand(t, binary, fake, runDir, "5s", filepath.Join(repositoryRoot(t), "examples", "brief.md"))
	command.Env = append(command.Env, "WRITE_UUTER_TEST_WORKER_ACTIVE_TIMEOUT=250ms")
	command.Args = append(command.Args, "--tmux", wrapper)
	output, err := command.CombinedOutput()
	if err == nil {
		t.Fatalf("CLI unexpectedly succeeded: %s", output)
	}
	state := readWorkflow(t, runDir)
	if state.Status != "blocked" || !strings.Contains(state.BlockReason, "termination failed") {
		t.Fatalf("unexpected workflow: %+v", state)
	}
	records := readInvocationRecords(t, fixtureDir)
	for _, record := range records {
		if record.Role == "story_editor" {
			t.Fatal("controller started another worker after termination failure")
		}
	}
	assertProcessesGone(t, records)
	assertNoArticle(t, runDir)
}

func TestBlackBoxControllerRejectsSymlinkedDurableArtifactDirectory(t *testing.T) {
	binary, fake, runDir, fixtureDir := prepareScenario(t, "slow_evidence")
	command := newRunCommand(t, binary, fake, runDir, "5s", filepath.Join(repositoryRoot(t), "examples", "brief.md"))
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	waitForPath(t, filepath.Join(runDir, "drafts", "article-001.md"))
	victim := filepath.Join(t.TempDir(), "victim")
	if err := os.Mkdir(victim, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(runDir, "reviews")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(victim, filepath.Join(runDir, "reviews")); err != nil {
		t.Fatal(err)
	}
	if err := command.Wait(); err == nil {
		t.Fatal("CLI unexpectedly followed symlinked durable path")
	}
	state := readWorkflow(t, runDir)
	if state.Status != "blocked" || !strings.Contains(state.BlockReason, "symlink") {
		t.Fatalf("unexpected workflow: %+v", state)
	}
	entries, err := os.ReadDir(victim)
	if err != nil || len(entries) != 0 {
		t.Fatalf("controller wrote outside run through symlink: %v, %v", entries, err)
	}
	assertProcessesGone(t, readInvocationRecords(t, fixtureDir))
	assertNoArticle(t, runDir)
}

func TestBlackBoxPublicationRollsBackWhenSucceededStateCannotPersist(t *testing.T) {
	binary, fake, runDir, fixtureDir := prepareScenario(t, "happy")
	command := newRunCommand(t, binary, fake, runDir, "5s", filepath.Join(repositoryRoot(t), "examples", "brief.md"))
	command.Env = append(command.Env, "WRITE_UUTER_TEST_FAIL_SUCCESS_SAVE=1")
	output, err := command.CombinedOutput()
	if err == nil {
		t.Fatalf("CLI unexpectedly succeeded: %s", output)
	}
	state := readWorkflow(t, runDir)
	if state.Status != "blocked" || !strings.Contains(state.BlockReason, "persist succeeded workflow") {
		t.Fatalf("unexpected workflow: %+v", state)
	}
	assertNoArticle(t, runDir)
	assertProcessesGone(t, readInvocationRecords(t, fixtureDir))
}

func TestBlackBoxInvalidBriefReportsAllProblemsWithoutRun(t *testing.T) {
	binary, _ := buildBinaries(t)
	temporary := t.TempDir()
	brief := filepath.Join(temporary, "invalid.md")
	if err := os.WriteFile(brief, []byte("# Brief\n\n## Question\n   \n## Source hints\noptional\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runDir := filepath.Join(temporary, "run")
	command := exec.Command(binary, "run", "--brief", brief, "--run-dir", runDir, "--prompts-dir", filepath.Join(repositoryRoot(t), "prompts"))
	output, err := command.CombinedOutput()
	if err == nil {
		t.Fatal("CLI unexpectedly succeeded")
	}
	text := string(output)
	for _, expected := range []string{
		"empty section: Question", "missing section: Audience", "missing section: Provisional takeaway",
		"missing section: Scope", "missing section: Out of scope", "missing section: Publication target",
		"missing section: Constraints", "missing section: Done when",
	} {
		if !strings.Contains(text, expected) {
			t.Errorf("diagnostic missing %q:\n%s", expected, text)
		}
	}
	if _, err := os.Stat(runDir); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("invalid brief created run directory: %v", err)
	}
}

func TestBlackBoxExplicitPromptDirectoryNeverFallsBack(t *testing.T) {
	binary, fake := buildBinaries(t)
	temporary := t.TempDir()
	runDir := filepath.Join(temporary, "run")
	missing := filepath.Join(temporary, "missing-prompts")
	command := exec.Command(binary, "run",
		"--brief", filepath.Join(repositoryRoot(t), "examples", "brief.md"),
		"--run-dir", runDir,
		"--codex", fake,
		"--prompts-dir", missing,
	)
	command.Dir = repositoryRoot(t)
	output, err := command.CombinedOutput()
	if err == nil || !strings.Contains(string(output), "explicit prompt directory") {
		t.Fatalf("explicit missing override silently fell back: %v\n%s", err, output)
	}
	if _, err := os.Stat(runDir); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("invalid explicit prompts created a run: %v", err)
	}
	emptyRunDir := filepath.Join(temporary, "empty-run")
	empty := exec.Command(binary, "run",
		"--brief", filepath.Join(repositoryRoot(t), "examples", "brief.md"),
		"--run-dir", emptyRunDir,
		"--codex", fake,
		"--prompts-dir=",
	)
	empty.Dir = repositoryRoot(t)
	output, err = empty.CombinedOutput()
	if err == nil || !strings.Contains(string(output), "explicit prompt directory is empty") {
		t.Fatalf("explicit empty override silently fell back: %v\n%s", err, output)
	}
	if _, err := os.Stat(emptyRunDir); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("empty explicit prompts created a run: %v", err)
	}
}

func TestBlackBoxBriefHeadingsAreCaseInsensitive(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(repositoryRoot(t), "examples", "brief.md"))
	if err != nil {
		t.Fatal(err)
	}
	lower := string(data)
	for _, heading := range []string{"Question", "Audience", "Provisional takeaway", "Scope", "Out of scope", "Publication target", "Constraints", "Done when", "Source hints"} {
		lower = strings.Replace(lower, "## "+heading, "## "+strings.ToLower(heading), 1)
	}
	briefDir := t.TempDir()
	briefPath := filepath.Join(briefDir, "lowercase.md")
	if err := os.WriteFile(briefPath, []byte(lower), 0o644); err != nil {
		t.Fatal(err)
	}
	run := executeScenarioWithBrief(t, "happy", "5s", briefPath)
	if run.err != nil {
		t.Fatalf("lowercase headings failed: %v\n%s", run.err, run.output)
	}
	if state := readWorkflow(t, run.runDir); state.Status != "succeeded" {
		t.Fatalf("unexpected workflow: %+v", state)
	}
	assertProcessesGone(t, readInvocationRecords(t, run.fixtureDir))
}

func TestBlackBoxExistingRunRemainsUnchanged(t *testing.T) {
	binary, fake := buildBinaries(t)
	temporary := t.TempDir()
	runDir := filepath.Join(temporary, "existing")
	if err := os.Mkdir(runDir, 0o755); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(runDir, "keep.txt")
	if err := os.WriteFile(marker, []byte("unchanged"), 0o600); err != nil {
		t.Fatal(err)
	}
	command := exec.Command(binary, "run",
		"--brief", filepath.Join(repositoryRoot(t), "examples", "brief.md"),
		"--run-dir", runDir,
		"--codex", fake,
		"--prompts-dir", filepath.Join(repositoryRoot(t), "prompts"),
	)
	if output, err := command.CombinedOutput(); err == nil {
		t.Fatalf("CLI unexpectedly succeeded: %s", output)
	}
	data, err := os.ReadFile(marker)
	if err != nil || string(data) != "unchanged" {
		t.Fatalf("existing contents changed: %q, %v", data, err)
	}
	entries, err := os.ReadDir(runDir)
	if err != nil || len(entries) != 1 || entries[0].Name() != "keep.txt" {
		t.Fatalf("existing directory changed: %v, %v", entries, err)
	}
}

type scenarioRun struct {
	runDir     string
	fixtureDir string
	output     string
	err        error
}

func executeScenario(t *testing.T, scenario string) scenarioRun {
	t.Helper()
	return executeScenarioWithTimeout(t, scenario, "5s")
}

func executeScenarioWithTimeout(t *testing.T, scenario, timeout string) scenarioRun {
	t.Helper()
	return executeScenarioWithBrief(t, scenario, timeout, filepath.Join(repositoryRoot(t), "examples", "brief.md"))
}

func executeScenarioWithBrief(t *testing.T, scenario, timeout, briefPath string) scenarioRun {
	t.Helper()
	binary, fake, runDir, fixtureDir := prepareScenario(t, scenario)
	command := newRunCommand(t, binary, fake, runDir, timeout, briefPath)
	output, err := command.CombinedOutput()
	return scenarioRun{runDir: runDir, fixtureDir: fixtureDir, output: string(output), err: err}
}

func prepareScenario(t *testing.T, scenario string) (binary, fake, runDir, fixtureDir string) {
	t.Helper()
	binary, fakeTemplate := buildBinaries(t)
	temporary := t.TempDir()
	fixtureDir = filepath.Join(temporary, "fake")
	if err := os.MkdirAll(filepath.Join(fixtureDir, "logs"), 0o755); err != nil {
		t.Fatal(err)
	}
	fake = filepath.Join(fixtureDir, "fake-codex")
	copyFile(t, fakeTemplate, fake, 0o755)
	if err := os.WriteFile(filepath.Join(fixtureDir, "scenario"), []byte(scenario), 0o644); err != nil {
		t.Fatal(err)
	}
	return binary, fake, filepath.Join(temporary, "run"), fixtureDir
}

func newRunCommand(t *testing.T, binary, fake, runDir, timeout, briefPath string) *exec.Cmd {
	t.Helper()
	command := exec.Command(binary, "run",
		"--brief", briefPath,
		"--run-dir", runDir,
		"--codex", fake,
		"--timeout", timeout,
		"--prompts-dir", filepath.Join(repositoryRoot(t), "prompts"),
	)
	command.Dir = repositoryRoot(t)
	command.Env = append(os.Environ(), "WRITE_UUTER_FAKE_LOG_DIR="+filepath.Join(filepath.Dir(fake), "logs"))
	return command
}

func replaceCommandEnv(environment []string, name, value string) []string {
	prefix := name + "="
	result := make([]string, 0, len(environment)+1)
	for _, item := range environment {
		if !strings.HasPrefix(item, prefix) {
			result = append(result, item)
		}
	}
	return append(result, prefix+value)
}

func buildBinaries(t *testing.T) (string, string) {
	t.Helper()
	buildOnce.Do(func() {
		buildDir, buildErr = os.MkdirTemp("", "write-uuter-blackbox-*")
		if buildErr != nil {
			return
		}
		root := repositoryRoot(t)
		for _, item := range []struct {
			output string
			input  string
		}{
			{filepath.Join(buildDir, "write-uuter"), "./cmd/write-uuter"},
			{filepath.Join(buildDir, "fake-codex"), "./internal/app/testdata/fakecodex"},
		} {
			command := exec.Command("go", "build", "-o", item.output, item.input)
			command.Dir = root
			if output, err := command.CombinedOutput(); err != nil {
				buildErr = fmt.Errorf("build %s: %w: %s", item.input, err, output)
				return
			}
		}
	})
	if buildErr != nil {
		t.Fatal(buildErr)
	}
	return filepath.Join(buildDir, "write-uuter"), filepath.Join(buildDir, "fake-codex")
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate repository root")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func readWorkflow(t *testing.T, runDir string) workflowState {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(runDir, "workflow.json"))
	if err != nil {
		t.Fatal(err)
	}
	var state workflowState
	if err := json.Unmarshal(data, &state); err != nil {
		t.Fatal(err)
	}
	return state
}

func readInvocationRecords(t *testing.T, fixtureDir string) []invocationRecord {
	t.Helper()
	paths, err := filepath.Glob(filepath.Join(fixtureDir, "logs", "*.json"))
	if err != nil {
		t.Fatal(err)
	}
	var records []invocationRecord
	for _, path := range paths {
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			t.Fatal(readErr)
		}
		var record invocationRecord
		if unmarshalErr := json.Unmarshal(data, &record); unmarshalErr != nil {
			t.Fatal(unmarshalErr)
		}
		records = append(records, record)
	}
	return records
}

func assertPromptMarkers(t *testing.T, record invocationRecord, present, absent []string) {
	t.Helper()
	for _, marker := range present {
		if !strings.Contains(record.Prompt, marker) {
			t.Errorf("%s prompt missing marker %q", record.Lens, marker)
		}
	}
	for _, marker := range absent {
		if strings.Contains(record.Prompt, marker) {
			t.Errorf("%s prompt leaked marker %q", record.Lens, marker)
		}
	}
}

func assertReviewerFilesystem(t *testing.T, record invocationRecord) {
	t.Helper()
	files := make(map[string]bool, len(record.WorkspaceFiles))
	for _, name := range record.WorkspaceFiles {
		files[name] = true
		for _, forbidden := range []string{"reviews/", "pm-decisions/", ".control/", "workflow.json"} {
			if strings.Contains(name, forbidden) {
				t.Errorf("%s reviewer filesystem leaked %s", record.Lens, name)
			}
		}
	}
	for _, common := range []string{"context/", "context/brief.md", "context/article.md", "context/revision.txt", "result.json", "report.md"} {
		if !files[common] {
			t.Errorf("%s reviewer workspace missing %s: %v", record.Lens, common, record.WorkspaceFiles)
		}
	}
	want := map[string][]string{
		"evidence": {"context/evidence/", "context/evidence/sources.md", "context/claim-ledger.md"},
		"story":    {"context/outline.md"},
		"clarity":  {"context/clarity-fields.md"},
		"copy":     {},
	}
	for _, expected := range want[record.Lens] {
		if !files[expected] {
			t.Errorf("%s reviewer workspace missing lens input %s", record.Lens, expected)
		}
	}
	for lens, lensFiles := range want {
		if lens == record.Lens {
			continue
		}
		for _, forbidden := range lensFiles {
			if files[forbidden] {
				t.Errorf("%s reviewer workspace leaked %s input %s", record.Lens, lens, forbidden)
			}
		}
	}
}

func assertNoArticle(t *testing.T, runDir string) {
	t.Helper()
	if _, err := os.Stat(filepath.Join(runDir, "article.md")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("blocked run contains success-only article.md: %v", err)
	}
}

func waitForPath(t *testing.T, path string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(path); err == nil {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", path)
}

func killPersistentPM(t *testing.T, runDir string) {
	t.Helper()
	readyPaths, err := filepath.Glob(filepath.Join(filepath.Dir(runDir), ".write-uuter-private-*", "control", "ready", "001-pm.ready"))
	if err != nil || len(readyPaths) != 1 {
		t.Fatalf("locate persistent PM ready marker: %v, %v", readyPaths, err)
	}
	data, err := os.ReadFile(readyPaths[0])
	if err != nil {
		t.Fatal(err)
	}
	var identity struct {
		PID int `json:"pid"`
	}
	if err := json.Unmarshal(data, &identity); err != nil {
		t.Fatal(err)
	}
	process, err := os.FindProcess(identity.PID)
	if err != nil {
		t.Fatal(err)
	}
	if err := process.Signal(syscall.SIGKILL); err != nil {
		t.Fatal(err)
	}
	controlDir := filepath.Dir(filepath.Dir(readyPaths[0]))
	waitForPath(t, filepath.Join(controlDir, "exits", "001-pm.exit"))
}

func assertProcessesGone(t *testing.T, records []invocationRecord) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for {
		var alive []int
		for _, record := range records {
			process, err := os.FindProcess(record.PID)
			if err == nil && process.Signal(syscall.Signal(0)) == nil {
				alive = append(alive, record.PID)
			}
		}
		if len(alive) == 0 {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("run-owned fake Codex processes still alive: %v", alive)
		}
		time.Sleep(25 * time.Millisecond)
	}
}

func assertPIDGone(t *testing.T, pid int) {
	t.Helper()
	if pid <= 0 {
		t.Fatalf("invalid detached child PID %d", pid)
	}
	deadline := time.Now().Add(2 * time.Second)
	for {
		process, err := os.FindProcess(pid)
		if err != nil || process.Signal(syscall.Signal(0)) != nil {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("detached child process %d is still alive", pid)
		}
		time.Sleep(25 * time.Millisecond)
	}
}

func assertExactFiles(t *testing.T, first, second string) {
	t.Helper()
	a, err := os.ReadFile(first)
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(second)
	if err != nil {
		t.Fatal(err)
	}
	if string(a) != string(b) {
		t.Errorf("%s is not an exact copy of %s", second, first)
	}
}

func copyFile(t *testing.T, source, target string, mode os.FileMode) {
	t.Helper()
	data, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, data, mode); err != nil {
		t.Fatal(err)
	}
}

func revision(data []byte) string {
	digest := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(digest[:])
}
