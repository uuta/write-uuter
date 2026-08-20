package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

type Config struct {
	BriefPath       string
	RunDir          string
	CodexExecutable string
	TmuxExecutable  string
	AgentTimeout    time.Duration
	PromptsDir      string
}

type decisionBinding struct {
	RequestID string
	Digest    string
}

type controller struct {
	config           Config
	brief            briefDocument
	runDir           string
	promptsDir       string
	workflow         Workflow
	store            *artifactStore
	runtime          *tmuxRuntime
	pm               invocation
	reachedLenses    map[int][]string
	decisionBindings map[int]map[string]decisionBinding
}

var jsonFencePattern = regexp.MustCompile("(?s)```json\\s*(\\{.*?\\})\\s*```")

func Run(config Config) error {
	if strings.TrimSpace(config.BriefPath) == "" || strings.TrimSpace(config.RunDir) == "" {
		return fmt.Errorf("--brief and --run-dir are required")
	}
	if config.AgentTimeout <= 0 {
		return fmt.Errorf("--timeout must be greater than zero")
	}
	if config.CodexExecutable == "" {
		config.CodexExecutable = "codex"
	}
	if config.TmuxExecutable == "" {
		config.TmuxExecutable = "tmux"
	}
	briefData, err := os.ReadFile(config.BriefPath)
	if err != nil {
		return fmt.Errorf("read brief: %w", err)
	}
	brief, err := parseBrief(string(briefData))
	if err != nil {
		return err
	}
	promptsDir, err := resolvePromptsDir(config.PromptsDir)
	if err != nil {
		return err
	}
	runDir, err := filepath.Abs(config.RunDir)
	if err != nil {
		return fmt.Errorf("resolve run directory: %w", err)
	}
	if _, err := os.Lstat(runDir); err == nil {
		return fmt.Errorf("run directory already exists: %s", runDir)
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("inspect run directory: %w", err)
	}
	control := &controller{
		config: config, brief: brief, runDir: runDir, promptsDir: promptsDir,
		reachedLenses: make(map[int][]string), decisionBindings: make(map[int]map[string]decisionBinding),
	}
	if err := control.initialize(briefData); err != nil {
		return err
	}
	defer control.store.Close()
	return control.execute()
}

func (control *controller) initialize(briefData []byte) error {
	parent := filepath.Dir(control.runDir)
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return fmt.Errorf("create run parent: %w", err)
	}
	temporary, err := os.MkdirTemp(parent, "."+filepath.Base(control.runDir)+".tmp-*")
	if err != nil {
		return fmt.Errorf("create temporary run workspace: %w", err)
	}
	keepTemporary := false
	defer func() {
		if !keepTemporary {
			_ = os.RemoveAll(temporary)
		}
	}()
	store, err := openArtifactStore(temporary)
	if err != nil {
		return err
	}
	keepStore := false
	defer func() {
		if !keepStore {
			_ = store.Close()
		}
	}()
	for _, relative := range []string{"evidence/assets", "drafts", "reviews", "pm-decisions", ".control/prompts", ".control/exits", ".control/logs"} {
		if err := store.mkdirAll(relative, 0o755); err != nil {
			return fmt.Errorf("initialize workspace: %w", err)
		}
	}
	if err := store.writeAtomic("brief.md", briefData, 0o644); err != nil {
		return fmt.Errorf("copy brief: %w", err)
	}
	now := time.Now().UTC()
	control.workflow = Workflow{
		SchemaVersion: workflowSchemaVersion, Status: "running", Phase: "initializing",
		ArtifactPaths: map[string]string{
			"brief": "brief.md", "workflow": "workflow.json", "sources": "evidence/sources.md",
			"firsthand": "evidence/firsthand.md", "assets": "evidence/assets", "claim_ledger": "claim-ledger.md",
			"outline": "outline.md", "drafts": "drafts", "reviews": "reviews",
			"pm_decisions": "pm-decisions", "article": "article.md",
		},
		StartedAt: now, UpdatedAt: now,
	}
	if err := store.writeJSON("workflow.json", control.workflow); err != nil {
		return fmt.Errorf("write initial workflow: %w", err)
	}
	if err := waitForCommitBarrier(); err != nil {
		return err
	}
	if err := renameNoReplace(temporary, control.runDir); err != nil {
		if _, targetErr := os.Lstat(control.runDir); targetErr == nil {
			return fmt.Errorf("run directory already exists: %s", control.runDir)
		}
		return fmt.Errorf("atomically commit run workspace without replacement: %w", err)
	}
	keepTemporary = true
	keepStore = true
	control.store = store
	return nil
}

