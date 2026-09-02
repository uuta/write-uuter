package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type finding struct {
	ID                 string `json:"id"`
	Severity           string `json:"severity"`
	Location           string `json:"location"`
	Problem            string `json:"problem"`
	SuggestedDirection string `json:"suggested_direction"`
}

type reviewResult struct {
	Status           string    `json:"status"`
	Lens             string    `json:"lens"`
	ReviewedRevision string    `json:"reviewed_revision"`
	Findings         []finding `json:"findings"`
}

type request struct {
	RequestID        string `json:"request_id"`
	Candidate        int    `json:"candidate"`
	Lens             string `json:"lens"`
	ReviewedRevision string `json:"reviewed_revision"`
	ReviewDigest     string `json:"review_digest"`
	RequestPath      string `json:"request_path"`
	ContextDirectory string `json:"context_directory"`
	OutputPath       string `json:"output_path"`
}

type decision struct {
	FindingID string `json:"finding_id"`
	Decision  string `json:"decision"`
	Reason    string `json:"reason,omitempty"`
}

type decisionRecord struct {
	RequestID    string     `json:"request_id"`
	ReviewDigest string     `json:"review_digest"`
	Decisions    []decision `json:"decisions"`
}

type decisionDocument struct {
	ReviewedRevision string                    `json:"reviewed_revision"`
	Lenses           map[string]decisionRecord `json:"lenses"`
}

type invocationLog struct {
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
	Args           []string          `json:"args"`
	Executable     string            `json:"executable"`
	ExecutableTag  string            `json:"executable_tag"`
	Environment    map[string]string `json:"environment"`
}

// executableTag reports which fixture executable this process was copied from.
// The harness appends the marker to each fake so a staged copy still proves
// which of --codex/--claude selected it.
const executableTagMarker = "#write-uuter-fake-tag:"

// secretSentinelPrefix is the marker the harness embeds in every fixture
// Cloudflare credential value. The fixture scans every surface it can observe
// for it, so a leak into an environment value, an argument, the prompt, or a
// staged workspace file is reported instead of merely being unlikely.
const secretSentinelPrefix = "write-uuter-fixture-secret-"

func probeSecretLeak(prompt, workDir string) string {
	for _, entry := range os.Environ() {
		if strings.Contains(entry, secretSentinelPrefix) {
			name, _, _ := strings.Cut(entry, "=")
			return "LEAKED:env:" + name
		}
	}
	for index, argument := range os.Args {
		if strings.Contains(argument, secretSentinelPrefix) {
			return "LEAKED:arg:" + strconv.Itoa(index)
		}
	}
	if strings.Contains(prompt, secretSentinelPrefix) {
		return "LEAKED:prompt"
	}
	leak := ""
	_ = filepath.WalkDir(workDir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || leak != "" {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr == nil && strings.Contains(string(data), secretSentinelPrefix) {
			relative, _ := filepath.Rel(workDir, path)
			leak = "LEAKED:file:" + filepath.ToSlash(relative)
		}
		return nil
	})
	if leak != "" {
		return leak
	}
	return "CLEAN"
}

func executableTag() string {
	path, err := os.Executable()
	if err != nil {
		return "unknown:" + err.Error()
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "unknown:" + err.Error()
	}
	window := data
	if len(window) > 256 {
		window = window[len(window)-256:]
	}
	index := strings.LastIndex(string(window), executableTagMarker)
	if index < 0 {
		return "untagged"
	}
	return strings.TrimSpace(strings.SplitN(string(window[index+len(executableTagMarker):]), "\n", 2)[0])
}

// environmentProbe records only whether a credential- or provider-selecting
// variable reached this process. Values are never recorded.
func environmentProbe() map[string]string {
	probe := map[string]string{}
	for _, name := range []string{
		"ANTHROPIC_API_KEY", "ANTHROPIC_AUTH_TOKEN", "ANTHROPIC_BASE_URL", "ANTHROPIC_MODEL",
		"CLAUDE_CODE_USE_BEDROCK", "CLAUDE_CODE_USE_VERTEX", "CLAUDE_CODE_USE_FOUNDRY",
		"AWS_BEARER_TOKEN_BEDROCK", "AWS_ACCESS_KEY_ID", "GOOGLE_APPLICATION_CREDENTIALS",
		"OPENAI_API_KEY", "CODEX_API_KEY",
		"CLOUDFLARE_ACCOUNT_ID", "CLOUDFLARE_API_TOKEN",
	} {
		if _, found := os.LookupEnv(name); found {
			probe[name] = "PRESENT"
		} else {
			probe[name] = "ABSENT"
		}
	}
	for _, name := range []string{"HOME", "CLAUDE_CONFIG_DIR", "CLAUDE_CODE_TMPDIR", "CODEX_HOME"} {
		if value := os.Getenv(name); value != "" {
			probe[name] = value
		} else {
			probe[name] = "ABSENT"
		}
	}
	return probe
}

