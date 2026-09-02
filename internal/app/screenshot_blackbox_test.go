package app_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/uuta/write-uuter/internal/captureprotocol"
)

// The fixture credentials carry a sentinel prefix the fake agent also knows,
// so a leak into any observable surface is detected rather than assumed away.
const (
	fixtureAccountID = "write-uuter-fixture-secret-account-a1"
	fixtureAPIToken  = "write-uuter-fixture-secret-token-b2"
	secretSentinel   = "write-uuter-fixture-secret-"
)

type screenshotManifestFile struct {
	SchemaVersion int `json:"schema_version"`
	Screenshots   []struct {
		ID           string   `json:"id"`
		Path         string   `json:"path"`
		RequestedURL string   `json:"requested_url"`
		FinalURL     string   `json:"final_url"`
		Selector     string   `json:"selector"`
		CapturedAt   string   `json:"captured_at"`
		Supports     []string `json:"supports"`
		Reason       string   `json:"reason"`
		Backend      string   `json:"backend"`
		MediaType    string   `json:"media_type"`
		Viewport     struct {
			Width  int `json:"width"`
			Height int `json:"height"`
		} `json:"viewport"`
		FullPage         bool   `json:"full_page"`
		ByteSize         int    `json:"byte_size"`
		Width            int    `json:"width"`
		Height           int    `json:"height"`
		SHA256           string `json:"sha256"`
		Attempt          int    `json:"attempt"`
		EditorialOutcome *struct {
			RequestID string `json:"request_id"`
			Status    string `json:"status"`
			Reason    string `json:"reason"`
		} `json:"editorial_outcome"`
		PriorAttempts []struct {
			Attempt          int    `json:"attempt"`
			Path             string `json:"path"`
			Backend          string `json:"backend"`
			SHA256           string `json:"sha256"`
			EditorialOutcome *struct {
				RequestID string `json:"request_id"`
				Status    string `json:"status"`
				Reason    string `json:"reason"`
			} `json:"editorial_outcome"`
		} `json:"prior_attempts"`
	} `json:"screenshots"`
}

type capturedCall struct {
	AuthHead       string
	RequestedURL   string
	FinalURL       string
	Selector       string
	ViewportWidth  int
	ViewportHeight int
	Screenshot     map[string]any
	Methods        []string
}

type fakeBrowserSession struct {
	callIndex int
	image     []byte
}

// fakeBrowserRendering stands in for Cloudflare's DevTools session API and
// its WebSocket CDP connection. It records same-session navigation, final URL
// observation, optional selector lookup, viewport, screenshot, and cleanup.
type fakeBrowserRendering struct {
	server *httptest.Server

	mutex       sync.Mutex
	calls       []capturedCall
	sessions    map[string]*fakeBrowserSession
	inFlight    int
	maxInFlight int
	deleted     int
	hangMethod  string
	releaseHang chan struct{}

	respond func(index int, writer http.ResponseWriter)
}

func newFakeBrowserRendering(t *testing.T, respond func(index int, writer http.ResponseWriter)) *fakeBrowserRendering {
	t.Helper()
	fake := &fakeBrowserRendering{respond: respond, sessions: make(map[string]*fakeBrowserSession), releaseHang: make(chan struct{})}
	t.Cleanup(func() {
		select {
		case <-fake.releaseHang:
		default:
			close(fake.releaseHang)
		}
	})
	fake.server = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		launchPath := "/accounts/" + fixtureAccountID + "/browser-rendering/devtools/browser"
		if request.Method == http.MethodPost && request.URL.Path == launchPath {
			fake.mutex.Lock()
			index := len(fake.calls)
			fake.calls = append(fake.calls, capturedCall{AuthHead: request.Header.Get("Authorization")})
			fake.inFlight++
			if fake.inFlight > fake.maxInFlight {
				fake.maxInFlight = fake.inFlight
			}
			fake.mutex.Unlock()

			recorder := httptest.NewRecorder()
			fake.respond(index, recorder)
			response := recorder.Result()
			if response.StatusCode < 200 || response.StatusCode > 299 {
				fake.mutex.Lock()
				fake.inFlight--
				fake.mutex.Unlock()
				for name, values := range response.Header {
					writer.Header()[name] = append([]string(nil), values...)
				}
				writer.WriteHeader(response.StatusCode)
				_, _ = writer.Write(recorder.Body.Bytes())
				return
			}
			sessionID := fmt.Sprintf("session-%d", index+1)
			fake.mutex.Lock()
			fake.sessions[sessionID] = &fakeBrowserSession{callIndex: index, image: append([]byte(nil), recorder.Body.Bytes()...)}
			fake.mutex.Unlock()
			websocketURL := "ws" + strings.TrimPrefix(fake.server.URL, "http") + "/cdp/" + sessionID
			writer.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(writer).Encode(map[string]string{"sessionId": sessionID, "webSocketDebuggerUrl": websocketURL})
			return
		}
		if request.Method == http.MethodDelete && strings.HasPrefix(request.URL.Path, launchPath+"/") {
			sessionID := strings.TrimPrefix(request.URL.Path, launchPath+"/")
			fake.mutex.Lock()
			if _, exists := fake.sessions[sessionID]; exists {
				delete(fake.sessions, sessionID)
				fake.inFlight--
				fake.deleted++
			}
			fake.mutex.Unlock()
			writer.Header().Set("Content-Type", "application/json")
			_, _ = writer.Write([]byte(`{}`))
			return
		}
		if request.Method == http.MethodGet && strings.HasPrefix(request.URL.Path, launchPath+"/") {
			fake.serveCDP(writer, request, strings.TrimPrefix(request.URL.Path, launchPath+"/"))
			return
		}
		http.NotFound(writer, request)
	}))
	t.Cleanup(fake.server.Close)
	return fake
}

func (fake *fakeBrowserRendering) serveCDP(writer http.ResponseWriter, request *http.Request, sessionID string) {
	fake.mutex.Lock()
	session := fake.sessions[sessionID]
	fake.mutex.Unlock()
	if session == nil {
		http.NotFound(writer, request)
		return
	}
	connection, err := websocket.Accept(writer, request, nil)
	if err != nil {
		return
	}
	defer connection.CloseNow()
	ctx := context.Background()
	for {
		_, payload, err := connection.Read(ctx)
		if err != nil {
			return
		}
		var command struct {
			ID        int64          `json:"id"`
			Method    string         `json:"method"`
			Params    map[string]any `json:"params"`
			SessionID string         `json:"sessionId"`
		}
		if json.Unmarshal(payload, &command) != nil {
			return
		}
		fake.mutex.Lock()
		call := &fake.calls[session.callIndex]
		call.Methods = append(call.Methods, command.Method)
		fake.mutex.Unlock()
		if fake.hangMethod == command.Method {
			<-fake.releaseHang
			return
		}
		result := any(map[string]any{})
		switch command.Method {
		case "Target.createTarget":
			result = map[string]any{"targetId": "target-1"}
		case "Target.attachToTarget":
			result = map[string]any{"sessionId": "target-session-1"}
		case "Emulation.setDeviceMetricsOverride":
			fake.mutex.Lock()
			call.ViewportWidth = int(command.Params["width"].(float64))
			call.ViewportHeight = int(command.Params["height"].(float64))
			fake.mutex.Unlock()
		case "Page.navigate":
			requested, _ := command.Params["url"].(string)
			fake.mutex.Lock()
			call.RequestedURL = requested
			call.FinalURL = requested + "?observed-final=1"
			fake.mutex.Unlock()
			result = map[string]any{"frameId": "frame-1"}
			fake.writeCDP(ctx, connection, map[string]any{"method": "Page.loadEventFired", "sessionId": command.SessionID, "params": map[string]any{"timestamp": 1}})
		case "Page.getNavigationHistory":
			fake.mutex.Lock()
			finalURL := call.FinalURL
			fake.mutex.Unlock()
			result = map[string]any{"currentIndex": 0, "entries": []map[string]any{{"id": 1, "url": finalURL, "title": "fixture", "transitionType": "typed"}}}
		case "DOM.getDocument":
			result = map[string]any{"root": map[string]any{"nodeId": 1}}
		case "DOM.querySelector":
			selector, _ := command.Params["selector"].(string)
			fake.mutex.Lock()
			call.Selector = selector
			fake.mutex.Unlock()
			result = map[string]any{"nodeId": 2}
		case "DOM.getBoxModel":
			result = map[string]any{"model": map[string]any{"border": []float64{10, 20, 74, 20, 74, 68, 10, 68}}}
		case "Page.captureScreenshot":
			fake.mutex.Lock()
			call.Screenshot = command.Params
			fake.mutex.Unlock()
			result = map[string]any{"data": base64.StdEncoding.EncodeToString(session.image)}
		}
		fake.writeCDP(ctx, connection, map[string]any{"id": command.ID, "result": result, "sessionId": command.SessionID})
	}
}

