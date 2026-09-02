package app

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/uuta/write-uuter/internal/captureimage"
	"github.com/uuta/write-uuter/internal/captureprotocol"
)

// Screenshot acquisition contract.
//
// The Researcher may ask for public page screenshots; the controller - never
// an agent - performs the capture. Every documented bound below is part of the
// contract and is asserted by the black-box suite.
const (
	// screenshotRequestArtifact is the optional Researcher-owned request.
	screenshotRequestArtifact = "evidence/screenshot-requests.json"
	// screenshotManifestArtifact is the controller-generated manifest.
	screenshotManifestArtifact = "evidence/screenshots.json"
	// screenshotAssetDir is controller-owned. The Researcher may not write it.
	screenshotAssetDir = "evidence/assets/screenshots"

	screenshotManifestVersion = 3
	// screenshotMediaType is the only accepted response media type.
	screenshotMediaType = "image/png"

	// screenshotMaxRequests is the per-run request ceiling.
	screenshotMaxRequests = 5
	// screenshotMaxBytes is the accepted per-image byte ceiling (10 MiB).
	screenshotMaxBytes = captureimage.MaxBytes
	// screenshotViewportWidth/Height is the fixed documented viewport. It is
	// not configurable: a stable viewport keeps captures comparable.
	screenshotViewportWidth  = 1280
	screenshotViewportHeight = 800
	// screenshotMinDimension/screenshotMaxDimension bound accepted image
	// dimensions, so a 0-pixel or absurd image is rejected as invalid.
	screenshotMinDimension = captureimage.MinDimension
	screenshotMaxDimension = captureimage.MaxDimension
	// screenshotMaxPixels bounds the decoded allocation. Per-axis limits alone
	// do not: 20000x20000 is inside them and still allocates gigabytes.
	screenshotMaxPixels = captureimage.MaxPixels
	// screenshotRequestTimeout bounds one capture request end to end. The only
	// retry is a new invocation after a request-keyed editorial rejection, and
	// each logical request is capped at two evaluated attempts.
	screenshotRequestTimeout = 60 * time.Second
)

var pngSignature = []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}

// screenshotRequestDocument is the strict shape of the optional Researcher
// artifact. Unknown fields and duplicate JSON keys are rejected recursively.
type screenshotRequestDocument struct {
	Screenshots []screenshotRequestEntry `json:"screenshots"`
}

// screenshotRequestEntry decodes one raw entry. Selector is a pointer so that
// an explicitly empty selector is rejected rather than silently treated as an
// absent one.
type screenshotRequestEntry struct {
	ID       string   `json:"id"`
	URL      string   `json:"url"`
	Reason   string   `json:"reason"`
	Supports []string `json:"supports"`
	Selector *string  `json:"selector,omitempty"`
}

// ScreenshotRequest is one validated request. Every field has already passed
// the contract by the time a value of this type exists.
type ScreenshotRequest struct {
	ID           string
	URL          string
	Reason       string
	Supports     []string
	Selector     string
	PriorAttempt *captureprotocol.PriorAttempt
}

// ScreenshotManifest is controller-generated. No agent writes it.
type ScreenshotManifest struct {
	SchemaVersion int                `json:"schema_version"`
	Screenshots   []ScreenshotRecord `json:"screenshots"`
}

type ScreenshotViewport struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

type ScreenshotRecord struct {
	ID               string                      `json:"id"`
	Path             string                      `json:"path"`
	RequestedURL     string                      `json:"requested_url"`
	FinalURL         string                      `json:"final_url"`
	Selector         string                      `json:"selector,omitempty"`
	CapturedAt       time.Time                   `json:"captured_at"`
	Supports         []string                    `json:"supports"`
	Reason           string                      `json:"reason"`
	Backend          string                      `json:"backend"`
	MediaType        string                      `json:"media_type"`
	Viewport         ScreenshotViewport          `json:"viewport"`
	FullPage         bool                        `json:"full_page"`
	ByteSize         int                         `json:"byte_size"`
	Width            int                         `json:"width"`
	Height           int                         `json:"height"`
	SHA256           string                      `json:"sha256"`
	ActionSummary    []string                    `json:"action_summary,omitempty"`
	TraceReference   string                      `json:"trace_reference,omitempty"`
	Attempt          int                         `json:"attempt"`
	EditorialOutcome *ScreenshotEditorialOutcome `json:"editorial_outcome,omitempty"`
	PriorAttempts    []ScreenshotAttemptRecord   `json:"prior_attempts,omitempty"`
}

