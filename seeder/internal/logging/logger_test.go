package logging

import (
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestNewLogger(t *testing.T) {
	tests := []struct {
		name      string
		level     string
		format    string
		wantError bool
	}{
		{
			name:      "valid json debug",
			level:     "debug",
			format:    "json",
			wantError: false,
		},
		{
			name:      "valid console info",
			level:     "info",
			format:    "console",
			wantError: false,
		},
		{
			name:      "valid json warn",
			level:     "warn",
			format:    "json",
			wantError: false,
		},
		{
			name:      "valid console error",
			level:     "error",
			format:    "console",
			wantError: false,
		},
		{
			name:      "invalid level",
			level:     "invalid",
			format:    "json",
			wantError: true,
		},
		{
			name:      "invalid format",
			level:     "info",
			format:    "invalid",
			wantError: true,
		},
		{
			name:      "case insensitive level",
			level:     "INFO",
			format:    "json",
			wantError: false,
		},
		{
			name:      "case insensitive format",
			level:     "info",
			format:    "JSON",
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, err := NewLogger(tt.level, tt.format)
			if (err != nil) != tt.wantError {
				t.Errorf("NewLogger() error = %v, wantError %v", err, tt.wantError)
				return
			}
			if !tt.wantError && logger == nil {
				t.Error("expected logger to be non-nil")
			}
		})
	}
}

func TestLoggerLevels(t *testing.T) {
	tests := []struct {
		level        string
		expectedZap  zapcore.Level
		shouldEnable []zapcore.Level
	}{
		{
			level:        "debug",
			expectedZap:  zapcore.DebugLevel,
			shouldEnable: []zapcore.Level{zapcore.DebugLevel, zapcore.InfoLevel, zapcore.WarnLevel, zapcore.ErrorLevel},
		},
		{
			level:        "info",
			expectedZap:  zapcore.InfoLevel,
			shouldEnable: []zapcore.Level{zapcore.InfoLevel, zapcore.WarnLevel, zapcore.ErrorLevel},
		},
		{
			level:        "warn",
			expectedZap:  zapcore.WarnLevel,
			shouldEnable: []zapcore.Level{zapcore.WarnLevel, zapcore.ErrorLevel},
		},
		{
			level:        "error",
			expectedZap:  zapcore.ErrorLevel,
			shouldEnable: []zapcore.Level{zapcore.ErrorLevel},
		},
	}

	for _, tt := range tests {
		t.Run(tt.level, func(t *testing.T) {
			logger, err := NewLogger(tt.level, "json")
			if err != nil {
				t.Fatalf("NewLogger() error = %v", err)
			}

			// Test that the logger was created
			if logger == nil {
				t.Fatal("expected non-nil logger")
			}

			// Basic smoke test - ensure logger doesn't panic
			logger.Info("test message")
			logger.Debug("debug message")
			logger.Warn("warn message")
			logger.Error("error message")
		})
	}
}

func TestLoggerFormats(t *testing.T) {
	formats := []string{"json", "console"}

	for _, format := range formats {
		t.Run(format, func(t *testing.T) {
			logger, err := NewLogger("info", format)
			if err != nil {
				t.Fatalf("NewLogger() error = %v", err)
			}

			if logger == nil {
				t.Fatal("expected non-nil logger")
			}

			// Smoke test
			logger.Info("test message",
				zap.String("format", format),
				zap.String("key", "value"),
			)
		})
	}
}
