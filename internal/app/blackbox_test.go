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
	PID            int      `json:"pid"`
	Role           string   `json:"role"`
	Lens           string   `json:"lens"`
	Candidate      int      `json:"candidate"`
	Revision       string   `json:"revision"`
	Invocation     string   `json:"invocation"`
	Prompt         string   `json:"prompt"`
	Workspace      string   `json:"workspace"`
	WorkspaceFiles []string `json:"workspace_files"`
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
	for _, record := range records {
		if strings.HasPrefix(record.Role, "reviewer_") {
			reviewers = append(reviewers, record)
		}
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
	assertProcessesGone(t, readInvocationRecords(t, run.fixtureDir))
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

func TestBlackBoxAtomicCommitDoesNotReplaceCompetingTarget(t *testing.T) {
	binary, fake, runDir, _ := prepareScenario(t, "happy")
	barrier := filepath.Join(t.TempDir(), "commit")
	command := newRunCommand(t, binary, fake, runDir, "5s", filepath.Join(repositoryRoot(t), "examples", "brief.md"))
	command.Env = append(os.Environ(), "WRITE_UUTER_TEST_COMMIT_BARRIER="+barrier)
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
	if _, err := os.Stat(filepath.Join(run.runDir, ".control", "launch-agent.sh")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("launch-critical script was exposed in durable agent-visible run: %v", err)
	}
	if state := readWorkflow(t, run.runDir); state.Status != "succeeded" {
		t.Fatalf("unexpected workflow: %+v", state)
	}
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
	command := newRunCommand(t, binary, fake, runDir, "1s", filepath.Join(repositoryRoot(t), "examples", "brief.md"))
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
	command.Env = append(os.Environ(), "WRITE_UUTER_TEST_FAIL_SUCCESS_SAVE=1")
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
	return command
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