type ScreenshotEditorialOutcome struct {
	RequestID string `json:"request_id"`
	Status    string `json:"status"`
	Reason    string `json:"reason"`
}

type ScreenshotAttemptRecord struct {
	Attempt          int                         `json:"attempt"`
	Path             string                      `json:"path"`
	FinalURL         string                      `json:"final_url"`
	CapturedAt       time.Time                   `json:"captured_at"`
	Backend          string                      `json:"backend"`
	MediaType        string                      `json:"media_type"`
	Viewport         ScreenshotViewport          `json:"viewport"`
	FullPage         bool                        `json:"full_page"`
	ByteSize         int                         `json:"byte_size"`
	Width            int                         `json:"width"`
	Height           int                         `json:"height"`
	SHA256           string                      `json:"sha256"`
	ActionSummary    []string                    `json:"action_summary,omitempty"`
	TraceReference   string                      `json:"trace_reference,omitempty"`
	EditorialOutcome *ScreenshotEditorialOutcome `json:"editorial_outcome"`
}

// screenshotAssetPath is the durable relative path of one captured image.
func screenshotAssetPath(id string) string {
	return screenshotAssetDir + "/" + id + ".png"
}

func screenshotAttemptAssetPath(id string, attempt int) string {
	if attempt <= 1 {
		return screenshotAssetPath(id)
	}
	// Initial request assets always occupy a single file directly below
	// screenshotAssetDir. Later attempts live in a controller-owned namespace
	// below that directory, so no valid single-component request ID can derive
	// the same path as another request's initial asset.
	return fmt.Sprintf("%s/attempts/%s/attempt-%03d.png", screenshotAssetDir, id, attempt)
}

// parseScreenshotRequests validates the Researcher artifact completely. Every
// rejection here happens before the Story Editor and the Writer run.
func parseScreenshotRequests(data, claimLedger []byte) ([]ScreenshotRequest, error) {
	if strings.TrimSpace(string(data)) == "" {
		return nil, fmt.Errorf("%s is empty", screenshotRequestArtifact)
	}
	// Require the control field to be present and a real array before the
	// typed decode, so an absent or null list cannot decode to "no requests".
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return nil, fmt.Errorf("invalid %s: %w", screenshotRequestArtifact, err)
	}
	list, found := fields["screenshots"]
	if !found {
		return nil, fmt.Errorf("%s is missing required field %q", screenshotRequestArtifact, "screenshots")
	}
	if trimmed := bytes.TrimSpace(list); len(trimmed) == 0 || trimmed[0] != '[' {
		return nil, fmt.Errorf("%s field %q must be an array", screenshotRequestArtifact, "screenshots")
	}
	var document screenshotRequestDocument
	if err := decodeStrictJSON(data, &document); err != nil {
		return nil, fmt.Errorf("invalid %s: %w", screenshotRequestArtifact, err)
	}
	if len(document.Screenshots) > screenshotMaxRequests {
		return nil, fmt.Errorf("%s requests %d screenshots, the limit is %d",
			screenshotRequestArtifact, len(document.Screenshots), screenshotMaxRequests)
	}
	seen := make(map[string]bool, len(document.Screenshots))
	requests := make([]ScreenshotRequest, 0, len(document.Screenshots))
	for index, entry := range document.Screenshots {
		if err := validateScreenshotID(entry.ID); err != nil {
			return nil, fmt.Errorf("%s entry %d: %w", screenshotRequestArtifact, index, err)
		}
		// IDs become file names, and the run may live on a case-insensitive
		// file system, where "shot-001" and "SHOT-001" are one path. Comparing
		// case-insensitively stops the second capture silently overwriting the
		// first while the manifest still claims two distinct screenshots.
		folded := strings.ToLower(entry.ID)
		if seen[folded] {
			return nil, fmt.Errorf("%s duplicates screenshot ID %q; IDs become file names and are compared case-insensitively",
				screenshotRequestArtifact, entry.ID)
		}
		seen[folded] = true
		normalized, err := validatePublicPageURL(entry.URL)
		if err != nil {
			return nil, fmt.Errorf("%s entry %q: %w", screenshotRequestArtifact, entry.ID, err)
		}
		if strings.TrimSpace(entry.Reason) == "" {
			return nil, fmt.Errorf("%s entry %q has an empty reason", screenshotRequestArtifact, entry.ID)
		}
		if len(entry.Supports) == 0 {
			return nil, fmt.Errorf("%s entry %q supports no claim", screenshotRequestArtifact, entry.ID)
		}
		supported := make(map[string]bool, len(entry.Supports))
		for _, claim := range entry.Supports {
			if err := validateClaimID(claim); err != nil {
				return nil, fmt.Errorf("%s entry %q: %w", screenshotRequestArtifact, entry.ID, err)
			}
			if supported[claim] {
				return nil, fmt.Errorf("%s entry %q repeats claim %q", screenshotRequestArtifact, entry.ID, claim)
			}
			supported[claim] = true
			if !claimIDPresent(claimLedger, claim) {
				return nil, fmt.Errorf("%s entry %q references claim %q, which is not in claim-ledger.md",
					screenshotRequestArtifact, entry.ID, claim)
			}
		}
		selector := ""
		if entry.Selector != nil {
			if err := validateScreenshotSelector(*entry.Selector); err != nil {
				return nil, fmt.Errorf("%s entry %q: %w", screenshotRequestArtifact, entry.ID, err)
			}
			selector = *entry.Selector
		}
		requests = append(requests, ScreenshotRequest{
			ID: entry.ID, URL: normalized, Reason: entry.Reason,
			Supports: entry.Supports, Selector: selector,
		})
	}
	return requests, nil
}

