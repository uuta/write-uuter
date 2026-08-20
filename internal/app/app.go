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

type controller struct {
	config     Config
	brief      briefDocument
	runDir     string
	promptsDir string
	workflow   Workflow
	runtime    *tmuxRuntime
	pm         invocation
	promptSeq  int
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

	control := &controller{config: config, brief: brief, runDir: runDir, promptsDir: promptsDir}
	if err := control.initialize(briefData); err != nil {
		return err
	}
	if err := control.execute(); err != nil {
		return err
	}
	return nil
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

	directories := []string{
		"evidence/assets",
		"drafts",
		"reviews",
		"pm-decisions",
		".control/prompts",
		".control/exits",
		".control/logs",
	}
	for _, relative := range directories {
		if err := os.MkdirAll(filepath.Join(temporary, relative), 0o755); err != nil {
			return fmt.Errorf("initialize workspace: %w", err)
		}
	}
	if err := os.WriteFile(filepath.Join(temporary, "brief.md"), briefData, 0o644); err != nil {
		return fmt.Errorf("copy brief: %w", err)
	}
	launcher := `#!/bin/sh
set +e
"$WRITE_UUTER_CODEX" -s workspace-write -a never -C "$WRITE_UUTER_RUN_DIR" exec --ephemeral --skip-git-repo-check - < "$WRITE_UUTER_PROMPT" > "$WRITE_UUTER_LOG_FILE" 2>&1
status=$?
printf '%s\n' "$status" > "$WRITE_UUTER_EXIT_FILE"
exit "$status"
`
	if err := os.WriteFile(filepath.Join(temporary, ".control", "launch-agent.sh"), []byte(launcher), 0o755); err != nil {
		return fmt.Errorf("write agent launcher: %w", err)
	}
	now := time.Now().UTC()
	control.workflow = Workflow{
		SchemaVersion: workflowSchemaVersion,
		Status:        "running",
		Phase:         "initializing",
		ArtifactPaths: map[string]string{
			"brief":        "brief.md",
			"workflow":     "workflow.json",
			"sources":      "evidence/sources.md",
			"firsthand":    "evidence/firsthand.md",
			"assets":       "evidence/assets",
			"claim_ledger": "claim-ledger.md",
			"outline":      "outline.md",
			"drafts":       "drafts",
			"reviews":      "reviews",
			"pm_decisions": "pm-decisions",
			"article":      "article.md",
		},
		StartedAt: now,
		UpdatedAt: now,
	}
	if err := writeJSONAtomic(filepath.Join(temporary, "workflow.json"), control.workflow); err != nil {
		return fmt.Errorf("write initial workflow: %w", err)
	}
	if _, err := os.Lstat(control.runDir); err == nil {
		return fmt.Errorf("run directory already exists: %s", control.runDir)
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("inspect run directory before commit: %w", err)
	}
	if err := os.Rename(temporary, control.runDir); err != nil {
		return fmt.Errorf("atomically commit run workspace: %w", err)
	}
	keepTemporary = true
	return nil
}

