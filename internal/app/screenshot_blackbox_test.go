package app_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// The fixture credentials carry a sentinel prefix the fake agent also knows,
// so a leak into any observable surface is detected rather than assumed away.
const (
	fixtureAccountID = "write-uuter-fixture-secret-account-a1"
	fixtureAPIToken  = "write-uuter-fixture-secret-token-b2"
	secretSentinel   = "write-uuter-fixture-secret-"
)

type screenshotManifestFile struct {
	SchemaVersion int    `json:"schema_version"`
	Engine        string `json:"engine"`
	Viewport      struct {
		Width  int `json:"width"`
		Height int `json:"height"`
	} `json:"viewport"`
	Screenshots []struct {
		ID           string   `json:"id"`
		Path         string   `json:"path"`
		RequestedURL string   `json:"requested_url"`
		Selector     string   `json:"selector"`
		CapturedAt   string   `json:"captured_at"`
		Supports     []string `json:"supports"`
		Reason       string   `json:"reason"`
		Engine       string   `json:"engine"`
		MediaType    string   `json:"media_type"`
		ByteSize     int      `json:"byte_size"`
		Width        int      `json:"width"`
		Height       int      `json:"height"`
		SHA256       string   `json:"sha256"`
	} `json:"screenshots"`
}

type capturedCall struct {
	Method   string
	Path     string
	AuthHead string
	Body     map[string]any
}

// fakeBrowserRendering stands in for the Cloudflare Chromium quick action. It
// records every call so the test can prove the exact endpoint, header, body,
// and ordering the controller used.
type fakeBrowserRendering struct {
	server *httptest.Server

	mutex       sync.Mutex
	calls       []capturedCall
	inFlight    int
	maxInFlight int

	respond func(index int, writer http.ResponseWriter)
}

func newFakeBrowserRendering(t *testing.T, respond func(index int, writer http.ResponseWriter)) *fakeBrowserRendering {
	t.Helper()
	fake := &fakeBrowserRendering{respond: respond}
	fake.server = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		body, _ := io.ReadAll(io.LimitReader(request.Body, 1<<20))
		decoded := map[string]any{}
		_ = json.Unmarshal(body, &decoded)
		fake.mutex.Lock()
		index := len(fake.calls)
		fake.calls = append(fake.calls, capturedCall{
			Method: request.Method, Path: request.URL.Path,
			AuthHead: request.Header.Get("Authorization"), Body: decoded,
		})
		fake.inFlight++
		if fake.inFlight > fake.maxInFlight {
			fake.maxInFlight = fake.inFlight
		}
		fake.mutex.Unlock()
		defer func() {
			fake.mutex.Lock()
			fake.inFlight--
			fake.mutex.Unlock()
		}()
		fake.respond(index, writer)
	}))
	t.Cleanup(fake.server.Close)
	return fake
}