// validateScreenshotID keeps every ID safe as a single path component and
// stable as a manifest key.
func validateScreenshotID(id string) error {
	if id == "" {
		return fmt.Errorf("screenshot ID is empty")
	}
	if len(id) > 64 {
		return fmt.Errorf("screenshot ID %q is longer than 64 characters", id)
	}
	for index, character := range id {
		switch {
		case character >= 'a' && character <= 'z',
			character >= 'A' && character <= 'Z',
			character >= '0' && character <= '9':
		case (character == '-' || character == '_') && index > 0:
		default:
			return fmt.Errorf("screenshot ID %q is not filename-safe", id)
		}
	}
	return nil
}

func validateClaimID(claim string) error {
	if claim == "" {
		return fmt.Errorf("claim ID is empty")
	}
	if len(claim) > 64 {
		return fmt.Errorf("claim ID %q is longer than 64 characters", claim)
	}
	for index, character := range claim {
		switch {
		case character >= 'a' && character <= 'z',
			character >= 'A' && character <= 'Z',
			character >= '0' && character <= '9':
		case (character == '-' || character == '_' || character == '.') && index > 0:
		default:
			return fmt.Errorf("claim ID %q is not a plain identifier", claim)
		}
	}
	return nil
}

// claimIDPresent reports whether the ledger names the claim as a whole token,
// so "claim-004" never matches "claim-0041".
func claimIDPresent(ledger []byte, claim string) bool {
	text := string(ledger)
	for offset := 0; ; {
		index := strings.Index(text[offset:], claim)
		if index < 0 {
			return false
		}
		start := offset + index
		end := start + len(claim)
		if !claimTokenRune(text, start-1) && !claimTokenRune(text, end) {
			return true
		}
		offset = start + 1
	}
}

func claimTokenRune(text string, index int) bool {
	if index < 0 || index >= len(text) {
		return false
	}
	character := text[index]
	switch {
	case character >= 'a' && character <= 'z',
		character >= 'A' && character <= 'Z',
		character >= '0' && character <= '9',
		character == '-', character == '_', character == '.':
		return true
	}
	return false
}

// validateScreenshotSelector accepts a bounded, printable CSS selector. It is
// the only page-targeting option in this slice: no clicks, waits, scripts,
// cookies, or navigation steps are accepted.
func validateScreenshotSelector(selector string) error {
	if strings.TrimSpace(selector) != selector || selector == "" {
		return fmt.Errorf("selector must be non-empty and free of surrounding whitespace")
	}
	if len(selector) > 256 {
		return fmt.Errorf("selector is longer than 256 characters")
	}
	for _, character := range selector {
		if character < 0x20 || character == 0x7f {
			return fmt.Errorf("selector contains a control character")
		}
	}
	return nil
}