func (fake *fakeBrowserRendering) writeCDP(ctx context.Context, connection *websocket.Conn, message any) {
	payload, _ := json.Marshal(message)
	_ = connection.Write(ctx, websocket.MessageText, payload)
}

func (fake *fakeBrowserRendering) recorded() ([]capturedCall, int) {
	fake.mutex.Lock()
	defer fake.mutex.Unlock()
	return append([]capturedCall(nil), fake.calls...), fake.maxInFlight
}

func (fake *fakeBrowserRendering) cleanupState() (active, deleted int) {
	fake.mutex.Lock()
	defer fake.mutex.Unlock()
	return len(fake.sessions), fake.deleted
}

func respondPNG(width, height int) func(int, http.ResponseWriter) {
	return func(_ int, writer http.ResponseWriter) {
		writer.Header().Set("Content-Type", "image/png")
		// A header a careless implementation might copy into an error.
		writer.Header().Set("X-Fixture-Echo", "account "+fixtureAccountID)
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write(fixturePNG(width, height))
	}
}

// fixturePNG renders a deterministic image whose bytes depend on its size.
func fixturePNG(width, height int) []byte {
	canvas := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			canvas.Set(x, y, color.RGBA{R: uint8(x * 3), G: uint8(y * 5), B: 0x40, A: 0xff})
		}
	}
	var buffer bytes.Buffer
	if err := png.Encode(&buffer, canvas); err != nil {
		panic(err)
	}
	return buffer.Bytes()
}

type screenshotOptions struct {
	credentials    bool
	baseURL        string
	timeout        string
	runTimeout     string
	omitRunner     bool
	runner         string
	runnerScenario string
	runnerTimeout  string
	runnerLog      string
	trackerDelay   string
}

func runScreenshotScenario(t *testing.T, scenario string, options screenshotOptions) scenarioRun {
	t.Helper()
	binary, fake, runDir, fixtureDir := prepareScenario(t, scenario)
	runTimeout := options.runTimeout
	if runTimeout == "" {
		runTimeout = "10s"
	}
	command := newRunCommand(t, binary, fake, runDir, runTimeout, filepath.Join(repositoryRoot(t), "examples", "brief.md"))
	command.Env = withoutEnv(command.Env, "CLOUDFLARE_ACCOUNT_ID", "CLOUDFLARE_API_TOKEN",
		"WRITE_UUTER_CAPTURE_RUNNER", "WRITE_UUTER_TEST_BROWSER_RENDERING_BASE_URL", "WRITE_UUTER_TEST_SCREENSHOT_TIMEOUT",
		"WRITE_UUTER_CAPTURE_RUNNER_DEADLINE_UNIX_MS", "WRITE_UUTER_TEST_CAPTURE_INVOCATION_LOG",
		"WRITE_UUTER_TEST_CAPTURE_TRACKER_DELAY")
	if !options.omitRunner && (options.baseURL != "" || options.runner != "") {
		runner := options.runner
		if runner == "" {
			runner = filepath.Join(buildDir, "write-uuter-cloudflare-capture")
		}
		command.Env = append(command.Env, "WRITE_UUTER_CAPTURE_RUNNER="+runner)
	}
	if options.runnerScenario != "" {
		command.Env = append(command.Env, "WRITE_UUTER_TEST_CAPTURE_SCENARIO="+options.runnerScenario)
	}
	if options.runnerTimeout != "" {
		command.Env = append(command.Env, "WRITE_UUTER_TEST_CAPTURE_RUNNER_TIMEOUT="+options.runnerTimeout)
	}
	if options.runnerLog != "" {
		command.Env = append(command.Env, "WRITE_UUTER_TEST_CAPTURE_INVOCATION_LOG="+options.runnerLog)
	}
	if options.trackerDelay != "" {
		command.Env = append(command.Env, "WRITE_UUTER_TEST_CAPTURE_TRACKER_DELAY="+options.trackerDelay)
	}
	if options.credentials {
		command.Env = append(command.Env,
			"CLOUDFLARE_ACCOUNT_ID="+fixtureAccountID,
			"CLOUDFLARE_API_TOKEN="+fixtureAPIToken)
	}
	if options.baseURL != "" {
		command.Env = append(command.Env, "WRITE_UUTER_TEST_BROWSER_RENDERING_BASE_URL="+options.baseURL)
	}
	if options.timeout != "" {
		command.Env = append(command.Env, "WRITE_UUTER_TEST_SCREENSHOT_TIMEOUT="+options.timeout)
	}
	output, err := command.CombinedOutput()
	return scenarioRun{runDir: runDir, fixtureDir: fixtureDir, output: string(output), err: err}
}

func withoutEnv(environment []string, names ...string) []string {
	blocked := make(map[string]bool, len(names))
	for _, name := range names {
		blocked[name] = true
	}
	result := make([]string, 0, len(environment))
	for _, entry := range environment {
		name, _, _ := strings.Cut(entry, "=")
		if blocked[name] {
			continue
		}
		result = append(result, entry)
	}
	return result
}

// A run whose Researcher asked for nothing must behave exactly as before: no
// browser request, no Cloudflare credential, and no screenshot artifact.
func TestBlackBoxScreenshotNoRequestKeepsExistingBehaviour(t *testing.T) {
	for _, scenario := range []string{"happy", "shot_empty"} {
		t.Run(scenario, func(t *testing.T) {
			fake := newFakeBrowserRendering(t, respondPNG(64, 48))
			run := runScreenshotScenario(t, scenario, screenshotOptions{baseURL: fake.server.URL, omitRunner: true})
			if run.err != nil {
				t.Fatalf("CLI failed without Cloudflare credentials: %v\n%s", run.err, run.output)
			}
			state := readWorkflow(t, run.runDir)
			if state.Status != "succeeded" || state.Phase != "complete" {
				t.Fatalf("unexpected workflow: %+v", state)
			}
			if calls, _ := fake.recorded(); len(calls) != 0 {
				t.Fatalf("browser rendering was called for a run with no request: %+v", calls)
			}
			for _, relative := range []string{"evidence/screenshots.json", "evidence/screenshot-requests.json", "evidence/assets/screenshots"} {
				if _, err := os.Lstat(filepath.Join(run.runDir, relative)); err == nil {
					t.Errorf("%s exists for a run with no screenshot request", relative)
				}
			}
			assertNoScreenshotContext(t, run.fixtureDir)
			assertProcessesGone(t, readInvocationRecords(t, run.fixtureDir))
		})
	}
}