func waitForCommitBarrier() error {
	barrier := os.Getenv("WRITE_UUTER_TEST_COMMIT_BARRIER")
	if barrier == "" {
		return nil
	}
	if err := os.WriteFile(barrier+".ready", []byte("ready\n"), 0o600); err != nil {
		return err
	}
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(barrier + ".continue"); err == nil {
			return nil
		}
		time.Sleep(10 * time.Millisecond)
	}
	return fmt.Errorf("test commit barrier timed out")
}

func (control *controller) execute() error {
	runtime, err := newTmuxRuntime(control.config.TmuxExecutable, control.config.CodexExecutable, control.config.AgentTimeout)
	if err != nil {
		return control.block(err.Error())
	}
	control.runtime = runtime
	pmPrompt, err := control.buildPMPrompt()
	if err != nil {
		return control.block(err.Error())
	}
	control.pm, err = control.runtime.prepareInvocation("pm", "", 0, "", pmPrompt)
	if err != nil {
		return control.block(err.Error())
	}
	pmStore, err := control.runtime.workspaceStore(control.pm)
	if err != nil {
		return control.block(err.Error())
	}
	if err := pmStore.mkdirAll("inbox", 0o700); err == nil {
		err = pmStore.mkdirAll("outbox", 0o700)
	}
	_ = pmStore.Close()
	if err != nil {
		return control.block(err.Error())
	}
	if err := control.runtime.startPM(control.pm); err != nil {
		return control.block(err.Error())
	}
	if err := control.runResearcher(); err != nil {
		return control.block(err.Error())
	}
	if err := control.runStoryEditor(); err != nil {
		return control.block(err.Error())
	}
	if err := control.runWriter(1); err != nil {
		return control.block(err.Error())
	}
	for candidate := 1; candidate <= 3; candidate++ {
		mustFix, blockReason, err := control.reviewCandidate(candidate)
		if err != nil {
			return control.block(err.Error())
		}
		if blockReason != "" {
			return control.block(blockReason)
		}
		if !mustFix {
			return control.succeed(candidate)
		}
		if candidate == 3 {
			return control.block("review budget exhausted: candidate article-003 has validated must-fix findings")
		}
		if err := control.runWriter(candidate + 1); err != nil {
			return control.block(err.Error())
		}
	}
	return control.block("review loop ended without a terminal result")
}

func (control *controller) runResearcher() error {
	base, err := loadPrompt(control.promptsDir, "researcher.md")
	if err != nil {
		return err
	}
	briefPath, err := filepath.Abs(control.config.BriefPath)
	if err != nil {
		return err
	}
	prompt := base + fmt.Sprintf("\n\nResolve relative source hints from `%s`. Write outputs relative to this isolated workspace.", filepath.Dir(briefPath)) + contextBlock("brief.md", []byte(control.brief.Raw))
	return control.runWorker("researcher", "", 0, "", "research", prompt,
		func(workspace *artifactStore) error {
			return workspace.writeAtomic("context/brief.md", []byte(control.brief.Raw), 0o444)
		},
		func(workspace *artifactStore) error {
			if _, err := workspace.readNonEmpty("evidence/sources.md"); err != nil {
				return err
			}
			ledger, err := workspace.readNonEmpty("claim-ledger.md")
			if err != nil {
				return err
			}
			lower := strings.ToLower(string(ledger))
			for _, classification := range []string{"fact", "firsthand observation", "inference", "opinion", "unresolved"} {
				if !strings.Contains(lower, classification) {
					return fmt.Errorf("claim-ledger.md does not distinguish %s", classification)
				}
			}
			return nil
		},
		func(workspace *artifactStore) error {
			if _, err := control.store.copyRegularFrom(workspace, "evidence/sources.md", "evidence/sources.md", 0o644); err != nil {
				return err
			}
			if _, err := control.store.copyRegularFrom(workspace, "claim-ledger.md", "claim-ledger.md", 0o644); err != nil {
				return err
			}
			if _, err := workspace.readRegular("evidence/firsthand.md"); err == nil {
				if _, err := control.store.copyRegularFrom(workspace, "evidence/firsthand.md", "evidence/firsthand.md", 0o644); err != nil {
					return err
				}
			} else if !errors.Is(err, os.ErrNotExist) {
				return err
			}
			return control.store.copyTreeFrom(workspace, "evidence/assets", "evidence/assets")
		})
}

