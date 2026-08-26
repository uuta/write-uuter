package app

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"image/png"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
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

	screenshotManifestVersion = 1
	// screenshotEngine names the only capture engine in this slice.
	screenshotEngine = "cloudflare-chromium"
	// screenshotMediaType is the only accepted response media type.
	screenshotMediaType = "image/png"

	// screenshotMaxRequests is the per-run request ceiling.
	screenshotMaxRequests = 5
	// screenshotMaxBytes is the accepted per-image byte ceiling (10 MiB).
	screenshotMaxBytes = 10 << 20
	// screenshotViewportWidth/Height is the fixed documented viewport. It is
	// not configurable: a stable viewport keeps captures comparable.
	screenshotViewportWidth  = 1280
	screenshotViewportHeight = 800
	// screenshotMinDimension/screenshotMaxDimension bound accepted image
	// dimensions, so a 0-pixel or absurd image is rejected as invalid.
	screenshotMinDimension = 1
	screenshotMaxDimension = 20000
	// screenshotMaxPixels bounds the decoded allocation. Per-axis limits alone
	// do not: 20000x20000 is inside them and still allocates gigabytes.
	screenshotMaxPixels = 40_000_000
	// screenshotRequestTimeout bounds one capture request end to end. There is
	// no automatic retry: a retry could hide duplicate billing or an unstable
	// page, so a timeout blocks the run instead.
	screenshotRequestTimeout = 60 * time.Second

	// screenshotAPIBaseURL is the Cloudflare API base. The Chromium quick
	// action is POST /accounts/{account_id}/browser-rendering/screenshot.
	screenshotAPIBaseURL = "https://api.cloudflare.com/client/v4"

	screenshotAccountEnv = "CLOUDFLARE_ACCOUNT_ID"
	screenshotTokenEnv   = "CLOUDFLARE_API_TOKEN"

	// screenshotBaseURLEnv redirects the API base for local testing. It is
	// accepted only for a loopback origin: the request it retargets carries the
	// bearer token, so an ambient value pointing anywhere else would be a
	// credential exfiltration channel rather than a test seam.
	screenshotBaseURLEnv = "WRITE_UUTER_TEST_BROWSER_RENDERING_BASE_URL"
)

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
	ID       string
	URL      string
	Reason   string
	Supports []string
	Selector string
}

// ScreenshotManifest is controller-generated. No agent writes it.
type ScreenshotManifest struct {
	SchemaVersion int                `json:"schema_version"`
	Engine        string             `json:"engine"`
	Viewport      ScreenshotViewport `json:"viewport"`
	Screenshots   []ScreenshotRecord `json:"screenshots"`
}

type ScreenshotViewport struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

type ScreenshotRecord struct {
	ID           string    `json:"id"`
	Path         string    `json:"path"`
	RequestedURL string    `json:"requested_url"`
	Selector     string    `json:"selector,omitempty"`
	CapturedAt   time.Time `json:"captured_at"`
	Supports     []string  `json:"supports"`
	Reason       string    `json:"reason"`
	Engine       string    `json:"engine"`
	MediaType    string    `json:"media_type"`
	ByteSize     int       `json:"byte_size"`
	Width        int       `json:"width"`
	Height       int       `json:"height"`
	SHA256       string    `json:"sha256"`
}