// One accepted request must produce a validated PNG, a complete generated
// manifest, and read-only Writer/Evidence context - and nothing else.
func TestBlackBoxScreenshotCaptureProducesManifestDigestAndRoleContext(t *testing.T) {
	fake := newFakeBrowserRendering(t, respondPNG(64, 48))
	run := runScreenshotScenario(t, "shot_selector", screenshotOptions{credentials: true, baseURL: fake.server.URL})
	if run.err != nil {
		t.Fatalf("CLI failed: %v\n%s", run.err, run.output)
	}
	if state := readWorkflow(t, run.runDir); state.Status != "succeeded" {
		t.Fatalf("unexpected workflow: %+v", state)
	}

	calls, maxInFlight := fake.recorded()
	if len(calls) != 1 {
		t.Fatalf("got %d browser rendering calls, want 1", len(calls))
	}
	call := calls[0]
	if call.AuthHead != "Bearer "+fixtureAPIToken {
		t.Errorf("session launch did not carry the adapter-owned bearer token")
	}
	if call.RequestedURL != "https://example.com/report" || call.Selector != "main" {
		t.Errorf("same-session navigation/selector were not preserved: %+v", call)
	}
	if call.FinalURL == call.RequestedURL || call.FinalURL != "https://example.com/report?observed-final=1" {
		t.Errorf("final URL was not observed independently in the capture session: %+v", call)
	}
	if call.ViewportWidth != 1280 || call.ViewportHeight != 800 {
		t.Errorf("viewport is not the fixed documented viewport: %+v", call)
	}
	if call.Screenshot["format"] != "png" || call.Screenshot["captureBeyondViewport"] != true || call.Screenshot["clip"] == nil {
		t.Errorf("selector screenshot options were not preserved: %+v", call.Screenshot)
	}
	for _, required := range []string{"Target.createTarget", "Target.attachToTarget", "Page.navigate", "Page.getNavigationHistory", "DOM.querySelector", "Page.captureScreenshot"} {
		if !slices.Contains(call.Methods, required) {
			t.Errorf("same-session CDP sequence omitted %s: %v", required, call.Methods)
		}
	}
	if maxInFlight != 1 {
		t.Errorf("captures were concurrent: max in flight = %d", maxInFlight)
	}

	manifest := readScreenshotManifest(t, run.runDir)
	if manifest.SchemaVersion != 3 {
		t.Errorf("unexpected manifest header: %+v", manifest)
	}
	if len(manifest.Screenshots) != 1 {
		t.Fatalf("got %d manifest entries, want 1", len(manifest.Screenshots))
	}
	record := manifest.Screenshots[0]
	imagePath := filepath.Join(run.runDir, "evidence", "assets", "screenshots", "shot-001.png")
	imageBytes, err := os.ReadFile(imagePath)
	if err != nil {
		t.Fatalf("captured image missing: %v", err)
	}
	if !bytes.Equal(imageBytes, fixturePNG(64, 48)) {
		t.Errorf("stored screenshot bytes differ from the captured response")
	}
	if record.ID != "shot-001" || record.Path != "evidence/assets/screenshots/shot-001.png" {
		t.Errorf("unexpected record identity: %+v", record)
	}
	if record.RequestedURL != "https://example.com/report" || record.FinalURL != call.FinalURL || record.FinalURL == record.RequestedURL || record.Selector != "main" {
		t.Errorf("record lost the requested URL/selector: %+v", record)
	}
	if len(record.Supports) != 1 || record.Supports[0] != "claim-004" || strings.TrimSpace(record.Reason) == "" {
		t.Errorf("record lost claim linkage or rationale: %+v", record)
	}
	if record.Backend != "cloudflare-chromium" || record.MediaType != "image/png" {
		t.Errorf("record lost provenance: %+v", record)
	}
	if record.Viewport.Width != 1280 || record.Viewport.Height != 800 || record.FullPage {
		t.Errorf("record lost viewport/full-page provenance: %+v", record)
	}
	if record.ByteSize != len(imageBytes) || record.Width != 64 || record.Height != 48 {
		t.Errorf("record size/dimensions disagree with the stored image: %+v", record)
	}
	if record.SHA256 != revision(imageBytes) {
		t.Errorf("record digest %q does not match the stored bytes", record.SHA256)
	}
	if _, err := time.Parse(time.RFC3339Nano, record.CapturedAt); err != nil {
		t.Errorf("captured_at is not a timestamp: %q", record.CapturedAt)
	}

	for _, item := range []struct {
		relative string
		mode     os.FileMode
	}{
		{"evidence/screenshots.json", 0o444},
		{"evidence/screenshot-requests.json", 0o444},
		{"evidence/assets/screenshots/shot-001.png", 0o444},
	} {
		info, statErr := os.Lstat(filepath.Join(run.runDir, item.relative))
		if statErr != nil {
			t.Fatalf("%s missing: %v", item.relative, statErr)
		}
		if info.Mode().Perm() != item.mode {
			t.Errorf("%s mode = %v, want %v", item.relative, info.Mode().Perm(), item.mode)
		}
	}

	records := readInvocationRecords(t, run.fixtureDir)
	sawWriter, sawEvidence, sawVisualEditor := false, false, false
	for _, invocation := range records {
		files := workspaceFileSet(invocation)
		hasManifest := files["context/evidence/screenshots.json"]
		hasImage := files["context/evidence/assets/screenshots/shot-001.png"]
		// The assembly Writer invocation places the already validated plan and
		// is identified by the plan staged into its workspace, so only the
		// prose draft invocation owns the screenshot context.
		assembly := files["context/visual-plan.json"]
		switch {
		case invocation.Role == "writer" && !assembly:
			sawWriter = true
			if !hasManifest || !hasImage {
				t.Errorf("writer did not receive read-only screenshot context: %v", invocation.WorkspaceFiles)
			}
			if !strings.Contains(invocation.Prompt, screenshotContextMarker) || !strings.Contains(invocation.Prompt, "shot-001") {
				t.Errorf("writer prompt does not carry the generated screenshot manifest")
			}
		case invocation.Lens == "evidence":
			sawEvidence = true
			if !hasManifest || !hasImage {
				t.Errorf("evidence reviewer did not receive read-only screenshot context: %v", invocation.WorkspaceFiles)
			}
		case invocation.Role == "visual_editor":
			sawVisualEditor = true
			// A captured screenshot is a placeable visual input, so the
			// Visual Editor gets the manifest describing what each capture
			// supports. It reads the image from the staged input pool.
			if !hasManifest {
				t.Errorf("visual editor did not receive the screenshot manifest: %v", invocation.WorkspaceFiles)
			}
			if !files["context/visual-inputs/shot-001.png"] {
				t.Errorf("visual editor did not receive the staged screenshot input: %v", invocation.WorkspaceFiles)
			}
		default:
			if hasManifest || hasImage {
				t.Errorf("%s/%s received screenshot context it does not own", invocation.Role, invocation.Lens)
			}
		}
		if strings.Contains(invocation.Prompt, string(pngSignatureBytes)) {
			t.Errorf("%s prompt embedded raw image bytes", invocation.Role)
		}
	}
	if !sawWriter || !sawEvidence || !sawVisualEditor {
		t.Fatalf("writer/evidence/visual editor invocations were not observed")
	}
	assertNoCredentialLeak(t, run, records)
	assertProcessesGone(t, records)
}

// Several requests are captured strictly one at a time, and the documented
// five-item ceiling is enforced before any capture happens.
func TestBlackBoxScreenshotSequentialCaptureAndFiveItemLimit(t *testing.T) {
	t.Run("multiple", func(t *testing.T) {
		fake := newFakeBrowserRendering(t, func(index int, writer http.ResponseWriter) {
			// A slow first response would overlap a concurrent second one.
			if index == 0 {
				time.Sleep(150 * time.Millisecond)
			}
			respondPNG(32+index, 24+index)(index, writer)
		})
		run := runScreenshotScenario(t, "shot_multi", screenshotOptions{credentials: true, baseURL: fake.server.URL})
		if run.err != nil {
			t.Fatalf("CLI failed: %v\n%s", run.err, run.output)
		}
		calls, maxInFlight := fake.recorded()
		if len(calls) != 3 {
			t.Fatalf("got %d calls, want 3", len(calls))
		}
		if maxInFlight != 1 {
			t.Fatalf("captures overlapped: max in flight = %d", maxInFlight)
		}
		wantOrder := []string{"https://example.com/report", "https://example.com/changelog", "https://example.com/pricing"}
		for index, want := range wantOrder {
			if calls[index].RequestedURL != want {
				t.Errorf("call %d requested %v, want %q", index, calls[index].RequestedURL, want)
			}
		}
		manifest := readScreenshotManifest(t, run.runDir)
		if len(manifest.Screenshots) != 3 {
			t.Fatalf("got %d manifest entries, want 3", len(manifest.Screenshots))
		}
		for index, record := range manifest.Screenshots {
			data, err := os.ReadFile(filepath.Join(run.runDir, record.Path))
			if err != nil {
				t.Fatalf("screenshot %s missing: %v", record.ID, err)
			}
			if !bytes.Equal(data, fixturePNG(32+index, 24+index)) || record.SHA256 != revision(data) {
				t.Errorf("screenshot %s is not the exact captured image", record.ID)
			}
		}
		assertProcessesGone(t, readInvocationRecords(t, run.fixtureDir))
	})

	t.Run("at_limit", func(t *testing.T) {
		fake := newFakeBrowserRendering(t, respondPNG(16, 16))
		run := runScreenshotScenario(t, "shot_five", screenshotOptions{credentials: true, baseURL: fake.server.URL})
		if run.err != nil {
			t.Fatalf("five requests were rejected: %v\n%s", run.err, run.output)
		}
		if calls, _ := fake.recorded(); len(calls) != 5 {
			t.Fatalf("got %d calls, want 5", len(calls))
		}
	})

	t.Run("over_limit", func(t *testing.T) {
		fake := newFakeBrowserRendering(t, respondPNG(16, 16))
		run := runScreenshotScenario(t, "shot_six", screenshotOptions{credentials: true, baseURL: fake.server.URL})
		assertBlockedBeforeWriter(t, run, "the limit is 5")
		if calls, _ := fake.recorded(); len(calls) != 0 {
			t.Fatalf("an over-limit request still reached the browser: %+v", calls)
		}
	})
}

// Missing credentials block only the non-empty request path.
func TestBlackBoxScreenshotMissingRunnerCredentialsBlockOnlyRequestedCaptures(t *testing.T) {
	fake := newFakeBrowserRendering(t, respondPNG(16, 16))
	run := runScreenshotScenario(t, "shot_one", screenshotOptions{baseURL: fake.server.URL})
	assertBlockedBeforeWriter(t, run, "capture runner exited with status 1")
	state := readWorkflow(t, run.runDir)
	if strings.Contains(state.BlockReason, "CLOUDFLARE") {
		t.Errorf("block reason crossed the provider credential boundary: %q", state.BlockReason)
	}
	if calls, _ := fake.recorded(); len(calls) != 0 {
		t.Fatalf("a credential-less run still called the browser: %+v", calls)
	}
}