func (fake *fakeBrowserRendering) recorded() ([]capturedCall, int) {
	fake.mutex.Lock()
	defer fake.mutex.Unlock()
	return append([]capturedCall(nil), fake.calls...), fake.maxInFlight
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
	credentials bool
	baseURL     string
	timeout     string
	runTimeout  string
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
		"WRITE_UUTER_TEST_BROWSER_RENDERING_BASE_URL", "WRITE_UUTER_TEST_SCREENSHOT_TIMEOUT")
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
			run := runScreenshotScenario(t, scenario, screenshotOptions{baseURL: fake.server.URL})
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
	if call.Method != http.MethodPost {
		t.Errorf("method = %q, want POST", call.Method)
	}
	if want := "/accounts/" + fixtureAccountID + "/browser-rendering/screenshot"; call.Path != want {
		t.Errorf("path = %q, want %q", call.Path, want)
	}
	if call.AuthHead != "Bearer "+fixtureAPIToken {
		t.Errorf("authorization header was not the controller-owned bearer token")
	}
	if call.Body["url"] != "https://example.com/report" || call.Body["selector"] != "main" {
		t.Errorf("unexpected request body: %+v", call.Body)
	}
	viewport, _ := call.Body["viewport"].(map[string]any)
	if viewport == nil || viewport["width"] != float64(1280) || viewport["height"] != float64(800) {
		t.Errorf("viewport is not the fixed documented viewport: %+v", call.Body["viewport"])
	}
	options, _ := call.Body["screenshotOptions"].(map[string]any)
	if options == nil || options["type"] != "png" || options["fullPage"] != false {
		t.Errorf("screenshot options are not the documented PNG viewport capture: %+v", call.Body["screenshotOptions"])
	}
	for _, forbidden := range []string{"cookies", "authenticate", "setExtraHTTPHeaders", "addScriptTag", "addStyleTag", "html", "userAgent", "waitForSelector"} {
		if _, present := call.Body[forbidden]; present {
			t.Errorf("request carried unsupported browser action %q", forbidden)
		}
	}
	if maxInFlight != 1 {
		t.Errorf("captures were concurrent: max in flight = %d", maxInFlight)
	}

	manifest := readScreenshotManifest(t, run.runDir)
	if manifest.SchemaVersion != 1 || manifest.Engine != "cloudflare-chromium" {
		t.Errorf("unexpected manifest header: %+v", manifest)
	}
	if manifest.Viewport.Width != 1280 || manifest.Viewport.Height != 800 {
		t.Errorf("manifest viewport = %+v", manifest.Viewport)
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
	if record.RequestedURL != "https://example.com/report" || record.Selector != "main" {
		t.Errorf("record lost the requested URL/selector: %+v", record)
	}
	if len(record.Supports) != 1 || record.Supports[0] != "claim-004" || strings.TrimSpace(record.Reason) == "" {
		t.Errorf("record lost claim linkage or rationale: %+v", record)
	}
	if record.Engine != "cloudflare-chromium" || record.MediaType != "image/png" {
		t.Errorf("record lost provenance: %+v", record)
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
			if calls[index].Body["url"] != want {
				t.Errorf("call %d requested %v, want %q", index, calls[index].Body["url"], want)
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
func TestBlackBoxScreenshotMissingCredentialsBlockOnlyRequestedCaptures(t *testing.T) {
	fake := newFakeBrowserRendering(t, respondPNG(16, 16))
	run := runScreenshotScenario(t, "shot_one", screenshotOptions{baseURL: fake.server.URL})
	assertBlockedBeforeWriter(t, run, "CLOUDFLARE_ACCOUNT_ID")
	state := readWorkflow(t, run.runDir)
	if !strings.Contains(state.BlockReason, "CLOUDFLARE_API_TOKEN") {
		t.Errorf("block reason does not name the missing token variable: %q", state.BlockReason)
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
				_, _ = writer.Write(fixturePNG(16, 16))
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
			assertBlockedBeforeWriter(t, run, testCase.want)
			if _, err := os.Lstat(filepath.Join(run.runDir, "evidence", "assets", "screenshots")); err == nil {
				t.Errorf("a failed capture left partial screenshot assets behind")
			}
			state := readWorkflow(t, run.runDir)
			if !strings.Contains(state.BlockReason, "https://example.com/report") {
				t.Errorf("block reason does not name the requested page: %q", state.BlockReason)
			}
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
	assertBlockedBeforeWriter(t, run, "must address a loopback host")
	if calls, _ := fake.recorded(); len(calls) != 0 {
		t.Fatalf("a redirected run still issued a capture: %+v", calls)
	}
	assertNoCredentialLeak(t, run, readInvocationRecords(t, run.fixtureDir))
}

// pngSignatureBytes is the PNG magic number. A prompt must never contain it:
// images are staged as files, never inlined into an agent assignment.
var pngSignatureBytes = []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}

// The capture path must remain a remote HTTP call. No local browser engine or
// driver may be introduced behind it. (An MCP server cannot appear either: the
// controller launches no process for a capture, which the failure-path tests
// above observe directly.)
func TestScreenshotSliceReferencesNoLocalBrowserEngine(t *testing.T) {
	// "cloudflare-chromium" is the remote engine name and is expected; a local
	// driver or a locally launched browser binary is not.
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
