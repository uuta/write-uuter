package app

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const testClaimLedger = "# Claim ledger\n\n- Fact (claim-004): a public page shows it.\n- Fact (claim-0041): a different claim.\n- Firsthand observation: none.\n- Inference: labeled.\n- Opinion: labeled.\n- Unresolved: none.\n"

func testPNG(t *testing.T, width, height int) []byte {
	t.Helper()
	canvas := image.NewRGBA(image.Rect(0, 0, width, height))
	canvas.Set(0, 0, color.RGBA{R: 1, G: 2, B: 3, A: 255})
	var buffer bytes.Buffer
	if err := png.Encode(&buffer, canvas); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

func TestScreenshotAttemptPathsCannotCollideWithRequestAssetPaths(t *testing.T) {
	retry := screenshotAttemptAssetPath("shot-001", 2)
	otherRequest := screenshotAttemptAssetPath("shot-001-attempt-002", 1)
	if retry == otherRequest {
		t.Fatalf("derived retry path %q collides with another request asset", retry)
	}
	if retry != "evidence/assets/screenshots/attempts/shot-001/attempt-002.png" {
		t.Fatalf("retry path = %q, want reserved attempt namespace", retry)
	}
}

func TestTerminalScreenshotRejectionCannotBeOverwritten(t *testing.T) {
	terminal := &ScreenshotEditorialOutcome{RequestID: "shot-001", Status: "rejected", Reason: "second attempt is unrelated"}
	control := controller{
		screenshotRequests: []ScreenshotRequest{{ID: "shot-001"}},
		screenshotRecords: map[string]*ScreenshotRecord{
			"shot-001": {ID: "shot-001", Attempt: 2, EditorialOutcome: terminal},
		},
	}
	if _, err := control.recordScreenshotEditorialOutcomes([]ScreenshotEditorialOutcome{{
		RequestID: "shot-001", Status: "usable", Reason: "later candidate tries to revive it",
	}}); err == nil || !strings.Contains(err.Error(), "durable non-placement is terminal") {
		t.Fatalf("later usable outcome did not fail at terminal rejection: %v", err)
	}
	if got := control.screenshotRecords["shot-001"].EditorialOutcome; got != terminal || got.Status != "rejected" || got.Reason != "second attempt is unrelated" {
		t.Fatalf("terminal rejection was overwritten: %+v", got)
	}
}

func TestScreenshotEditorialReasonByteBoundary(t *testing.T) {
	available := map[string]visualInput{
		"shot-001": {ID: "shot-001", Origin: visualInputOriginScreenshot},
	}
	for _, test := range []struct {
		name    string
		length  int
		wantErr bool
	}{
		{name: "1024 accepted", length: 1024},
		{name: "1025 rejected", length: 1025, wantErr: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := validateScreenshotEditorialOutcomes([]ScreenshotEditorialOutcome{{
				RequestID: "shot-001", Status: "usable", Reason: strings.Repeat("x", test.length),
			}}, available, nil)
			if test.wantErr && (err == nil || !strings.Contains(err.Error(), "longer than 1024 bytes")) {
				t.Fatalf("reason length %d did not fail at documented boundary: %v", test.length, err)
			}
			if !test.wantErr && err != nil {
				t.Fatalf("reason length %d rejected: %v", test.length, err)
			}
		})
	}
}