// Every rejected request shape blocks before the Writer and before any capture.
func TestBlackBoxScreenshotInvalidRequestsNeverReachTheWriter(t *testing.T) {
	for _, testCase := range []struct{ scenario, want string }{
		{"shot_bad_json", "invalid evidence/screenshot-requests.json"},
		{"shot_missing_field", "missing required field"},
		{"shot_null_list", "must be an array"},
		{"shot_dup_key", "duplicate JSON key"},
		{"shot_dup_id", "duplicates screenshot ID"},
		{"shot_case_id", "compared case-insensitively"},
		{"shot_unknown_field", "unknown field"},
		{"shot_unknown_claim", "not in claim-ledger.md"},
		{"shot_unsafe_url", "URL embeds credentials"},
		{"shot_unsafe_id", "not filename-safe"},
		{"shot_asset_dir", "controller-owned"},
	} {
		t.Run(testCase.scenario, func(t *testing.T) {
			fake := newFakeBrowserRendering(t, respondPNG(16, 16))
			run := runScreenshotScenario(t, testCase.scenario, screenshotOptions{credentials: true, baseURL: fake.server.URL})
			assertBlockedBeforeWriter(t, run, testCase.want)
			if calls, _ := fake.recorded(); len(calls) != 0 {
				t.Fatalf("an invalid request still reached the browser: %+v", calls)
			}
			assertNoCredentialLeak(t, run, readInvocationRecords(t, run.fixtureDir))
		})
	}
}

// Every capture-side failure blocks before the Writer with a credential-free,
// inspectable reason and leaves no partial evidence behind.
func TestBlackBoxScreenshotCaptureFailuresNeverReachTheWriter(t *testing.T) {
	oversized := append(fixturePNG(16, 16), bytes.Repeat([]byte{0x41}, 10<<20)...)
	for _, testCase := range []struct {
		name    string
		want    string
		timeout string
		respond func(int, http.ResponseWriter)
	}{
		{
			name: "non_2xx", want: "returned HTTP 401",
			respond: func(_ int, writer http.ResponseWriter) {
				writer.Header().Set("Content-Type", "application/json")
				writer.WriteHeader(http.StatusUnauthorized)
				// The body deliberately echoes the account ID.
				fmt.Fprintf(writer, `{"success":false,"errors":[{"message":"bad token for account %s"}]}`, fixtureAccountID)
			},
		},
		{
			name: "not_an_image", want: "is not a PNG",
			respond: func(_ int, writer http.ResponseWriter) {
				writer.Header().Set("Content-Type", "image/png")
				writer.WriteHeader(http.StatusOK)
				_, _ = writer.Write([]byte("<html>not an image</html>"))
			},
		},
		{
			name: "truncated_png", want: "not a decodable PNG",
			respond: func(_ int, writer http.ResponseWriter) {
				writer.Header().Set("Content-Type", "image/png")
				writer.WriteHeader(http.StatusOK)
				full := fixturePNG(64, 48)
				_, _ = writer.Write(full[:len(full)-40])
			},
		},
		{
			name: "empty_body", want: "image is empty",
			respond: func(_ int, writer http.ResponseWriter) {
				writer.Header().Set("Content-Type", "image/png")
				writer.WriteHeader(http.StatusOK)
			},
		},
		{
			name: "wrong_media_type", want: "returned media type",
			respond: func(_ int, writer http.ResponseWriter) {
				writer.Header().Set("Content-Type", "image/jpeg")
				writer.WriteHeader(http.StatusOK)
				_, _ = writer.Write([]byte{0xff, 0xd8, 0xff, 0xd9})
			},
		},
		{
			name: "oversized", want: "exceeds the 10485760 byte limit",
			respond: func(_ int, writer http.ResponseWriter) {
				writer.Header().Set("Content-Type", "image/png")
				writer.WriteHeader(http.StatusOK)
				_, _ = writer.Write(oversized)
			},
		},
		{
			name: "timeout", want: "timed out", timeout: "300ms",
			respond: func(_ int, writer http.ResponseWriter) {
				time.Sleep(3 * time.Second)
				respondPNG(16, 16)(0, writer)
			},
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			fake := newFakeBrowserRendering(t, testCase.respond)
			run := runScreenshotScenario(t, "shot_one", screenshotOptions{
				credentials: true, baseURL: fake.server.URL, timeout: testCase.timeout,
			})
			assertBlockedBeforeWriter(t, run, "capture runner exited with status 1")
			if _, err := os.Lstat(filepath.Join(run.runDir, "evidence", "assets", "screenshots")); err == nil {
				t.Errorf("a failed capture left partial screenshot assets behind")
			}
			state := readWorkflow(t, run.runDir)
			// The upstream error body is never copied into a durable artifact,
			// so its free text must be absent even before scrubbing applies.
			for _, upstream := range []string{"bad token", "not an image", "<html>"} {
				if strings.Contains(state.BlockReason, upstream) {
					t.Errorf("block reason echoed upstream response text %q: %q", upstream, state.BlockReason)
				}
			}
			assertNoCredentialLeak(t, run, readInvocationRecords(t, run.fixtureDir))
			assertProcessesGone(t, readInvocationRecords(t, run.fixtureDir))
		})
	}
}

func assertBlockedBeforeWriter(t *testing.T, run scenarioRun, wantReason string) {
	t.Helper()
	if run.err == nil {
		t.Fatalf("run succeeded but should have blocked: %s", run.output)
	}
	state := readWorkflow(t, run.runDir)
	if state.Status != "blocked" {
		t.Fatalf("status = %q, want blocked (%s)", state.Status, run.output)
	}
	if !strings.Contains(state.BlockReason, wantReason) {
		t.Fatalf("block reason %q does not contain %q", state.BlockReason, wantReason)
	}
	for _, forbidden := range []string{"drafts/article-001.md", "article.md", "outline.md", "evidence/screenshots.json"} {
		if _, err := os.Lstat(filepath.Join(run.runDir, forbidden)); err == nil {
			t.Errorf("%s exists although the run blocked before drafting", forbidden)
		}
	}
	for _, record := range readInvocationRecords(t, run.fixtureDir) {
		if record.Role == "writer" || record.Role == "story_editor" {
			t.Errorf("%s ran although the screenshot step should have blocked first", record.Role)
		}
	}
}

// assertNoCredentialLeak checks every surface the issue names: retained
// artifacts, prompts, process arguments, agent environments, and errors.
func assertNoCredentialLeak(t *testing.T, run scenarioRun, records []invocationRecord) {
	t.Helper()
	if strings.Contains(run.output, secretSentinel) {
		t.Errorf("controller output contained credential material")
	}
	_ = filepath.WalkDir(run.runDir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr == nil && bytes.Contains(data, []byte(secretSentinel)) {
			relative, _ := filepath.Rel(run.runDir, path)
			t.Errorf("retained artifact %s contained credential material", relative)
		}
		return nil
	})
	for _, record := range records {
		if strings.Contains(record.Prompt, secretSentinel) {
			t.Errorf("%s prompt contained credential material", record.Role)
		}
		if strings.Contains(strings.Join(record.Args, " "), secretSentinel) {
			t.Errorf("%s process arguments contained credential material", record.Role)
		}
		for name, value := range record.Environment {
			if strings.Contains(value, secretSentinel) {
				t.Errorf("%s environment variable %s contained credential material", record.Role, name)
			}
		}
		for _, name := range []string{"CLOUDFLARE_ACCOUNT_ID", "CLOUDFLARE_API_TOKEN"} {
			if record.Environment[name] != "ABSENT" {
				t.Errorf("%s agent environment carried %s (%s)", record.Role, name, record.Environment[name])
			}
		}
		if scan := record.Isolation["secret_scan"]; scan != "CLEAN" {
			t.Errorf("%s observed credential material: %s", record.Role, scan)
		}
	}
}

func assertNoScreenshotContext(t *testing.T, fixtureDir string) {
	t.Helper()
	for _, record := range readInvocationRecords(t, fixtureDir) {
		for _, name := range record.WorkspaceFiles {
			if strings.Contains(name, "screenshots") {
				t.Errorf("%s received %s for a run with no screenshot request", record.Role, name)
			}
		}
		if strings.Contains(record.Prompt, screenshotContextMarker) {
			t.Errorf("%s prompt was given a screenshot manifest that was never generated", record.Role)
		}
	}
}

func workspaceFileSet(record invocationRecord) map[string]bool {
	files := make(map[string]bool, len(record.WorkspaceFiles))
	for _, name := range record.WorkspaceFiles {
		files[name] = true
	}
	return files
}

