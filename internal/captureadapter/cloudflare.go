package captureadapter

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/coder/websocket"
	"github.com/uuta/write-uuter/internal/captureimage"
	"github.com/uuta/write-uuter/internal/captureprotocol"
)

const (
	backend             = "cloudflare-chromium"
	apiBaseURL          = "https://api.cloudflare.com/client/v4"
	accountEnv          = "CLOUDFLARE_ACCOUNT_ID"
	tokenEnv            = "CLOUDFLARE_API_TOKEN"
	testBaseURLEnv      = "WRITE_UUTER_TEST_BROWSER_RENDERING_BASE_URL"
	testTimeoutEnv      = "WRITE_UUTER_TEST_SCREENSHOT_TIMEOUT"
	requestTimeout      = 60 * time.Second
	sessionCleanupLimit = 2 * time.Second
	providerJSONLimit   = 1 << 20
	cdpMessageLimit     = 16 << 20
	viewportWidth       = 1280
	viewportHeight      = 800
)

var sessionIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$`)

type apiErrorEnvelope struct {
	Errors []struct {
		Code int `json:"code"`
	} `json:"errors"`
}

type browserSession struct {
	SessionID            string `json:"sessionId"`
	WebSocketDebuggerURL string `json:"webSocketDebuggerUrl"`
}

type browserSessionEnvelope struct {
	Result browserSession `json:"result"`
}

type cloudflareAPI struct {
	client  *http.Client
	baseURL string
	account string
	token   string
}

// RunCloudflare executes the current capture protocol in the working directory.
// Provider diagnostics are reduced to HTTP status and documented numeric error
// codes; bodies, headers, account IDs, tokens, and CDP details never cross the
// adapter boundary.
func RunCloudflare(arguments []string) error {
	if len(arguments) != 1 || arguments[0] != captureprotocol.VersionArgument {
		return fmt.Errorf("unsupported capture protocol invocation")
	}
	requestData, err := os.ReadFile(captureprotocol.RequestFile)
	if err != nil {
		return fmt.Errorf("read capture request")
	}
	var document captureprotocol.RequestDocument
	decoder := json.NewDecoder(bytes.NewReader(requestData))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&document); err != nil || document.SchemaVersion != captureprotocol.Version || len(document.Requests) == 0 {
		return fmt.Errorf("invalid capture request protocol")
	}
	if err := ensureJSONEOF(decoder); err != nil {
		return fmt.Errorf("invalid capture request protocol")
	}

	// Validate the adapter-only override before reading credentials. An invalid
	// ambient endpoint therefore cannot influence where a token is sent.
	baseURL, err := resolveBaseURL()
	if err != nil {
		return err
	}
	account := strings.TrimSpace(os.Getenv(accountEnv))
	token := strings.TrimSpace(os.Getenv(tokenEnv))
	if account == "" || token == "" || strings.ContainsAny(account, "/?#") || !printableCredential(account, 128) || !printableCredential(token, 4096) {
		return fmt.Errorf("capture adapter credentials are missing or invalid")
	}
	timeout := requestTimeout
	if injected, parseErr := time.ParseDuration(os.Getenv(testTimeoutEnv)); parseErr == nil && injected > 0 {
		timeout = injected
	}
	invocationDeadline, err := runnerDeadline()
	if err != nil {
		return err
	}
	client := &http.Client{
		Transport:     &http.Transport{DisableKeepAlives: true, Proxy: http.ProxyFromEnvironment},
		CheckRedirect: func(*http.Request, []*http.Request) error { return errors.New("redirect refused") },
	}
	provider := cloudflareAPI{client: client, baseURL: baseURL, account: account, token: token}
	if err := os.Mkdir(captureprotocol.AssetsDirectory, 0o700); err != nil {
		return fmt.Errorf("create capture assets directory")
	}

	results := make([]captureprotocol.Result, 0, len(document.Requests))
	for index, request := range document.Requests {
		requestBudget := timeout
		if !invocationDeadline.IsZero() {
			remaining := time.Until(invocationDeadline)
			if remaining < requestBudget {
				requestBudget = remaining
			}
		}
		if requestBudget <= 0 {
			return fmt.Errorf("capture invocation deadline expired")
		}
		data, width, height, finalURL, actions, err := provider.capture(request, requestBudget)
		if err != nil {
			return fmt.Errorf("capture request %d failed: %w", index+1, err)
		}
		name := fmt.Sprintf("capture-%03d.png", index+1)
		relative := filepath.ToSlash(filepath.Join(captureprotocol.AssetsDirectory, name))
		if err := os.WriteFile(relative, data, 0o600); err != nil {
			return fmt.Errorf("write capture asset")
		}
		digest := sha256.Sum256(data)
		results = append(results, captureprotocol.Result{
			RequestID: request.RequestID, RequestedURL: request.PublicURL, FinalURL: finalURL,
			CapturedAt: time.Now().UTC().Format(time.RFC3339Nano), Backend: backend,
			MediaType: captureprotocol.PNGMediaType,
			Viewport:  captureprotocol.Viewport{Width: viewportWidth, Height: viewportHeight}, FullPage: false,
			ImagePath: relative, ByteSize: int64(len(data)), Width: width, Height: height,
			SHA256: "sha256:" + hex.EncodeToString(digest[:]), SupportedClaimIDs: request.SupportedClaimIDs,
			Rationale: request.Reason, ActionSummary: actions,
		})
	}
	resultData, err := json.MarshalIndent(captureprotocol.ResultDocument{SchemaVersion: captureprotocol.Version, Results: results}, "", "  ")
	if err != nil {
		return fmt.Errorf("encode capture results")
	}
	if err := os.WriteFile(captureprotocol.ResultFile, append(resultData, '\n'), 0o600); err != nil {
		return fmt.Errorf("write capture results")
	}
	return nil
}

func runnerDeadline() (time.Time, error) {
	raw := strings.TrimSpace(os.Getenv(captureprotocol.RunnerDeadlineEnv))
	if raw == "" {
		return time.Time{}, nil
	}
	milliseconds, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || milliseconds <= 0 {
		return time.Time{}, fmt.Errorf("invalid capture invocation deadline")
	}
	deadline := time.UnixMilli(milliseconds)
	if !deadline.After(time.Now()) {
		return time.Time{}, fmt.Errorf("capture invocation deadline expired")
	}
	return deadline, nil
}

func (provider cloudflareAPI) capture(request captureprotocol.Request, timeout time.Duration) (data []byte, width, height int, finalURL string, actions []string, returnErr error) {
	cleanupLimit := sessionCleanupLimit
	if quarter := timeout / 4; quarter < cleanupLimit {
		cleanupLimit = quarter
	}
	if cleanupLimit < time.Millisecond {
		cleanupLimit = time.Millisecond
	}
	workLimit := timeout - cleanupLimit
	if workLimit < time.Millisecond {
		workLimit = time.Millisecond
	}
	ctx, cancel := context.WithTimeout(context.Background(), workLimit)
	defer cancel()

	session, err := provider.launchBrowser(ctx)
	if err != nil {
		return nil, 0, 0, "", nil, err
	}
	defer func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), cleanupLimit)
		defer cleanupCancel()
		if err := provider.closeBrowser(cleanupCtx, session.SessionID); err != nil {
			returnErr = errors.Join(returnErr, err)
		}
	}()
	connectionURL := provider.baseURL + "/accounts/" + provider.account + "/browser-rendering/devtools/browser/" + session.SessionID
	connectionHeader := http.Header{"Authorization": []string{"Bearer " + provider.token}, "Accept": []string{"*/*"}}
	connection, _, err := websocket.Dial(ctx, connectionURL, &websocket.DialOptions{HTTPClient: provider.client, HTTPHeader: connectionHeader})
	if err != nil {
		return nil, 0, 0, "", nil, fmt.Errorf("provider browser connection failed")
	}
	connection.SetReadLimit(cdpMessageLimit)
	defer connection.CloseNow()
	cdp := newCDPClient(connection)

	var target struct {
		TargetID string `json:"targetId"`
	}
	if err := cdp.call(ctx, "Target.createTarget", map[string]any{"url": "about:blank"}, "", &target); err != nil || target.TargetID == "" {
		return nil, 0, 0, "", nil, fmt.Errorf("provider browser target creation failed")
	}
	var attached struct {
		SessionID string `json:"sessionId"`
	}
	if err := cdp.call(ctx, "Target.attachToTarget", map[string]any{"targetId": target.TargetID, "flatten": true}, "", &attached); err != nil || attached.SessionID == "" {
		return nil, 0, 0, "", nil, fmt.Errorf("provider browser target attachment failed")
	}
	if err := cdp.call(ctx, "Page.enable", nil, attached.SessionID, nil); err != nil {
		return nil, 0, 0, "", nil, fmt.Errorf("provider browser setup failed")
	}
	if err := cdp.call(ctx, "Emulation.setDeviceMetricsOverride", map[string]any{
		"width": viewportWidth, "height": viewportHeight, "deviceScaleFactor": 1, "mobile": false,
	}, attached.SessionID, nil); err != nil {
		return nil, 0, 0, "", nil, fmt.Errorf("provider browser setup failed")
	}
	cdp.clearEvent("Page.loadEventFired", attached.SessionID)
	var navigation struct {
		ErrorText string `json:"errorText"`
	}
	if err := cdp.call(ctx, "Page.navigate", map[string]any{"url": request.PublicURL}, attached.SessionID, &navigation); err != nil || navigation.ErrorText != "" {
		return nil, 0, 0, "", nil, fmt.Errorf("provider navigation failed")
	}
	if err := cdp.waitEvent(ctx, "Page.loadEventFired", attached.SessionID); err != nil {
		return nil, 0, 0, "", nil, fmt.Errorf("provider navigation timed out")
	}
	var history struct {
		CurrentIndex int `json:"currentIndex"`
		Entries      []struct {
			URL string `json:"url"`
		} `json:"entries"`
	}
	if err := cdp.call(ctx, "Page.getNavigationHistory", nil, attached.SessionID, &history); err != nil || history.CurrentIndex < 0 || history.CurrentIndex >= len(history.Entries) || strings.TrimSpace(history.Entries[history.CurrentIndex].URL) == "" {
		return nil, 0, 0, "", nil, fmt.Errorf("provider did not report the final navigation URL")
	}
	finalURL = history.Entries[history.CurrentIndex].URL

	parameters := map[string]any{"format": "png", "fromSurface": true, "captureBeyondViewport": false}
	actions = []string{"navigate to the requested public URL", "observe the final URL from the capture session"}
	if request.Selector != "" {
		clip, err := selectorClip(ctx, cdp, attached.SessionID, request.Selector)
		if err != nil {
			return nil, 0, 0, "", nil, err
		}
		parameters["clip"] = clip
		parameters["captureBeyondViewport"] = true
		actions = append(actions, "capture the requested selector as a PNG")
	} else {
		actions = append(actions, "capture the visible viewport as a PNG")
	}
	var screenshot struct {
		Data string `json:"data"`
	}
	if err := cdp.call(ctx, "Page.captureScreenshot", parameters, attached.SessionID, &screenshot); err != nil || screenshot.Data == "" {
		return nil, 0, 0, "", nil, fmt.Errorf("provider screenshot failed")
	}
	data, err = base64.StdEncoding.DecodeString(screenshot.Data)
	if err != nil {
		return nil, 0, 0, "", nil, fmt.Errorf("provider returned an invalid or oversized PNG")
	}
	width, height, err = validateProviderPNG(data)
	if err != nil {
		return nil, 0, 0, "", nil, fmt.Errorf("provider returned an invalid PNG: %w", err)
	}
	return data, width, height, finalURL, actions, nil
}

func validateProviderPNG(data []byte) (int, int, error) {
	if len(data) > captureimage.MaxBytes {
		return 0, 0, fmt.Errorf("image exceeds the byte limit")
	}
	return captureimage.ValidatePNG(data)
}

func selectorClip(ctx context.Context, cdp *cdpClient, sessionID, selector string) (map[string]any, error) {
	var document struct {
		Root struct {
			NodeID int `json:"nodeId"`
		} `json:"root"`
	}
	if err := cdp.call(ctx, "DOM.getDocument", map[string]any{"depth": 0}, sessionID, &document); err != nil || document.Root.NodeID == 0 {
		return nil, fmt.Errorf("provider could not inspect the requested selector")
	}
	var queried struct {
		NodeID int `json:"nodeId"`
	}
	if err := cdp.call(ctx, "DOM.querySelector", map[string]any{"nodeId": document.Root.NodeID, "selector": selector}, sessionID, &queried); err != nil || queried.NodeID == 0 {
		return nil, fmt.Errorf("provider did not find the requested selector")
	}
	var box struct {
		Model struct {
			Border []float64 `json:"border"`
		} `json:"model"`
	}
	if err := cdp.call(ctx, "DOM.getBoxModel", map[string]any{"nodeId": queried.NodeID}, sessionID, &box); err != nil || len(box.Model.Border) != 8 {
		return nil, fmt.Errorf("provider could not measure the requested selector")
	}
	minX, minY := box.Model.Border[0], box.Model.Border[1]
	maxX, maxY := minX, minY
	for index := 0; index < len(box.Model.Border); index += 2 {
		minX = math.Min(minX, box.Model.Border[index])
		maxX = math.Max(maxX, box.Model.Border[index])
		minY = math.Min(minY, box.Model.Border[index+1])
		maxY = math.Max(maxY, box.Model.Border[index+1])
	}
	x, y := math.Floor(minX), math.Floor(minY)
	width, height := math.Ceil(maxX)-x, math.Ceil(maxY)-y
	if x < 0 || y < 0 || width < 1 || height < 1 || width > captureimage.MaxDimension || height > captureimage.MaxDimension || width*height > captureimage.MaxPixels {
		return nil, fmt.Errorf("provider reported invalid selector dimensions")
	}
	return map[string]any{"x": x, "y": y, "width": width, "height": height, "scale": 1}, nil
}

func (provider cloudflareAPI) launchBrowser(ctx context.Context) (browserSession, error) {
	var raw json.RawMessage
	if err := provider.doJSON(ctx, http.MethodPost, "/accounts/"+provider.account+"/browser-rendering/devtools/browser", []byte("{}"), &raw); err != nil {
		return browserSession{}, err
	}
	var session browserSession
	if err := json.Unmarshal(raw, &session); err != nil || session.SessionID == "" {
		var envelope browserSessionEnvelope
		if envelopeErr := json.Unmarshal(raw, &envelope); envelopeErr != nil {
			return browserSession{}, fmt.Errorf("provider returned an invalid browser session")
		}
		session = envelope.Result
	}
	if !sessionIDPattern.MatchString(session.SessionID) {
		return browserSession{}, fmt.Errorf("provider returned an invalid browser session")
	}
	return session, nil
}

func (provider cloudflareAPI) closeBrowser(ctx context.Context, sessionID string) error {
	if err := provider.doJSON(ctx, http.MethodDelete, "/accounts/"+provider.account+"/browser-rendering/devtools/browser/"+sessionID, nil, nil); err != nil {
		return fmt.Errorf("provider browser cleanup failed")
	}
	return nil
}

func (provider cloudflareAPI) doJSON(ctx context.Context, method, path string, body []byte, destination any) error {
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	request, err := http.NewRequestWithContext(ctx, method, provider.baseURL+path, reader)
	if err != nil {
		return fmt.Errorf("build provider request")
	}
	request.Header.Set("Authorization", "Bearer "+provider.token)
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	request.Header.Set("Accept", "application/json")
	response, err := provider.client.Do(request)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return fmt.Errorf("provider request timed out")
		}
		return fmt.Errorf("provider request failed")
	}
	defer response.Body.Close()
	responseBody, err := io.ReadAll(io.LimitReader(response.Body, providerJSONLimit+1))
	if err != nil || len(responseBody) > providerJSONLimit {
		return fmt.Errorf("provider response read failed")
	}
	if response.StatusCode < 200 || response.StatusCode > 299 {
		return fmt.Errorf("provider returned HTTP %d (%s)", response.StatusCode, apiErrorCodes(responseBody))
	}
	if destination != nil {
		if err := json.Unmarshal(responseBody, destination); err != nil {
			return fmt.Errorf("provider returned invalid JSON")
		}
	}
	return nil
}

func ensureJSONEOF(decoder *json.Decoder) error {
	if err := decoder.Decode(new(any)); err != io.EOF {
		return fmt.Errorf("multiple JSON values")
	}
	return nil
}

func printableCredential(value string, limit int) bool {
	if value == "" || len(value) > limit {
		return false
	}
	for _, character := range value {
		if character < 0x21 || character > 0x7e {
			return false
		}
	}
	return true
}

func resolveBaseURL() (string, error) {
	override := strings.TrimSpace(os.Getenv(testBaseURLEnv))
	if override == "" {
		return apiBaseURL, nil
	}
	parsed, err := url.Parse(override)
	if err != nil || parsed.Opaque != "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return "", fmt.Errorf("invalid test provider origin")
	}
	address := parsed.Hostname()
	if address != "localhost" && address != "127.0.0.1" && address != "::1" {
		return "", fmt.Errorf("test provider origin is not loopback")
	}
	return strings.TrimSuffix(override, "/"), nil
}

func apiErrorCodes(body []byte) string {
	var envelope apiErrorEnvelope
	if err := json.Unmarshal(body, &envelope); err != nil || len(envelope.Errors) == 0 {
		return "no documented error code"
	}
	codes := make([]string, 0, len(envelope.Errors))
	for _, item := range envelope.Errors {
		codes = append(codes, strconv.Itoa(item.Code))
	}
	return "Cloudflare error codes " + strings.Join(codes, ", ")
}
