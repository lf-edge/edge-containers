package registry

import (
	"testing"
)

func TestGetLayerMediaType(t *testing.T) {
	tests := []struct {
		legacy   bool
		input    string
		expected string
	}{
		{false, MimeTypeECIKernel, MimeTypeECIKernel},
		{true, MimeTypeECIKernel, MimeTypeOCIImageLayer},
	}
	for i, tt := range tests {
		out := GetLayerMediaType(tt.input, tt.legacy)
		if out != tt.expected {
			t.Logf("%d: mismatched mimeType, actual %s expected %s", i, out, tt.expected)
		}
	}
}
func TestGetConfigMediaType(t *testing.T) {
	tests := []struct {
		legacy   bool
		input    string
		expected string
	}{
		{false, MimeTypeECIConfig, MimeTypeECIKernel},
		{true, MimeTypeECIConfig, MimeTypeOCIImageConfig},
	}
	for i, tt := range tests {
		out := GetConfigMediaType(tt.input, tt.legacy)
		if out != tt.expected {
			t.Logf("%d: mismatched mimeType, actual %s expected %s", i, out, tt.expected)
		}
	}
}