func (control *controller) execute() (result error) {
	runtime, err := newTmuxRuntime(control.config.TmuxExecutable, control.config.CodexExecutable, control.runDir)
	if err != nil {
		return control.block(err.Error())
	}
	control.runtime = runtime
	defer func() {
		if cleanupError := control.runtime.cleanup(); cleanupError != nil && result == nil {
			result = control.block(cleanupError.Error())
		}
	}()

	pmPrompt, err := control.buildPMPrompt()
	if err != nil {
		return control.block(err.Error())
	}
	pmPromptPath, err := control.writeAssignment("pm", pmPrompt)
	if err != nil {
		return control.block(err.Error())
	}
	control.pm, err = control.runtime.startPM(pmPromptPath)
	if err != nil {
		return control.block(err.Error())
	}

	if err := control.runResearcher(); err != nil {
		return control.block(err.Error())
	}
	if err := control.runStoryEditor(); err != nil {
		return control.block(err.Error())
	}
	if err := control.runWriter(1, ""); err != nil {
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
			if status, exited := exitStatus(control.pm.ExitPath); exited {
				return control.block(fmt.Sprintf("PM exited unexpectedly with status %d before terminal success", status))
			}
			candidatePath := control.draftPath(candidate)
			if err := copyExact(candidatePath, filepath.Join(control.runDir, "article.md")); err != nil {
				return control.block(fmt.Sprintf("finalize article: %v", err))
			}
			if err := control.runtime.cleanup(); err != nil {
				return control.block(err.Error())
			}
			now := time.Now().UTC()
			control.workflow.Status = "succeeded"
			control.workflow.Phase = "complete"
			control.workflow.ActiveRole = ""
			control.workflow.CompletedAt = &now
			control.workflow.BlockReason = ""
			if err := control.saveWorkflow(); err != nil {
				return err
			}
			return nil
		}
		if candidate == 3 {
			return control.block("review budget exhausted: candidate article-003 has validated must-fix findings")
		}
		if err := control.runWriter(candidate+1, control.decisionPath(candidate)); err != nil {
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
	briefPath, pathErr := filepath.Abs(control.config.BriefPath)
	if pathErr != nil {
		return pathErr
	}
	prompt := base + fmt.Sprintf("\n\nResolve relative source hints from `%s`.", filepath.Dir(briefPath)) + contextBlock("brief.md", []byte(control.brief.Raw))
	return control.runWorker("researcher", "", 0, "", "research", prompt, func() error {
		sources, err := readNonEmpty(filepath.Join(control.runDir, "evidence", "sources.md"))
		if err != nil {
			return err
		}
		_ = sources
		ledger, err := readNonEmpty(filepath.Join(control.runDir, "claim-ledger.md"))
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
	})
}

func (control *controller) runStoryEditor() error {
	base, err := loadPrompt(control.promptsDir, "story-editor.md")
	if err != nil {
		return err
	}
	prompt := base + contextBlock("brief.md", []byte(control.brief.Raw))
	for _, relative := range []string{"evidence/sources.md", "claim-ledger.md"} {
		data, readErr := os.ReadFile(filepath.Join(control.runDir, relative))
		if readErr != nil {
			return readErr
		}
		prompt += contextBlock(relative, data)
	}
	if firsthand, readErr := os.ReadFile(filepath.Join(control.runDir, "evidence", "firsthand.md")); readErr == nil {
		prompt += contextBlock("evidence/firsthand.md", firsthand)
	} else if !errors.Is(readErr, os.ErrNotExist) {
		return readErr
	}
	return control.runWorker("story_editor", "", 0, "", "story", prompt, func() error {
		outline, err := readNonEmpty(filepath.Join(control.runDir, "outline.md"))
		if err != nil {
			return err
		}
		lower := strings.ToLower(string(outline))
		for _, field := range []string{"purpose", "supporting evidence", "reader takeaway"} {
			if !strings.Contains(lower, field) {
				return fmt.Errorf("outline.md is missing section field %q", field)
			}
		}
		return nil
	})
}

func (control *controller) runWriter(candidate int, decisionPath string) error {
	base, err := loadPrompt(control.promptsDir, "writer.md")
	if err != nil {
		return err
	}
	targetRelative := filepath.ToSlash(filepath.Join("drafts", fmt.Sprintf("article-%03d.md", candidate)))
	prompt := base + fmt.Sprintf("\n\n## Assignment\n\nWrite candidate %03d to `%s`. Do not write or edit any other candidate.", candidate, targetRelative)
	prompt += contextBlock("brief.md", []byte(control.brief.Raw))
	for _, relative := range []string{"evidence/sources.md", "claim-ledger.md", "outline.md"} {
		data, readErr := os.ReadFile(filepath.Join(control.runDir, relative))
		if readErr != nil {
			return readErr
		}
		prompt += contextBlock(relative, data)
	}
	if candidate > 1 {
		previous := control.draftPath(candidate - 1)
		previousData, readErr := os.ReadFile(previous)
		if readErr != nil {
			return readErr
		}
		decisionData, readErr := os.ReadFile(decisionPath)
		if readErr != nil {
			return readErr
		}
		prompt += contextBlock(filepath.ToSlash(filepath.Join("drafts", filepath.Base(previous))), previousData)
		prompt += contextBlock(filepath.ToSlash(filepath.Join("pm-decisions", filepath.Base(decisionPath))), decisionData)
	}
	target := control.draftPath(candidate)
	err = control.runWorker("writer", "", candidate, "", "writing", prompt, func() error {
		data, readErr := readNonEmpty(target)
		if readErr != nil {
			return readErr
		}
		if strings.Contains(strings.ToLower(string(data)), "todo") {
			return fmt.Errorf("candidate contains unresolved TODO placeholder")
		}
		return nil
	})
	if err != nil {
		return err
	}
	data, err := os.ReadFile(target)
	if err != nil {
		return err
	}
	control.workflow.CurrentCandidate = candidate
	control.workflow.CurrentRevision = revisionFor(data)
	control.workflow.ActiveRole = ""
	return control.saveWorkflow()
}

func (control *controller) reviewCandidate(candidate int) (bool, string, error) {
	lenses := []string{"evidence", "story", "clarity", "copy"}
	for _, lens := range lenses {
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
		for _, decision := range decisions {
			switch decision.Decision {
			case "needs_human_judgment":
				return false, fmt.Sprintf("human judgment required for finding %s: %s", decision.FindingID, decision.Reason), nil
			case "valid_must_fix":
				return true, "", nil
			}
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
	candidatePath := control.draftPath(candidate)
	candidateData, err := os.ReadFile(candidatePath)
	if err != nil {
		return result, err
	}
	revision := revisionFor(candidateData)
	if revision != control.workflow.CurrentRevision {
		return result, fmt.Errorf("candidate revision changed outside the controller")
	}
	reviewDirectory := filepath.Join(control.runDir, "reviews", fmt.Sprintf("article-%03d", candidate), lens)
	if err := os.MkdirAll(reviewDirectory, 0o755); err != nil {
		return result, err
	}
	prompt := base + fmt.Sprintf("\n\n## Assignment\n\nLens: `%s`\nCandidate: `article-%03d`\nRevision: `%s`\nWrite only `%s/result.json` and `%s/report.md`; never edit a draft.", lens, candidate, revision, filepath.ToSlash(filepath.Join("reviews", fmt.Sprintf("article-%03d", candidate), lens)), filepath.ToSlash(filepath.Join("reviews", fmt.Sprintf("article-%03d", candidate), lens)))
	prompt += contextBlock("brief.md", []byte(control.brief.Raw))
	prompt += contextBlock(fmt.Sprintf("drafts/article-%03d.md", candidate), candidateData)
	switch lens {
	case "evidence":
		for _, relative := range []string{"evidence/sources.md", "claim-ledger.md"} {
			data, readErr := os.ReadFile(filepath.Join(control.runDir, relative))
			if readErr != nil {
				return result, readErr
			}
			prompt += contextBlock(relative, data)
		}
		if firsthand, readErr := os.ReadFile(filepath.Join(control.runDir, "evidence", "firsthand.md")); readErr == nil {
			prompt += contextBlock("evidence/firsthand.md", firsthand)
		} else if !errors.Is(readErr, os.ErrNotExist) {
			return result, readErr
		}
	case "story":
		outline, readErr := os.ReadFile(filepath.Join(control.runDir, "outline.md"))
		if readErr != nil {
			return result, readErr
		}
		prompt += contextBlock("outline.md", outline)
	case "clarity":
		clarity := fmt.Sprintf("Audience:\n%s\n\nConstraints:\n%s", control.brief.Sections["Audience"], control.brief.Sections["Constraints"])
		prompt += contextBlock("clarity-fields", []byte(clarity))
	case "copy":
		if stylePath := findStyleGuide(control.promptsDir); stylePath != "" {
			style, readErr := os.ReadFile(stylePath)
			if readErr != nil {
				return result, readErr
			}
			prompt += contextBlock("style-guide.md", style)
		}
	}

	control.workflow.ReviewAttemptCount++
	if err := control.saveWorkflow(); err != nil {
		return result, err
	}
	resultPath := filepath.Join(reviewDirectory, "result.json")
	reportPath := filepath.Join(reviewDirectory, "report.md")
	err = control.runWorker("reviewer_"+lens, lens, candidate, revision, "reviewing", prompt, func() error {
		if changed, hashErr := os.ReadFile(candidatePath); hashErr != nil {
			return hashErr
		} else if revisionFor(changed) != revision {
			return fmt.Errorf("%s reviewer edited the candidate draft", lens)
		}
		validated, validateErr := validateReview(resultPath, reportPath, lens, revision)
		if validateErr == nil {
			result = validated
		}
		return validateErr
	})
	return result, err
}

func validateReview(resultPath, reportPath, lens, revision string) (ReviewResult, error) {
	var result ReviewResult
	data, err := readNonEmpty(resultPath)
	if err != nil {
		return result, err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return result, errNotReady
	}
	for _, required := range []string{"status", "lens", "reviewed_revision", "findings"} {
		if _, found := fields[required]; !found {
			return result, fmt.Errorf("review result is missing required field %q", required)
		}
	}
	if strings.TrimSpace(string(fields["findings"])) == "null" {
		return result, fmt.Errorf("review findings must be an array")
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return result, errNotReady
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
	report, err := readNonEmpty(reportPath)
	if err != nil {
		return result, err
	}
	reportText := string(report)
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
	request := pmRequest{
		Candidate:        candidate,
		Lens:             lens,
		ReviewedRevision: control.workflow.CurrentRevision,
		ResultPath:       filepath.ToSlash(filepath.Join("reviews", fmt.Sprintf("article-%03d", candidate), lens, "result.json")),
		ReportPath:       filepath.ToSlash(filepath.Join("reviews", fmt.Sprintf("article-%03d", candidate), lens, "report.md")),
		DecisionPath:     filepath.ToSlash(filepath.Join("pm-decisions", fmt.Sprintf("article-%03d.md", candidate))),
	}
	requestPath := filepath.Join(control.runDir, ".control", "pm-request.json")
	if err := writeJSONAtomic(requestPath, request); err != nil {
		return nil, err
	}
	defer os.Remove(requestPath)
	control.workflow.ActiveRole = "pm"
	if err := control.saveWorkflow(); err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), control.config.AgentTimeout)
	defer cancel()
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()
	for {
		decisions, err := control.validatePMDecision(candidate, lens, result)
		if err == nil {
			control.workflow.ActiveRole = ""
			if saveErr := control.saveWorkflow(); saveErr != nil {
				return nil, saveErr
			}
			return decisions, nil
		}
		if !errors.Is(err, errNotReady) {
			return nil, err
		}
		if status, exited := exitStatus(control.pm.ExitPath); exited {
			return nil, fmt.Errorf("PM exited unexpectedly with status %d before deciding %s review", status, lens)
		}
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("PM timed out deciding %s review: %w", lens, ctx.Err())
		case <-ticker.C:
		}
	}
}

func (control *controller) validatePMDecision(candidate int, lens string, result ReviewResult) ([]PMDecision, error) {
	data, err := readNonEmpty(control.decisionPath(candidate))
	if err != nil {
		return nil, err
	}
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
	decisions, found := document.Lenses[lens]
	if !found {
		return nil, errNotReady
	}
	findings := make(map[string]Finding, len(result.Findings))
	for _, finding := range result.Findings {
		findings[finding.ID] = finding
	}
	seen := make(map[string]bool)
	for _, decision := range decisions {
		if _, exists := findings[decision.FindingID]; !exists {
			return nil, fmt.Errorf("PM decision references unknown finding %q", decision.FindingID)
		}
		if seen[decision.FindingID] {
			return nil, fmt.Errorf("PM decision duplicates finding %q", decision.FindingID)
		}
		seen[decision.FindingID] = true
		switch decision.Decision {
		case "valid_must_fix", "valid_optional", "needs_human_judgment":
		case "invalid":
			if strings.TrimSpace(decision.Reason) == "" {
				return nil, fmt.Errorf("invalid decision for %s requires a reason", decision.FindingID)
			}
		default:
			return nil, fmt.Errorf("invalid PM classification %q", decision.Decision)
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
		return nil, fmt.Errorf("PM decision missing findings: %s", strings.Join(missing, ", "))
	}
	return decisions, nil
}

func (control *controller) runWorker(role, lens string, candidate int, revision, phase, prompt string, validate func() error) error {
	promptPath, err := control.writeAssignment(role, prompt)
	if err != nil {
		return err
	}
	control.workflow.Phase = phase
	control.workflow.ActiveRole = role
	if err := control.saveWorkflow(); err != nil {
		return err
	}
	worker, err := control.runtime.startWorker(role, lens, candidate, revision, promptPath)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), control.config.AgentTimeout)
	defer cancel()
	if err := control.runtime.waitForContract(ctx, control.pm, worker, validate); err != nil {
		return err
	}
	control.workflow.ActiveRole = ""
	return control.saveWorkflow()
}