func (control *controller) runStoryEditor() error {
	base, err := loadPrompt(control.promptsDir, "story-editor.md")
	if err != nil {
		return err
	}
	prompt := base + contextBlock("brief.md", []byte(control.brief.Raw))
	inputs := []string{"evidence/sources.md", "claim-ledger.md"}
	for _, relative := range inputs {
		data, readErr := control.store.readRegular(relative)
		if readErr != nil {
			return readErr
		}
		prompt += contextBlock(relative, data)
	}
	return control.runWorker("story_editor", "", 0, "", "story", prompt,
		func(workspace *artifactStore) error {
			if err := workspace.writeAtomic("context/brief.md", []byte(control.brief.Raw), 0o444); err != nil {
				return err
			}
			for _, relative := range inputs {
				data, readErr := control.store.readRegular(relative)
				if readErr != nil {
					return readErr
				}
				if err := workspace.writeAtomic(filepath.Join("context", relative), data, 0o444); err != nil {
					return err
				}
			}
			return nil
		},
		func(workspace *artifactStore) error {
			outline, readErr := workspace.readNonEmpty("outline.md")
			if readErr != nil {
				return readErr
			}
			lower := strings.ToLower(string(outline))
			for _, field := range []string{"purpose", "supporting evidence", "reader takeaway"} {
				if !strings.Contains(lower, field) {
					return fmt.Errorf("outline.md is missing section field %q", field)
				}
			}
			return nil
		},
		func(workspace *artifactStore) error {
			_, err := control.store.copyRegularFrom(workspace, "outline.md", "outline.md", 0o644)
			return err
		})
}

func (control *controller) runWriter(candidate int) error {
	base, err := loadPrompt(control.promptsDir, "writer.md")
	if err != nil {
		return err
	}
	target := fmt.Sprintf("drafts/article-%03d.md", candidate)
	prompt := base + fmt.Sprintf("\n\n## Assignment\n\nWrite candidate %03d to `%s` in this isolated workspace.", candidate, target)
	inputs := []string{"brief.md", "evidence/sources.md", "claim-ledger.md", "outline.md"}
	if candidate > 1 {
		inputs = append(inputs, fmt.Sprintf("drafts/article-%03d.md", candidate-1), fmt.Sprintf("pm-decisions/article-%03d.md", candidate-1))
	}
	for _, relative := range inputs {
		data, readErr := control.store.readRegular(relative)
		if readErr != nil {
			return readErr
		}
		prompt += contextBlock(relative, data)
	}
	err = control.runWorker("writer", "", candidate, "", "writing", prompt,
		func(workspace *artifactStore) error {
			for _, relative := range inputs {
				data, readErr := control.store.readRegular(relative)
				if readErr != nil {
					return readErr
				}
				if err := workspace.writeAtomic(filepath.Join("context", relative), data, 0o444); err != nil {
					return err
				}
			}
			return nil
		},
		func(workspace *artifactStore) error {
			data, readErr := workspace.readNonEmpty(target)
			if readErr != nil {
				return readErr
			}
			if strings.Contains(strings.ToLower(string(data)), "todo") {
				return fmt.Errorf("candidate contains unresolved TODO placeholder")
			}
			return nil
		},
		func(workspace *artifactStore) error {
			data, copyErr := control.store.copyRegularFrom(workspace, target, target, 0o644)
			if copyErr != nil {
				return copyErr
			}
			control.workflow.CurrentCandidate = candidate
			control.workflow.CurrentRevision = revisionFor(data)
			return control.saveWorkflow()
		})
	return err
}

func (control *controller) reviewCandidate(candidate int) (bool, string, error) {
	for _, lens := range []string{"evidence", "story", "clarity", "copy"} {
		result, err := control.runReviewer(candidate, lens)
		if err != nil {
			return false, "", err
		}
		if result.Status == "blocked" {
			return false, fmt.Sprintf("%s reviewer blocked candidate %03d", lens, candidate), nil
		}
		decisions, err := control.requestPMDecision(candidate, lens, result)
		if err != nil {
			return false, "", err
		}
		var mustFix *PMDecision
		for index := range decisions {
			decision := decisions[index]
			if decision.Decision == "needs_human_judgment" {
				return false, fmt.Sprintf("human judgment required for finding %s: %s", decision.FindingID, decision.Reason), nil
			}
			if decision.Decision == "valid_must_fix" && mustFix == nil {
				mustFix = &decision
			}
		}
		if mustFix != nil {
			return true, "", nil
		}
	}
	return false, "", nil
}