// runAuthStatus answers the controller Max preflight. The scenario file decides
// which authentication state this fixture reports.
func runAuthStatus(scenario string) {
	switch scenario {
	case "auth_nonzero":
		fmt.Fprintln(os.Stderr, "fake claude: auth status failed")
		os.Exit(3)
	case "auth_malformed":
		fmt.Println("not json at all")
	case "auth_logged_out":
		fmt.Println(`{"loggedIn":false,"authMethod":"none","subscriptionType":"none"}`)
	case "auth_api_key":
		fmt.Println(`{"loggedIn":true,"authMethod":"apiKey","apiProvider":"firstParty","subscriptionType":"none"}`)
	case "auth_not_max":
		fmt.Println(`{"loggedIn":true,"authMethod":"claude.ai","apiProvider":"firstParty","subscriptionType":"pro"}`)
	case "auth_missing_field":
		fmt.Println(`{"loggedIn":true,"authMethod":"claude.ai"}`)
	case "auth_duplicate_key":
		fmt.Println(`{"loggedIn":true,"loggedIn":false,"authMethod":"claude.ai","subscriptionType":"max"}`)
	case "auth_two_documents":
		fmt.Println(`{"loggedIn":true,"authMethod":"claude.ai","subscriptionType":"max"}{"loggedIn":false}`)
	case "auth_oversized":
		// A wrong or broken executable can emit unbounded output. The
		// controller must bound what it keeps and say so, rather than grow
		// with the response or report it as a malformed status document.
		filler := strings.Repeat("x", 1024)
		for index := 0; index < 512; index++ {
			fmt.Println(filler)
		}
	default:
		// The real CLI also returns account identity fields. They are present
		// here so the controller is exercised against, and must ignore, them.
		fmt.Println(`{"loggedIn":true,"authMethod":"claude.ai","apiProvider":"firstParty","email":"fixture@example.invalid","orgId":"00000000-0000-0000-0000-000000000000","orgName":"fixture","subscriptionType":"max"}`)
	}
}

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "__fork_escape":
			runForkEscape()
			return
		case "__fork_escape_child":
			startDetachedChild()
			return
		case "__privilege_reexec":
			runPrivilegeReexecProbe()
			return
		}
	}
	if len(os.Args) > 2 && os.Args[1] == "auth" && os.Args[2] == "status" {
		runAuthStatus(fixtureScenario(""))
		return
	}
	prompt, _ := io.ReadAll(os.Stdin)
	role := os.Getenv("WRITE_UUTER_ROLE")
	lens := os.Getenv("WRITE_UUTER_LENS")
	candidate, _ := strconv.Atoi(os.Getenv("WRITE_UUTER_CANDIDATE"))
	revision := os.Getenv("WRITE_UUTER_REVISION")
	workDir := os.Getenv("WRITE_UUTER_WORK_DIR")
	invocation := os.Getenv("WRITE_UUTER_INVOCATION")
	executable, _ := os.Executable()
	scenario := fixtureScenario(workDir)
	logPath := filepath.Join(workDir, ".write-uuter-test-log.json")
	isolation := map[string]string{}
	writeLog := func() {
		writeJSON(logPath, invocationLog{
			PID: os.Getpid(), Role: role, Lens: lens, Candidate: candidate,
			Revision: revision, Invocation: invocation, Prompt: string(prompt),
			Workspace: workDir, WorkspaceFiles: workspaceFiles(workDir), Isolation: isolation,
			Args: os.Args[1:], Executable: executable, ExecutableTag: executableTag(),
			Environment: environmentProbe(),
		})
	}
	isolation["secret_scan"] = probeSecretLeak(string(prompt), workDir)
	writeLog()
	defer writeLog()

	if scenario == "model_unavailable" && role == "writer" {
		// The provider CLI refuses the declared model. The controller must
		// block with the effective profile instead of retrying elsewhere.
		fmt.Fprintln(os.Stderr, "fake provider: model is not available for this account")
		writeLog()
		os.Exit(7)
	}
	switch role {
	case "pm":
		runPM(workDir, scenario)
	case "researcher":
		if scenario == "detached_child_success" || scenario == "detached_child_block" || scenario == "timeout_detached" {
			startDetachedChild()
			if scenario == "timeout_detached" {
				mustWrite(filepath.Join(workDir, ".write-uuter-detached.ready"), "ready\n")
			}
		}
		if scenario == "timeout" || scenario == "timeout_detached" {
			time.Sleep(time.Minute)
		}
		if scenario == "pm_exit_during_worker" {
			time.Sleep(500 * time.Millisecond)
		}
		mustWrite(filepath.Join(workDir, "evidence", "sources.md"), "# Sources\n\nEVIDENCE_ONLY_MARKER\n\n- Local repository documentation, accessed 2026-08-20.\n")
		mustWrite(filepath.Join(workDir, "claim-ledger.md"), "# Claim ledger\n\n- Fact: supported.\n- Firsthand observation: none.\n- Inference: labeled.\n- Opinion: labeled.\n- Unresolved: none.\n")
		if scenario == "launcher_attack" {
			privateRoot := filepath.Dir(filepath.Dir(workDir))
			attackPath := filepath.Join(privateRoot, "control", "agent-runner")
			if err := os.WriteFile(attackPath, []byte("replaced\n"), 0o500); err == nil {
				os.Exit(91)
			} else {
				isolation["actual_launcher_write"] = err.Error()
			}
		}
		if scenario == "asset_root_symlink" {
			mustWrite(filepath.Join(workDir, "empty-assets", ".keep"), "keep\n")
			_ = os.Remove(filepath.Join(workDir, "empty-assets", ".keep"))
			if err := os.Symlink("../empty-assets", filepath.Join(workDir, "evidence", "assets")); err != nil {
				panic(err)
			}
		}
		if strings.HasPrefix(scenario, "shot_") {
			writeScreenshotScenario(workDir, scenario)
		}
		if scenario == "asset_nested_symlink" {
			if err := os.MkdirAll(filepath.Join(workDir, "evidence", "assets"), 0o755); err != nil {
				panic(err)
			}
			if err := os.MkdirAll(filepath.Join(workDir, "empty-nested"), 0o755); err != nil {
				panic(err)
			}
			if err := os.Symlink("../../empty-nested", filepath.Join(workDir, "evidence", "assets", "nested")); err != nil {
				panic(err)
			}
		}
	case "story_editor":
		mustWrite(filepath.Join(workDir, "outline.md"), "# Outline\n\n## Workflow STORY_ONLY_MARKER\n\n- Purpose: Explain the workflow.\n- Supporting evidence: Repository docs.\n- Reader takeaway: Durable gates make runs inspectable.\n")
	case "visual_editor":
		runVisualEditor(workDir, scenario, candidate, revision)
	case "writer":
		if fileExists(filepath.Join(workDir, "context", "visual-plan.json")) {
			runWriterAssembly(workDir, scenario, candidate)
			return
		}
		target := filepath.Join(workDir, "drafts", fmt.Sprintf("article-%03d-prose.md", candidate))
		if scenario == "partial" {
			mustWrite(target, "# Partial but initially valid\n")
			time.Sleep(300 * time.Millisecond)
		}
		mustWrite(target, proseDraft(candidate))
	default:
		if strings.HasPrefix(role, "reviewer_") {
			runReviewer(workDir, scenario, candidate, lens, revision)
			return
		}
		os.Exit(3)
	}
}