func TestValidateCaptureWorkspaceRequiresExactDeclaredAssetPath(t *testing.T) {
	declarations := map[string]bool{"assets/declared.png": true}
	folded := map[string]bool{"assets/declared.png": true}

	t.Run("actual_case_variant_is_undeclared", func(t *testing.T) {
		workspace := t.TempDir()
		if err := os.Mkdir(filepath.Join(workspace, "assets"), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(workspace, "assets", "DECLARED.png"), []byte("extra"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := validateCaptureWorkspaceEntries(workspace, declarations, folded); err == nil || !strings.Contains(err.Error(), "undeclared file") {
			t.Fatalf("case-variant file passed exact declaration membership: %v", err)
		}
	})

	t.Run("two_case_variants_on_case_sensitive_filesystem", func(t *testing.T) {
		workspace := t.TempDir()
		assets := filepath.Join(workspace, "assets")
		if err := os.Mkdir(assets, 0o700); err != nil {
			t.Fatal(err)
		}
		lower := filepath.Join(assets, "declared.png")
		upper := filepath.Join(assets, "DECLARED.png")
		if err := os.WriteFile(lower, []byte("declared"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(upper, []byte("extra"), 0o600); err != nil {
			t.Fatal(err)
		}
		lowerInfo, lowerErr := os.Stat(lower)
		upperInfo, upperErr := os.Stat(upper)
		if lowerErr != nil || upperErr != nil {
			t.Fatalf("inspect case variants: lower=%v upper=%v", lowerErr, upperErr)
		}
		if os.SameFile(lowerInfo, upperInfo) {
			t.Skip("filesystem is case-insensitive")
		}
		if err := validateCaptureWorkspaceEntries(workspace, declarations, folded); err == nil || !strings.Contains(err.Error(), "undeclared file") {
			t.Fatalf("extra case variant passed on case-sensitive filesystem: %v", err)
		}
	})
}

func TestValidatePublicPageURLAcceptsOnlyPublicHTTPSPages(t *testing.T) {
	for _, accepted := range []struct{ raw, want string }{
		{"https://example.com/report", "https://example.com/report"},
		{"https://example.com:443/report?q=1#top", "https://example.com/report?q=1#top"},
		{"https://Sub.Example.CO.UK/a", "https://sub.example.co.uk/a"},
		{"https://example.com", "https://example.com"},
	} {
		got, err := validatePublicPageURL(accepted.raw)
		if err != nil {
			t.Errorf("public URL %q rejected: %v", accepted.raw, err)
			continue
		}
		if got != accepted.want {
			t.Errorf("normalized %q to %q, want %q", accepted.raw, got, accepted.want)
		}
	}
	for _, rejected := range []string{
		"", "http://example.com/a", "ftp://example.com/a", "file:///etc/passwd",
		"https://user:pass@example.com/a", "https://user@example.com/a",
		"https://localhost/a", "https://api.localhost/a", "https://printer.local/a",
		"https://wiki.internal/a", "https://box.lan/a", "https://thing.home/a",
		"https://127.0.0.1/a", "https://10.0.0.5/a", "https://[::1]/a", "https://[fd00::1]/a",
		"https://intranet/a", "https://singlelabel/a", "https://example.com:8443/a",
		"https://exa mple.com/a", "https://example.com/a\nHost: evil",
		"https:example.com", "https://example.123/a", "https://-bad.example.com/a",
		"https://.example.com/a",
	} {
		if got, err := validatePublicPageURL(rejected); err == nil {
			t.Errorf("unsafe URL %q accepted as %q", rejected, got)
		}
	}
}

func TestParseScreenshotRequestsEnforcesTheDocumentedShape(t *testing.T) {
	valid := `{"screenshots":[{"id":"shot-001","url":"https://example.com/a","reason":"why","supports":["claim-004"],"selector":"main"}]}`
	requests, err := parseScreenshotRequests([]byte(valid), []byte(testClaimLedger))
	if err != nil {
		t.Fatalf("valid request rejected: %v", err)
	}
	if len(requests) != 1 || requests[0].ID != "shot-001" || requests[0].Selector != "main" {
		t.Fatalf("unexpected parse result: %+v", requests)
	}

	empty, err := parseScreenshotRequests([]byte(`{"screenshots":[]}`), []byte(testClaimLedger))
	if err != nil || len(empty) != 0 {
		t.Fatalf("empty request list rejected: %v (%+v)", err, empty)
	}

	entry := func(id, extra string) string {
		return `{"id":"` + id + `","url":"https://example.com/a","reason":"why","supports":["claim-004"]` + extra + `}`
	}
	overLimit := make([]string, 0, 6)
	for index := 0; index < 6; index++ {
		overLimit = append(overLimit, entry(string(rune('a'+index)), ""))
	}
	for name, payload := range map[string]string{
		"empty document":      ``,
		"blank document":      `   `,
		"not an object":       `[]`,
		"missing list":        `{"shots":[]}`,
		"null list":           `{"screenshots":null}`,
		"unknown top field":   `{"screenshots":[],"quota":3}`,
		"unknown entry field": `{"screenshots":[` + entry("shot-001", `,"crop":true`) + `]}`,
		"duplicate key":       `{"screenshots":[{"id":"a","id":"b","url":"https://example.com/a","reason":"w","supports":["claim-004"]}]}`,
		"duplicate id":        `{"screenshots":[` + entry("shot-001", "") + `,` + entry("shot-001", "") + `]}`,
		"over limit":          `{"screenshots":[` + strings.Join(overLimit, ",") + `]}`,
		"empty id":            `{"screenshots":[{"id":"","url":"https://example.com/a","reason":"w","supports":["claim-004"]}]}`,
		"unsafe id":           `{"screenshots":[{"id":"../x","url":"https://example.com/a","reason":"w","supports":["claim-004"]}]}`,
		"dotted id":           `{"screenshots":[{"id":"a.png","url":"https://example.com/a","reason":"w","supports":["claim-004"]}]}`,
		"empty reason":        `{"screenshots":[{"id":"a","url":"https://example.com/a","reason":"  ","supports":["claim-004"]}]}`,
		"no supports":         `{"screenshots":[{"id":"a","url":"https://example.com/a","reason":"w","supports":[]}]}`,
		"repeated support":    `{"screenshots":[{"id":"a","url":"https://example.com/a","reason":"w","supports":["claim-004","claim-004"]}]}`,
		"unknown claim":       `{"screenshots":[{"id":"a","url":"https://example.com/a","reason":"w","supports":["claim-999"]}]}`,
		"prefix claim":        `{"screenshots":[{"id":"a","url":"https://example.com/a","reason":"w","supports":["claim-00"]}]}`,
		"unsafe url":          `{"screenshots":[{"id":"a","url":"https://localhost/a","reason":"w","supports":["claim-004"]}]}`,
		"empty selector":      `{"screenshots":[{"id":"a","url":"https://example.com/a","reason":"w","supports":["claim-004"],"selector":""}]}`,
		"control selector":    "{\"screenshots\":[{\"id\":\"a\",\"url\":\"https://example.com/a\",\"reason\":\"w\",\"supports\":[\"claim-004\"],\"selector\":\"ma\\u0000in\"}]}",
		"padded selector":     `{"screenshots":[{"id":"a","url":"https://example.com/a","reason":"w","supports":["claim-004"],"selector":" main "}]}`,
		"two documents":       `{"screenshots":[]}{"screenshots":[]}`,
	} {
		if _, err := parseScreenshotRequests([]byte(payload), []byte(testClaimLedger)); err == nil {
			t.Errorf("%s was accepted: %s", name, payload)
		}
	}

	// A longer ledger entry must not satisfy a shorter requested claim ID.
	if _, err := parseScreenshotRequests(
		[]byte(`{"screenshots":[{"id":"a","url":"https://example.com/a","reason":"w","supports":["claim-0041"]}]}`),
		[]byte(testClaimLedger)); err != nil {
		t.Errorf("an exact ledger claim was rejected: %v", err)
	}
}

func TestValidatePNGRejectsAnythingButACompleteImage(t *testing.T) {
	width, height, err := validatePNG(testPNG(t, 12, 7))
	if err != nil || width != 12 || height != 7 {
		t.Fatalf("valid PNG rejected: %dx%d %v", width, height, err)
	}
	complete := testPNG(t, 12, 7)
	for name, payload := range map[string][]byte{
		"empty":     nil,
		"text":      []byte("<html>not an image</html>"),
		"truncated": complete[:len(complete)-20],
		"header only": append(append([]byte(nil), pngSignature...),
			bytes.Repeat([]byte{0}, 25)...),
		"jpeg-ish": {0xff, 0xd8, 0xff, 0xe0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
			0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
	} {
		if _, _, err := validatePNG(payload); err == nil {
			t.Errorf("%s was accepted as a screenshot", name)
		}
	}
}

func TestControllerCommandEnvironmentDropsScreenshotCredentials(t *testing.T) {
	t.Setenv("CLOUDFLARE_ACCOUNT_ID", "account-id")
	t.Setenv("CLOUDFLARE_API_TOKEN", "token-value")
	t.Setenv("FUTURE_CAPTURE_PROVIDER_SECRET", "future-secret")
	for _, entry := range controllerCommandEnvironment() {
		if strings.Contains(entry, "account-id") || strings.Contains(entry, "token-value") || strings.Contains(entry, "future-secret") {
			t.Fatalf("controller helper environment carried capture-runner credential material: %q", entry)
		}
	}
	for _, entry := range agentEnvironment(providerCodex, t.TempDir(), t.TempDir(), "researcher", "", 1, "rev", "inv") {
		if strings.Contains(entry, "account-id") || strings.Contains(entry, "token-value") {
			t.Fatalf("agent environment carried screenshot credential material: %q", entry)
		}
	}
}

// Regression: the declared canvas must be rejected before png.Decode allocates
// from it. A tiny body may declare an enormous image.
func TestValidatePNGBoundsDeclaredDimensionsBeforeDecoding(t *testing.T) {
	header := func(width, height uint32) []byte {
		data := append([]byte(nil), pngSignature...)
		chunk := make([]byte, 25)
		binary.BigEndian.PutUint32(chunk[0:4], 13)
		copy(chunk[4:8], "IHDR")
		binary.BigEndian.PutUint32(chunk[8:12], width)
		binary.BigEndian.PutUint32(chunk[12:16], height)
		chunk[16] = 8 // bit depth
		chunk[17] = 6 // truecolour with alpha
		return append(data, chunk...)
	}
	for name, item := range map[string]struct {
		payload []byte
		want    string
	}{
		"oversized width":  {header(1<<20, 8), "outside the accepted"},
		"oversized height": {header(8, 1<<20), "outside the accepted"},
		"zero width":       {header(0, 8), "outside the accepted"},
		"pixel bomb":       {header(20000, 20000), "more than the accepted"},
	} {
		start := time.Now()
		_, _, err := validatePNG(item.payload)
		if err == nil {
			t.Errorf("%s was accepted", name)
			continue
		}
		if !strings.Contains(err.Error(), item.want) {
			t.Errorf("%s rejected by the wrong gate: %v", name, err)
		}
		// A decode-then-check implementation would allocate first; the bound
		// must be reached without touching the pixel data at all.
		if elapsed := time.Since(start); elapsed > time.Second {
			t.Errorf("%s took %s, which means the header was decoded first", name, elapsed)
		}
	}

	// A well-formed image inside every bound still decodes normally.
	if width, height, err := validatePNG(testPNG(t, 9, 4)); err != nil || width != 9 || height != 4 {
		t.Fatalf("valid image rejected after the reordering: %dx%d %v", width, height, err)
	}
}

// Regression: IDs become file names. On a case-insensitive file system two IDs
// differing only in case are one path, so the second capture would overwrite
// the first while the manifest still advertised two screenshots.
func TestParseScreenshotRequestsRejectsCaseInsensitiveIDCollisions(t *testing.T) {
	entry := func(id string) string {
		return `{"id":"` + id + `","url":"https://example.com/a","reason":"w","supports":["claim-004"]}`
	}
	for _, pair := range [][2]string{
		{"shot-001", "SHOT-001"},
		{"Shot_A", "shot_a"},
		{"AB", "aB"},
	} {
		payload := `{"screenshots":[` + entry(pair[0]) + `,` + entry(pair[1]) + `]}`
		requests, err := parseScreenshotRequests([]byte(payload), []byte(testClaimLedger))
		if err == nil {
			t.Errorf("colliding IDs %q/%q were accepted as %d screenshots", pair[0], pair[1], len(requests))
			continue
		}
		if !strings.Contains(err.Error(), "duplicates screenshot ID") {
			t.Errorf("colliding IDs %q/%q rejected by the wrong gate: %v", pair[0], pair[1], err)
		}
	}
	// Genuinely distinct IDs are still accepted.
	payload := `{"screenshots":[` + entry("shot-001") + `,` + entry("shot-002") + `]}`
	if _, err := parseScreenshotRequests([]byte(payload), []byte(testClaimLedger)); err != nil {
		t.Fatalf("distinct IDs rejected: %v", err)
	}
}