// screenshotAssetPath is the durable relative path of one captured image.
func screenshotAssetPath(id string) string {
	return screenshotAssetDir + "/" + id + ".png"
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

// screenshotCredentials never leaves the controller process. It is not staged
// into a workspace, an environment, a process argument, or a prompt.
type screenshotCredentials struct {
	accountID string
	apiToken  string
}

func loadScreenshotCredentials() (screenshotCredentials, error) {
	var credentials screenshotCredentials
	account := strings.TrimSpace(os.Getenv(screenshotAccountEnv))
	token := strings.TrimSpace(os.Getenv(screenshotTokenEnv))
	var missing []string
	if account == "" {
		missing = append(missing, screenshotAccountEnv)
	}
	if token == "" {
		missing = append(missing, screenshotTokenEnv)
	}
	if len(missing) != 0 {
		return credentials, fmt.Errorf("screenshot capture requires %s; set it in the controller environment",
			strings.Join(missing, " and "))
	}
	if err := validateCredentialToken(screenshotAccountEnv, account, 128); err != nil {
		return credentials, err
	}
	if strings.ContainsAny(account, "/?#") {
		return credentials, fmt.Errorf("%s contains a URL path character", screenshotAccountEnv)
	}
	if err := validateCredentialToken(screenshotTokenEnv, token, 4096); err != nil {
		return credentials, err
	}
	return screenshotCredentials{accountID: account, apiToken: token}, nil
}

// validateCredentialToken reports only the variable name, never its value.
func validateCredentialToken(name, value string, limit int) error {
	if len(value) > limit {
		return fmt.Errorf("%s is longer than %d characters", name, limit)
	}
	for _, character := range value {
		if character < 0x21 || character > 0x7e {
			return fmt.Errorf("%s contains a character that is not printable ASCII", name)
		}
	}
	return nil
}

// screenshotClient owns the authenticated Cloudflare call. No agent process
// ever constructs it, receives it, or sees any value inside it.
type screenshotClient struct {
	credentials screenshotCredentials
	baseURL     string
	timeout     time.Duration
	http        *http.Client
}

// resolveScreenshotBaseURL returns the API base for this run. It is validated
// before any credential is read, so a hostile ambient override fails the run
// instead of ever being combined with the bearer token.
func resolveScreenshotBaseURL() (string, error) {
	override := strings.TrimSpace(os.Getenv(screenshotBaseURLEnv))
	if override == "" {
		return screenshotAPIBaseURL, nil
	}
	parsed, err := url.Parse(override)
	if err != nil {
		return "", fmt.Errorf("%s is not a parseable URL", screenshotBaseURLEnv)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", fmt.Errorf("%s scheme %q is not http or https", screenshotBaseURLEnv, parsed.Scheme)
	}
	if parsed.Opaque != "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", fmt.Errorf("%s must be a bare origin with an optional path", screenshotBaseURLEnv)
	}
	host := parsed.Hostname()
	address := net.ParseIP(host)
	if host != "localhost" && (address == nil || !address.IsLoopback()) {
		return "", fmt.Errorf("%s must address a loopback host, but names %q; the redirected request carries the API token",
			screenshotBaseURLEnv, host)
	}
	return strings.TrimSuffix(override, "/"), nil
}

func newScreenshotClient(credentials screenshotCredentials, baseURL string) *screenshotClient {
	timeout := screenshotRequestTimeout
	if injected, err := time.ParseDuration(os.Getenv("WRITE_UUTER_TEST_SCREENSHOT_TIMEOUT")); err == nil && injected > 0 {
		timeout = injected
	}
	return &screenshotClient{
		credentials: credentials,
		baseURL:     baseURL,
		timeout:     timeout,
		http: &http.Client{
			// Keep-alives are disabled so the transport never replays a POST
			// onto a reused connection: one request means one capture.
			Transport: &http.Transport{DisableKeepAlives: true, Proxy: http.ProxyFromEnvironment},
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return errors.New("browser rendering endpoint redirected")
			},
		},
	}
}

// scrub removes credential material from any text that is about to become an
// error, a log line, or a durable artifact. Transport errors embed the request
// URL, which contains the account ID, so this is not optional.
func (client *screenshotClient) scrub(text string) string {
	for _, secret := range []string{client.credentials.apiToken, client.credentials.accountID} {
		if secret != "" {
			text = strings.ReplaceAll(text, secret, "[redacted]")
		}
	}
	return text
}

func (client *screenshotClient) scrubbed(format string, arguments ...any) error {
	return errors.New(client.scrub(fmt.Sprintf(format, arguments...)))
}

type screenshotAPIRequest struct {
	URL               string                  `json:"url"`
	Selector          string                  `json:"selector,omitempty"`
	Viewport          ScreenshotViewport      `json:"viewport"`
	ScreenshotOptions screenshotAPIImageParam `json:"screenshotOptions"`
}

type screenshotAPIImageParam struct {
	Type     string `json:"type"`
	FullPage bool   `json:"fullPage"`
}

type capturedImage struct {
	data   []byte
	width  int
	height int
}

