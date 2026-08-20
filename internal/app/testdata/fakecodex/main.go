package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
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
	logDir := filepath.Join(fixtureDir, "logs")
	_ = os.MkdirAll(logDir, 0o755)
	logPath := filepath.Join(logDir, fmt.Sprintf("%s-%d.json", invocation, os.Getpid()))
	writeLog := func() {
		writeJSON(logPath, invocationLog{
			PID: os.Getpid(), Role: role, Lens: lens, Candidate: candidate,
			Revision: revision, Invocation: invocation, Prompt: string(prompt),
			Workspace: workDir, WorkspaceFiles: workspaceFiles(workDir),
		})
	}
	writeLog()
	defer writeLog()

	switch role {
	case "pm":
		runPM(workDir, scenario)
	case "researcher":
		if scenario == "timeout" {
			time.Sleep(time.Minute)
		}
		mustWrite(filepath.Join(workDir, "evidence", "sources.md"), "# Sources\n\nEVIDENCE_ONLY_MARKER\n\n- Local repository documentation, accessed 2026-08-20.\n")
		mustWrite(filepath.Join(workDir, "claim-ledger.md"), "# Claim ledger\n\n- Fact: supported.\n- Firsthand observation: none.\n- Inference: labeled.\n- Opinion: labeled.\n- Unresolved: none.\n")
		if scenario == "launcher_attack" {
			mustWrite(filepath.Join(workDir, "launch-agent.sh"), "#!/bin/sh\nexit 99\n")
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
		(scenario == "mixed" && candidate == 1 && lens == "evidence")
	if needsFinding {
		result.Status = "fix_required"
		result.Findings = []finding{standardFinding(lens + "-001")}
		if scenario == "optional_invalid" || scenario == "mixed" {
			result.Findings = append(result.Findings, standardFinding(lens+"-002"))
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
			case "human":
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
		if scenario == "slow_final" && current.Lens == "copy" {
			time.Sleep(500 * time.Millisecond)
		}
		jsonData, _ := json.MarshalIndent(document, "", "  ")
		mustWrite(filepath.Join(workDir, current.OutputPath), "# PM decisions\n\n```json\n"+string(jsonData)+"\n```\n")
		for fileExists(requestPath) {
			time.Sleep(10 * time.Millisecond)
		}
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
