package registry

import (
	"testing"
)

func TestGetLayerMediaType(t *testing.T) {
	tests := []struct {
		format   Format
		input    string
		expected string
	}{
		{FormatArtifacts, MimeTypeECIKernel, MimeTypeECIKernel},
		{FormatLegacy, MimeTypeECIKernel, MimeTypeOCIImageLayer},
	}
	for i, tt := range tests {
		out := GetLayerMediaType(tt.input, tt.format)
		if out != tt.expected {
			t.Logf("%d: mismatched mimeType, actual %s expected %s", i, out, tt.expected)
		}
	}
}
func TestGetConfigMediaType(t *testing.T) {
	tests := []struct {
		format   Format
		input    string
		expected string
	}{
		{FormatArtifacts, MimeTypeECIConfig, MimeTypeECIKernel},
		{FormatLegacy, MimeTypeECIConfig, MimeTypeOCIImageConfig},
	}
	for i, tt := range tests {
		out := GetConfigMediaType(tt.input, tt.format)
		if out != tt.expected {
			t.Logf("%d: mismatched mimeType, actual %s expected %s", i, out, tt.expected)
		}
	}
}
