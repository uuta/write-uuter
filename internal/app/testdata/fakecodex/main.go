package main

import (
	"encoding/json"
	"fmt"
	"io"
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
	Candidate        int    `json:"candidate"`
	Lens             string `json:"lens"`
	ReviewedRevision string `json:"reviewed_revision"`
	ResultPath       string `json:"result_path"`
	DecisionPath     string `json:"decision_path"`
}

type decision struct {
	FindingID string `json:"finding_id"`
	Decision  string `json:"decision"`
	Reason    string `json:"reason,omitempty"`
}

type decisionDocument struct {
	ReviewedRevision string                `json:"reviewed_revision"`
	Lenses           map[string][]decision `json:"lenses"`
}

type invocationLog struct {
	PID        int    `json:"pid"`
	Role       string `json:"role"`
	Lens       string `json:"lens"`
	Candidate  int    `json:"candidate"`
	Revision   string `json:"revision"`
	Invocation string `json:"invocation"`
	Prompt     string `json:"prompt"`
}

func main() {
	prompt, _ := io.ReadAll(os.Stdin)
	role := os.Getenv("WRITE_UUTER_ROLE")
	lens := os.Getenv("WRITE_UUTER_LENS")
	candidate, _ := strconv.Atoi(os.Getenv("WRITE_UUTER_CANDIDATE"))
	revision := os.Getenv("WRITE_UUTER_REVISION")
	runDir := os.Getenv("WRITE_UUTER_RUN_DIR")
	invocation := os.Getenv("WRITE_UUTER_INVOCATION")
	executable, _ := os.Executable()
	fixtureDir := filepath.Dir(executable)
	scenarioBytes, _ := os.ReadFile(filepath.Join(fixtureDir, "scenario"))
	scenario := strings.TrimSpace(string(scenarioBytes))
	logDir := filepath.Join(fixtureDir, "logs")
	_ = os.MkdirAll(logDir, 0o755)
	writeJSON(filepath.Join(logDir, fmt.Sprintf("%s-%d.json", invocation, os.Getpid())), invocationLog{
		PID: os.Getpid(), Role: role, Lens: lens, Candidate: candidate,
		Revision: revision, Invocation: invocation, Prompt: string(prompt),
	})

	switch role {
	case "pm":
		runPM(runDir, scenario)
	case "researcher":
		if scenario == "timeout" {
			time.Sleep(time.Minute)
		}
		mustWrite(filepath.Join(runDir, "evidence", "sources.md"), "# Sources\n\nEVIDENCE_ONLY_MARKER\n\n- Local repository documentation, accessed 2026-08-20.\n")
		mustWrite(filepath.Join(runDir, "claim-ledger.md"), "# Claim ledger\n\n- Fact: supported.\n- Firsthand observation: none.\n- Inference: labeled.\n- Opinion: labeled.\n- Unresolved: none.\n")
	case "story_editor":
		mustWrite(filepath.Join(runDir, "outline.md"), "# Outline\n\n## Workflow STORY_ONLY_MARKER\n\n- Purpose: Explain the workflow.\n- Supporting evidence: Repository docs.\n- Reader takeaway: Durable gates make runs inspectable.\n")
	case "writer":
		mustWrite(filepath.Join(runDir, "drafts", fmt.Sprintf("article-%03d.md", candidate)), fmt.Sprintf("# An inspectable editorial workflow\n\nCANDIDATE_ONLY_MARKER version %03d turns a brief into durable evidence, an outline, a draft, and sequential reviews.\n", candidate))
	default:
		if strings.HasPrefix(role, "reviewer_") {
			runReviewer(runDir, scenario, candidate, lens, revision)
			return
		}
		os.Exit(3)
	}
}

func runReviewer(runDir, scenario string, candidate int, lens, revision string) {
	result := reviewResult{Status: "clean", Lens: lens, ReviewedRevision: revision, Findings: []finding{}}
	if scenario == "stale" && candidate == 1 && lens == "evidence" {
		result.ReviewedRevision = "sha256:stale"
	}
	needsFinding := (scenario == "mustfix_once" && candidate == 1 && lens == "evidence") ||
		(scenario == "budget" && lens == "evidence") ||
		(scenario == "human" && candidate == 1 && lens == "evidence")
	if needsFinding {
		result.Status = "fix_required"
		result.Findings = []finding{{
			ID:                 lens + "-001",
			Severity:           "must_fix",
			Location:           "section: opening",
			Problem:            "The opening needs a verified detail.",
			SuggestedDirection: "Add the supported workflow detail.",
		}}
	}
	directory := filepath.Join(runDir, "reviews", fmt.Sprintf("article-%03d", candidate), lens)
	mustJSON(filepath.Join(directory, "result.json"), result)
	var report strings.Builder
	report.WriteString("# Review report\n\n")
	if len(result.Findings) == 0 {
		report.WriteString("No findings.\n")
	}
	for _, item := range result.Findings {
		fmt.Fprintf(&report, "- ID: %s\n- Severity: %s\n- Location: %s\n- Problem: %s\n- Suggested direction: %s\n", item.ID, item.Severity, item.Location, item.Problem, item.SuggestedDirection)
	}
	mustWrite(filepath.Join(directory, "report.md"), report.String())
}

func runPM(runDir, scenario string) {
	requestPath := filepath.Join(runDir, ".control", "pm-request.json")
	for {
		var workflow struct {
			Status string `json:"status"`
		}
		if readJSON(filepath.Join(runDir, "workflow.json"), &workflow) == nil && (workflow.Status == "succeeded" || workflow.Status == "blocked") {
			return
		}
		var current request
		if readJSON(requestPath, &current) != nil {
			time.Sleep(20 * time.Millisecond)
			continue
		}
		var result reviewResult
		if readJSON(filepath.Join(runDir, filepath.FromSlash(current.ResultPath)), &result) != nil {
			time.Sleep(20 * time.Millisecond)
			continue
		}
		decisionPath := filepath.Join(runDir, filepath.FromSlash(current.DecisionPath))
		document := decisionDocument{ReviewedRevision: current.ReviewedRevision, Lenses: map[string][]decision{}}
		if data, err := os.ReadFile(decisionPath); err == nil {
			if parsed, ok := parseDecisionDocument(data); ok && parsed.ReviewedRevision == current.ReviewedRevision {
				document = parsed
			}
		}
		if _, alreadyDone := document.Lenses[current.Lens]; !alreadyDone {
			items := make([]decision, 0, len(result.Findings))
			for _, item := range result.Findings {
				classification := "valid_optional"
				reason := "The finding is useful but not required."
				switch scenario {
				case "mustfix_once", "budget":
					classification = "valid_must_fix"
					reason = "The supported correction is required."
				case "human":
					classification = "needs_human_judgment"
					reason = "Editorial intent must be chosen by a human."
				}
				items = append(items, decision{FindingID: item.ID, Decision: classification, Reason: reason})
			}
			document.Lenses[current.Lens] = items
			jsonData, _ := json.MarshalIndent(document, "", "  ")
			mustWrite(decisionPath, "# PM decisions\n\n```json\n"+string(jsonData)+"\n```\n")
		}
		for {
			var next request
			if readJSON(requestPath, &next) != nil || next.Lens != current.Lens || next.Candidate != current.Candidate {
				break
			}
			time.Sleep(10 * time.Millisecond)
		}
	}
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