func (control *controller) runReviewer(candidate int, lens string) (ReviewResult, error) {
	var result ReviewResult
	base, err := loadPrompt(control.promptsDir, "reviewer-"+lens+".md")
	if err != nil {
		return result, err
	}
	outputContract, err := loadPrompt(control.promptsDir, "reviewer-output.md")
	if err != nil {
		return result, err
	}
	base += "\n\n" + outputContract
	candidateRelative := fmt.Sprintf("drafts/article-%03d.md", candidate)
	candidateData, err := control.store.readRegular(candidateRelative)
	if err != nil {
		return result, err
	}
	revision := revisionFor(candidateData)
	if revision != control.workflow.CurrentRevision {
		return result, fmt.Errorf("candidate revision changed outside the controller")
	}
	prompt := base + fmt.Sprintf("\n\n## Assignment\n\nLens: `%s`\nCandidate: `article-%03d`\nRevision: `%s`\nThe `context/` directory contains every permitted input and no other run artifact. Write only `result.json` and `report.md` in this isolated workspace; never edit `context/article.md`.", lens, candidate, revision)
	contextFiles, err := control.reviewerContext(candidate, lens, candidateData, revision)
	if err != nil {
		return result, err
	}
	contextNames := make([]string, 0, len(contextFiles))
	for label := range contextFiles {
		contextNames = append(contextNames, label)
	}
	sort.Strings(contextNames)
	for _, label := range contextNames {
		data := contextFiles[label]
		prompt += contextBlock(label, data)
	}
	control.workflow.ReviewAttemptCount++
	if err := control.saveWorkflow(); err != nil {
		return result, err
	}
	resultData := []byte(nil)
	reportData := []byte(nil)
	err = control.runWorker("reviewer_"+lens, lens, candidate, revision, "reviewing", prompt,
		func(workspace *artifactStore) error {
			for _, relative := range contextNames {
				data := contextFiles[relative]
				target := filepath.Join("context", reviewerContextName(relative))
				if err := workspace.writeAtomic(target, data, 0o444); err != nil {
					return err
				}
			}
			return nil
		},
		func(workspace *artifactStore) error {
			stagedCandidate, readErr := workspace.readRegular("context/article.md")
			if readErr != nil || revisionFor(stagedCandidate) != revision {
				return fmt.Errorf("%s reviewer edited the candidate input", lens)
			}
			resultData, readErr = workspace.readNonEmpty("result.json")
			if readErr != nil {
				return readErr
			}
			reportData, readErr = workspace.readNonEmpty("report.md")
			if readErr != nil {
				return readErr
			}
			validated, validateErr := validateReviewData(resultData, reportData, lens, revision)
			if validateErr == nil {
				result = validated
			}
			return validateErr
		},
		func(_ *artifactStore) error {
			basePath := filepath.Join("reviews", fmt.Sprintf("article-%03d", candidate), lens)
			if err := control.store.writeAtomic(filepath.Join(basePath, "result.json"), resultData, 0o644); err != nil {
				return err
			}
			return control.store.writeAtomic(filepath.Join(basePath, "report.md"), reportData, 0o644)
		})
	return result, err
}

func reviewerContextName(label string) string {
	switch label {
	case "brief.md":
		return "brief.md"
	default:
		if strings.HasPrefix(label, "drafts/article-") {
			return "article.md"
		}
		return label
	}
}

func (control *controller) reviewerContext(candidate int, lens string, candidateData []byte, revision string) (map[string][]byte, error) {
	files := map[string][]byte{
		"brief.md": []byte(control.brief.Raw),
		fmt.Sprintf("drafts/article-%03d.md", candidate): candidateData,
		"revision.txt": []byte(revision + "\n"),
	}
	read := func(relative string) error {
		data, err := control.store.readRegular(relative)
		if err != nil {
			return err
		}
		files[relative] = data
		return nil
	}
	switch lens {
	case "evidence":
		for _, relative := range []string{"evidence/sources.md", "claim-ledger.md"} {
			if err := read(relative); err != nil {
				return nil, err
			}
		}
		if data, err := control.store.readRegular("evidence/firsthand.md"); err == nil {
			files["evidence/firsthand.md"] = data
		} else if !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
	case "story":
		if err := read("outline.md"); err != nil {
			return nil, err
		}
	case "clarity":
		files["clarity-fields.md"] = []byte(fmt.Sprintf("Audience:\n%s\n\nConstraints:\n%s\n", control.brief.Sections["Audience"], control.brief.Sections["Constraints"]))
	case "copy":
		if stylePath := findStyleGuide(control.promptsDir); stylePath != "" {
			data, err := os.ReadFile(stylePath)
			if err != nil {
				return nil, err
			}
			files["style-guide.md"] = data
		}
	}
	return files, nil
}