func readScreenshotManifest(t *testing.T, runDir string) screenshotManifestFile {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(runDir, "evidence", "screenshots.json"))
	if err != nil {
		t.Fatalf("read generated manifest: %v", err)
	}
	var manifest screenshotManifestFile
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatalf("generated manifest is not valid JSON: %v", err)
	}
	return manifest
}

// screenshotContextMarker is how a generated manifest appears in a prompt as
// supplied context, distinct from a role prompt merely naming the artifact.
const screenshotContextMarker = `<write-uuter-context name="evidence/screenshots.json">`

// A non-loopback base URL override must fail the run: the redirected request
// would carry the API token to whatever host the ambient value named.
func TestBlackBoxScreenshotRejectsNonLoopbackEndpointOverride(t *testing.T) {
	fake := newFakeBrowserRendering(t, respondPNG(16, 16))
	run := runScreenshotScenario(t, "shot_one", screenshotOptions{
		credentials: true, baseURL: "https://attacker.example/client/v4",
	})
	assertBlockedBeforeWriter(t, run, "capture runner exited with status 1")
	if calls, _ := fake.recorded(); len(calls) != 0 {
		t.Fatalf("a redirected run still issued a capture: %+v", calls)
	}
	assertNoCredentialLeak(t, run, readInvocationRecords(t, run.fixtureDir))
}

func TestBlackBoxExternalCaptureRunnerProtocolAndInvalidOutputs(t *testing.T) {
	fakeRunner := filepath.Join(buildDir, "fake-capture-runner")
	// Populate buildDir before constructing the path above is used.
	_, _ = buildBinaries(t)
	fakeRunner = filepath.Join(buildDir, "fake-capture-runner")

	t.Run("fake_success", func(t *testing.T) {
		run := runScreenshotScenario(t, "shot_selector", screenshotOptions{runner: fakeRunner})
		if run.err != nil {
			t.Fatalf("fake external runner failed: %v\n%s", run.err, run.output)
		}
		manifest := readScreenshotManifest(t, run.runDir)
		if len(manifest.Screenshots) != 1 || manifest.Screenshots[0].Backend != "fake-backend" || manifest.Screenshots[0].FinalURL != "https://example.com/report" {
			t.Fatalf("external provenance was not retained: %+v", manifest)
		}
		assertNoCapturePrivateRoots(t, run.runDir)
	})

	invalid := []struct {
		scenario string
		want     string
		research string
	}{
		{"unknown-field", "unknown field", "shot_one"},
		{"duplicate-field", "duplicate JSON key", "shot_one"},
		{"omitted-full-page", "missing required field", "shot_one"},
		{"null-full-page", "must be a JSON boolean", "shot_one"},
		{"case-variant-full-page", "unknown field", "shot_one"},
		{"full-page-alias-collision", "unknown field", "shot_one"},
		{"wrong-version", "does not match protocol", "shot_one"},
		{"malformed-result", "invalid capture runner result.json", "shot_one"},
		{"missing-result-file", "did not produce a regular result.json", "shot_one"},
		{"missing-result", "returned 0 results for 1 requests", "shot_one"},
		{"duplicate-result", "returned 2 results for 1 requests", "shot_one"},
		{"mismatch-id", "request ID", "shot_one"},
		{"mismatch-url", "requested URL", "shot_one"},
		{"unsafe-final-url", "invalid final URL", "shot_one"},
		{"bad-media", "media type", "shot_one"},
		{"bad-backend", "backend identifier", "shot_one"},
		{"bad-timestamp", "invalid timestamp", "shot_one"},
		{"bad-viewport", "viewport dimensions", "shot_one"},
		{"mismatch-claims", "claim binding", "shot_one"},
		{"mismatch-rationale", "claim binding", "shot_one"},
		{"bad-trace", "exactly one", "shot_one"},
		{"traversal", "clean relative path", "shot_one"},
		{"absolute", "clean relative path", "shot_one"},
		{"unsafe-path", "clean relative path", "shot_one"},
		{"extra-file", "undeclared file", "shot_one"},
		{"extra-root-file", "undeclared file", "shot_one"},
		{"extra-directory", "undeclared directory", "shot_one"},
		{"missing-asset", "not a regular no-follow file", "shot_one"},
		{"symlink", "not a regular no-follow file", "shot_one"},
		{"special-file", "not a regular no-follow file", "shot_one"},
		{"invalid-png", "not a PNG", "shot_one"},
		{"oversized-png", "accepted byte size", "shot_one"},
		{"digest-mismatch", "digest does not match", "shot_one"},
		{"size-mismatch", "size or dimensions", "shot_one"},
		{"dimensions-mismatch", "size or dimensions", "shot_one"},
		{"duplicate-asset", "duplicate asset path", "shot_multi"},
		{"mutate-request", "modified or removed read-only request.json", "shot_one"},
	}
	for _, testCase := range invalid {
		t.Run(testCase.scenario, func(t *testing.T) {
			run := runScreenshotScenario(t, testCase.research, screenshotOptions{runner: fakeRunner, runnerScenario: testCase.scenario})
			assertBlockedBeforeWriter(t, run, testCase.want)
			assertNoCapturePrivateRoots(t, run.runDir)
		})
	}
}

func TestBlackBoxCapturedScreenshotPlacementBindsAcceptedRevision(t *testing.T) {
	_, _ = buildBinaries(t)
	fakeRunner := filepath.Join(buildDir, "fake-capture-runner")
	run := runScreenshotScenario(t, "shot_place", screenshotOptions{runner: fakeRunner})
	if run.err != nil {
		t.Fatalf("placed screenshot run failed: %v\n%s", run.err, run.output)
	}
	evidence, err := os.ReadFile(filepath.Join(run.runDir, "evidence/assets/screenshots/shot-001.png"))
	if err != nil {
		t.Fatal(err)
	}
	var inputs struct {
		Inputs []struct {
			ID, Origin, Source, SHA256 string
		} `json:"inputs"`
	}
	inputData, err := os.ReadFile(filepath.Join(run.runDir, "visual-inputs.json"))
	if err != nil || json.Unmarshal(inputData, &inputs) != nil {
		t.Fatalf("read screenshot visual-input binding: %v %s", err, inputData)
	}
	var input struct{ ID, Origin, Source, SHA256 string }
	for _, candidate := range inputs.Inputs {
		if candidate.ID == "shot-001" {
			input = candidate
		}
	}
	if input.ID != "shot-001" || input.Origin != "screenshot" || input.Source != "evidence/assets/screenshots/shot-001.png" || input.SHA256 != revision(evidence) {
		t.Fatalf("visual-inputs.json lost screenshot origin/digest binding: %+v", input)
	}
	article, err := os.ReadFile(filepath.Join(run.runDir, "article.md"))
	if err != nil {
		t.Fatal(err)
	}
	placedPath := "visuals/article-001/assets/shot-001.png"
	if !strings.Contains(string(article), "]("+placedPath+")") {
		t.Fatalf("accepted article did not embed captured screenshot %s:\n%s", placedPath, article)
	}
	manifest := readVisualManifest(t, run.runDir, 1)
	if len(manifest.Assets) != 1 || manifest.Assets[0].Origin != "screenshot" || manifest.Assets[0].Source != input.Source || manifest.Assets[0].SHA256 != input.SHA256 {
		t.Fatalf("visual manifest lost placed screenshot provenance: %+v", manifest.Assets)
	}
	wantRevision := candidateRevisionFromDisk(t, run.runDir, manifest)
	state := readWorkflow(t, run.runDir)
	if state.CurrentRevision != wantRevision || manifest.ReviewedRevision != wantRevision {
		t.Fatalf("accepted revision is not bound to placed screenshot: workflow=%s manifest=%s want=%s", state.CurrentRevision, manifest.ReviewedRevision, wantRevision)
	}
	assertReviewsBoundTo(t, run.runDir, 1, wantRevision)
	sawContentGate := false
	for _, invocation := range readInvocationRecords(t, run.fixtureDir) {
		if invocation.Role == "visual_editor" && strings.Contains(invocation.Prompt, "visible-content validation") &&
			strings.Contains(invocation.Prompt, "claim-004") && strings.Contains(invocation.Prompt, "generic errors") {
			sawContentGate = true
		}
	}
	if !sawContentGate {
		t.Fatal("Visual Editor did not receive the screenshot usability gate with claim/request context")
	}
}