// capture performs exactly one Cloudflare Chromium screenshot request. It never
// retries, never falls back to another engine, and never sends cookies,
// credentials, extra headers, scripts, or navigation steps for the page.
func (client *screenshotClient) capture(parent context.Context, request ScreenshotRequest) (capturedImage, error) {
	var captured capturedImage
	payload, err := json.Marshal(screenshotAPIRequest{
		URL:               request.URL,
		Selector:          request.Selector,
		Viewport:          ScreenshotViewport{Width: screenshotViewportWidth, Height: screenshotViewportHeight},
		ScreenshotOptions: screenshotAPIImageParam{Type: "png", FullPage: false},
	})
	if err != nil {
		return captured, fmt.Errorf("encode screenshot request %q: %w", request.ID, err)
	}
	ctx, cancel := context.WithTimeout(parent, client.timeout)
	defer cancel()
	endpoint := client.baseURL + "/accounts/" + client.credentials.accountID + "/browser-rendering/screenshot"
	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return captured, client.scrubbed("build screenshot request %q: %v", request.ID, err)
	}
	httpRequest.Header.Set("Authorization", "Bearer "+client.credentials.apiToken)
	httpRequest.Header.Set("Content-Type", "application/json")
	httpRequest.Header.Set("Accept", screenshotMediaType)
	response, err := client.http.Do(httpRequest)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return captured, client.scrubbed("screenshot %q timed out after %s capturing %s",
				request.ID, client.timeout, request.URL)
		}
		return captured, client.scrubbed("screenshot %q request failed for %s: %v", request.ID, request.URL, err)
	}
	defer func() {
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 4096))
		_ = response.Body.Close()
	}()
	body, err := io.ReadAll(io.LimitReader(response.Body, screenshotMaxBytes+1))
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return captured, client.scrubbed("screenshot %q timed out after %s reading %s",
				request.ID, client.timeout, request.URL)
		}
		return captured, client.scrubbed("screenshot %q response read failed for %s: %v", request.ID, request.URL, err)
	}
	if response.StatusCode < 200 || response.StatusCode > 299 {
		// Only the status code and the documented numeric error codes are
		// reported. Upstream free text is never copied into a durable
		// artifact: scrubbing is a denylist and cannot prove that an arbitrary
		// response body is credential-free. No response header is ever read.
		return captured, client.scrubbed("screenshot %q for %s returned HTTP %d (%s)",
			request.ID, request.URL, response.StatusCode, apiErrorCodes(body))
	}
	if len(body) > screenshotMaxBytes {
		return captured, client.scrubbed("screenshot %q for %s exceeds the %d byte limit",
			request.ID, request.URL, screenshotMaxBytes)
	}
	if mediaType := strings.TrimSpace(strings.SplitN(response.Header.Get("Content-Type"), ";", 2)[0]); mediaType != screenshotMediaType {
		return captured, client.scrubbed("screenshot %q for %s returned media type %q, want %q",
			request.ID, request.URL, mediaType, screenshotMediaType)
	}
	width, height, err := validatePNG(body)
	if err != nil {
		return captured, client.scrubbed("screenshot %q for %s: %v", request.ID, request.URL, err)
	}
	return capturedImage{data: body, width: width, height: height}, nil
}

// screenshotAPIErrorEnvelope is the documented Cloudflare error shape. Only
// the numeric codes are ever extracted from it.
type screenshotAPIErrorEnvelope struct {
	Errors []struct {
		Code int `json:"code"`
	} `json:"errors"`
}

// apiErrorCodes renders the documented numeric error codes of a failed
// response and nothing else. Every value it can return is generated here, so
// the diagnostic is credential-free by construction rather than by filtering.
func apiErrorCodes(body []byte) string {
	var envelope screenshotAPIErrorEnvelope
	if err := json.Unmarshal(body, &envelope); err != nil || len(envelope.Errors) == 0 {
		return "no documented error code in the response body"
	}
	codes := make([]string, 0, len(envelope.Errors))
	for _, item := range envelope.Errors {
		codes = append(codes, strconv.Itoa(item.Code))
	}
	return "Cloudflare error codes " + strings.Join(codes, ", ")
}

var pngSignature = []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}

