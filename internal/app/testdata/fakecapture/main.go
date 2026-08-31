package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/uuta/write-uuter/internal/captureprotocol"
)

func main() {
	if len(os.Args) != 2 || os.Args[1] != captureprotocol.VersionArgument {
		os.Exit(90)
	}
	scenario := os.Getenv("WRITE_UUTER_TEST_CAPTURE_SCENARIO")
	if scenario == "fast-detached-nonzero" {
		startDetachedChild()
		_, _ = os.Stderr.WriteString("secret-output:" + os.Getenv("WRITE_UUTER_CAPTURE_SECRET"))
		os.Exit(7)
	}
	if scenario == "nonzero" {
		_, _ = os.Stderr.WriteString("secret-output:" + os.Getenv("WRITE_UUTER_CAPTURE_SECRET"))
		os.Exit(7)
	}
	if scenario == "timeout" {
		time.Sleep(30 * time.Second)
		return
	}
	if scenario == "descriptor-child" || scenario == "detached-child" {
		time.Sleep(30 * time.Second)
		return
	}
	if scenario == "descriptor-descendant" {
		child := exec.Command(os.Args[0], captureprotocol.VersionArgument)
		child.Env = append(os.Environ(), "WRITE_UUTER_TEST_CAPTURE_SCENARIO=descriptor-child")
		child.Stdout = os.Stdout
		child.Stderr = os.Stderr
		child.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
		if child.Start() != nil {
			os.Exit(93)
		}
		if marker := os.Getenv("WRITE_UUTER_TEST_CAPTURE_CHILD_PID"); marker != "" {
			_ = os.WriteFile(marker, []byte(strconv.Itoa(child.Process.Pid)+"\n"), 0o600)
		}
		time.Sleep(30 * time.Second)
		return
	}
	requestData, err := os.ReadFile(captureprotocol.RequestFile)
	if err != nil {
		os.Exit(91)
	}
	var request captureprotocol.RequestDocument
	if json.Unmarshal(requestData, &request) != nil {
		os.Exit(92)
	}
	logInvocation(request)
	if scenario == "mutate-request" {
		_ = os.Chmod(captureprotocol.RequestFile, 0o600)
		_ = os.WriteFile(captureprotocol.RequestFile, []byte("{}\n"), 0o600)
	}
	if scenario == "partial" {
		_ = os.WriteFile(captureprotocol.ResultFile, []byte(`{"schema_version":2,"results":[`), 0o600)
		os.Exit(8)
	}
	_ = os.Mkdir(captureprotocol.AssetsDirectory, 0o700)
	pngData := fixturePNG()
	results := make([]captureprotocol.Result, 0, len(request.Requests))
	for index, item := range request.Requests {
		name := filepath.ToSlash(filepath.Join(captureprotocol.AssetsDirectory, "capture-"+item.RequestID+".png"))
		data := pngData
		if (scenario == "unusable-success" || scenario == "retry-exhaust" || scenario == "retry-success") && index == 0 && item.PriorAttempt == nil {
			data = blankPNG()
		}
		if scenario == "retry-exhaust" && index == 0 {
			data = blankPNG()
		}
		if scenario == "path-collision" {
			switch {
			case item.PriorAttempt != nil:
				data = replacementPNG()
			case item.RequestID == "shot-001":
				data = blankPNG()
			default:
				data = pngData
			}
		}
		if scenario == "invalid-png" && index == 0 {
			data = []byte("not png")
		}
		if scenario == "oversized-png" && index == 0 {
			data = append(append([]byte(nil), pngData...), bytes.Repeat([]byte{'x'}, (10<<20)+1)...)
		}
		_ = os.WriteFile(name, data, 0o600)
		digest := sha256.Sum256(data)
		result := captureprotocol.Result{
			RequestID: item.RequestID, RequestedURL: item.PublicURL, FinalURL: item.PublicURL,
			CapturedAt: "2026-08-30T01:02:03Z", Backend: "fake-backend", MediaType: captureprotocol.PNGMediaType,
			Viewport: captureprotocol.Viewport{Width: 1280, Height: 800}, FullPage: false,
			ImagePath: name, ByteSize: int64(len(data)), Width: 3, Height: 2,
			SHA256: "sha256:" + hex.EncodeToString(digest[:]), SupportedClaimIDs: item.SupportedClaimIDs,
			Rationale: item.Reason, ActionSummary: []string{"capture fixture"},
		}
		if item.PriorAttempt != nil {
			result.Backend = "fake-backend-second-attempt"
		}
		results = append(results, result)
	}
	if len(results) > 0 {
		switch scenario {
		case "mismatch-id":
			results[0].RequestID = "wrong-id"
		case "mismatch-url":
			results[0].RequestedURL = "https://other.example/path"
		case "unsafe-final-url":
			results[0].FinalURL = "https://localhost/private"
		case "bad-media":
			results[0].MediaType = "image/jpeg"
		case "bad-backend":
			results[0].Backend = "bad backend"
		case "bad-timestamp":
			results[0].CapturedAt = "yesterday"
		case "bad-viewport":
			results[0].Viewport.Width = 0
		case "mismatch-claims":
			results[0].SupportedClaimIDs = []string{"claim-other"}
		case "mismatch-rationale":
			results[0].Rationale = "changed"
		case "bad-trace":
			results[0].TraceReference = "also-present"
		case "traversal":
			results[0].ImagePath = "assets/../outside.png"
		case "absolute":
			results[0].ImagePath = "/tmp/outside.png"
		case "unsafe-path":
			results[0].ImagePath = `assets\capture.png`
		case "digest-mismatch":
			results[0].SHA256 = "sha256:" + strings.Repeat("0", 64)
		case "size-mismatch":
			results[0].ByteSize++
		case "dimensions-mismatch":
			results[0].Width++
		case "duplicate-asset":
			if len(results) > 1 {
				results[1].ImagePath = results[0].ImagePath
			}
		}
	}
	switch scenario {
	case "missing-result":
		results = results[:len(results)-1]
	case "duplicate-result":
		results = append(results, results[0])
	case "extra-file":
		_ = os.WriteFile(filepath.Join(captureprotocol.AssetsDirectory, "extra.txt"), []byte("extra"), 0o600)
	case "extra-root-file":
		_ = os.WriteFile("debug.log", []byte("secret"), 0o600)
	case "extra-directory":
		_ = os.Mkdir(filepath.Join(captureprotocol.AssetsDirectory, "unused"), 0o700)
	case "missing-asset":
		_ = os.Remove(results[0].ImagePath)
	case "symlink":
		_ = os.Remove(results[0].ImagePath)
		_ = os.Symlink(captureprotocol.RequestFile, results[0].ImagePath)
	case "special-file":
		_ = os.Remove(results[0].ImagePath)
		_ = syscall.Mkfifo(results[0].ImagePath, 0o600)
	case "replace-after-exit":
		marker := os.Getenv("WRITE_UUTER_TEST_CAPTURE_CHILD_PID")
		child := exec.Command(os.Args[0], captureprotocol.VersionArgument)
		child.Env = append(os.Environ(), "WRITE_UUTER_TEST_CAPTURE_SCENARIO=delayed-child", "WRITE_UUTER_TEST_CAPTURE_TARGET="+results[0].ImagePath)
		if child.Start() == nil && marker != "" {
			_ = os.WriteFile(marker, []byte(strconv.Itoa(child.Process.Pid)+"\n"), 0o600)
		}
	case "delayed-child":
		time.Sleep(500 * time.Millisecond)
		_ = os.WriteFile(os.Getenv("WRITE_UUTER_TEST_CAPTURE_TARGET"), []byte("replacement"), 0o600)
		return
	}
	document := captureprotocol.ResultDocument{SchemaVersion: captureprotocol.Version, Results: results}
	data, _ := json.MarshalIndent(document, "", "  ")
	data = append(data, '\n')
	if scenario == "missing-result-file" {
		return
	}
	if scenario == "malformed-result" {
		data = []byte(`{"schema_version":`)
	}
	if scenario == "unknown-field" {
		data = bytes.Replace(data, []byte(`"schema_version": 2`), []byte(`"schema_version": 2, "unknown": true`), 1)
	}
	if scenario == "duplicate-field" {
		data = bytes.Replace(data, []byte(`"schema_version": 2`), []byte(`"schema_version": 2, "schema_version": 2`), 1)
	}
	if scenario == "wrong-version" {
		data = bytes.Replace(data, []byte(`"schema_version": 2`), []byte(`"schema_version": 999`), 1)
	}
	if scenario == "omitted-full-page" {
		data = bytes.Replace(data, []byte("      \"full_page\": false,\n"), nil, 1)
	}
	if scenario == "null-full-page" {
		data = bytes.Replace(data, []byte(`"full_page": false`), []byte(`"full_page": null`), 1)
	}
	if scenario == "case-variant-full-page" {
		data = bytes.Replace(data, []byte(`"full_page": false`), []byte(`"FULL_PAGE": false`), 1)
	}
	if scenario == "full-page-alias-collision" {
		data = bytes.Replace(data, []byte(`"full_page": false`), []byte(`"full_page": false, "FULL_PAGE": true`), 1)
	}
	_ = os.WriteFile(captureprotocol.ResultFile, data, 0o600)
	if scenario == "fast-detached-success" {
		startDetachedChild()
	}
}