func runReviewer(workDir, scenario string, candidate int, lens, revision string) {
	if scenario == "filesystem_isolation" {
		probeIsolation(workDir)
	}
	if scenario == "slow_evidence" && candidate == 1 && lens == "evidence" {
		time.Sleep(500 * time.Millisecond)
	}
	if scenario == "mutate" && candidate == 1 && lens == "evidence" {
		mustWrite(filepath.Join(workDir, "context", "article.md"), "mutated\n")
	}
	result := reviewResult{Status: "clean", Lens: lens, ReviewedRevision: revision, Findings: []finding{}}
	if candidate == 1 && lens == "evidence" && (scenario == "duplicate_review" || scenario == "unknown_review") {
		mustWrite(filepath.Join(workDir, "report.md"), "# Review report\n\nNo findings.\n")
		payload := fmt.Sprintf("{\"status\":\"clean\",\"lens\":%q,\"reviewed_revision\":%q,\"findings\":[]}", lens, revision)
		if scenario == "duplicate_review" {
			payload = fmt.Sprintf("{\"status\":\"clean\",\"status\":\"clean\",\"lens\":%q,\"reviewed_revision\":%q,\"findings\":[]}", lens, revision)
		} else {
			payload = strings.TrimSuffix(payload, "}") + ",\"unexpected\":true}"
		}
		mustWrite(filepath.Join(workDir, "result.json"), payload+"\n")
		return
	}
	if scenario == "stale" && candidate == 1 && lens == "evidence" {
		result.ReviewedRevision = "sha256:stale"
	}
	if scenario == "invalid_review" && candidate == 1 && lens == "evidence" {
		result.Lens = "story"
	}
	needsFinding := (scenario == "mustfix_once" && candidate == 1 && lens == "evidence") ||
		(scenario == "shot_retry_terminal_later_place" && candidate == 1 && lens == "evidence") ||
		(scenario == "shot_retry_terminal_later_compliant" && candidate == 1 && lens == "evidence") ||
		(scenario == "budget" && lens == "evidence") ||
		(scenario == "human" && candidate == 1 && lens == "evidence") ||
		(scenario == "optional_invalid" && candidate == 1 && lens == "evidence") ||
		(scenario == "invalid_no_reason" && candidate == 1 && lens == "evidence") ||
		(scenario == "mixed" && candidate == 1 && lens == "evidence") ||
		(scenario == "rewrite_history" && candidate == 1 && lens == "evidence") ||
		(scenario == "duplicate_pm" && candidate == 1 && lens == "evidence") ||
		(scenario == "unknown_pm" && candidate == 1 && lens == "evidence") ||
		(scenario == "duplicate_nested_finding" && candidate == 1 && lens == "evidence") ||
		(scenario == "unknown_nested_finding" && candidate == 1 && lens == "evidence") ||
		(scenario == "duplicate_pm_decision" && candidate == 1 && lens == "evidence") ||
		(scenario == "unknown_pm_lens" && candidate == 1 && lens == "evidence") ||
		(scenario == "detached_child_block" && candidate == 1 && lens == "evidence") ||
		(scenario == "unbulleted_report" && candidate == 1 && lens == "evidence") ||
		(scenario == "whitespace_finding" && candidate == 1 && lens == "evidence") ||
		(scenario == "incomplete_report" && candidate == 1 && lens == "evidence")
	if needsFinding {
		result.Status = "fix_required"
		result.Findings = []finding{standardFinding(lens + "-001")}
		if scenario == "optional_invalid" || scenario == "mixed" || scenario == "incomplete_report" {
			result.Findings = append(result.Findings, standardFinding(lens+"-002"))
		}
		if scenario == "whitespace_finding" {
			result.Findings[0].Problem = "   "
		}
	}
	var report strings.Builder
	report.WriteString("# Review report\n\n")
	if len(result.Findings) == 0 {
		report.WriteString("No findings.\n")
	}
	for _, item := range result.Findings {
		fmt.Fprintf(&report, "- ID: %s\n- Severity: %s\n- Location: %s\n- Problem: %s\n- Suggested direction: %s\n", item.ID, item.Severity, item.Location, item.Problem, item.SuggestedDirection)
	}
	if scenario == "incomplete_report" {
		report.Reset()
		report.WriteString("# Review report\n\n")
		fmt.Fprintf(&report, "- ID: %s\n- Severity: %s\n- Location: %s\n- Problem: %s\n- Suggested direction: %s\n", result.Findings[0].ID, result.Findings[0].Severity, result.Findings[0].Location, result.Findings[0].Problem, result.Findings[0].SuggestedDirection)
		fmt.Fprintf(&report, "- ID: %s\n", result.Findings[1].ID)
	}
	if scenario == "unbulleted_report" && candidate == 1 && lens == "evidence" {
		report.Reset()
		report.WriteString("# Review report\n\n")
		item := result.Findings[0]
		fmt.Fprintf(&report, "id: %s\n\nseverity: %s\n\nlocation: %s\n\nproblem: %s\n\nsuggested_direction: %s\n", item.ID, item.Severity, item.Location, item.Problem, item.SuggestedDirection)
	}
	mustWrite(filepath.Join(workDir, "report.md"), report.String())
	if scenario == "symlink_output" && candidate == 1 && lens == "evidence" {
		mustJSON(filepath.Join(workDir, "outside-result.json"), result)
		if err := os.Symlink("outside-result.json", filepath.Join(workDir, "result.json")); err != nil {
			panic(err)
		}
		return
	}
	if scenario == "duplicate_nested_finding" || scenario == "unknown_nested_finding" {
		data, _ := json.MarshalIndent(result, "", "  ")
		if scenario == "duplicate_nested_finding" {
			data = []byte(strings.Replace(string(data), "\"severity\": \"must_fix\"", "\"severity\": \"must_fix\",\n      \"severity\": \"must_fix\"", 1))
		} else {
			data = []byte(strings.Replace(string(data), "\"severity\": \"must_fix\"", "\"severity\": \"must_fix\",\n      \"unexpected\": true", 1))
		}
		mustWrite(filepath.Join(workDir, "result.json"), string(data)+"\n")
		return
	}
	mustJSON(filepath.Join(workDir, "result.json"), result)
}

// proseDraft is the deterministic Writer prose draft. Its second section is a
// deliberately long sequential explanation, so a visual pass that replaces it
// with a diagram or an image measurably shortens the article.
func proseDraft(candidate int) string {
	return fmt.Sprintf("# An inspectable editorial workflow\n\n"+
		"CANDIDATE_ONLY_MARKER COMPLETE_MARKER version %03d turns a brief into durable evidence, an outline, a draft, and sequential reviews.\n\n"+
		"## How the stages connect\n\n%s\n", candidate, longExplanation)
}

const longExplanation = "The controller first reads the brief and validates every required section. " +
	"It then starts one persistent PM process for the whole run. " +
	"The Researcher runs next and records the sources and the claim ledger. " +
	"The Story Editor then builds an outline from that evidence. " +
	"The Writer then drafts one candidate from that outline. " +
	"The Evidence reviewer then inspects the candidate. " +
	"The Story reviewer then inspects the same candidate. " +
	"The Clarity reviewer then inspects it again. " +
	"The Copy reviewer inspects it last. " +
	"The controller then copies the accepted candidate to the published article."

const fakeMermaid = "flowchart TD\n" +
	"    brief[Brief] --> research[Researcher]\n" +
	"    research --> outline[Story Editor]\n" +
	"    outline --> draft[Writer prose draft]\n" +
	"    draft --> visual[Visual Editor]\n" +
	"    visual --> assembly[Writer assembly]\n" +
	"    assembly --> review[Four review lenses]\n" +
	"    review --> article[article.md]"

type visualOpportunity struct {
	ID        string `json:"id"`
	Location  string `json:"location"`
	Action    string `json:"action"`
	Rationale string `json:"rationale"`
	Mermaid   string `json:"mermaid,omitempty"`
	AssetID   string `json:"asset_id,omitempty"`
	AltText   string `json:"alt_text,omitempty"`
}

type visualPlan struct {
	SchemaVersion      int                          `json:"schema_version"`
	SourceRevision     string                       `json:"source_revision"`
	Opportunities      []visualOpportunity          `json:"opportunities"`
	ScreenshotOutcomes []screenshotEditorialOutcome `json:"screenshot_outcomes,omitempty"`
}

type screenshotEditorialOutcome struct {
	RequestID string `json:"request_id"`
	Status    string `json:"status"`
	Reason    string `json:"reason"`
}

type visualInputManifest struct {
	SchemaVersion int `json:"schema_version"`
	Inputs        []struct {
		ID         string `json:"id"`
		Origin     string `json:"origin"`
		Source     string `json:"source"`
		MediaType  string `json:"media_type"`
		ByteSize   int    `json:"byte_size"`
		SHA256     string `json:"sha256"`
		StagedPath string `json:"staged_path"`
	} `json:"inputs"`
}

