package utils

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"strings"
)

// CompressScreenshot compresses a base64-encoded screenshot image
func CompressScreenshot(base64Data string, quality int) (string, error) {
	// Remove data URL prefix if present (e.g., "data:image/png;base64,")
	if idx := strings.Index(base64Data, ","); idx != -1 {
		base64Data = base64Data[idx+1:]
	}

	// Decode base64
	imageData, err := base64.StdEncoding.DecodeString(base64Data)
	if err != nil {
		return "", fmt.Errorf("base64 decode failed: %w", err)
	}

	// Decode image
	img, format, err := image.Decode(bytes.NewReader(imageData))
	if err != nil {
		return "", fmt.Errorf("image decode failed: %w", err)
	}

	// If already JPEG with reasonable size, return as-is
	if format == "jpeg" && len(imageData) < 500000 { // 500KB threshold
		return base64Data, nil
	}

	// Re-encode as JPEG with specified quality
	var buf bytes.Buffer
	err = jpeg.Encode(&buf, img, &jpeg.Options{Quality: quality})
	if err != nil {
		return "", fmt.Errorf("jpeg encode failed: %w", err)
	}

	// Encode to base64
	compressed := base64.StdEncoding.EncodeToString(buf.Bytes())

	// Add data URL prefix back
	return "data:image/jpeg;base64," + compressed, nil
}

// CompressScreenshotPNG compresses a PNG image (alternative method)
func CompressScreenshotPNG(base64Data string) (string, error) {
	// Remove data URL prefix if present
	if idx := strings.Index(base64Data, ","); idx != -1 {
		base64Data = base64Data[idx+1:]
	}

	// Decode base64
	imageData, err := base64.StdEncoding.DecodeString(base64Data)
	if err != nil {
		return "", fmt.Errorf("base64 decode failed: %w", err)
	}

	// Decode image
	img, _, err := image.Decode(bytes.NewReader(imageData))
	if err != nil {
		return "", fmt.Errorf("image decode failed: %w", err)
	}

	// Re-encode as PNG with compression
	var buf bytes.Buffer
	encoder := png.Encoder{
		CompressionLevel: png.BestCompression,
	}
	err = encoder.Encode(&buf, img)
	if err != nil {
		return "", fmt.Errorf("png encode failed: %w", err)
	}

	// Encode to base64
	compressed := base64.StdEncoding.EncodeToString(buf.Bytes())

	// Add data URL prefix back
	return "data:image/png;base64," + compressed, nil
}