package app

import "time"

const workflowSchemaVersion = 1

type Workflow struct {
	SchemaVersion        int               `json:"schema_version"`
	Status               string            `json:"status"`
	Phase                string            `json:"phase"`
	CurrentCandidate     int               `json:"current_candidate"`
	CurrentProseRevision string            `json:"current_prose_revision"`
	CurrentRevision      string            `json:"current_revision"`
	ActiveRole           string            `json:"active_role"`
	ArtifactPaths        map[string]string `json:"artifact_paths"`
	ModelPolicyDigest    string            `json:"model_policy_digest"`
	ReviewAttemptCount   int               `json:"review_attempt_count"`
	StartedAt            time.Time         `json:"started_at"`
	UpdatedAt            time.Time         `json:"updated_at"`
	CompletedAt          *time.Time        `json:"completed_at,omitempty"`
	BlockReason          string            `json:"block_reason,omitempty"`
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
	ReviewedRevision string                      `json:"reviewed_revision"`
	Lenses           map[string]PMDecisionRecord `json:"lenses"`
}

type PMDecisionRecord struct {
	RequestID    string       `json:"request_id"`
	ReviewDigest string       `json:"review_digest"`
	Decisions    []PMDecision `json:"decisions"`
}

type pmRequest struct {
	RequestID        string `json:"request_id"`
	Candidate        int    `json:"candidate"`
	Lens             string `json:"lens"`
	ReviewedRevision string `json:"reviewed_revision"`
	ReviewDigest     string `json:"review_digest"`
	ResultPath       string `json:"result_path"`
	ReportPath       string `json:"report_path"`
	DecisionPath     string `json:"decision_path"`
	RequestPath      string `json:"request_path"`
	ContextDirectory string `json:"context_directory"`
	OutputPath       string `json:"output_path"`
}

// InvocationAudit is the immutable record published for every launched
// invocation before the process is considered ready. Its values come from the
// same validated profile that built the process arguments. It never records
// authentication, environment, prompt, or secret material.
type InvocationAudit struct {
	Invocation        string `json:"invocation"`
	Role              string `json:"role"`
	Lens              string `json:"lens,omitempty"`
	Candidate         int    `json:"candidate,omitempty"`
	Provider          string `json:"provider"`
	Model             string `json:"model"`
	ReasoningEffort   string `json:"reasoning_effort"`
	ModelPolicyDigest string `json:"model_policy_digest"`
}
