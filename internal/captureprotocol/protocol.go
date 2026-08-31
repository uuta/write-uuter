package captureprotocol

const (
	Version          = 2
	VersionArgument  = "--protocol-version=2"
	RequestFile      = "request.json"
	ResultFile       = "result.json"
	AssetsDirectory  = "assets"
	PNGMediaType     = "image/png"
	MaxActionSteps   = 32
	MaxActionStepLen = 256
	// RunnerDeadlineEnv carries the controller-owned absolute deadline for one
	// invocation. It is orchestration metadata, not a provider option and not
	// part of either durable JSON artifact.
	RunnerDeadlineEnv = "WRITE_UUTER_CAPTURE_RUNNER_DEADLINE_UNIX_MS"
)

type RequestDocument struct {
	SchemaVersion int       `json:"schema_version"`
	Requests      []Request `json:"requests"`
}

type Request struct {
	RequestID         string        `json:"request_id"`
	PublicURL         string        `json:"public_url"`
	Selector          string        `json:"selector,omitempty"`
	Reason            string        `json:"reason"`
	SupportedClaimIDs []string      `json:"supported_claim_ids"`
	PriorAttempt      *PriorAttempt `json:"prior_attempt,omitempty"`
}

// PriorAttempt is provider-neutral feedback for the single bounded capture
// retry. It lets runner policy avoid silently repeating editorially rejected
// evidence without telling the runner which backend to select.
type PriorAttempt struct {
	Attempt          int                `json:"attempt"`
	RequestID        string             `json:"request_id"`
	FinalURL         string             `json:"final_url"`
	CapturedAt       string             `json:"captured_at"`
	Backend          string             `json:"backend"`
	MediaType        string             `json:"media_type"`
	Viewport         Viewport           `json:"viewport"`
	FullPage         bool               `json:"full_page"`
	ByteSize         int64              `json:"byte_size"`
	Width            int                `json:"width"`
	Height           int                `json:"height"`
	SHA256           string             `json:"sha256"`
	EditorialOutcome EditorialRejection `json:"editorial_outcome"`
}

type EditorialRejection struct {
	RequestID string `json:"request_id"`
	Status    string `json:"status"`
	Reason    string `json:"reason"`
}

type ResultDocument struct {
	SchemaVersion int      `json:"schema_version"`
	Results       []Result `json:"results"`
}

type Result struct {
	RequestID         string   `json:"request_id"`
	RequestedURL      string   `json:"requested_url"`
	FinalURL          string   `json:"final_url"`
	CapturedAt        string   `json:"captured_at"`
	Backend           string   `json:"backend"`
	MediaType         string   `json:"media_type"`
	Viewport          Viewport `json:"viewport"`
	FullPage          bool     `json:"full_page"`
	ImagePath         string   `json:"image_path"`
	ByteSize          int64    `json:"byte_size"`
	Width             int      `json:"width"`
	Height            int      `json:"height"`
	SHA256            string   `json:"sha256"`
	SupportedClaimIDs []string `json:"supported_claim_ids"`
	Rationale         string   `json:"rationale"`
	ActionSummary     []string `json:"action_summary,omitempty"`
	TraceReference    string   `json:"trace_reference,omitempty"`
}

type Viewport struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}