func validateReviewData(resultData, reportData []byte, lens, revision string) (ReviewResult, error) {
	var result ReviewResult
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(resultData, &fields); err != nil {
		return result, fmt.Errorf("invalid reviewer JSON: %w", err)
	}
	for _, required := range []string{"status", "lens", "reviewed_revision", "findings"} {
		if _, found := fields[required]; !found {
			return result, fmt.Errorf("review result is missing required field %q", required)
		}
	}
	if strings.TrimSpace(string(fields["findings"])) == "null" {
		return result, fmt.Errorf("review findings must be an array")
	}
	if err := json.Unmarshal(resultData, &result); err != nil {
		return result, err
	}
	if result.Lens != lens {
		return result, fmt.Errorf("review lens mismatch: got %q, want %q", result.Lens, lens)
	}
	if result.ReviewedRevision != revision {
		return result, fmt.Errorf("stale review revision: got %q, want %q", result.ReviewedRevision, revision)
	}
	if result.Status != "clean" && result.Status != "fix_required" && result.Status != "blocked" {
		return result, fmt.Errorf("invalid reviewer status %q", result.Status)
	}
	if result.Status == "clean" && len(result.Findings) != 0 {
		return result, fmt.Errorf("clean review contains findings")
	}
	if result.Status == "fix_required" && len(result.Findings) == 0 {
		return result, fmt.Errorf("fix_required review has no findings")
	}
	seen := make(map[string]bool)
	for _, finding := range result.Findings {
		if finding.ID == "" || finding.Severity == "" || finding.Location == "" || finding.Problem == "" || finding.SuggestedDirection == "" {
			return result, fmt.Errorf("review finding fields must be non-empty")
		}
		if seen[finding.ID] {
			return result, fmt.Errorf("duplicate finding ID %q", finding.ID)
		}
		seen[finding.ID] = true
	}
	reportText := string(reportData)
	if strings.Count(reportText, "- ID: ") != len(result.Findings) {
		return result, fmt.Errorf("human report finding count does not match result.json")
	}
	for _, finding := range result.Findings {
		for _, expected := range []string{finding.ID, finding.Severity, finding.Location, finding.Problem, finding.SuggestedDirection} {
			if !strings.Contains(reportText, expected) {
				return result, fmt.Errorf("human report does not match finding %s", finding.ID)
			}
		}
	}
	return result, nil
}

func (control *controller) requestPMDecision(candidate int, lens string, result ReviewResult) ([]PMDecision, error) {
	resultPath := filepath.Join("reviews", fmt.Sprintf("article-%03d", candidate), lens, "result.json")
	reportPath := filepath.Join("reviews", fmt.Sprintf("article-%03d", candidate), lens, "report.md")
	resultData, err := control.store.readRegular(resultPath)
	if err != nil {
		return nil, err
	}
	reportData, err := control.store.readRegular(reportPath)
	if err != nil {
		return nil, err
	}
	requestID := randomToken()
	requestPath := filepath.Join("inbox", requestID+".json")
	contextDirectory := filepath.Join("requests", requestID)
	outputPath := filepath.Join("outbox", requestID+".md")
	request := pmRequest{
		RequestID: requestID, Candidate: candidate, Lens: lens,
		ReviewedRevision: control.workflow.CurrentRevision, ReviewDigest: reviewDigest(resultData, reportData),
		ResultPath: filepath.Join(contextDirectory, "result.json"), ReportPath: filepath.Join(contextDirectory, "report.md"),
		DecisionPath:     fmt.Sprintf("pm-decisions/article-%03d.md", candidate),
		RequestPath:      requestPath,
		ContextDirectory: contextDirectory, OutputPath: outputPath,
	}
	pmStore, err := control.runtime.workspaceStore(control.pm)
	if err != nil {
		return nil, err
	}
	defer pmStore.Close()
	if err := control.stagePMContext(pmStore, request, resultData, reportData); err != nil {
		return nil, err
	}
	if err := pmStore.writeJSON(requestPath, request); err != nil {
		return nil, err
	}
	defer pmStore.remove(requestPath)
	control.workflow.ActiveRole = "pm"
	if err := control.saveWorkflow(); err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), control.config.AgentTimeout)
	defer cancel()
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()
	for {
		data, readErr := pmStore.readNonEmpty(outputPath)
		if readErr == nil {
			decisions, validateErr := control.validatePMDecisionData(data, candidate, lens, result, request)
			if validateErr == nil {
				if err := control.store.writeAtomic(fmt.Sprintf("pm-decisions/article-%03d.md", candidate), data, 0o644); err != nil {
					return nil, err
				}
				if control.decisionBindings[candidate] == nil {
					control.decisionBindings[candidate] = make(map[string]decisionBinding)
				}
				control.decisionBindings[candidate][lens] = decisionBinding{RequestID: requestID, Digest: request.ReviewDigest}
				control.reachedLenses[candidate] = append(control.reachedLenses[candidate], lens)
				control.workflow.ActiveRole = ""
				if err := control.saveWorkflow(); err != nil {
					return nil, err
				}
				return decisions, nil
			}
			if !errors.Is(validateErr, errNotReady) {
				return nil, validateErr
			}
		} else if !errors.Is(readErr, errNotReady) {
			return nil, readErr
		}
		if status, exited := control.runtime.exitStatus(control.pm); exited {
			return nil, fmt.Errorf("PM exited unexpectedly with status %d before deciding %s review", status, lens)
		}
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("PM timed out deciding %s review: %w", lens, ctx.Err())
		case <-ticker.C:
		}
	}
}

