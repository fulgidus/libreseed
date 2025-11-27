package logging

import (
	"fmt"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// NewLogger creates a new zap logger with the specified level and format
func NewLogger(level, format string) (*zap.Logger, error) {
	// Parse log level
	var zapLevel zapcore.Level
	if err := zapLevel.UnmarshalText([]byte(level)); err != nil {
		return nil, fmt.Errorf("invalid log level %q: %w", level, err)
	}

	// Create config based on format (case-insensitive)
	var cfg zap.Config
	switch strings.ToLower(format) {
	case "json":
		cfg = zap.NewProductionConfig()
	case "console":
		cfg = zap.NewDevelopmentConfig()
		cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	default:
		return nil, fmt.Errorf("invalid log format %q, must be 'json' or 'console'", format)
	}

	// Set level
	cfg.Level = zap.NewAtomicLevelAt(zapLevel)

	// Build logger
	logger, err := cfg.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build logger: %w", err)
	}

	return logger, nil
}