func startDetachedChild() {
	child := exec.Command(os.Args[0], captureprotocol.VersionArgument)
	child.Env = append(os.Environ(), "WRITE_UUTER_TEST_CAPTURE_SCENARIO=detached-child")
	child.Stdin = nil
	child.Stdout = os.Stdout
	child.Stderr = os.Stderr
	child.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if child.Start() != nil {
		os.Exit(93)
	}
	marker := os.Getenv("WRITE_UUTER_TEST_CAPTURE_CHILD_PID")
	if marker == "" || os.WriteFile(marker, []byte(strconv.Itoa(child.Process.Pid)+"\n"), 0o600) != nil {
		os.Exit(94)
	}
}

func logInvocation(request captureprotocol.RequestDocument) {
	path := os.Getenv("WRITE_UUTER_TEST_CAPTURE_INVOCATION_LOG")
	if path == "" {
		return
	}
	workingDirectory, _ := os.Getwd()
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		os.Exit(94)
	}
	defer file.Close()
	_ = json.NewEncoder(file).Encode(struct {
		Workspace string                          `json:"workspace"`
		Request   captureprotocol.RequestDocument `json:"request"`
	}{Workspace: workingDirectory, Request: request})
}

func fixturePNG() []byte {
	canvas := image.NewRGBA(image.Rect(0, 0, 3, 2))
	canvas.Set(0, 0, color.RGBA{R: 1, G: 2, B: 3, A: 255})
	var buffer bytes.Buffer
	_ = png.Encode(&buffer, canvas)
	return buffer.Bytes()
}

func blankPNG() []byte {
	canvas := image.NewRGBA(image.Rect(0, 0, 3, 2))
	for y := 0; y < 2; y++ {
		for x := 0; x < 3; x++ {
			canvas.Set(x, y, color.White)
		}
	}
	var buffer bytes.Buffer
	_ = png.Encode(&buffer, canvas)
	return buffer.Bytes()
}

func replacementPNG() []byte {
	canvas := image.NewRGBA(image.Rect(0, 0, 3, 2))
	for y := 0; y < 2; y++ {
		for x := 0; x < 3; x++ {
			canvas.Set(x, y, color.RGBA{R: 0x20, G: uint8(0x40 + x), B: uint8(0x80 + y), A: 0xff})
		}
	}
	var buffer bytes.Buffer
	_ = png.Encode(&buffer, canvas)
	return buffer.Bytes()
}
