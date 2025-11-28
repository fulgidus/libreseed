// Package cli provides command-line interface commands for the seeder.
package cli

import (
	"testing"
)

// TestTruncateString tests the truncateString helper function
func TestTruncateString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxLen   int
		expected string
	}{
		{
			name:     "string shorter than maxLen",
			input:    "hello",
			maxLen:   10,
			expected: "hello",
		},
		{
			name:     "string equal to maxLen",
			input:    "hello",
			maxLen:   5,
			expected: "hello",
		},
		{
			name:     "string longer than maxLen - truncated with ellipsis",
			input:    "hello world",
			maxLen:   8,
			expected: "hello...",
		},
		{
			name:     "empty string",
			input:    "",
			maxLen:   10,
			expected: "",
		},
		{
			name:     "maxLen of 3 - exactly fits ellipsis",
			input:    "hello",
			maxLen:   3,
			expected: "hel",
		},
		{
			name:     "maxLen of 2 - less than ellipsis length",
			input:    "hello",
			maxLen:   2,
			expected: "he",
		},
		{
			name:     "maxLen of 1",
			input:    "hello",
			maxLen:   1,
			expected: "h",
		},
		{
			name:     "maxLen of 0",
			input:    "hello",
			maxLen:   0,
			expected: "",
		},
		{
			name:     "maxLen of 4 - truncate with ellipsis",
			input:    "hello world",
			maxLen:   4,
			expected: "h...",
		},
		{
			name:     "unicode string truncation - byte based",
			input:    "hello世界",
			maxLen:   8,
			expected: "hello...", // byte-based: "hello世界" is 11 bytes, truncates at byte 5 + "..."
		},
		{
			name:     "long infohash-like string",
			input:    "abc123def456ghi789jkl012mno345pqr678stu901",
			maxLen:   16,
			expected: "abc123def456g...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncateString(tt.input, tt.maxLen)
			if result != tt.expected {
				t.Errorf("truncateString(%q, %d) = %q, want %q",
					tt.input, tt.maxLen, result, tt.expected)
			}
		})
	}
}

// TestFormatBytes tests the formatBytes helper function
func TestFormatBytes(t *testing.T) {
	tests := []struct {
		name     string
		bytes    int64
		expected string
	}{
		{
			name:     "zero bytes",
			bytes:    0,
			expected: "0 B",
		},
		{
			name:     "1 byte",
			bytes:    1,
			expected: "1 B",
		},
		{
			name:     "1023 bytes - just under 1 KB",
			bytes:    1023,
			expected: "1023 B",
		},
		{
			name:     "1024 bytes - exactly 1 KB",
			bytes:    1024,
			expected: "1.0 KB",
		},
		{
			name:     "1536 bytes - 1.5 KB",
			bytes:    1536,
			expected: "1.5 KB",
		},
		{
			name:     "1 MB - 1048576 bytes",
			bytes:    1048576,
			expected: "1.0 MB",
		},
		{
			name:     "1.5 MB",
			bytes:    1572864,
			expected: "1.5 MB",
		},
		{
			name:     "1 GB - 1073741824 bytes",
			bytes:    1073741824,
			expected: "1.0 GB",
		},
		{
			name:     "1 TB - 1099511627776 bytes",
			bytes:    1099511627776,
			expected: "1.0 TB",
		},
		{
			name:     "1 PB - 1125899906842624 bytes",
			bytes:    1125899906842624,
			expected: "1.0 PB",
		},
		{
			name:     "1 EB - 1152921504606846976 bytes",
			bytes:    1152921504606846976,
			expected: "1.0 EB",
		},
		{
			name:     "realistic torrent size - 4.7 GB DVD",
			bytes:    5046586573,
			expected: "4.7 GB",
		},
		{
			name:     "realistic torrent size - 700 MB CD",
			bytes:    734003200,
			expected: "700.0 MB",
		},
		{
			name:     "realistic uploaded amount - 25 GB",
			bytes:    26843545600,
			expected: "25.0 GB",
		},
		{
			name:     "small piece size - 256 KB",
			bytes:    262144,
			expected: "256.0 KB",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatBytes(tt.bytes)
			if result != tt.expected {
				t.Errorf("formatBytes(%d) = %q, want %q",
					tt.bytes, result, tt.expected)
			}
		})
	}
}