func TestBlackBoxUnusableScreenshotIsExplicitlyNotPlaced(t *testing.T) {
	_, _ = buildBinaries(t)
	run := runScreenshotScenario(t, "shot_unusable", screenshotOptions{
		runner: filepath.Join(buildDir, "fake-capture-runner"), runnerScenario: "unusable-success",
	})
	if run.err != nil {
		t.Fatalf("explicit non-placement run failed: %v\n%s", run.err, run.output)
	}
	article, err := os.ReadFile(filepath.Join(run.runDir, "article.md"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(article), "shot-001.png") {
		t.Fatalf("unusable screenshot was silently placed:\n%s", article)
	}
	plan, err := os.ReadFile(filepath.Join(run.runDir, "visuals/article-001/plan.md"))
	if err != nil || !bytes.Contains(plan, []byte("Explicit non-placement")) || !bytes.Contains(plan, []byte("unrelated")) {
		t.Fatalf("unusable capture lacks an explicit durable non-placement: %v\n%s", err, plan)
	}
	manifest := readVisualManifest(t, run.runDir, 1)
	if len(manifest.Assets) != 0 {
		t.Fatalf("unusable screenshot entered the accepted revision: %+v", manifest.Assets)
	}
	screenshots := readScreenshotManifest(t, run.runDir)
	if len(screenshots.Screenshots) != 1 || screenshots.Screenshots[0].Attempt != 2 ||
		screenshots.Screenshots[0].EditorialOutcome == nil || screenshots.Screenshots[0].EditorialOutcome.Status != "rejected" ||
		len(screenshots.Screenshots[0].PriorAttempts) != 1 || screenshots.Screenshots[0].PriorAttempts[0].EditorialOutcome == nil {
		t.Fatalf("retry exhaustion lacks request-keyed durable non-placement: %+v", screenshots.Screenshots)
	}
}

type captureInvocationLog struct {
	Workspace string                          `json:"workspace"`
	Request   captureprotocol.RequestDocument `json:"request"`
}

func readCaptureInvocationLog(t *testing.T, path string) []captureInvocationLog {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var records []captureInvocationLog
	for _, line := range bytes.Split(bytes.TrimSpace(data), []byte{'\n'}) {
		var record captureInvocationLog
		if err := json.Unmarshal(line, &record); err != nil {
			t.Fatalf("decode capture invocation log: %v", err)
		}
		records = append(records, record)
	}
	return records
}

func TestBlackBoxEditorialRejectionRetriesOnceWithNeutralPriorProvenance(t *testing.T) {
	_, _ = buildBinaries(t)
	logPath := filepath.Join(t.TempDir(), "capture-invocations.jsonl")
	run := runScreenshotScenario(t, "shot_retry_success", screenshotOptions{
		runner: filepath.Join(buildDir, "fake-capture-runner"), runnerScenario: "retry-success", runnerLog: logPath,
	})
	if run.err != nil {
		t.Fatalf("retry-success run failed: %v\n%s", run.err, run.output)
	}
	invocations := readCaptureInvocationLog(t, logPath)
	if len(invocations) != 2 {
		t.Fatalf("capture invocations = %d, want exactly 2", len(invocations))
	}
	if invocations[0].Workspace == invocations[1].Workspace {
		t.Fatalf("retry reused private workspace %q", invocations[0].Workspace)
	}
	for _, invocation := range invocations {
		if _, err := os.Lstat(invocation.Workspace); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("private workspace survived runner exit: %s (%v)", invocation.Workspace, err)
		}
	}
	first := invocations[0].Request.Requests[0]
	second := invocations[1].Request.Requests[0]
	if first.PriorAttempt != nil || second.PriorAttempt == nil {
		t.Fatalf("prior-attempt handoff = first:%+v second:%+v", first.PriorAttempt, second.PriorAttempt)
	}
	prior := second.PriorAttempt
	if prior.RequestID != "shot-001" || prior.Attempt != 1 || prior.EditorialOutcome.RequestID != "shot-001" ||
		prior.EditorialOutcome.Status != "rejected" || strings.TrimSpace(prior.EditorialOutcome.Reason) == "" ||
		prior.Backend != "fake-backend" || prior.SHA256 == "" {
		t.Fatalf("retry lost request-keyed rejection/provenance: %+v", prior)
	}
	encodedSecond, _ := json.Marshal(second)
	for _, backendDirective := range []string{"preferred_backend", "fallback_backend", "cloudflare"} {
		if bytes.Contains(bytes.ToLower(encodedSecond), []byte(backendDirective)) {
			t.Fatalf("retry request leaked backend selection directive %q: %s", backendDirective, encodedSecond)
		}
	}
	manifest := readScreenshotManifest(t, run.runDir)
	record := manifest.Screenshots[0]
	if record.Attempt != 2 || record.Backend != "fake-backend-second-attempt" || record.EditorialOutcome == nil || record.EditorialOutcome.Status != "usable" || len(record.PriorAttempts) != 1 {
		t.Fatalf("retry-success durable lifecycle is incomplete: %+v", record)
	}
	firstBytes, err := os.ReadFile(filepath.Join(run.runDir, record.PriorAttempts[0].Path))
	if err != nil {
		t.Fatal(err)
	}
	secondBytes, err := os.ReadFile(filepath.Join(run.runDir, record.Path))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(firstBytes, secondBytes) || revision(firstBytes) != record.PriorAttempts[0].SHA256 || revision(secondBytes) != record.SHA256 {
		t.Fatalf("attempt bytes/provenance are not independently retained")
	}
	visual := readVisualManifest(t, run.runDir, 1)
	if len(visual.Assets) != 1 || visual.Assets[0].SHA256 != record.SHA256 || visual.Assets[0].SHA256 == record.PriorAttempts[0].SHA256 {
		t.Fatalf("accepted revision placed rejected pixels or lost replacement binding: %+v", visual.Assets)
	}
	visualEditors := 0
	for _, invocation := range readInvocationRecords(t, run.fixtureDir) {
		if invocation.Role == "visual_editor" {
			visualEditors++
		}
	}
	if visualEditors != 2 {
		t.Fatalf("editorial evaluations = %d, want exactly 2", visualEditors)
	}
	assertNoCapturePrivateRoots(t, run.runDir)
}

func TestBlackBoxEditorialRetryExhaustionStopsWithoutPlacementOrLoop(t *testing.T) {
	_, _ = buildBinaries(t)
	logPath := filepath.Join(t.TempDir(), "capture-invocations.jsonl")
	run := runScreenshotScenario(t, "shot_retry_exhaust", screenshotOptions{
		runner: filepath.Join(buildDir, "fake-capture-runner"), runnerScenario: "retry-exhaust", runnerLog: logPath,
	})
	if run.err != nil {
		t.Fatalf("retry-exhaustion run failed: %v\n%s", run.err, run.output)
	}
	if invocations := readCaptureInvocationLog(t, logPath); len(invocations) != 2 {
		t.Fatalf("capture retry loop count = %d, want exactly 2", len(invocations))
	}
	record := readScreenshotManifest(t, run.runDir).Screenshots[0]
	if record.Attempt != 2 || record.EditorialOutcome == nil || record.EditorialOutcome.Status != "rejected" || len(record.PriorAttempts) != 1 {
		t.Fatalf("exhaustion lacks explicit durable non-placement: %+v", record)
	}
	article, err := os.ReadFile(filepath.Join(run.runDir, "article.md"))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(article, []byte("shot-001")) || len(readVisualManifest(t, run.runDir, 1).Assets) != 0 {
		t.Fatalf("rejected capture entered accepted article revision:\n%s", article)
	}
	visualEditors := 0
	for _, invocation := range readInvocationRecords(t, run.fixtureDir) {
		if invocation.Role == "visual_editor" {
			visualEditors++
		}
	}
	if visualEditors != 2 {
		t.Fatalf("editorial evaluations = %d, want exactly 2", visualEditors)
	}
	assertNoCapturePrivateRoots(t, run.runDir)
}

