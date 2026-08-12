package service

import (
	"testing"
	"time"
)

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		input    int64
		expected string
	}{
		{0, "0 B"},
		{500, "500 B"},
		{1024, "1.00 KB"},
		{1572864, "1.50 MB"},
		{1073741824, "1.00 GB"},
		{2684354560, "2.50 GB"},
	}

	for _, tt := range tests {
		result := FormatBytes(tt.input)
		if result != tt.expected {
			t.Errorf("FormatBytes(%d) = %s, expected %s", tt.input, result, tt.expected)
		}
	}
}

func TestFormatHandshake(t *testing.T) {
	now := time.Now().Unix()

	if res := FormatHandshake(0); res != "Offline" {
		t.Errorf("Expected 'Offline' for timestamp 0, got %s", res)
	}

	if res := FormatHandshake(now - 30); res != "Just now" {
		t.Errorf("Expected 'Just now' for 30s ago, got %s", res)
	}

	if res := FormatHandshake(now - 300); res != "5m ago" {
		t.Errorf("Expected '5m ago' for 300s ago, got %s", res)
	}

	if res := FormatHandshake(now - 7200); res != "2h ago" {
		t.Errorf("Expected '2h ago' for 7200s ago, got %s", res)
	}
}