// stagedVisualInputID returns the first controller-staged visual input ID, so
// the existing-asset scenario places a real staged asset rather than a guess.
func stagedVisualInputID(workDir string) string {
	var manifest visualInputManifest
	if readJSON(filepath.Join(workDir, "context", "visual-inputs.json"), &manifest) != nil || len(manifest.Inputs) == 0 {
		return ""
	}
	return manifest.Inputs[0].ID
}

func stagedScreenshotInputID(workDir string) string {
	var manifest visualInputManifest
	if readJSON(filepath.Join(workDir, "context", "visual-inputs.json"), &manifest) != nil {
		return ""
	}
	for _, input := range manifest.Inputs {
		if input.Origin == "screenshot" {
			return input.ID
		}
	}
	return ""
}

func stagedScreenshotInputIDs(workDir string) []string {
	var manifest visualInputManifest
	if readJSON(filepath.Join(workDir, "context", "visual-inputs.json"), &manifest) != nil {
		return nil
	}
	var ids []string
	for _, input := range manifest.Inputs {
		if input.Origin == "screenshot" {
			ids = append(ids, input.ID)
		}
	}
	return ids
}

func stagedScreenshotAttempt(workDir, requestID string) int {
	var manifest struct {
		Screenshots []struct {
			ID      string `json:"id"`
			Attempt int    `json:"attempt"`
		} `json:"screenshots"`
	}
	if readJSON(filepath.Join(workDir, "context", "evidence", "screenshots.json"), &manifest) != nil {
		return 0
	}
	for _, screenshot := range manifest.Screenshots {
		if screenshot.ID == requestID {
			return screenshot.Attempt
		}
	}
	return 0
}

// runVisualEditor emits one deterministic plan. Each case is either an
// accepted plan shape or a documented controller rejection.
func runVisualEditor(workDir, scenario string, candidate int, revision string) {
	if scenario == "visual_timeout" {
		time.Sleep(time.Minute)
	}
	if scenario == "visual_edit_prose" {
		mustWrite(filepath.Join(workDir, "context", "drafts", fmt.Sprintf("article-%03d-prose.md", candidate)), "mutated\n")
	}
	plan := visualPlan{SchemaVersion: 1, SourceRevision: revision}
	switch scenario {
	case "visual_none":
		plan.Opportunities = []visualOpportunity{
			{ID: "vis-001", Location: "section: How the stages connect", Action: "restructure_text",
				Rationale: "The sequence reads as one long sentence run and is clearer split into short steps."},
			{ID: "vis-002", Location: "section: opening", Action: "none",
				Rationale: "The opening is two sentences long and a visual would add nothing."},
		}
	case "visual_asset", "visual_wrong_asset_target":
		assetID := stagedVisualInputID(workDir)
		plan.Opportunities = []visualOpportunity{
			{ID: "vis-001", Location: "section: How the stages connect", Action: "existing_local_asset",
				AssetID: assetID, AltText: "Staged diagram of the controller stages in order",
				Rationale: "A staged diagram already shows the stage order the paragraph spells out."},
			{ID: "vis-002", Location: "section: opening", Action: "none",
				Rationale: "The opening is two sentences long and a visual would add nothing."},
		}
	case "shot_place":
		assetID := stagedScreenshotInputID(workDir)
		plan.Opportunities = []visualOpportunity{
			{ID: "vis-001", Location: "section: How the stages connect", Action: "existing_local_asset",
				AssetID: assetID, AltText: "Captured report page supporting claim 004",
				Rationale: "The visible captured report matches the request reason, supports claim-004, and belongs beside this explanation."},
			{ID: "vis-002", Location: "section: opening", Action: "none",
				Rationale: "The opening is two sentences long and a visual would add nothing."},
		}
	case "shot_unusable", "shot_retry_exhaust", "shot_retry_terminal_later_place", "shot_retry_terminal_later_compliant":
		if scenario == "shot_retry_terminal_later_place" && candidate > 1 {
			plan.Opportunities = []visualOpportunity{
				{ID: "vis-001", Location: "section: How the stages connect", Action: "existing_local_asset",
					AssetID: "shot-001", AltText: "A twice rejected capture that must remain unavailable",
					Rationale: "This deliberately tries to place terminally rejected pixels from an earlier candidate."},
			}
			plan.ScreenshotOutcomes = append(plan.ScreenshotOutcomes, screenshotEditorialOutcome{
				RequestID: "shot-001", Status: "usable", Reason: "This deliberately attempts to overwrite the terminal rejection.",
			})
			break
		}
		if scenario == "shot_retry_terminal_later_compliant" && candidate > 1 {
			plan.Opportunities = []visualOpportunity{
				{ID: "vis-001", Location: "section: How the stages connect", Action: "none",
					Rationale: "The terminal screenshot is not staged, so this candidate leaves it durably unplaced."},
			}
			break
		}
		plan.Opportunities = []visualOpportunity{
			{ID: "vis-001", Location: "section: How the stages connect", Action: "none",
				Rationale: "Explicit non-placement: the capture is unrelated to claim-004, the request reason, and this workflow article context."},
		}
	case "shot_retry_path_collision":
		plan.Opportunities = []visualOpportunity{
			{ID: "vis-001", Location: "section: How the stages connect", Action: "none",
				Rationale: "Both request-keyed captures were evaluated; neither improves this explanation enough to place."},
		}
	case "shot_retry_success":
		assetID := stagedScreenshotInputID(workDir)
		if stagedScreenshotAttempt(workDir, assetID) == 1 {
			plan.Opportunities = []visualOpportunity{{ID: "vis-001", Location: "section: How the stages connect", Action: "none",
				Rationale: "Explicit first-attempt rejection: the capture does not visibly support claim-004 or the request reason."}}
		} else {
			plan.Opportunities = []visualOpportunity{{ID: "vis-001", Location: "section: How the stages connect", Action: "existing_local_asset",
				AssetID: assetID, AltText: "Replacement report page supporting claim 004",
				Rationale: "The replacement visibly matches the request reason, claim-004, and the surrounding article context."}}
		}
	case "visual_unstaged":
		plan.Opportunities = []visualOpportunity{
			{ID: "vis-001", Location: "section: How the stages connect", Action: "existing_local_asset",
				AssetID: "vin-999", AltText: "An asset the controller never staged",
				Rationale: "Placing an unstaged asset must be rejected."},
		}
	case "visual_bad_action":
		plan.Opportunities = []visualOpportunity{
			{ID: "vis-001", Location: "section: How the stages connect", Action: "screenshot",
				Rationale: "An action outside the supported vocabulary must be rejected."},
		}
	default:
		plan.Opportunities = []visualOpportunity{
			{ID: "vis-001", Location: "section: How the stages connect", Action: "mermaid", Mermaid: fakeMermaid,
				Rationale: "The paragraph describes one sequence of stages, which a flow diagram shows at a glance."},
			{ID: "vis-002", Location: "section: opening", Action: "none",
				Rationale: "The opening is two sentences long and a visual would add nothing."},
		}
	}
	for _, requestID := range stagedScreenshotInputIDs(workDir) {
		status := "usable"
		reason := "The visible capture matches its request reason, supported claims, and intended article context."
		if scenario == "shot_unusable" || scenario == "shot_retry_exhaust" || scenario == "shot_retry_terminal_later_place" || scenario == "shot_retry_terminal_later_compliant" ||
			(scenario == "shot_retry_success" && stagedScreenshotAttempt(workDir, requestID) == 1) ||
			(scenario == "shot_retry_path_collision" && requestID == "shot-001" && stagedScreenshotAttempt(workDir, requestID) == 1) {
			status = "rejected"
			reason = "The visible capture is unusable or unrelated to its request reason, supported claims, and article context."
		}
		plan.ScreenshotOutcomes = append(plan.ScreenshotOutcomes, screenshotEditorialOutcome{RequestID: requestID, Status: status, Reason: reason})
	}
	if scenario == "visual_stale" {
		plan.SourceRevision = "sha256:stale"
	}
	var report strings.Builder
	report.WriteString("# Visual and reading-flow plan\n\n")
	for _, opportunity := range plan.Opportunities {
		fmt.Fprintf(&report, "- ID: %s\n- Location: %s\n- Action: %s\n- Rationale: %s\n\n",
			opportunity.ID, opportunity.Location, opportunity.Action, opportunity.Rationale)
	}
	for _, outcome := range plan.ScreenshotOutcomes {
		fmt.Fprintf(&report, "- Screenshot request: %s\n- Editorial status: %s\n- Editorial reason: %s\n\n", outcome.RequestID, outcome.Status, outcome.Reason)
	}
	if scenario != "visual_missing_report" {
		mustWrite(filepath.Join(workDir, "plan.md"), report.String())
	}
	if scenario == "visual_bad_json" {
		mustWrite(filepath.Join(workDir, "plan.json"), "{\"schema_version\": 1, \"opportunities\": [\n")
		return
	}
	mustJSON(filepath.Join(workDir, "plan.json"), plan)
}