func TestBlackBoxRetryExhaustionRemainsTerminalAcrossCandidates(t *testing.T) {
	_, _ = buildBinaries(t)
	logPath := filepath.Join(t.TempDir(), "capture-invocations.jsonl")
	run := runScreenshotScenario(t, "shot_retry_terminal_later_place", screenshotOptions{
		runner: filepath.Join(buildDir, "fake-capture-runner"), runnerScenario: "retry-exhaust", runnerLog: logPath,
	})
	if run.err == nil || !strings.Contains(run.output, "asset \"shot-001\" is not a controller-staged visual input") {
		t.Fatalf("later candidate was not blocked from reviving terminal pixels: %v\n%s", run.err, run.output)
	}
	if invocations := readCaptureInvocationLog(t, logPath); len(invocations) != 2 {
		t.Fatalf("terminal request capture count = %d, want exactly 2", len(invocations))
	}
	record := readScreenshotManifest(t, run.runDir).Screenshots[0]
	if record.Attempt != 2 || record.EditorialOutcome == nil || record.EditorialOutcome.Status != "rejected" ||
		!strings.Contains(record.EditorialOutcome.Reason, "unusable or unrelated") || len(record.PriorAttempts) != 1 {
		t.Fatalf("later candidate changed durable exhaustion: %+v", record)
	}
	if _, err := os.Stat(filepath.Join(run.runDir, "article.md")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("twice-rejected pixels reached a published article: %v", err)
	}
	if _, err := os.Stat(filepath.Join(run.runDir, "visuals/article-002/assets/shot-001.png")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("later candidate placed terminal screenshot: %v", err)
	}
	visualEditors := 0
	laterAttempted := false
	for _, invocation := range readInvocationRecords(t, run.fixtureDir) {
		if invocation.Role == "visual_editor" {
			visualEditors++
			laterAttempted = laterAttempted || invocation.Candidate == 2
		}
	}
	if visualEditors != 3 || !laterAttempted {
		t.Fatalf("visual-editor lifecycle = %d calls, later attempted=%v; want two evaluations then one blocked later candidate", visualEditors, laterAttempted)
	}
	assertNoCapturePrivateRoots(t, run.runDir)
}

func TestBlackBoxCompliantLaterCandidateCompletesAfterRetryExhaustion(t *testing.T) {
	_, _ = buildBinaries(t)
	logPath := filepath.Join(t.TempDir(), "capture-invocations.jsonl")
	run := runScreenshotScenario(t, "shot_retry_terminal_later_compliant", screenshotOptions{
		runner: filepath.Join(buildDir, "fake-capture-runner"), runnerScenario: "retry-exhaust", runnerLog: logPath,
	})
	if run.err != nil {
		t.Fatalf("compliant later candidate failed after terminal exhaustion: %v\n%s", run.err, run.output)
	}
	if invocations := readCaptureInvocationLog(t, logPath); len(invocations) != 2 {
		t.Fatalf("terminal request capture count = %d, want exactly 2", len(invocations))
	}
	record := readScreenshotManifest(t, run.runDir).Screenshots[0]
	if record.Attempt != 2 || record.EditorialOutcome == nil || record.EditorialOutcome.Status != "rejected" || len(record.PriorAttempts) != 1 {
		t.Fatalf("successful later candidate changed durable exhaustion: %+v", record)
	}
	article, err := os.ReadFile(filepath.Join(run.runDir, "article.md"))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(article, []byte("shot-001")) || len(readVisualManifest(t, run.runDir, 2).Assets) != 0 {
		t.Fatalf("terminal screenshot entered the accepted later candidate: %s", article)
	}
	visualEditors := 0
	compliantLaterEditor := false
	for _, invocation := range readInvocationRecords(t, run.fixtureDir) {
		if invocation.Role != "visual_editor" {
			continue
		}
		visualEditors++
		if invocation.Candidate == 2 {
			compliantLaterEditor = strings.Contains(invocation.Prompt, "origin is `screenshot`") &&
				strings.Contains(invocation.Prompt, "terminal non-placement")
		}
	}
	if visualEditors != 3 || !compliantLaterEditor {
		t.Fatalf("later editor did not receive/follow the staged-only contract: calls=%d compliant=%v", visualEditors, compliantLaterEditor)
	}
	assertNoCapturePrivateRoots(t, run.runDir)
}

func TestBlackBoxRetryAssetNamespaceCannotOverwriteAnotherRequest(t *testing.T) {
	_, _ = buildBinaries(t)
	logPath := filepath.Join(t.TempDir(), "capture-invocations.jsonl")
	run := runScreenshotScenario(t, "shot_retry_path_collision", screenshotOptions{
		runner: filepath.Join(buildDir, "fake-capture-runner"), runnerScenario: "path-collision", runnerLog: logPath,
	})
	if run.err != nil {
		t.Fatalf("collision regression run failed: %v\n%s", run.err, run.output)
	}
	invocations := readCaptureInvocationLog(t, logPath)
	if len(invocations) != 2 || len(invocations[0].Request.Requests) != 2 || len(invocations[1].Request.Requests) != 1 ||
		invocations[1].Request.Requests[0].RequestID != "shot-001" {
		t.Fatalf("unexpected request-keyed retry lifecycle: %+v", invocations)
	}
	manifest := readScreenshotManifest(t, run.runDir)
	if len(manifest.Screenshots) != 2 {
		t.Fatalf("screenshot records = %d, want 2", len(manifest.Screenshots))
	}
	first, other := manifest.Screenshots[0], manifest.Screenshots[1]
	if first.ID != "shot-001" || first.Attempt != 2 || first.Path != "evidence/assets/screenshots/attempts/shot-001/attempt-002.png" ||
		other.ID != "shot-001-attempt-002" || other.Attempt != 1 || other.Path != "evidence/assets/screenshots/shot-001-attempt-002.png" ||
		first.Path == other.Path {
		t.Fatalf("attempt paths are not request-disjoint: first=%+v other=%+v", first, other)
	}
	firstBytes, err := os.ReadFile(filepath.Join(run.runDir, first.Path))
	if err != nil {
		t.Fatal(err)
	}
	otherBytes, err := os.ReadFile(filepath.Join(run.runDir, other.Path))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(firstBytes, otherBytes) || revision(firstBytes) != first.SHA256 || revision(otherBytes) != other.SHA256 {
		t.Fatalf("retry overwrote or rebound another request's accepted evidence")
	}
	if first.EditorialOutcome == nil || first.EditorialOutcome.Status != "usable" || other.EditorialOutcome == nil || other.EditorialOutcome.Status != "usable" {
		t.Fatalf("both request-keyed records were not evaluated after retry: first=%+v other=%+v", first.EditorialOutcome, other.EditorialOutcome)
	}
	assertNoCapturePrivateRoots(t, run.runDir)
}

