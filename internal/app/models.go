package app

import "time"

const workflowSchemaVersion = 1

type Workflow struct {
	SchemaVersion      int               `json:"schema_version"`
	Status             string            `json:"status"`
	Phase              string            `json:"phase"`
	CurrentCandidate   int               `json:"current_candidate"`
	CurrentRevision    string            `json:"current_revision"`
	ActiveRole         string            `json:"active_role"`
	ArtifactPaths      map[string]string `json:"artifact_paths"`
	ReviewAttemptCount int               `json:"review_attempt_count"`
	StartedAt          time.Time         `json:"started_at"`
	UpdatedAt          time.Time         `json:"updated_at"`
	CompletedAt        *time.Time        `json:"completed_at,omitempty"`
	BlockReason        string            `json:"block_reason,omitempty"`
}

type Finding struct {
	ID                 string `json:"id"`
	Severity           string `json:"severity"`
	Location           string `json:"location"`
	Problem            string `json:"problem"`
	SuggestedDirection string `json:"suggested_direction"`
}

type ReviewResult struct {
	Status           string    `json:"status"`
	Lens             string    `json:"lens"`
	ReviewedRevision string    `json:"reviewed_revision"`
	Findings         []Finding `json:"findings"`
}

type PMDecision struct {
	FindingID string `json:"finding_id"`
	Decision  string `json:"decision"`
	Reason    string `json:"reason,omitempty"`
}

type PMDecisionDocument struct {
	ReviewedRevision string                  `json:"reviewed_revision"`
	Lenses           map[string][]PMDecision `json:"lenses"`
}

type pmRequest struct {
	Candidate        int    `json:"candidate"`
	Lens             string `json:"lens"`
	ReviewedRevision string `json:"reviewed_revision"`
	ResultPath       string `json:"result_path"`
	ReportPath       string `json:"report_path"`
	DecisionPath     string `json:"decision_path"`
}