// runWriterAssembly applies whatever plan the controller validated, so the
// fixture exercises the real assembly contract instead of a fixed article.
func runWriterAssembly(workDir, scenario string, candidate int) {
	var plan visualPlan
	if err := readJSON(filepath.Join(workDir, "context", "visual-plan.json"), &plan); err != nil {
		panic(err)
	}
	var article strings.Builder
	fmt.Fprintf(&article, "# An inspectable editorial workflow\n\n"+
		"CANDIDATE_ONLY_MARKER COMPLETE_MARKER version %03d turns a brief into durable evidence, an outline, a draft, and sequential reviews.\n\n"+
		"## How the stages connect\n\n", candidate)
	restructured := false
	for _, opportunity := range plan.Opportunities {
		switch opportunity.Action {
		case "mermaid":
			article.WriteString("```mermaid\n" + opportunity.Mermaid + "\n```\n\n")
		case "existing_local_asset":
			path := stagedAssetPath(workDir, candidate, opportunity.AssetID)
			if scenario == "visual_wrong_asset_target" {
				// A relative path that is not the staged one the plan bound.
				path = "assets/" + filepath.Base(path)
			}
			fmt.Fprintf(&article, "![%s](%s)\n\n", opportunity.AltText, path)
		case "restructure_text":
			restructured = true
		}
	}
	switch {
	case scenario == "visual_duplicate_prose":
		article.WriteString(longExplanation + "\n")
	case restructured:
		article.WriteString("- The controller validates the brief and starts the PM.\n" +
			"- Research, outline, prose draft, visual plan, and assembly follow in order.\n" +
			"- Four sequential lenses review the assembled candidate before publication.\n")
	default:
		article.WriteString("Each stage hands the next a validated artifact.\n")
	}
	article.WriteString(scenarioArticleExtras(scenario))
	mustWrite(filepath.Join(workDir, "drafts", fmt.Sprintf("article-%03d.md", candidate)), article.String())
}

// unplannedInlineImage is the one assembly defect Go checks for: an image
// written in the supported inline form whose target the validated plan never
// placed. It follows the planned visual, so it also proves the scan does not
// stop at the placement it already matched.
const unplannedInlineImage = "\n![An unplanned diagram](visuals/article-999/assets/unplanned.png)\n"

// scenarioArticleExtras appends the Markdown one assembly scenario adds.
func scenarioArticleExtras(scenario string) string {
	if scenario == "visual_unplanned_inline" {
		return unplannedInlineImage
	}
	return ""
}

// stagedAssetPath resolves the exact relative path the controller staged for
// one planned asset, extension included.
func stagedAssetPath(workDir string, candidate int, assetID string) string {
	directory := fmt.Sprintf("visuals/article-%03d/assets", candidate)
	entries, err := os.ReadDir(filepath.Join(workDir, "context", directory))
	if err != nil {
		return directory + "/" + assetID
	}
	for _, entry := range entries {
		if strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name())) == assetID {
			return directory + "/" + entry.Name()
		}
	}
	return directory + "/" + assetID
}

func standardFinding(id string) finding {
	return finding{ID: id, Severity: "must_fix", Location: "section: opening", Problem: "The opening needs a verified detail.", SuggestedDirection: "Add the supported workflow detail."}
}