func TestBlackBoxExternalCaptureRunnerFailuresAndCleanup(t *testing.T) {
	_, _ = buildBinaries(t)
	fakeRunner := filepath.Join(buildDir, "fake-capture-runner")

	t.Run("missing_runner", func(t *testing.T) {
		run := runScreenshotScenario(t, "shot_one", screenshotOptions{omitRunner: true})
		assertBlockedBeforeWriter(t, run, "WRITE_UUTER_CAPTURE_RUNNER")
		assertNoCapturePrivateRoots(t, run.runDir)
	})
	for _, scenario := range []string{"nonzero", "partial"} {
		t.Run(scenario, func(t *testing.T) {
			t.Setenv("WRITE_UUTER_CAPTURE_SECRET", secretSentinel+"runner")
			run := runScreenshotScenario(t, "shot_one", screenshotOptions{runner: fakeRunner, runnerScenario: scenario})
			assertBlockedBeforeWriter(t, run, "capture runner exited with status")
			if strings.Contains(run.output, secretSentinel) || strings.Contains(readWorkflow(t, run.runDir).BlockReason, secretSentinel) {
				t.Fatal("runner output crossed the credential-safe diagnostic boundary")
			}
			assertNoCapturePrivateRoots(t, run.runDir)
		})
	}
	t.Run("timeout", func(t *testing.T) {
		run := runScreenshotScenario(t, "shot_one", screenshotOptions{runner: fakeRunner, runnerScenario: "timeout", runnerTimeout: "200ms"})
		assertBlockedBeforeWriter(t, run, "capture runner timed out")
		assertNoCapturePrivateRoots(t, run.runDir)
	})
	t.Run("inner_deadline_closes_remote_session_before_outer_cleanup", func(t *testing.T) {
		fake := newFakeBrowserRendering(t, respondPNG(64, 48))
		fake.hangMethod = "Page.navigate"
		run := runScreenshotScenario(t, "shot_one", screenshotOptions{
			credentials: true, baseURL: fake.server.URL, timeout: "3s",
			runnerTimeout: "3s", runTimeout: "7s",
		})
		assertBlockedBeforeWriter(t, run, "capture runner")
		active, deleted := fake.cleanupState()
		if active != 0 || deleted != 1 {
			t.Fatalf("remote cleanup state after equal outer/adapter timeout: active=%d deleted=%d", active, deleted)
		}
		assertNoCapturePrivateRoots(t, run.runDir)
	})
	t.Run("descriptor_holding_descendant", func(t *testing.T) {
		marker := filepath.Join(t.TempDir(), "descendant.pid")
		t.Setenv("WRITE_UUTER_TEST_CAPTURE_CHILD_PID", marker)
		started := time.Now()
		run := runScreenshotScenario(t, "shot_one", screenshotOptions{
			runner: fakeRunner, runnerScenario: "descriptor-descendant", runnerTimeout: "2s", runTimeout: "8s",
		})
		elapsed := time.Since(started)
		assertBlockedBeforeWriter(t, run, "capture runner timed out")
		if elapsed >= 4*time.Second {
			t.Fatalf("capture deadline was not bounded: elapsed %s for a 2s deadline", elapsed)
		}
		pidData, err := os.ReadFile(marker)
		if err != nil {
			t.Fatalf("descriptor-holding descendant was not observed: %v", err)
		}
		pid, err := strconv.Atoi(strings.TrimSpace(string(pidData)))
		if err != nil {
			t.Fatalf("invalid descendant pid %q: %v", pidData, err)
		}
		assertPIDGone(t, pid)
		assertNoCapturePrivateRoots(t, run.runDir)
	})
	t.Run("post_acceptance_replacement", func(t *testing.T) {
		marker := filepath.Join(t.TempDir(), "child.pid")
		t.Setenv("WRITE_UUTER_TEST_CAPTURE_CHILD_PID", marker)
		run := runScreenshotScenario(t, "shot_place", screenshotOptions{runner: fakeRunner, runnerScenario: "replace-after-exit"})
		if run.err != nil {
			t.Fatalf("run failed: %v\n%s", run.err, run.output)
		}
		data, err := os.ReadFile(filepath.Join(run.runDir, "evidence/assets/screenshots/shot-001.png"))
		if err != nil || bytes.Equal(data, []byte("replacement")) {
			t.Fatalf("accepted evidence was replaced: %v", err)
		}
		if _, err := png.Decode(bytes.NewReader(data)); err != nil {
			t.Fatalf("accepted evidence is no longer the validated PNG: %v", err)
		}
		articlePath := filepath.Join(run.runDir, "article.md")
		articleBefore, err := os.ReadFile(articlePath)
		if err != nil {
			t.Fatal(err)
		}
		manifestBefore := readVisualManifest(t, run.runDir, 1)
		if len(manifestBefore.Assets) != 1 || manifestBefore.Assets[0].SHA256 != revision(data) {
			t.Fatalf("placed revision was not bound to accepted evidence: %+v", manifestBefore.Assets)
		}
		revisionBefore := readWorkflow(t, run.runDir).CurrentRevision
		pidData, err := os.ReadFile(marker)
		if err != nil {
			t.Fatalf("replacement child was not observed: %v", err)
		}
		pid, err := strconv.Atoi(strings.TrimSpace(string(pidData)))
		if err != nil {
			t.Fatal(err)
		}
		if err := syscall.Kill(pid, 0); err != syscall.ESRCH {
			t.Fatalf("capture runner child %d survived acceptance cleanup: %v", pid, err)
		}
		time.Sleep(600 * time.Millisecond)
		evidenceAfter, evidenceErr := os.ReadFile(filepath.Join(run.runDir, "evidence/assets/screenshots/shot-001.png"))
		articleAfter, articleErr := os.ReadFile(articlePath)
		if evidenceErr != nil || articleErr != nil || !bytes.Equal(evidenceAfter, data) || !bytes.Equal(articleAfter, articleBefore) {
			t.Fatalf("post-exit mutation changed durable evidence or article: evidence=%v article=%v", evidenceErr, articleErr)
		}
		manifestAfter := readVisualManifest(t, run.runDir, 1)
		if !reflect.DeepEqual(manifestAfter, manifestBefore) || readWorkflow(t, run.runDir).CurrentRevision != revisionBefore {
			t.Fatalf("post-exit mutation changed accepted placed revision")
		}
		assertNoCapturePrivateRoots(t, run.runDir)
	})
}

func TestBlackBoxFastCaptureRunnerExitReachesNormalResultAndStatusHandling(t *testing.T) {
	_, _ = buildBinaries(t)
	fakeRunner := filepath.Join(buildDir, "fake-capture-runner")

	t.Run("valid_result", func(t *testing.T) {
		for attempt := 0; attempt < 5; attempt++ {
			run := runScreenshotScenario(t, "shot_one", screenshotOptions{
				runner: fakeRunner, runnerScenario: "fast-success", trackerDelay: "20ms",
			})
			if run.err != nil {
				t.Fatalf("fast valid runner attempt %d failed: %v\n%s", attempt+1, run.err, run.output)
			}
			manifest := readScreenshotManifest(t, run.runDir)
			if len(manifest.Screenshots) != 1 || manifest.Screenshots[0].Backend != "fake-backend" {
				t.Fatalf("fast valid runner attempt %d did not reach result validation: %+v", attempt+1, manifest)
			}
			assertNoCapturePrivateRoots(t, run.runDir)
		}
	})

	t.Run("nonzero_status", func(t *testing.T) {
		t.Setenv("WRITE_UUTER_CAPTURE_SECRET", secretSentinel+"fast-runner")
		for attempt := 0; attempt < 5; attempt++ {
			run := runScreenshotScenario(t, "shot_one", screenshotOptions{
				runner: fakeRunner, runnerScenario: "nonzero", trackerDelay: "20ms",
			})
			assertBlockedBeforeWriter(t, run, "capture runner exited with status 7")
			if strings.Contains(run.output, "process ownership") || strings.Contains(run.output, secretSentinel) ||
				strings.Contains(readWorkflow(t, run.runDir).BlockReason, secretSentinel) {
				t.Fatalf("fast nonzero attempt %d bypassed sanitized status handling: %s", attempt+1, run.output)
			}
			assertNoCapturePrivateRoots(t, run.runDir)
		}
	})
}

func TestBlackBoxFastCaptureRunnerDetachedDescendantsAreAlwaysRemoved(t *testing.T) {
	_, _ = buildBinaries(t)
	fakeRunner := filepath.Join(buildDir, "fake-capture-runner")

	for _, testCase := range []struct {
		name           string
		runnerScenario string
		wantStatus     string
	}{
		{name: "valid_result", runnerScenario: "fast-detached-success"},
		{name: "nonzero_status", runnerScenario: "fast-detached-nonzero", wantStatus: "capture runner exited with status 7"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			for attempt := 0; attempt < 5; attempt++ {
				marker := filepath.Join(t.TempDir(), fmt.Sprintf("detached-%d.pid", attempt))
				t.Setenv("WRITE_UUTER_TEST_CAPTURE_CHILD_PID", marker)
				run := runScreenshotScenario(t, "shot_one", screenshotOptions{
					runner: fakeRunner, runnerScenario: testCase.runnerScenario, trackerDelay: "20ms",
				})
				if testCase.wantStatus == "" {
					if run.err != nil {
						t.Fatalf("fast detached success attempt %d failed: %v\n%s", attempt+1, run.err, run.output)
					}
					if manifest := readScreenshotManifest(t, run.runDir); len(manifest.Screenshots) != 1 || manifest.Screenshots[0].Backend != "fake-backend" {
						t.Fatalf("fast detached success attempt %d lost its valid result: %+v", attempt+1, manifest)
					}
				} else {
					assertBlockedBeforeWriter(t, run, testCase.wantStatus)
				}
				pidData, err := os.ReadFile(marker)
				if err != nil {
					t.Fatalf("fast detached attempt %d did not record a descendant: %v", attempt+1, err)
				}
				pid, err := strconv.Atoi(strings.TrimSpace(string(pidData)))
				if err != nil {
					t.Fatalf("fast detached attempt %d recorded invalid PID %q: %v", attempt+1, pidData, err)
				}
				assertPIDGone(t, pid)
				assertNoCapturePrivateRoots(t, run.runDir)
			}
		})
	}
}

func assertNoCapturePrivateRoots(t *testing.T, runDir string) {
	t.Helper()
	paths, err := filepath.Glob(filepath.Join(filepath.Dir(runDir), ".write-uuter-capture-*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) != 0 {
		t.Fatalf("capture runner private workspace survived cleanup: %v", paths)
	}
}

// pngSignatureBytes is the PNG magic number. A prompt must never contain it:
// images are staged as files, never inlined into an agent assignment.
var pngSignatureBytes = []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}

// Browser execution must remain outside the controller. The controller may
// launch only the configured external runner; no local browser engine, driver,
// per-agent browser, or MCP server may be introduced into article workflow
// code. Provider-specific transport stays inside the external adapter.
func TestScreenshotSliceReferencesNoLocalBrowserEngine(t *testing.T) {
	// "cloudflare-chromium" is recorded external-backend provenance and is
	// expected; a controller-owned driver or browser binary is not.
	forbidden := []string{"playwright", "puppeteer", "chromedriver", "chrome-headless",
		"exec.command(\"chrom", "/applications/google chrome", "headless_shell"}
	_ = filepath.WalkDir(filepath.Join(repositoryRoot(t), "internal"), func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil
		}
		lowered := strings.ToLower(string(data))
		for _, marker := range forbidden {
			if strings.Contains(lowered, marker) {
				t.Errorf("%s references a local browser engine or MCP server (%q)", path, marker)
			}
		}
		return nil
	})
}
