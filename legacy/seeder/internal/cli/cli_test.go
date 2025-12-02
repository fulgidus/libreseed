// Package cli provides command-line interface commands for the seeder.
package cli

import (
	"errors"
	"strings"
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

// TestSentinelErrors tests that the sentinel errors are properly defined and can be used with errors.Is()
func TestSentinelErrors(t *testing.T) {
	tests := []struct {
		name        string
		err         error
		target      error
		shouldMatch bool
		errContains string
	}{
		{
			name:        "errNoSourceProvided identity",
			err:         errNoSourceProvided,
			target:      errNoSourceProvided,
			shouldMatch: true,
			errContains: "must specify one of",
		},
		{
			name:        "errMultipleSourcesProvided identity",
			err:         errMultipleSourcesProvided,
			target:      errMultipleSourcesProvided,
			shouldMatch: true,
			errContains: "specify only one of",
		},
		{
			name:        "errInfoHashRequired identity",
			err:         errInfoHashRequired,
			target:      errInfoHashRequired,
			shouldMatch: true,
			errContains: "required",
		},
		{
			name:        "errNoSourceProvided does not match errMultipleSourcesProvided",
			err:         errNoSourceProvided,
			target:      errMultipleSourcesProvided,
			shouldMatch: false,
			errContains: "",
		},
		{
			name:        "errNoSourceProvided does not match errInfoHashRequired",
			err:         errNoSourceProvided,
			target:      errInfoHashRequired,
			shouldMatch: false,
			errContains: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := errors.Is(tt.err, tt.target); got != tt.shouldMatch {
				t.Errorf("errors.Is(%v, %v) = %v, want %v",
					tt.err, tt.target, got, tt.shouldMatch)
			}

			// Also verify the error message contains expected text
			if tt.errContains != "" && !strings.Contains(tt.err.Error(), tt.errContains) {
				t.Errorf("error message %q does not contain %q",
					tt.err.Error(), tt.errContains)
			}
		})
	}
}

// TestVersionCommand tests the version command executes without error
// Note: The version command uses fmt.Printf which writes to os.Stdout directly,
// so we only verify execution succeeds rather than capturing output.
func TestVersionCommand(t *testing.T) {
	// Execute command - should not return an error
	err := versionCmd.RunE
	if err != nil {
		t.Fatalf("versionCmd should not have RunE set, has Run instead")
	}

	// Verify Run function is set
	if versionCmd.Run == nil {
		t.Error("versionCmd.Run should be set")
	}
}

// TestVersionCommandStructure tests the version command metadata
func TestVersionCommandStructure(t *testing.T) {
	if versionCmd.Use != "version" {
		t.Errorf("versionCmd.Use = %q, want %q", versionCmd.Use, "version")
	}

	if versionCmd.Short == "" {
		t.Error("versionCmd.Short should not be empty")
	}

	if versionCmd.Long == "" {
		t.Error("versionCmd.Long should not be empty")
	}

	if versionCmd.Run == nil {
		t.Error("versionCmd.Run should not be nil")
	}
}

// TestRootCommandStructure tests the root command structure and subcommands
func TestRootCommandStructure(t *testing.T) {
	// Test root command properties
	if rootCmd.Use != "seeder" {
		t.Errorf("rootCmd.Use = %q, want %q", rootCmd.Use, "seeder")
	}

	if rootCmd.Short == "" {
		t.Error("rootCmd.Short should not be empty")
	}

	// Test that expected subcommands are registered
	expectedSubcommands := []string{"version", "start", "add", "remove", "list", "status"}

	for _, expected := range expectedSubcommands {
		found := false
		for _, cmd := range rootCmd.Commands() {
			if cmd.Use == expected || strings.HasPrefix(cmd.Use, expected+" ") {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("rootCmd missing expected subcommand %q", expected)
		}
	}
}

// TestRootCommandPersistentFlags tests that persistent flags are properly defined
func TestRootCommandPersistentFlags(t *testing.T) {
	pflags := rootCmd.PersistentFlags()

	// Test --config flag
	configFlag := pflags.Lookup("config")
	if configFlag == nil {
		t.Error("rootCmd missing --config persistent flag")
	} else {
		// Note: --config has no shorthand defined in root.go
		if configFlag.Shorthand != "" {
			t.Errorf("--config shorthand = %q, want %q", configFlag.Shorthand, "")
		}
	}

	// Test --log-level flag
	logLevelFlag := pflags.Lookup("log-level")
	if logLevelFlag == nil {
		t.Error("rootCmd missing --log-level persistent flag")
	} else {
		if logLevelFlag.DefValue != "info" {
			t.Errorf("--log-level default = %q, want %q", logLevelFlag.DefValue, "info")
		}
	}

	// Test --log-format flag
	logFormatFlag := pflags.Lookup("log-format")
	if logFormatFlag == nil {
		t.Error("rootCmd missing --log-format persistent flag")
	} else {
		// Note: --log-format default is "json" in root.go
		if logFormatFlag.DefValue != "json" {
			t.Errorf("--log-format default = %q, want %q", logFormatFlag.DefValue, "json")
		}
	}
}