func runPM(workDir, scenario string) {
	if scenario == "pm_exit_before_ready" {
		os.Exit(23)
	}
	mustWrite(filepath.Join(workDir, "pm-ready"), "ready\n")
	if scenario == "pm_exit_during_worker" {
		time.Sleep(500 * time.Millisecond)
		os.Exit(23)
	}
	for {
		requestPath, err := nextRequestPath(filepath.Join(workDir, "inbox"))
		if err != nil {
			time.Sleep(20 * time.Millisecond)
			continue
		}
		var current request
		if readJSON(requestPath, &current) != nil {
			time.Sleep(20 * time.Millisecond)
			continue
		}
		if filepath.Clean(filepath.Join(workDir, current.RequestPath)) != filepath.Clean(requestPath) {
			os.Exit(4)
		}
		var result reviewResult
		if readJSON(filepath.Join(workDir, current.ContextDirectory, "result.json"), &result) != nil {
			time.Sleep(20 * time.Millisecond)
			continue
		}
		document := decisionDocument{ReviewedRevision: current.ReviewedRevision, Lenses: map[string]decisionRecord{}}
		previousPath := filepath.Join(workDir, current.ContextDirectory, "previous-decision.md")
		if data, err := os.ReadFile(previousPath); err == nil {
			if parsed, ok := parseDecisionDocument(data); ok && parsed.ReviewedRevision == current.ReviewedRevision && !(scenario == "drop_history" && current.Lens == "story") {
				document = parsed
			}
		}
		items := make([]decision, 0, len(result.Findings))
		for index, item := range result.Findings {
			classification := "valid_optional"
			reason := "The finding is useful but not required."
			switch scenario {
			case "mustfix_once", "budget", "shot_retry_terminal_later_place", "shot_retry_terminal_later_compliant":
				classification = "valid_must_fix"
				reason = "The supported correction is required."
			case "human", "detached_child_block":
				classification = "needs_human_judgment"
				reason = "Editorial intent must be chosen by a human."
			case "optional_invalid":
				if index == 1 {
					classification = "invalid"
					reason = "The claimed problem is not present."
				}
			case "invalid_no_reason":
				classification = "invalid"
				reason = ""
			case "mixed":
				if index == 0 {
					classification = "valid_must_fix"
				} else {
					classification = "needs_human_judgment"
					reason = "Editorial intent must be chosen by a human."
				}
			}
			items = append(items, decision{FindingID: item.ID, Decision: classification, Reason: reason})
		}
		document.Lenses[current.Lens] = decisionRecord{RequestID: current.RequestID, ReviewDigest: current.ReviewDigest, Decisions: items}
		if scenario == "prepopulate" && current.Lens == "evidence" {
			document.Lenses["story"] = decisionRecord{RequestID: "future", ReviewDigest: "sha256:future", Decisions: []decision{}}
		}
		if scenario == "rewrite_history" && current.Lens == "story" {
			record := document.Lenses["evidence"]
			record.Decisions[0].Decision = "valid_must_fix"
			record.Decisions[0].Reason = "Rewritten after acceptance."
			document.Lenses["evidence"] = record
		}
		if scenario == "slow_final" && current.Lens == "copy" {
			time.Sleep(500 * time.Millisecond)
		}
		jsonData, _ := json.MarshalIndent(document, "", "  ")
		if scenario == "duplicate_pm" && current.Lens == "evidence" {
			jsonData = []byte(strings.Replace(string(jsonData), "{", "{\n  \"reviewed_revision\": \"sha256:contradictory\",", 1))
		}
		if scenario == "unknown_pm" && current.Lens == "evidence" {
			jsonData = []byte(strings.Replace(string(jsonData), "{", "{\n  \"unexpected\": true,", 1))
		}
		if scenario == "duplicate_pm_decision" && current.Lens == "evidence" {
			jsonData = []byte(strings.Replace(string(jsonData), "\"decision\": \"valid_optional\"", "\"decision\": \"valid_optional\",\n          \"decision\": \"valid_optional\"", 1))
		}
		if scenario == "unknown_pm_lens" && current.Lens == "evidence" {
			jsonData = []byte(strings.Replace(string(jsonData), "\"review_digest\":", "\"unexpected\": true,\n      \"review_digest\":", 1))
		}
		if scenario == "missing_pm_decisions" && current.Lens == "evidence" {
			jsonData = []byte(strings.Replace(string(jsonData), ",\n      \"decisions\": []", "", 1))
		}
		if scenario == "null_pm_decisions" && current.Lens == "evidence" {
			jsonData = []byte(strings.Replace(string(jsonData), "\"decisions\": []", "\"decisions\": null", 1))
		}
		payload := "```json\n" + string(jsonData) + "\n```\n"
		if scenario == "multiple_pm_documents" && current.Lens == "evidence" {
			payload += "```json\n{}\n```\n"
		}
		mustWrite(filepath.Join(workDir, current.OutputPath), payload)
		if scenario == "final_pm_exit" && current.Lens == "copy" {
			os.Exit(9)
		}
		for fileExists(requestPath) {
			time.Sleep(10 * time.Millisecond)
		}
	}
}

// fixtureScenario resolves the scenario for this process. The controller stages
// it into each workspace; a preflight invocation has no workspace and reads the
// fixture directory instead.
func fixtureScenario(workDir string) string {
	if workDir != "" {
		if data, err := os.ReadFile(filepath.Join(workDir, ".write-uuter-test-scenario")); err == nil {
			return strings.TrimSpace(string(data))
		}
	}
	executable, err := os.Executable()
	if err != nil {
		return ""
	}
	data, _ := os.ReadFile(filepath.Join(filepath.Dir(executable), "scenario"))
	return strings.TrimSpace(string(data))
}

// probeClaudeIsolation records the Claude-specific boundary. The staged client
// may read the account record that identifies the logged-in session, but not
// the keychain stores, the rest of the user's Claude state, or the home
// directory; a model-invoked tool - anything reached through a shell - may
// read none of it and may not start the system keychain client at all.
// managedSettingsRoot contains the admin-managed Claude policy tree. The
// directory itself exists on every macOS host, so probing it proves denial
// rather than absence.
const managedSettingsRoot = "/Library/Application Support"

func probeClaudeIsolation(result map[string]string) {
	home := filepath.Join("/Users", os.Getenv("USER"))
	keychain := filepath.Join(home, "Library", "Keychains")
	for label, path := range map[string]string{
		"client_user_claude_config": filepath.Join(home, ".claude.json"),
		"client_user_claude_dir":    filepath.Join(home, ".claude", "settings.json"),
		"client_user_home":          home,
		"client_keychain_file":      keychain,
		// Admin-managed policy still applies under --safe-mode and can inject
		// provider routing, models, keys, and hooks, so neither the managed
		// tree nor the directory that would contain it may be readable. The
		// containing directory exists on every host, which keeps the check
		// from passing merely because the managed tree is absent.
		"client_managed_settings_root": managedSettingsRoot,
		"client_managed_settings":      filepath.Join(managedSettingsRoot, "ClaudeCode"),
	} {
		if _, err := os.ReadDir(path); err == nil {
			result[label] = "READ_SUCCEEDED"
		} else if _, err := os.ReadFile(path); err == nil {
			result[label] = "READ_SUCCEEDED"
		} else {
			result[label] = err.Error()
		}
	}
	for label, script := range map[string]string{
		"tool_user_claude_config":  "cat " + filepath.Join(home, ".claude.json"),
		"tool_keychain_file":       "cat " + filepath.Join(keychain, "login.keychain-db"),
		"tool_keychain_client":     "/usr/bin/security list-keychains",
		"tool_managed_settings":    "ls " + managedSettingsRoot,
		"tool_managed_settings_db": "cat " + filepath.Join(managedSettingsRoot, "ClaudeCode", "managed-settings.json"),
	} {
		command := exec.Command("/bin/sh", "-c", script)
		command.Stdout = io.Discard
		var failure strings.Builder
		command.Stderr = &failure
		if err := command.Run(); err == nil {
			result[label] = "LOOKUP_SUCCEEDED"
		} else {
			result[label] = err.Error() + ":" + failure.String()
		}
	}
}

