package app

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/png"
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

func TestScreenshotClientScrubsCredentialsFromDiagnostics(t *testing.T) {
	client := newScreenshotClient(screenshotCredentials{accountID: "acct-secret", apiToken: "tok-secret"}, screenshotAPIBaseURL)
	message := client.scrubbed("request to https://api.example/accounts/%s failed: token %s rejected", "acct-secret", "tok-secret").Error()
	if strings.Contains(message, "acct-secret") || strings.Contains(message, "tok-secret") {
		t.Fatalf("diagnostic retained credential material: %q", message)
	}
	if !strings.Contains(message, "[redacted]") {
		t.Fatalf("diagnostic did not mark the redaction: %q", message)
	}
}

func TestLoadScreenshotCredentialsNamesOnlyTheMissingVariable(t *testing.T) {
	t.Setenv(screenshotAccountEnv, "")
	t.Setenv(screenshotTokenEnv, "")
	_, err := loadScreenshotCredentials()
	if err == nil {
		t.Fatal("missing credentials were accepted")
	}
	if !strings.Contains(err.Error(), screenshotAccountEnv) || !strings.Contains(err.Error(), screenshotTokenEnv) {
		t.Fatalf("error does not name both variables: %v", err)
	}

	t.Setenv(screenshotAccountEnv, "account-id")
	t.Setenv(screenshotTokenEnv, "bad token with space")
	if _, err := loadScreenshotCredentials(); err == nil {
		t.Fatal("a token containing whitespace was accepted")
	} else if strings.Contains(err.Error(), "bad token") {
		t.Fatalf("rejection echoed the credential value: %v", err)
	}

	t.Setenv(screenshotAccountEnv, "acct/../other")
	t.Setenv(screenshotTokenEnv, "token")
	if _, err := loadScreenshotCredentials(); err == nil {
		t.Fatal("an account ID containing a path separator was accepted")
	}
}

func TestControllerCommandEnvironmentDropsScreenshotCredentials(t *testing.T) {
	t.Setenv(screenshotAccountEnv, "account-id")
	t.Setenv(screenshotTokenEnv, "token-value")
	for _, entry := range controllerCommandEnvironment() {
		if strings.HasPrefix(entry, screenshotAccountEnv+"=") || strings.HasPrefix(entry, screenshotTokenEnv+"=") {
			t.Fatalf("controller helper environment carried %q", entry)
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

// Regression: the base URL override retargets a request that carries the API
// token, so anything but a loopback origin must fail the run.
func TestResolveScreenshotBaseURLAcceptsOnlyLoopbackOverrides(t *testing.T) {
	t.Setenv(screenshotBaseURLEnv, "")
	base, err := resolveScreenshotBaseURL()
	if err != nil || base != screenshotAPIBaseURL {
		t.Fatalf("unset override did not resolve to the Cloudflare API: %q %v", base, err)
	}
	for _, accepted := range []string{
		"http://127.0.0.1:8080", "http://127.0.0.1:8080/", "https://127.0.0.1:9443/v4",
		"http://[::1]:8080", "http://localhost:8080",
	} {
		t.Setenv(screenshotBaseURLEnv, accepted)
		if _, err := resolveScreenshotBaseURL(); err != nil {
			t.Errorf("loopback override %q rejected: %v", accepted, err)
		}
	}
	for _, rejected := range []string{
		"https://attacker.example/v4", "http://169.254.169.254/latest",
		"http://10.0.0.5:8080", "https://api.cloudflare.com.evil.example/client/v4",
		"ftp://127.0.0.1/x", "file:///tmp/x", "http://user:pass@127.0.0.1/x",
		"http://127.0.0.1/x?token=1", "http://127.0.0.1/x#f", "not a url at all",
		"http://0.0.0.0:8080",
	} {
		t.Setenv(screenshotBaseURLEnv, rejected)
		if base, err := resolveScreenshotBaseURL(); err == nil {
			t.Errorf("unsafe override %q accepted as %q", rejected, base)
		}
	}
}

// Regression: an upstream error body must never be copied into a durable
// artifact. Only the status code and documented numeric codes are reported.
func TestNonSuccessDiagnosticsReportOnlyGeneratedText(t *testing.T) {
	secret := "sk-live-abcdefghijklmnopqrstuvwxyz"
	for name, body := range map[string]string{
		"documented envelope": `{"success":false,"errors":[{"code":10000,"message":"Authentication error for ` + secret + `"}]}`,
		"undocumented shape":  `{"detail":"` + secret + `"}`,
		"html error page":     `<html><body>` + secret + `</body></html>`,
		"empty body":          ``,
	} {
		rendered := apiErrorCodes([]byte(body))
		if strings.Contains(rendered, secret) || strings.Contains(rendered, "Authentication error") {
			t.Errorf("%s leaked upstream text into a diagnostic: %q", name, rendered)
		}
	}
	if got := apiErrorCodes([]byte(`{"errors":[{"code":10000},{"code":7003}]}`)); !strings.Contains(got, "10000") || !strings.Contains(got, "7003") {
		t.Errorf("documented error codes were dropped: %q", got)
	}
	if got := apiErrorCodes([]byte(`{"detail":"nope"}`)); !strings.Contains(got, "no documented error code") {
		t.Errorf("unparseable body produced an unexpected diagnostic: %q", got)
	}
}