// validatePublicPageURL accepts only a public HTTPS page on the default port
// with a DNS hostname. Credentials in the URL, non-HTTPS schemes, IP literals,
// and local/private name suffixes are rejected.
func validatePublicPageURL(raw string) (string, error) {
	if raw == "" {
		return "", fmt.Errorf("URL is empty")
	}
	if len(raw) > 2048 {
		return "", fmt.Errorf("URL is longer than 2048 characters")
	}
	for _, character := range raw {
		if character < 0x20 || character == 0x7f || character == ' ' {
			return "", fmt.Errorf("URL contains a control or space character")
		}
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("URL is not parseable")
	}
	if parsed.Scheme != "https" {
		return "", fmt.Errorf("URL scheme %q is not https", parsed.Scheme)
	}
	if parsed.Opaque != "" {
		return "", fmt.Errorf("URL is not in origin form")
	}
	if parsed.User != nil {
		return "", fmt.Errorf("URL embeds credentials")
	}
	host := parsed.Hostname()
	if host == "" {
		return "", fmt.Errorf("URL has no host")
	}
	if port := parsed.Port(); port != "" && port != "443" {
		return "", fmt.Errorf("URL port %q is not the default HTTPS port", port)
	}
	if strings.HasPrefix(parsed.Host, "[") || net.ParseIP(host) != nil {
		return "", fmt.Errorf("URL host is an IP literal")
	}
	if err := validatePublicHostname(host); err != nil {
		return "", err
	}
	// Re-render from the parsed value with the default port normalized away,
	// so the recorded URL is exactly what the controller requests.
	parsed.Host = strings.ToLower(host)
	return parsed.String(), nil
}

// screenshotBlockedHostSuffixes are name suffixes that never denote a public
// page. They are rejected regardless of what they currently resolve to.
var screenshotBlockedHostSuffixes = []string{"localhost", "local", "internal", "intranet", "lan", "home", "corp", "invalid", "onion"}

func validatePublicHostname(host string) error {
	host = strings.ToLower(strings.TrimSuffix(host, "."))
	if host == "" || len(host) > 253 {
		return fmt.Errorf("URL host is not a valid DNS name")
	}
	labels := strings.Split(host, ".")
	if len(labels) < 2 {
		return fmt.Errorf("URL host %q is not a public DNS name", host)
	}
	for _, label := range labels {
		if label == "" || len(label) > 63 {
			return fmt.Errorf("URL host is not a valid DNS name")
		}
		if label[0] == '-' || label[len(label)-1] == '-' {
			return fmt.Errorf("URL host is not a valid DNS name")
		}
		for _, character := range label {
			switch {
			case character >= 'a' && character <= 'z',
				character >= '0' && character <= '9',
				character == '-':
			default:
				return fmt.Errorf("URL host is not a valid DNS name")
			}
		}
	}
	last := labels[len(labels)-1]
	for _, character := range last {
		if character < 'a' || character > 'z' {
			return fmt.Errorf("URL host %q has a non-alphabetic top-level label", host)
		}
	}
	for _, blocked := range screenshotBlockedHostSuffixes {
		if last == blocked {
			return fmt.Errorf("URL host %q is not a public page", host)
		}
	}
	return nil
}

// validatePNG proves the bytes are a complete PNG with usable dimensions. A
// transport-level success is not enough: a truncated or non-image body is
// rejected here rather than persisted as evidence.
func validatePNG(data []byte) (int, int, error) {
	return captureimage.ValidatePNG(data)
}

// captureScreenshots runs between the Researcher and every later role. It is a
// no-op - and requires no capture runner - when the Researcher asked for
// nothing, so existing briefs are unaffected. Any failure here returns an
// error, which blocks the run before the Story Editor and the Writer start.
func (control *controller) captureScreenshots() error {
	if len(control.screenshotRequests) == 0 {
		return nil
	}
	control.workflow.Phase = "capturing screenshots"
	if err := control.saveWorkflow(); err != nil {
		return err
	}
	rollback := func() {
		_ = control.store.removeAll(screenshotAssetDir)
		_ = control.store.remove(screenshotManifestArtifact)
		control.screenshotRecords = make(map[string]*ScreenshotRecord)
		control.screenshotManifest = nil
	}
	captures, err := control.runCaptureRunner(control.screenshotRequests)
	if err != nil {
		rollback()
		return err
	}
	for index, captured := range captures {
		request := control.screenshotRequests[index]
		relative := screenshotAttemptAssetPath(request.ID, 1)
		if err := control.store.writeAtomic(relative, captured.data, 0o444); err != nil {
			rollback()
			return fmt.Errorf("persist screenshot %q: %w", request.ID, err)
		}
		record := &ScreenshotRecord{
			ID: request.ID, Path: relative, RequestedURL: request.URL, FinalURL: captured.protocol.FinalURL, Selector: request.Selector,
			CapturedAt: captured.time, Supports: request.Supports, Reason: request.Reason,
			Backend: captured.protocol.Backend, MediaType: captured.protocol.MediaType,
			Viewport: ScreenshotViewport{Width: captured.protocol.Viewport.Width, Height: captured.protocol.Viewport.Height},
			FullPage: captured.protocol.FullPage, ByteSize: len(captured.data),
			Width: captured.width, Height: captured.height, SHA256: revisionFor(captured.data),
			ActionSummary: captured.protocol.ActionSummary, TraceReference: captured.protocol.TraceReference,
			Attempt: 1,
		}
		control.screenshotRecords[request.ID] = record
	}
	if err := control.persistScreenshotManifest(); err != nil {
		rollback()
		return err
	}
	return nil
}