func (control *controller) stagePMContext(pmStore *artifactStore, request pmRequest, resultData, reportData []byte) error {
	files := map[string][]byte{
		"result.json": resultData,
		"report.md":   reportData,
		"brief.md":    []byte(control.brief.Raw),
	}
	for _, relative := range []string{"evidence/sources.md", "claim-ledger.md", "outline.md", fmt.Sprintf("drafts/article-%03d.md", request.Candidate)} {
		data, err := control.store.readRegular(relative)
		if err != nil {
			return err
		}
		files[relative] = data
	}
	if previous, err := control.store.readRegular(fmt.Sprintf("pm-decisions/article-%03d.md", request.Candidate)); err == nil {
		files["previous-decision.md"] = previous
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	for relative, data := range files {
		if err := pmStore.writeAtomic(filepath.Join(request.ContextDirectory, relative), data, 0o444); err != nil {
			return err
		}
	}
	return nil
}

func (control *controller) validatePMDecisionData(data []byte, candidate int, lens string, result ReviewResult, request pmRequest) ([]PMDecision, error) {
	match := jsonFencePattern.FindSubmatch(data)
	if len(match) != 2 {
		return nil, errNotReady
	}
	var document PMDecisionDocument
	if err := json.Unmarshal(match[1], &document); err != nil {
		return nil, errNotReady
	}
	if document.ReviewedRevision != control.workflow.CurrentRevision {
		return nil, fmt.Errorf("PM decision revision mismatch: got %q, want %q", document.ReviewedRevision, control.workflow.CurrentRevision)
	}
	expected := append(append([]string{}, control.reachedLenses[candidate]...), lens)
	if len(document.Lenses) != len(expected) {
		return nil, fmt.Errorf("PM decision lens history mismatch: got %d entries, want %d", len(document.Lenses), len(expected))
	}
	for _, expectedLens := range expected {
		record, found := document.Lenses[expectedLens]
		if !found {
			return nil, fmt.Errorf("PM decision missing reached lens %q", expectedLens)
		}
		resultPath := filepath.Join("reviews", fmt.Sprintf("article-%03d", candidate), expectedLens, "result.json")
		reportPath := filepath.Join("reviews", fmt.Sprintf("article-%03d", candidate), expectedLens, "report.md")
		resultData, err := control.store.readRegular(resultPath)
		if err != nil {
			return nil, err
		}
		reportData, err := control.store.readRegular(reportPath)
		if err != nil {
			return nil, err
		}
		validatedResult, err := validateReviewData(resultData, reportData, expectedLens, control.workflow.CurrentRevision)
		if err != nil {
			return nil, err
		}
		binding := control.decisionBindings[candidate][expectedLens]
		if expectedLens == lens {
			binding = decisionBinding{RequestID: request.RequestID, Digest: request.ReviewDigest}
		}
		if record.RequestID != binding.RequestID || record.ReviewDigest != binding.Digest {
			return nil, fmt.Errorf("PM decision for %s is not bound to the active review request", expectedLens)
		}
		if err := validateDecisionList(record.Decisions, validatedResult); err != nil {
			return nil, err
		}
	}
	return document.Lenses[lens].Decisions, nil
}

func validateDecisionList(decisions []PMDecision, result ReviewResult) error {
	findings := make(map[string]Finding, len(result.Findings))
	for _, finding := range result.Findings {
		findings[finding.ID] = finding
	}
	seen := make(map[string]bool)
	for _, decision := range decisions {
		if _, exists := findings[decision.FindingID]; !exists {
			return fmt.Errorf("PM decision references unknown finding %q", decision.FindingID)
		}
		if seen[decision.FindingID] {
			return fmt.Errorf("PM decision duplicates finding %q", decision.FindingID)
		}
		seen[decision.FindingID] = true
		switch decision.Decision {
		case "valid_must_fix", "valid_optional", "needs_human_judgment":
		case "invalid":
			if strings.TrimSpace(decision.Reason) == "" {
				return fmt.Errorf("invalid decision for %s requires a reason", decision.FindingID)
			}
		default:
			return fmt.Errorf("invalid PM classification %q", decision.Decision)
		}
	}
	if len(seen) != len(findings) {
		var missing []string
		for id := range findings {
			if !seen[id] {
				missing = append(missing, id)
			}
		}
		sort.Strings(missing)
		return fmt.Errorf("PM decision missing findings: %s", strings.Join(missing, ", "))
	}
	return nil
}

func (control *controller) runWorker(role, lens string, candidate int, revision, phase, prompt string, prepare, validate, commit func(*artifactStore) error) error {
	inv, err := control.runtime.prepareInvocation(role, lens, candidate, revision, prompt)
	if err != nil {
		return err
	}
	workspace, err := control.runtime.workspaceStore(inv)
	if err != nil {
		return err
	}
	defer workspace.Close()
	if err := prepare(workspace); err != nil {
		return err
	}
	control.workflow.Phase = phase
	control.workflow.ActiveRole = role
	if err := control.saveWorkflow(); err != nil {
		return err
	}
	if err := control.runtime.startWorker(inv); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), control.config.AgentTimeout)
	defer cancel()
	if err := control.runtime.waitForWorker(ctx, control.pm, inv); err != nil {
		return err
	}
	if err := validate(workspace); err != nil {
		return fmt.Errorf("%s artifact contract failed after process completion: %w", role, err)
	}
	if err := commit(workspace); err != nil {
		return err
	}
	control.workflow.ActiveRole = ""
	return control.saveWorkflow()
}