func (control *controller) buildPMPrompt() (string, error) {
	base, err := loadPrompt(control.promptsDir, "pm.md")
	if err != nil {
		return "", err
	}
	return base + fmt.Sprintf("\n\n## Run assignment\n\nThe run directory is `%s`. Monitor `.control/pm-request.json`, validate each requested review, and atomically update the requested `pm-decisions/article-00N.md`. Wait for one request at a time: use a bounded polling command that returns as soon as either a request exists or workflow status is terminal, then return to reasoning before deciding. After writing a decision, wait only until Go removes or replaces that request, then repeat. Do not start one shell loop that waits for the entire run. Continue until `workflow.json` becomes terminal. Do not create or edit drafts or reviewer reports.", control.runDir), nil
}

func (control *controller) writeAssignment(role, prompt string) (string, error) {
	control.promptSeq++
	path := filepath.Join(control.runDir, ".control", "prompts", fmt.Sprintf("%03d-%s.md", control.promptSeq, strings.ReplaceAll(role, "_", "-")))
	if err := writeFileAtomic(path, []byte(prompt), 0o600); err != nil {
		return "", err
	}
	return path, nil
}

func (control *controller) saveWorkflow() error {
	control.workflow.UpdatedAt = time.Now().UTC()
	return writeJSONAtomic(filepath.Join(control.runDir, "workflow.json"), control.workflow)
}

func (control *controller) block(reason string) error {
	if control.runDir == "" {
		return errors.New(reason)
	}
	if control.runtime != nil {
		_ = control.runtime.cleanup()
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
	return errors.New(reason)
}

func (control *controller) draftPath(candidate int) string {
	return filepath.Join(control.runDir, "drafts", fmt.Sprintf("article-%03d.md", candidate))
}

func (control *controller) decisionPath(candidate int) string {
	return filepath.Join(control.runDir, "pm-decisions", fmt.Sprintf("article-%03d.md", candidate))
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