func (control *controller) persistScreenshotManifest() error {
	manifest := ScreenshotManifest{SchemaVersion: screenshotManifestVersion, Screenshots: make([]ScreenshotRecord, 0, len(control.screenshotRequests))}
	for _, request := range control.screenshotRequests {
		record := control.screenshotRecords[request.ID]
		if record == nil {
			return fmt.Errorf("screenshot record %q is missing", request.ID)
		}
		manifest.Screenshots = append(manifest.Screenshots, *record)
	}
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("encode %s: %w", screenshotManifestArtifact, err)
	}
	data = append(data, '\n')
	if err := control.store.writeAtomic(screenshotManifestArtifact, data, 0o444); err != nil {
		return fmt.Errorf("write %s: %w", screenshotManifestArtifact, err)
	}
	control.screenshotManifest = append([]byte(nil), data...)
	return nil
}

func screenshotAttemptFromRecord(record *ScreenshotRecord) ScreenshotAttemptRecord {
	return ScreenshotAttemptRecord{
		Attempt: record.Attempt, Path: record.Path, FinalURL: record.FinalURL, CapturedAt: record.CapturedAt,
		Backend: record.Backend, MediaType: record.MediaType, Viewport: record.Viewport, FullPage: record.FullPage,
		ByteSize: record.ByteSize, Width: record.Width, Height: record.Height, SHA256: record.SHA256,
		ActionSummary: append([]string(nil), record.ActionSummary...), TraceReference: record.TraceReference,
		EditorialOutcome: record.EditorialOutcome,
	}
}

func screenshotEditorialRejectionExhausted(record *ScreenshotRecord) bool {
	return record != nil && record.Attempt >= 2 && record.EditorialOutcome != nil && record.EditorialOutcome.Status == "rejected"
}