func (control *controller) succeed(candidate int) error {
	if err := control.runtime.cleanup(); err != nil {
		return control.block(fmt.Sprintf("terminal cleanup failed: %v", err))
	}
	if err := control.runtime.archiveAll(control.store); err != nil {
		return control.block(fmt.Sprintf("archive controller audit files: %v", err))
	}
	candidateData, err := control.store.readRegular(fmt.Sprintf("drafts/article-%03d.md", candidate))
	if err != nil {
		return control.block(fmt.Sprintf("read accepted candidate: %v", err))
	}
	if got := revisionFor(candidateData); got != control.workflow.CurrentRevision {
		return control.block(fmt.Sprintf("accepted candidate changed before publication: got %s, want %s", got, control.workflow.CurrentRevision))
	}
	if err := control.validateFinalAudit(candidate); err != nil {
		return control.block(fmt.Sprintf("final review audit invalid: %v", err))
	}
	if err := control.runtime.closePrivate(); err != nil {
		return control.block(fmt.Sprintf("private runtime cleanup failed: %v", err))
	}
	control.runtime = nil
	if err := control.store.writeAtomic("article.md", candidateData, 0o644); err != nil {
		return control.block(fmt.Sprintf("stage final article: %v", err))
	}
	now := time.Now().UTC()
	control.workflow.Status = "succeeded"
	control.workflow.Phase = "complete"
	control.workflow.ActiveRole = ""
	control.workflow.CompletedAt = &now
	control.workflow.BlockReason = ""
	if err := control.saveWorkflow(); err != nil {
		_ = control.store.remove("article.md")
		return control.block(fmt.Sprintf("persist succeeded workflow: %v", err))
	}
	return nil
}

