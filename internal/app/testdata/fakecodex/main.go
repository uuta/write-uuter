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
}

func main() {
	prompt, _ := io.ReadAll(os.Stdin)
	role := os.Getenv("WRITE_UUTER_ROLE")
	lens := os.Getenv("WRITE_UUTER_LENS")
	candidate, _ := strconv.Atoi(os.Getenv("WRITE_UUTER_CANDIDATE"))
	revision := os.Getenv("WRITE_UUTER_REVISION")
	workDir := os.Getenv("WRITE_UUTER_WORK_DIR")
	invocation := os.Getenv("WRITE_UUTER_INVOCATION")
	executable, _ := os.Executable()
	fixtureDir := filepath.Dir(executable)
	scenarioBytes, _ := os.ReadFile(filepath.Join(fixtureDir, "scenario"))
	scenario := strings.TrimSpace(string(scenarioBytes))
	logDir := os.Getenv("WRITE_UUTER_FAKE_LOG_DIR")
	if logDir == "" {
		logDir = filepath.Join(fixtureDir, "logs")
	}
	_ = os.MkdirAll(logDir, 0o755)
	logPath := filepath.Join(logDir, fmt.Sprintf("%s-%d.json", invocation, os.Getpid()))
	isolation := map[string]string{}
	writeLog := func() {
		writeJSON(logPath, invocationLog{
			PID: os.Getpid(), Role: role, Lens: lens, Candidate: candidate,
			Revision: revision, Invocation: invocation, Prompt: string(prompt),
			Workspace: workDir, WorkspaceFiles: workspaceFiles(workDir), Isolation: isolation,
		})
	}
	writeLog()
	defer writeLog()

	switch role {
	case "pm":
		runPM(workDir, scenario)
	case "researcher":
		if scenario == "detached_child_success" || scenario == "detached_child_block" || scenario == "timeout_detached" {
			startDetachedChild()
		}
		if scenario == "timeout" || scenario == "timeout_detached" {
			time.Sleep(time.Minute)
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
	case "writer":
		target := filepath.Join(workDir, "drafts", fmt.Sprintf("article-%03d.md", candidate))
		if scenario == "partial" {
			mustWrite(target, "# Partial but initially valid\n")
			time.Sleep(300 * time.Millisecond)
		}
		mustWrite(target, fmt.Sprintf("# An inspectable editorial workflow\n\nCANDIDATE_ONLY_MARKER COMPLETE_MARKER version %03d turns a brief into durable evidence, an outline, a draft, and sequential reviews.\n", candidate))
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
	if scenario == "stale" && candidate == 1 && lens == "evidence" {
		result.ReviewedRevision = "sha256:stale"
	}
	if scenario == "invalid_review" && candidate == 1 && lens == "evidence" {
		result.Lens = "story"
	}
	needsFinding := (scenario == "mustfix_once" && candidate == 1 && lens == "evidence") ||
		(scenario == "budget" && lens == "evidence") ||
		(scenario == "human" && candidate == 1 && lens == "evidence") ||
		(scenario == "optional_invalid" && candidate == 1 && lens == "evidence") ||
		(scenario == "invalid_no_reason" && candidate == 1 && lens == "evidence") ||
		(scenario == "mixed" && candidate == 1 && lens == "evidence") ||
		(scenario == "rewrite_history" && candidate == 1 && lens == "evidence") ||
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
	mustJSON(filepath.Join(workDir, "result.json"), result)
}

func standardFinding(id string) finding {
	return finding{ID: id, Severity: "must_fix", Location: "section: opening", Problem: "The opening needs a verified detail.", SuggestedDirection: "Add the supported workflow detail."}
}

func runPM(workDir, scenario string) {
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
			case "mustfix_once", "budget":
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

func probeIsolation(workDir string) {
	privateRoot := filepath.Dir(filepath.Dir(workDir))
	runDir := filepath.Join(filepath.Dir(privateRoot), "run")
	paths := map[string]string{
		"durable":    filepath.Join(runDir, "brief.md"),
		"prior_lens": filepath.Join(runDir, "reviews", "article-001", "evidence", "report.md"),
		"host":       filepath.Join(os.Getenv("HOME"), ".codex", "RTK.md"),
	}
	result := map[string]string{}
	for label, path := range paths {
		if _, err := os.ReadFile(path); err == nil {
			result[label] = "READ_SUCCEEDED"
		} else {
			result[label] = err.Error()
		}
	}
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
	data, _ := json.MarshalIndent(result, "", "  ")
	mustWrite(filepath.Join(os.Getenv("WRITE_UUTER_FAKE_LOG_DIR"), "isolation-"+os.Getenv("WRITE_UUTER_INVOCATION")+".probe"), string(data)+"\n")
}

func startDetachedChild() {
	command := exec.Command("/bin/sh", "-c", "exec sleep 60")
	if err := command.Start(); err != nil {
		panic(err)
	}
	directory := os.Getenv("WRITE_UUTER_TEST_DETACHED_PID_DIR")
	if directory != "" {
		mustWrite(filepath.Join(directory, fmt.Sprintf("%d.pid", command.Process.Pid)), fmt.Sprintf("%d\n", command.Process.Pid))
	}
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

func writeJSON(path string, value any) {
	data, _ := json.MarshalIndent(value, "", "  ")
	_ = os.WriteFile(path, data, 0o644)
}

func mustWrite(path, content string) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		panic(err)
	}
	temporary, err := os.CreateTemp(filepath.Dir(path), ".fake-*")
	if err != nil {
		panic(err)
	}
	name := temporary.Name()
	if _, err := temporary.WriteString(content); err != nil {
		panic(err)
	}
	if err := temporary.Close(); err != nil {
		panic(err)
	}
	if err := os.Rename(name, path); err != nil {
		panic(err)
	}
}
