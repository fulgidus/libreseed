// Package daemon provides daemon lifecycle management and configuration.
package daemon

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// DaemonConfig holds the configuration for the libreseed daemon.
type DaemonConfig struct {
	// ListenAddr is the HTTP server address (default: "127.0.0.1:8080")
	ListenAddr string `yaml:"listen_addr"`

	// StorageDir is where packages are stored (default: ~/.libreseed/storage)
	StorageDir string `yaml:"storage_dir"`

	// DHTPort is the UDP port for DHT operations (default: 6881)
	DHTPort int `yaml:"dht_port"`

	// MaxUploadRate is the maximum upload rate in bytes/sec (0 = unlimited)
	MaxUploadRate int64 `yaml:"max_upload_rate"`

	// MaxDownloadRate is the maximum download rate in bytes/sec (0 = unlimited)
	MaxDownloadRate int64 `yaml:"max_download_rate"`

	// MaxConnections is the maximum number of concurrent peer connections
	MaxConnections int `yaml:"max_connections"`

	// EnableDHT enables or disables DHT participation
	EnableDHT bool `yaml:"enable_dht"`

	// EnablePEX enables or disables Peer Exchange
	EnablePEX bool `yaml:"enable_pex"`

	// AnnounceInterval is how often to announce to trackers
	AnnounceInterval time.Duration `yaml:"announce_interval"`

	// LogLevel is the logging verbosity (debug, info, warn, error)
	LogLevel string `yaml:"log_level"`
}

// DefaultConfig returns a DaemonConfig with sensible defaults.
func DefaultConfig() *DaemonConfig {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}

	return &DaemonConfig{
		ListenAddr:       "127.0.0.1:8080",
		StorageDir:       filepath.Join(homeDir, ".libreseed", "storage"),
		DHTPort:          6881,
		MaxUploadRate:    0, // unlimited
		MaxDownloadRate:  0, // unlimited
		MaxConnections:   100,
		EnableDHT:        true,
		EnablePEX:        true,
		AnnounceInterval: 30 * time.Minute,
		LogLevel:         "info",
	}
}

// Validate checks if the configuration is valid and returns an error if not.
func (c *DaemonConfig) Validate() error {
	if c.ListenAddr == "" {
		return fmt.Errorf("listen_addr cannot be empty")
	}

	if c.StorageDir == "" {
		return fmt.Errorf("storage_dir cannot be empty")
	}

	if c.DHTPort < 1024 || c.DHTPort > 65535 {
		return fmt.Errorf("dht_port must be between 1024 and 65535")
	}

	if c.MaxUploadRate < 0 {
		return fmt.Errorf("max_upload_rate cannot be negative")
	}

	if c.MaxDownloadRate < 0 {
		return fmt.Errorf("max_download_rate cannot be negative")
	}

	if c.MaxConnections < 1 {
		return fmt.Errorf("max_connections must be at least 1")
	}

	if c.AnnounceInterval < time.Minute {
		return fmt.Errorf("announce_interval must be at least 1 minute")
	}

	validLogLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}
	if !validLogLevels[c.LogLevel] {
		return fmt.Errorf("log_level must be one of: debug, info, warn, error")
	}

	return nil
}

// EnsureStorageDir creates the storage directory if it doesn't exist.
func (c *DaemonConfig) EnsureStorageDir() error {
	return os.MkdirAll(c.StorageDir, 0755)
}