func (control *controller) validateFinalAudit(candidate int) error {
	if len(control.reachedLenses[candidate]) != 4 {
		return fmt.Errorf("final candidate has %d reached lenses, want 4", len(control.reachedLenses[candidate]))
	}
	decisionData, err := control.store.readRegular(fmt.Sprintf("pm-decisions/article-%03d.md", candidate))
	if err != nil {
		return err
	}
	match := jsonFencePattern.FindSubmatch(decisionData)
	if len(match) != 2 {
		return fmt.Errorf("PM decision file has no JSON document")
	}
	var document PMDecisionDocument
	if err := json.Unmarshal(match[1], &document); err != nil {
		return err
	}
	if document.ReviewedRevision != control.workflow.CurrentRevision || len(document.Lenses) != 4 {
		return fmt.Errorf("PM decision final revision/history mismatch")
	}
	for _, lens := range []string{"evidence", "story", "clarity", "copy"} {
		resultPath := filepath.Join("reviews", fmt.Sprintf("article-%03d", candidate), lens, "result.json")
		reportPath := filepath.Join("reviews", fmt.Sprintf("article-%03d", candidate), lens, "report.md")
		resultData, err := control.store.readRegular(resultPath)
		if err != nil {
			return err
		}
		reportData, err := control.store.readRegular(reportPath)
		if err != nil {
			return err
		}
		result, err := validateReviewData(resultData, reportData, lens, control.workflow.CurrentRevision)
		if err != nil {
			return err
		}
		record, found := document.Lenses[lens]
		binding := control.decisionBindings[candidate][lens]
		if !found || record.RequestID != binding.RequestID || record.ReviewDigest != reviewDigest(resultData, reportData) || record.ReviewDigest != binding.Digest {
			return fmt.Errorf("PM decision binding mismatch for %s", lens)
		}
		if err := validateDecisionList(record.Decisions, result); err != nil {
			return err
		}
	}
	return nil
}

func (control *controller) buildPMPrompt() (string, error) {
	base, err := loadPrompt(control.promptsDir, "pm.md")
	if err != nil {
		return "", err
	}
	return base + "\n\n## Runtime protocol\n\nThis is an isolated PM workspace. Poll `inbox/` for one request-ID-named JSON file at a time. Record that exact inbox path, read only the request's `context_directory`, and write the complete decision document atomically to its unique `output_path`. Preserve every prior reached-lens record from `previous-decision.md` and add only the active lens with the request's exact request ID and review digest. After writing, wait until that exact request-specific inbox file is removed; a later request uses a different filename and cannot satisfy this acknowledgement. Then poll for the next file. Continue until the process is terminated by the controller.", nil
}

func (control *controller) saveWorkflow() error {
	if control.workflow.Status == "succeeded" && os.Getenv("WRITE_UUTER_TEST_FAIL_SUCCESS_SAVE") == "1" {
		return fmt.Errorf("injected succeeded workflow persistence failure")
	}
	control.workflow.UpdatedAt = time.Now().UTC()
	return control.store.writeJSON("workflow.json", control.workflow)
}

func (control *controller) block(reason string) error {
	cleanupSucceeded := true
	if control.runtime != nil {
		if cleanupErr := control.runtime.cleanup(); cleanupErr != nil {
			cleanupSucceeded = false
			reason += fmt.Sprintf("; cleanup failed: %v", cleanupErr)
		} else if archiveErr := control.runtime.archiveAll(control.store); archiveErr != nil {
			reason += fmt.Sprintf("; audit archive failed: %v", archiveErr)
		}
	}
	if removeErr := control.store.remove("article.md"); removeErr != nil {
		reason += fmt.Sprintf("; failed to remove success-only article: %v", removeErr)
	}
	now := time.Now().UTC()
	control.workflow.Status = "blocked"
	control.workflow.Phase = "blocked"
	control.workflow.ActiveRole = ""
	control.workflow.BlockReason = reason
	control.workflow.CompletedAt = &now
	if err := control.saveWorkflow(); err != nil {
		return fmt.Errorf("%s (also failed to persist blocked workflow: %v)", reason, err)
	}
	if cleanupSucceeded && control.runtime != nil {
		if err := control.runtime.closePrivate(); err != nil {
			return fmt.Errorf("%s; private runtime cleanup failed: %v", reason, err)
		}
	}
	return errors.New(reason)
}

func findStyleGuide(promptsDir string) string {
	repositoryRoot := filepath.Dir(promptsDir)
	for _, relative := range []string{"STYLE.md", "style-guide.md", "docs/style-guide.md"} {
		candidate := filepath.Join(repositoryRoot, relative)
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate
		}
	}
	return ""
}