func (control *controller) recordScreenshotEditorialOutcomes(outcomes []ScreenshotEditorialOutcome) ([]ScreenshotRequest, error) {
	if len(control.screenshotRequests) == 0 {
		return nil, nil
	}
	var retry []ScreenshotRequest
	for _, outcome := range outcomes {
		record := control.screenshotRecords[outcome.RequestID]
		if record == nil {
			return nil, fmt.Errorf("editorial outcome names unknown screenshot request %q", outcome.RequestID)
		}
		if screenshotEditorialRejectionExhausted(record) {
			if outcome.Status != "rejected" {
				return nil, fmt.Errorf("screenshot request %q already exhausted two editorially rejected attempts; its durable non-placement is terminal", outcome.RequestID)
			}
			// Preserve the exact terminal outcome and its reason. A later
			// candidate cannot rewrite already-evaluated attempt provenance.
			continue
		}
		copyOfOutcome := outcome
		record.EditorialOutcome = &copyOfOutcome
		if outcome.Status != "rejected" || record.Attempt >= 2 {
			continue
		}
		var original ScreenshotRequest
		found := false
		for _, request := range control.screenshotRequests {
			if request.ID == outcome.RequestID {
				original = request
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("editorial outcome lost screenshot request %q", outcome.RequestID)
		}
		original.PriorAttempt = &captureprotocol.PriorAttempt{
			Attempt: record.Attempt, RequestID: record.ID, FinalURL: record.FinalURL,
			CapturedAt: record.CapturedAt.UTC().Format(time.RFC3339Nano), Backend: record.Backend,
			MediaType: record.MediaType,
			Viewport:  captureprotocol.Viewport{Width: record.Viewport.Width, Height: record.Viewport.Height},
			FullPage:  record.FullPage, ByteSize: int64(record.ByteSize), Width: record.Width, Height: record.Height,
			SHA256: record.SHA256,
			EditorialOutcome: captureprotocol.EditorialRejection{
				RequestID: outcome.RequestID, Status: outcome.Status, Reason: outcome.Reason,
			},
		}
		retry = append(retry, original)
	}
	if err := control.persistScreenshotManifest(); err != nil {
		return nil, err
	}
	// A terminally rejected second attempt remains durable evidence, but its
	// pixels are no longer an adoptable input for any later candidate.
	if err := control.publishVisualInputs(); err != nil {
		return nil, err
	}
	return retry, nil
}

func (control *controller) retryRejectedScreenshots(requests []ScreenshotRequest) error {
	captures, err := control.runCaptureRunner(requests)
	if err != nil {
		return err
	}
	type writtenAttempt struct {
		path  string
		owned os.FileInfo
	}
	written := make([]writtenAttempt, 0, len(captures))
	rollback := func() {
		for _, item := range written {
			_ = control.store.removeOwned(item.path, item.owned)
		}
	}
	for index, captured := range captures {
		request := requests[index]
		record := control.screenshotRecords[request.ID]
		if record == nil || record.EditorialOutcome == nil || record.EditorialOutcome.Status != "rejected" || record.Attempt != 1 {
			rollback()
			return fmt.Errorf("screenshot retry %q has no first-attempt editorial rejection", request.ID)
		}
		relative := screenshotAttemptAssetPath(request.ID, 2)
		owned, err := control.store.writeAtomicNoReplace(relative, captured.data, 0o444)
		if owned != nil {
			written = append(written, writtenAttempt{path: relative, owned: owned})
		}
		if err != nil {
			rollback()
			return fmt.Errorf("persist screenshot retry %q: %w", request.ID, err)
		}
		record.PriorAttempts = append(record.PriorAttempts, screenshotAttemptFromRecord(record))
		record.Path = relative
		record.FinalURL = captured.protocol.FinalURL
		record.CapturedAt = captured.time
		record.Backend = captured.protocol.Backend
		record.MediaType = captured.protocol.MediaType
		record.Viewport = ScreenshotViewport{Width: captured.protocol.Viewport.Width, Height: captured.protocol.Viewport.Height}
		record.FullPage = captured.protocol.FullPage
		record.ByteSize = len(captured.data)
		record.Width = captured.width
		record.Height = captured.height
		record.SHA256 = revisionFor(captured.data)
		record.ActionSummary = append([]string(nil), captured.protocol.ActionSummary...)
		record.TraceReference = captured.protocol.TraceReference
		record.Attempt = 2
		record.EditorialOutcome = nil
	}
	if err := control.persistScreenshotManifest(); err != nil {
		rollback()
		return err
	}
	if err := control.publishVisualInputs(); err != nil {
		return err
	}
	return nil
}

// screenshotContext returns the validated read-only screenshot context for a
// later role, or nil when this run captured nothing. The manifest is text and
// joins the prompt; the images are staged as files and never enter a prompt.
func (control *controller) screenshotContext() (manifest []byte, images map[string][]byte, err error) {
	if len(control.screenshotManifest) == 0 {
		return nil, nil, nil
	}
	images = make(map[string][]byte, len(control.screenshotRequests))
	for _, request := range control.screenshotRequests {
		record := control.screenshotRecords[request.ID]
		if record == nil {
			return nil, nil, fmt.Errorf("screenshot record %q is missing", request.ID)
		}
		relative := record.Path
		data, readErr := control.store.readRegular(relative)
		if readErr != nil {
			return nil, nil, fmt.Errorf("read staged screenshot %s: %w", relative, readErr)
		}
		images[relative] = data
	}
	return control.screenshotManifest, images, nil
}

// stageScreenshotImages copies the validated images into a role workspace as
// read-only files. Their bytes are unchanged, so a later visual pass can use
// them in place.
func stageScreenshotImages(workspace *artifactStore, images map[string][]byte) error {
	for relative, data := range images {
		if err := workspace.writeAtomic(filepath.Join("context", relative), data, 0o444); err != nil {
			return err
		}
	}
	return nil
}