// validatePNG proves the bytes are a complete PNG with usable dimensions. A
// transport-level success is not enough: a truncated or non-image body is
// rejected here rather than persisted as evidence.
func validatePNG(data []byte) (int, int, error) {
	if len(data) == 0 {
		return 0, 0, fmt.Errorf("image is empty")
	}
	if len(data) < len(pngSignature)+25 || !bytes.Equal(data[:len(pngSignature)], pngSignature) {
		return 0, 0, fmt.Errorf("image is not a PNG")
	}
	header := data[len(pngSignature):]
	if binary.BigEndian.Uint32(header[0:4]) != 13 || !bytes.Equal(header[4:8], []byte("IHDR")) {
		return 0, 0, fmt.Errorf("image has no PNG header chunk")
	}
	width := int(binary.BigEndian.Uint32(header[8:12]))
	height := int(binary.BigEndian.Uint32(header[12:16]))
	// Bound the declared dimensions BEFORE decoding. png.Decode allocates from
	// the header, so a small compressed body may declare an enormous canvas;
	// checking afterwards would mean the allocation has already happened.
	if width < screenshotMinDimension || height < screenshotMinDimension ||
		width > screenshotMaxDimension || height > screenshotMaxDimension {
		return 0, 0, fmt.Errorf("image dimensions %dx%d are outside the accepted %d-%d range",
			width, height, screenshotMinDimension, screenshotMaxDimension)
	}
	// A per-axis bound alone still permits a 20000x20000 allocation, so the
	// pixel count is bounded too. The fixed capture viewport is far below this.
	if width*height > screenshotMaxPixels {
		return 0, 0, fmt.Errorf("image declares %d pixels, more than the accepted %d",
			width*height, screenshotMaxPixels)
	}
	// Only now is a full decode safe. It proves the stream is complete rather
	// than a valid prefix, and that the header did not lie about its canvas.
	decoded, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return 0, 0, fmt.Errorf("image is not a decodable PNG: %v", err)
	}
	bounds := decoded.Bounds()
	if bounds.Dx() != width || bounds.Dy() != height {
		return 0, 0, fmt.Errorf("image dimensions disagree with the PNG header")
	}
	return width, height, nil
}

// captureScreenshots runs between the Researcher and every later role. It is a
// no-op - and requires no Cloudflare credential - when the Researcher asked for
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
	// The endpoint is settled before the credentials are read, so an unsafe
	// override can never reach the same code path as the bearer token.
	baseURL, err := resolveScreenshotBaseURL()
	if err != nil {
		return err
	}
	credentials, err := loadScreenshotCredentials()
	if err != nil {
		return err
	}
	client := newScreenshotClient(credentials, baseURL)
	manifest := ScreenshotManifest{
		SchemaVersion: screenshotManifestVersion,
		Engine:        screenshotEngine,
		Viewport:      ScreenshotViewport{Width: screenshotViewportWidth, Height: screenshotViewportHeight},
		Screenshots:   make([]ScreenshotRecord, 0, len(control.screenshotRequests)),
	}
	rollback := func() {
		_ = control.store.removeAll(screenshotAssetDir)
		_ = control.store.remove(screenshotManifestArtifact)
	}
	// Requests are captured strictly one at a time, in artifact order.
	for _, request := range control.screenshotRequests {
		captured, captureErr := client.capture(context.Background(), request)
		if captureErr != nil {
			rollback()
			return captureErr
		}
		relative := screenshotAssetPath(request.ID)
		if err := control.store.writeAtomic(relative, captured.data, 0o444); err != nil {
			rollback()
			return fmt.Errorf("persist screenshot %q: %w", request.ID, err)
		}
		manifest.Screenshots = append(manifest.Screenshots, ScreenshotRecord{
			ID: request.ID, Path: relative, RequestedURL: request.URL, Selector: request.Selector,
			CapturedAt: time.Now().UTC(), Supports: request.Supports, Reason: request.Reason,
			Engine: screenshotEngine, MediaType: screenshotMediaType, ByteSize: len(captured.data),
			Width: captured.width, Height: captured.height, SHA256: revisionFor(captured.data),
		})
	}
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		rollback()
		return fmt.Errorf("encode %s: %w", screenshotManifestArtifact, err)
	}
	if err := control.store.writeAtomic(screenshotManifestArtifact, append(data, '\n'), 0o444); err != nil {
		rollback()
		return fmt.Errorf("write %s: %w", screenshotManifestArtifact, err)
	}
	control.screenshotManifest = append(data, '\n')
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
		relative := screenshotAssetPath(request.ID)
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
