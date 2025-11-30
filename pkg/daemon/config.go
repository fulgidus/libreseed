// Package daemon provides daemon lifecycle management and configuration.
package daemon

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// DaemonConfig holds the configuration for the libreseed daemon.
type DaemonConfig struct {
	// ListenAddr is the HTTP server address (default: "127.0.0.1:8080")
	ListenAddr string `yaml:"listen_addr"`

	// StorageDir is where packages are stored (default: ~/.local/share/libreseed/storage)
	StorageDir string `yaml:"storage_dir"`

	// DHTPort is the UDP port for DHT operations (default: 6881)
	DHTPort int `yaml:"dht_port"`

	// DHTBootstrapNodes is the list of DHT bootstrap nodes
	DHTBootstrapNodes []string `yaml:"dht_bootstrap_nodes"`

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
		ListenAddr: "127.0.0.1:9091",
		StorageDir: filepath.Join(homeDir, ".local", "share", "libreseed", "storage"),
		DHTPort:    6881,
		DHTBootstrapNodes: []string{
			"router.bittorrent.com:6881",
			"dht.transmissionbt.com:2710",
			"router.utorrent.com:6881",
		},
		MaxUploadRate:    0, // unlimited
		MaxDownloadRate:  0, // unlimited
		MaxConnections:   100,
		EnableDHT:        true,
		EnablePEX:        true,
		AnnounceInterval: 30 * time.Minute,
		LogLevel:         "info",
	}
}

// LoadFromEnv applies environment variable overrides to the configuration.
// This follows the 12-factor app methodology where environment variables
// take precedence over file-based configuration.
//
// Supported environment variables:
//   - LIBRESEED_LISTEN_ADDR: HTTP server address
//   - LIBRESEED_STORAGE_DIR: Storage directory path
//   - LIBRESEED_DHT_PORT: DHT UDP port
//   - LIBRESEED_DHT_BOOTSTRAP_NODES: Comma-separated list of bootstrap nodes
//   - LIBRESEED_MAX_UPLOAD_RATE: Maximum upload rate in bytes/sec
//   - LIBRESEED_MAX_DOWNLOAD_RATE: Maximum download rate in bytes/sec
//   - LIBRESEED_MAX_CONNECTIONS: Maximum peer connections
//   - LIBRESEED_ENABLE_DHT: Enable DHT (true/false)
//   - LIBRESEED_ENABLE_PEX: Enable PEX (true/false)
//   - LIBRESEED_ANNOUNCE_INTERVAL: Announce interval (e.g., "30m", "1h")
//   - LIBRESEED_LOG_LEVEL: Log level (debug/info/warn/error)
func (c *DaemonConfig) LoadFromEnv() error {
	if val := os.Getenv("LIBRESEED_LISTEN_ADDR"); val != "" {
		c.ListenAddr = val
	}

	if val := os.Getenv("LIBRESEED_STORAGE_DIR"); val != "" {
		c.StorageDir = val
	}

	if val := os.Getenv("LIBRESEED_DHT_PORT"); val != "" {
		port, err := strconv.Atoi(val)
		if err != nil {
			return fmt.Errorf("invalid LIBRESEED_DHT_PORT: %w", err)
		}
		c.DHTPort = port
	}

	if val := os.Getenv("LIBRESEED_DHT_BOOTSTRAP_NODES"); val != "" {
		nodes := strings.Split(val, ",")
		// Trim whitespace from each node
		for i := range nodes {
			nodes[i] = strings.TrimSpace(nodes[i])
		}
		c.DHTBootstrapNodes = nodes
	}

	if val := os.Getenv("LIBRESEED_MAX_UPLOAD_RATE"); val != "" {
		rate, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid LIBRESEED_MAX_UPLOAD_RATE: %w", err)
		}
		c.MaxUploadRate = rate
	}

	if val := os.Getenv("LIBRESEED_MAX_DOWNLOAD_RATE"); val != "" {
		rate, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid LIBRESEED_MAX_DOWNLOAD_RATE: %w", err)
		}
		c.MaxDownloadRate = rate
	}

	if val := os.Getenv("LIBRESEED_MAX_CONNECTIONS"); val != "" {
		conns, err := strconv.Atoi(val)
		if err != nil {
			return fmt.Errorf("invalid LIBRESEED_MAX_CONNECTIONS: %w", err)
		}
		c.MaxConnections = conns
	}

	if val := os.Getenv("LIBRESEED_ENABLE_DHT"); val != "" {
		enabled, err := strconv.ParseBool(val)
		if err != nil {
			return fmt.Errorf("invalid LIBRESEED_ENABLE_DHT: %w", err)
		}
		c.EnableDHT = enabled
	}

	if val := os.Getenv("LIBRESEED_ENABLE_PEX"); val != "" {
		enabled, err := strconv.ParseBool(val)
		if err != nil {
			return fmt.Errorf("invalid LIBRESEED_ENABLE_PEX: %w", err)
		}
		c.EnablePEX = enabled
	}

	if val := os.Getenv("LIBRESEED_ANNOUNCE_INTERVAL"); val != "" {
		interval, err := time.ParseDuration(val)
		if err != nil {
			return fmt.Errorf("invalid LIBRESEED_ANNOUNCE_INTERVAL: %w", err)
		}
		c.AnnounceInterval = interval
	}

	if val := os.Getenv("LIBRESEED_LOG_LEVEL"); val != "" {
		c.LogLevel = strings.ToLower(val)
	}

	return nil
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

	if c.EnableDHT && len(c.DHTBootstrapNodes) == 0 {
		return fmt.Errorf("dht_bootstrap_nodes cannot be empty when DHT is enabled")
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
