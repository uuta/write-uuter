package captureadapter

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/png"
	"strings"
	"testing"
)

func TestAPIErrorCodesExposeOnlyDocumentedNumericCodes(t *testing.T) {
	secret := "provider-secret-message"
	for _, testCase := range []struct {
		name string
		body string
		want string
	}{
		{"codes_only", `{"errors":[{"code":10000,"message":"` + secret + `"},{"code":10001}]}`, "Cloudflare error codes 10000, 10001"},
		{"missing_errors", `{"message":"` + secret + `"}`, "no documented error code"},
		{"malformed", `{"errors":[{"code":`, "no documented error code"},
		{"html", `<html>` + secret + `</html>`, "no documented error code"},
		{"empty", ``, "no documented error code"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			got := apiErrorCodes([]byte(testCase.body))
			if got != testCase.want {
				t.Fatalf("apiErrorCodes() = %q, want %q", got, testCase.want)
			}
			if strings.Contains(got, secret) {
				t.Fatalf("provider-controlled text escaped diagnostics: %q", got)
			}
		})
	}
}

func TestResolveBaseURLAllowsOnlyLoopbackTestOrigins(t *testing.T) {
	for _, testCase := range []struct {
		name    string
		value   string
		want    string
		wantErr string
	}{
		{"default", "", apiBaseURL, ""},
		{"ipv4", "http://127.0.0.1:43123", "http://127.0.0.1:43123", ""},
		{"localhost_trailing_slash", "https://localhost:8443/", "https://localhost:8443", ""},
		{"ipv6_path", "http://[::1]:8080/client/v4", "http://[::1]:8080/client/v4", ""},
		{"remote", "https://example.com", "", "not loopback"},
		{"metadata_service", "http://169.254.169.254", "", "not loopback"},
		{"private_network", "http://10.0.0.5", "", "not loopback"},
		{"wildcard_address", "http://0.0.0.0:8080", "", "not loopback"},
		{"cloudflare_suffix_attack", "https://api.cloudflare.com.evil.example", "", "not loopback"},
		{"localhost_suffix", "https://localhost.example.com", "", "not loopback"},
		{"numeric_alias", "http://2130706433", "", "not loopback"},
		{"userinfo", "http://user:pass@127.0.0.1", "", "invalid"},
		{"query", "http://127.0.0.1?token=x", "", "invalid"},
		{"fragment", "http://127.0.0.1/#x", "", "invalid"},
		{"unsupported_scheme", "ftp://127.0.0.1", "", "invalid"},
		{"missing_scheme", "127.0.0.1:8080", "", "invalid"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			t.Setenv(testBaseURLEnv, testCase.value)
			got, err := resolveBaseURL()
			if testCase.wantErr == "" {
				if err != nil || got != testCase.want {
					t.Fatalf("resolveBaseURL() = %q, %v; want %q", got, err, testCase.want)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), testCase.wantErr) {
				t.Fatalf("resolveBaseURL() error = %v, want %q", err, testCase.wantErr)
			}
		})
	}
}

func TestValidateProviderPNGBoundsIHDRBeforeDecode(t *testing.T) {
	valid := testPNG(4, 3)
	if width, height, err := validateProviderPNG(valid); err != nil || width != 4 || height != 3 {
		t.Fatalf("valid PNG rejected: %dx%d %v", width, height, err)
	}
	for _, testCase := range []struct {
		name   string
		width  uint32
		height uint32
		want   string
	}{
		{"axis", 20001, 1, "outside the accepted"},
		{"pixel_count", 10000, 5000, "more than the accepted"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			declared := append([]byte(nil), valid...)
			binary.BigEndian.PutUint32(declared[16:20], testCase.width)
			binary.BigEndian.PutUint32(declared[20:24], testCase.height)
			_, _, err := validateProviderPNG(declared)
			if err == nil || !strings.Contains(err.Error(), testCase.want) {
				t.Fatalf("declared %dx%d error = %v, want %q", testCase.width, testCase.height, err, testCase.want)
			}
		})
	}
}

func testPNG(width, height int) []byte {
	canvas := image.NewRGBA(image.Rect(0, 0, width, height))
	canvas.Set(0, 0, color.RGBA{R: 1, G: 2, B: 3, A: 255})
	var buffer bytes.Buffer
	if err := png.Encode(&buffer, canvas); err != nil {
		panic(err)
	}
	return buffer.Bytes()
}