func probeIsolation(workDir string) {
	privateRoot := filepath.Dir(filepath.Dir(workDir))
	runDir := filepath.Join(filepath.Dir(privateRoot), "run")
	paths := map[string]string{
		"durable":       filepath.Join(runDir, "brief.md"),
		"prior_lens":    filepath.Join(runDir, "reviews", "article-001", "evidence", "report.md"),
		"host":          "/etc/hosts",
		"usr_sibling":   "/usr/local/bin/docker",
		"codex_sibling": filepath.Join(privateRoot, "control", "agent-runner"),
	}
	result := map[string]string{}
	executable, executableErr := os.Executable()
	if executableErr != nil {
		result["exact_client_reexec"] = executableErr.Error()
	} else if output, err := exec.Command(executable, "__privilege_reexec").CombinedOutput(); err == nil {
		result["exact_client_reexec"] = "REEXEC_SUCCEEDED:" + string(output)
	} else {
		result["exact_client_reexec"] = err.Error() + ":" + string(output)
	}
	for label, path := range paths {
		if _, err := os.ReadFile(path); err == nil {
			result[label] = "READ_SUCCEEDED"
		} else {
			result[label] = err.Error()
		}
	}
	if device, err := os.Open("/dev/zero"); err == nil {
		var one [1]byte
		_, readErr := device.Read(one[:])
		_ = device.Close()
		if readErr == nil {
			result["dev_sibling"] = "READ_SUCCEEDED"
		} else {
			result["dev_sibling"] = readErr.Error()
		}
	} else {
		result["dev_sibling"] = err.Error()
	}
	authTool := exec.Command("/bin/cat", filepath.Join(os.Getenv("CODEX_HOME"), "auth.json"))
	if output, err := authTool.CombinedOutput(); err == nil {
		result["tool_auth"] = "READ_SUCCEEDED:" + string(output)
	} else {
		result["tool_auth"] = err.Error() + ":" + string(output)
	}
	networkTool := exec.Command("/usr/bin/nc", "-vz", "-w", "1", "127.0.0.1", "9")
	if output, err := networkTool.CombinedOutput(); err == nil {
		result["tool_network"] = "CONNECT_SUCCEEDED:" + string(output)
	} else {
		result["tool_network"] = err.Error() + ":" + string(output)
	}
	// These commands succeed outside Seatbelt, but their output could contain
	// user data. Discard it and record only whether the protected lookup crossed
	// the model-tool boundary.
	keychainTool := exec.Command("/usr/bin/security", "list-keychains")
	keychainTool.Stdout = io.Discard
	var keychainError strings.Builder
	keychainTool.Stderr = &keychainError
	if err := keychainTool.Run(); err == nil {
		result["tool_keychain"] = "LOOKUP_SUCCEEDED"
	} else {
		result["tool_keychain"] = err.Error() + ":" + keychainError.String()
	}
	pasteboardTool := exec.Command("/usr/bin/osascript", "-e", "use scripting additions", "-e", "clipboard info")
	pasteboardTool.Stdout = io.Discard
	var pasteboardError strings.Builder
	pasteboardTool.Stderr = &pasteboardError
	if err := pasteboardTool.Run(); err == nil {
		result["tool_pasteboard"] = "LOOKUP_SUCCEEDED"
	} else {
		result["tool_pasteboard"] = err.Error() + ":" + pasteboardError.String()
	}
	for _, name := range []string{"HTTP_PROXY", "HTTPS_PROXY"} {
		if value := os.Getenv(name); value != "" {
			result["proxy_"+strings.ToLower(name)] = value
		} else {
			result["proxy_"+strings.ToLower(name)] = "ABSENT"
		}
	}
	result["double_fork"] = probeForkEscape(workDir)
	for label, path := range map[string]string{
		"pm_workspace_parent": filepath.Join(privateRoot, "workspaces"),
		"controller":          filepath.Join(privateRoot, "control"),
	} {
		if _, err := os.ReadDir(path); err == nil {
			result[label] = "READ_SUCCEEDED"
		} else {
			result[label] = err.Error()
		}
	}
	if err := syscall.Kill(1, 0); err == nil {
		result["unrelated_process"] = "SIGNAL_SUCCEEDED"
	} else {
		result["unrelated_process"] = err.Error()
	}
	for _, name := range []string{"WRITE_UUTER_FAKE_LOG_DIR", "WRITE_UUTER_TEST_DETACHED_PID_DIR"} {
		if value := os.Getenv(name); value != "" {
			result[name] = value
		} else {
			result[name] = "ABSENT"
		}
	}
	if os.Getenv("CLAUDE_CODE_TMPDIR") != "" {
		probeClaudeIsolation(result)
	}
	data, _ := json.MarshalIndent(result, "", "  ")
	mustWrite(filepath.Join(workDir, ".write-uuter-isolation.probe"), string(data)+"\n")
}

func runPrivilegeReexecProbe() {
	result := map[string]string{}
	if err := exec.Command("/bin/true").Run(); err == nil {
		result["fork"] = "REACQUIRED"
	} else {
		result["fork"] = err.Error()
	}
	if _, err := os.ReadFile(filepath.Join(os.Getenv("CODEX_HOME"), "auth.json")); err == nil {
		result["auth"] = "REACQUIRED"
	} else {
		result["auth"] = err.Error()
	}
	if err := exec.Command("/usr/bin/nc", "-vz", "-w", "1", "127.0.0.1", "9").Run(); err == nil {
		result["network"] = "REACQUIRED"
	} else {
		result["network"] = err.Error()
	}
	data, _ := json.Marshal(result)
	fmt.Println(string(data))
}

func probeForkEscape(workDir string) string {
	executable, err := os.Executable()
	if err != nil {
		return err.Error()
	}
	data, err := os.ReadFile(executable)
	if err != nil {
		return err.Error()
	}
	helper := filepath.Join(workDir, "model-tool")
	if err := os.WriteFile(helper, data, 0o700); err != nil {
		return err.Error()
	}
	command := exec.Command(helper, "__fork_escape")
	output, err := command.CombinedOutput()
	if err == nil {
		return "ESCAPE_SUCCEEDED:" + string(output)
	}
	return err.Error() + ":" + string(output)
}

func runForkEscape() {
	command := exec.Command(os.Args[0], "__fork_escape_child")
	command.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := command.Start(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(71)
	}
	_ = command.Process.Release()
}

func startDetachedChild() {
	command := exec.Command("/bin/sleep", "60")
	command.Env = []string{"PATH=/usr/bin:/bin"}
	command.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := command.Start(); err != nil {
		panic(err)
	}
	mustWrite(filepath.Join(os.Getenv("WRITE_UUTER_WORK_DIR"), ".write-uuter-detached.pid"), fmt.Sprintf("%d\n", command.Process.Pid))
	pgid, err := syscall.Getpgid(command.Process.Pid)
	if err != nil {
		panic(err)
	}
	mustWrite(filepath.Join(os.Getenv("WRITE_UUTER_WORK_DIR"), ".write-uuter-detached.pgid"), fmt.Sprintf("%d\n", pgid))
	_ = command.Process.Release()
}

func nextRequestPath(inbox string) (string, error) {
	entries, err := os.ReadDir(inbox)
	if err != nil {
		return "", err
	}
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".json") {
			return filepath.Join(inbox, entry.Name()), nil
		}
	}
	return "", os.ErrNotExist
}

func fileExists(path string) bool {
	_, err := os.Lstat(path)
	return err == nil
}