// TestValidateAddSourcesCount tests the validation logic for add command sources
func TestValidateAddSourcesCount(t *testing.T) {
	tests := []struct {
		name        string
		file        string
		magnet      string
		infohash    string
		wantErr     bool
		errContains string
	}{
		{
			name:        "no sources provided",
			file:        "",
			magnet:      "",
			infohash:    "",
			wantErr:     true,
			errContains: "must specify one of",
		},
		{
			name:        "file only - valid",
			file:        "/path/to/file.torrent",
			magnet:      "",
			infohash:    "",
			wantErr:     false,
			errContains: "",
		},
		{
			name:        "magnet only - valid",
			file:        "",
			magnet:      "magnet:?xt=urn:btih:abc123",
			infohash:    "",
			wantErr:     false,
			errContains: "",
		},
		{
			name:        "infohash only - valid",
			file:        "",
			magnet:      "",
			infohash:    "abc123def456",
			wantErr:     false,
			errContains: "",
		},
		{
			name:        "file and magnet - invalid",
			file:        "/path/to/file.torrent",
			magnet:      "magnet:?xt=urn:btih:abc123",
			infohash:    "",
			wantErr:     true,
			errContains: "specify only one of",
		},
		{
			name:        "file and infohash - invalid",
			file:        "/path/to/file.torrent",
			magnet:      "",
			infohash:    "abc123def456",
			wantErr:     true,
			errContains: "specify only one of",
		},
		{
			name:        "magnet and infohash - invalid",
			file:        "",
			magnet:      "magnet:?xt=urn:btih:abc123",
			infohash:    "abc123def456",
			wantErr:     true,
			errContains: "specify only one of",
		},
		{
			name:        "all three sources - invalid",
			file:        "/path/to/file.torrent",
			magnet:      "magnet:?xt=urn:btih:abc123",
			infohash:    "abc123def456",
			wantErr:     true,
			errContains: "specify only one of",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateAddSources(tt.file, tt.magnet, tt.infohash)

			if tt.wantErr {
				if err == nil {
					t.Errorf("validateAddSources() expected error containing %q, got nil",
						tt.errContains)
					return
				}
				if tt.errContains != "" && !containsString(err.Error(), tt.errContains) {
					t.Errorf("validateAddSources() error = %q, want error containing %q",
						err.Error(), tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("validateAddSources() unexpected error: %v", err)
				}
			}
		})
	}
}

// TestValidateRemoveInfoHash tests the validation logic for remove command
func TestValidateRemoveInfoHash(t *testing.T) {
	tests := []struct {
		name        string
		infohash    string
		wantErr     bool
		errContains string
	}{
		{
			name:        "empty infohash",
			infohash:    "",
			wantErr:     true,
			errContains: "required",
		},
		{
			name:        "valid infohash",
			infohash:    "abc123def456ghi789jkl012mno345pqr678stu901",
			wantErr:     false,
			errContains: "",
		},
		{
			name:        "short infohash - still valid format",
			infohash:    "abc123",
			wantErr:     false,
			errContains: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateRemoveInfoHash(tt.infohash)

			if tt.wantErr {
				if err == nil {
					t.Errorf("validateRemoveInfoHash() expected error, got nil")
					return
				}
				if tt.errContains != "" && !containsString(err.Error(), tt.errContains) {
					t.Errorf("validateRemoveInfoHash() error = %q, want error containing %q",
						err.Error(), tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("validateRemoveInfoHash() unexpected error: %v", err)
				}
			}
		})
	}
}

// Helper functions for validation logic (extracted for testability)

// validateAddSources validates that exactly one source is provided for the add command
func validateAddSources(file, magnet, infohash string) error {
	sources := 0
	if file != "" {
		sources++
	}
	if magnet != "" {
		sources++
	}
	if infohash != "" {
		sources++
	}

	if sources == 0 {
		return errNoSourceProvided
	}
	if sources > 1 {
		return errMultipleSourcesProvided
	}

	return nil
}

// validateRemoveInfoHash validates that an infohash is provided for the remove command
func validateRemoveInfoHash(infohash string) error {
	if infohash == "" {
		return errInfoHashRequired
	}
	return nil
}

// containsString checks if s contains substr
func containsString(s, substr string) bool {
	return len(substr) == 0 || (len(s) >= len(substr) && containsSubstring(s, substr))
}

// containsSubstring is a simple substring check
func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
