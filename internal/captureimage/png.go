package captureimage

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image/png"
)

const (
	MaxBytes     = 10 << 20
	MinDimension = 1
	MaxDimension = 20000
	MaxPixels    = 40_000_000
)

var pngSignature = []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}

// ValidatePNG bounds the IHDR dimensions before png.Decode can allocate the
// declared canvas, then fully decodes the stream to reject truncated images.
// Both external adapters and the controller use it; the controller still
// validates independently after receiving runner-owned bytes.
func ValidatePNG(data []byte) (int, int, error) {
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
	if width < MinDimension || height < MinDimension || width > MaxDimension || height > MaxDimension {
		return 0, 0, fmt.Errorf("image dimensions %dx%d are outside the accepted %d-%d range", width, height, MinDimension, MaxDimension)
	}
	if int64(width)*int64(height) > MaxPixels {
		return 0, 0, fmt.Errorf("image declares %d pixels, more than the accepted %d", int64(width)*int64(height), MaxPixels)
	}
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