func parseDecisionDocument(data []byte) (decisionDocument, bool) {
	text := string(data)
	start := strings.Index(text, "```json")
	if start < 0 {
		return decisionDocument{}, false
	}
	text = text[start+len("```json"):]
	end := strings.Index(text, "```")
	if end < 0 {
		return decisionDocument{}, false
	}
	var document decisionDocument
	if json.Unmarshal([]byte(text[:end]), &document) != nil {
		return decisionDocument{}, false
	}
	return document, true
}

func workspaceFiles(root string) []string {
	var files []string
	_ = filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || path == root {
			return nil
		}
		relative, _ := filepath.Rel(root, path)
		if entry.IsDir() {
			files = append(files, filepath.ToSlash(relative)+"/")
		} else {
			files = append(files, filepath.ToSlash(relative))
		}
		return nil
	})
	return files
}

func readJSON(path string, target any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, target)
}

func mustJSON(path string, value any) {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		panic(err)
	}
	mustWrite(path, string(data)+"\n")
}

// writeJSON publishes the invocation log. The controller archives this file
// while the fake may still be rewriting it, so truncating in place can hand
// the test harness a partial document; publish it atomically instead.
func writeJSON(path string, value any) {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return
	}
	_ = writeAtomicFile(path, string(data))
}

func mustWrite(path, content string) {
	if err := writeAtomicFile(path, content); err != nil {
		panic(err)
	}
}

func writeAtomicFile(path, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(filepath.Dir(path), ".fake-*")
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
	if _, err := temporary.WriteString(content); err != nil {
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

// screenshotLedger names explicit claim IDs so a screenshot request can be
// linked to a real ledger entry. It still distinguishes all five required
// classifications.
const screenshotLedger = `# Claim ledger

- Fact (claim-004): the public report page shows the described interface.
- Fact (claim-005): the public changelog lists the release.
- Fact (claim-006): the public pricing page lists the tier.
- Fact (claim-007): the public status page lists the incident.
- Fact (claim-008): the public docs page lists the endpoint.
- Firsthand observation: none.
- Inference: labeled.
- Opinion: labeled.
- Unresolved: none.
`

// writeScreenshotScenario emits one deterministic Researcher screenshot
// request artifact. Each case is a documented controller rejection or an
// accepted request shape.
func writeScreenshotScenario(workDir, scenario string) {
	mustWrite(filepath.Join(workDir, "claim-ledger.md"), screenshotLedger)
	target := filepath.Join(workDir, "evidence", "screenshot-requests.json")
	entry := func(id, url, claim string) string {
		return fmt.Sprintf(`{"id":%q,"url":%q,"reason":"Shows the state described by %s","supports":[%q]}`, id, url, claim, claim)
	}
	switch scenario {
	case "shot_none":
		return
	case "shot_asset_dir":
		if err := os.MkdirAll(filepath.Join(workDir, "evidence", "assets", "screenshots"), 0o755); err != nil {
			panic(err)
		}
		return
	case "shot_empty":
		mustWrite(target, "{\"screenshots\": []}\n")
	case "shot_one", "shot_selector", "shot_place", "shot_unusable", "shot_retry_success", "shot_retry_exhaust", "shot_retry_terminal_later_place", "shot_retry_terminal_later_compliant":
		selector := ""
		if scenario == "shot_selector" {
			selector = `,"selector":"main"`
		}
		mustWrite(target, fmt.Sprintf("{\"screenshots\":[%s]}\n",
			strings.TrimSuffix(entry("shot-001", "https://example.com/report", "claim-004"), "}")+selector+"}"))
	case "shot_live":
		// Selected only by the manual authenticated smoke run. No automated
		// test picks it, so the suite never touches the network.
		mustWrite(target, entryDocument(entry("shot-001", "https://developers.cloudflare.com/browser-rendering/", "claim-004")))
	case "shot_multi":
		mustWrite(target, fmt.Sprintf("{\"screenshots\":[%s,%s,%s]}\n",
			entry("shot-001", "https://example.com/report", "claim-004"),
			entry("shot-002", "https://example.com/changelog", "claim-005"),
			entry("shot-003", "https://example.com/pricing", "claim-006")))
	case "shot_retry_path_collision":
		mustWrite(target, fmt.Sprintf("{\"screenshots\":[%s,%s]}\n",
			entry("shot-001", "https://example.com/report", "claim-004"),
			entry("shot-001-attempt-002", "https://example.com/changelog", "claim-005")))
	case "shot_five", "shot_six":
		entries := []string{
			entry("shot-001", "https://example.com/report", "claim-004"),
			entry("shot-002", "https://example.com/changelog", "claim-005"),
			entry("shot-003", "https://example.com/pricing", "claim-006"),
			entry("shot-004", "https://example.com/status", "claim-007"),
			entry("shot-005", "https://example.com/docs", "claim-008"),
		}
		if scenario == "shot_six" {
			entries = append(entries, entry("shot-006", "https://example.com/extra", "claim-004"))
		}
		mustWrite(target, fmt.Sprintf("{\"screenshots\":[%s]}\n", strings.Join(entries, ",")))
	case "shot_bad_json":
		mustWrite(target, "{\"screenshots\":[{\"id\":\"shot-001\"\n")
	case "shot_missing_field":
		mustWrite(target, "{\"shots\":[]}\n")
	case "shot_null_list":
		mustWrite(target, "{\"screenshots\":null}\n")
	case "shot_dup_key":
		mustWrite(target, `{"screenshots":[{"id":"shot-001","id":"shot-002","url":"https://example.com/report","reason":"r","supports":["claim-004"]}]}`+"\n")
	case "shot_dup_id":
		mustWrite(target, fmt.Sprintf("{\"screenshots\":[%s,%s]}\n",
			entry("shot-001", "https://example.com/report", "claim-004"),
			entry("shot-001", "https://example.com/changelog", "claim-005")))
	case "shot_case_id":
		mustWrite(target, fmt.Sprintf("{\"screenshots\":[%s,%s]}\n",
			entry("shot-001", "https://example.com/report", "claim-004"),
			entry("SHOT-001", "https://example.com/changelog", "claim-005")))
	case "shot_unknown_field":
		mustWrite(target, `{"screenshots":[{"id":"shot-001","url":"https://example.com/report","reason":"r","supports":["claim-004"],"crop":true}]}`+"\n")
	case "shot_unknown_claim":
		mustWrite(target, entryDocument(entry("shot-001", "https://example.com/report", "claim-999")))
	case "shot_unsafe_url":
		mustWrite(target, `{"screenshots":[{"id":"shot-001","url":"https://admin:secret@localhost/status","reason":"r","supports":["claim-004"]}]}`+"\n")
	case "shot_unsafe_id":
		mustWrite(target, `{"screenshots":[{"id":"../escape","url":"https://example.com/report","reason":"r","supports":["claim-004"]}]}`+"\n")
	default:
		panic("unknown screenshot scenario " + scenario)
	}
}

func entryDocument(entry string) string {
	return fmt.Sprintf("{\"screenshots\":[%s]}\n", entry)
}
