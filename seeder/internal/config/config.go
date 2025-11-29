package config

import (
	"errors"
	"fmt"
	"os"
	"slices"
	"time"

	"github.com/spf13/viper"
)

// Config represents the complete seeder configuration
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Network  NetworkConfig  `mapstructure:"network"`
	DHT      DHTConfig      `mapstructure:"dht"`
	Storage  StorageConfig  `mapstructure:"storage"`
	Limits   LimitsConfig   `mapstructure:"limits"`
	Manifest ManifestConfig `mapstructure:"manifest"`
	Log      LogConfig      `mapstructure:"log"`
}

// ServerConfig contains server-level settings
type ServerConfig struct {
	MetricsPort int `mapstructure:"metrics_port"`
	HealthPort  int `mapstructure:"health_port"`
}

// NetworkConfig contains network settings
type NetworkConfig struct {
	BindAddress string `mapstructure:"bind_address"`
	Port        int    `mapstructure:"port"`
	EnableIPv6  bool   `mapstructure:"enable_ipv6"`
}

// DHTConfig contains DHT settings
type DHTConfig struct {
	Enabled        bool     `mapstructure:"enabled"`
	BootstrapPeers []string `mapstructure:"bootstrap_peers"`
}

// StorageConfig contains storage settings
type StorageConfig struct {
	DataDir      string `mapstructure:"data_dir"`
	MaxDiskUsage string `mapstructure:"max_disk_usage"`
	CacheSizeMB  int    `mapstructure:"cache_size_mb"`
}

// LimitsConfig contains bandwidth and connection limits
type LimitsConfig struct {
	MaxUploadKBps     int `mapstructure:"max_upload_kbps"`
	MaxDownloadKBps   int `mapstructure:"max_download_kbps"`
	MaxConnections    int `mapstructure:"max_connections"`
	MaxActiveTorrents int `mapstructure:"max_active_torrents"`
}

// ManifestConfig contains manifest management settings
type ManifestConfig struct {
	Source          string        `mapstructure:"source"`
	RefreshInterval time.Duration `mapstructure:"refresh_interval"`
	WatchDir        string        `mapstructure:"watch_dir"`
}

// LogConfig contains logging settings
type LogConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
}

// DefaultConfig returns a configuration with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			MetricsPort: 9090,
			HealthPort:  8080,
		},
		Network: NetworkConfig{
			BindAddress: "0.0.0.0",
			Port:        6881,
			EnableIPv6:  true,
		},
		DHT: DHTConfig{
			Enabled: true,
			BootstrapPeers: []string{
				"router.bittorrent.com:6881",
				"dht.transmissionbt.com:6881",
			},
		},
		Storage: StorageConfig{
			DataDir:      "./data",
			MaxDiskUsage: "50GB",
			CacheSizeMB:  256,
		},
		Limits: LimitsConfig{
			MaxUploadKBps:     0, // 0 = unlimited
			MaxDownloadKBps:   0,
			MaxConnections:    200,
			MaxActiveTorrents: 100,
		},
		Manifest: ManifestConfig{
			Source:          "",
			RefreshInterval: 1 * time.Hour,
			WatchDir:        "./packages",
		},
		Log: LogConfig{
			Level:  "info",
			Format: "json",
		},
	}
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	// Validate network port
	if c.Network.Port < 1024 || c.Network.Port > 65535 {
		return fmt.Errorf("network.port must be between 1024 and 65535")
	}

	// Validate log level
	validLogLevels := []string{"debug", "info", "warn", "error"}
	if !slices.Contains(validLogLevels, c.Log.Level) {
		return fmt.Errorf("log.level must be one of: debug, info, warn, error")
	}

	// Validate log format
	if c.Log.Format != "json" && c.Log.Format != "console" {
		return fmt.Errorf("log.format must be 'json' or 'console'")
	}

	// Validate data directory
	if c.Storage.DataDir == "" {
		return fmt.Errorf("storage.data_dir cannot be empty")
	}

	// Check if data directory exists or can be created
	if _, err := os.Stat(c.Storage.DataDir); os.IsNotExist(err) {
		if err := os.MkdirAll(c.Storage.DataDir, 0755); err != nil {
			return fmt.Errorf("failed to create data directory: %w", err)
		}
	}

	return nil
}

// LoadConfig loads configuration from file, environment, and flags
func LoadConfig() (*Config, error) {
	// Get defaults and register them with Viper
	defaults := DefaultConfig()

	// Set Viper defaults so they're available during unmarshal
	// Server
	viper.SetDefault("server.metrics_port", defaults.Server.MetricsPort)
	viper.SetDefault("server.health_port", defaults.Server.HealthPort)

	// Network
	viper.SetDefault("network.bind_address", defaults.Network.BindAddress)
	viper.SetDefault("network.port", defaults.Network.Port)
	viper.SetDefault("network.enable_ipv6", defaults.Network.EnableIPv6)

	// DHT
	viper.SetDefault("dht.enabled", defaults.DHT.Enabled)
	viper.SetDefault("dht.bootstrap_peers", defaults.DHT.BootstrapPeers)

	// Storage
	viper.SetDefault("storage.data_dir", defaults.Storage.DataDir)
	viper.SetDefault("storage.max_disk_usage", defaults.Storage.MaxDiskUsage)
	viper.SetDefault("storage.cache_size_mb", defaults.Storage.CacheSizeMB)

	// Limits
	viper.SetDefault("limits.max_upload_kbps", defaults.Limits.MaxUploadKBps)
	viper.SetDefault("limits.max_download_kbps", defaults.Limits.MaxDownloadKBps)
	viper.SetDefault("limits.max_connections", defaults.Limits.MaxConnections)
	viper.SetDefault("limits.max_active_torrents", defaults.Limits.MaxActiveTorrents)

	// Manifest
	viper.SetDefault("manifest.source", defaults.Manifest.Source)
	viper.SetDefault("manifest.refresh_interval", defaults.Manifest.RefreshInterval)
	viper.SetDefault("manifest.watch_dir", defaults.Manifest.WatchDir)

	// Log
	viper.SetDefault("log.level", defaults.Log.Level)
	viper.SetDefault("log.format", defaults.Log.Format)

	// Try to read config file
	if err := viper.ReadInConfig(); err != nil {
		// Check if it's a "config file not found" error
		// This covers both ConfigFileNotFoundError (search paths) and file not exist (SetConfigFile)
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok && !errors.Is(err, os.ErrNotExist) {
			// Config file was found but another error occurred (e.g., parse error, permission denied)
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
		// Config file not found; ignore error and use defaults
	}

	// Unmarshal into config struct (now with Viper-managed defaults)
	cfg := &Config{}
	if err := viper.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return cfg, nil
}
