package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/spf13/viper"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	// Test server defaults
	if cfg.Server.MetricsPort != 9090 {
		t.Errorf("expected metrics port 9090, got %d", cfg.Server.MetricsPort)
	}
	if cfg.Server.HealthPort != 8080 {
		t.Errorf("expected health port 8080, got %d", cfg.Server.HealthPort)
	}

	// Test network defaults
	if cfg.Network.BindAddress != "0.0.0.0" {
		t.Errorf("expected bind address 0.0.0.0, got %s", cfg.Network.BindAddress)
	}
	if cfg.Network.Port != 6881 {
		t.Errorf("expected port 6881, got %d", cfg.Network.Port)
	}
	if !cfg.Network.EnableIPv6 {
		t.Error("expected IPv6 to be enabled by default")
	}

	// Test DHT defaults
	if !cfg.DHT.Enabled {
		t.Error("expected DHT to be enabled by default")
	}
	if len(cfg.DHT.BootstrapPeers) == 0 {
		t.Error("expected default bootstrap peers")
	}

	// Test storage defaults
	if cfg.Storage.DataDir != "./data" {
		t.Errorf("expected data dir ./data, got %s", cfg.Storage.DataDir)
	}
	if cfg.Storage.MaxDiskUsage != "50GB" {
		t.Errorf("expected max disk usage 50GB, got %s", cfg.Storage.MaxDiskUsage)
	}
	if cfg.Storage.CacheSizeMB != 256 {
		t.Errorf("expected cache size 256MB, got %d", cfg.Storage.CacheSizeMB)
	}

	// Test limits defaults
	if cfg.Limits.MaxUploadKBps != 0 {
		t.Errorf("expected unlimited upload (0), got %d", cfg.Limits.MaxUploadKBps)
	}
	if cfg.Limits.MaxDownloadKBps != 0 {
		t.Errorf("expected unlimited download (0), got %d", cfg.Limits.MaxDownloadKBps)
	}
	if cfg.Limits.MaxConnections != 200 {
		t.Errorf("expected max connections 200, got %d", cfg.Limits.MaxConnections)
	}
	if cfg.Limits.MaxActiveTorrents != 100 {
		t.Errorf("expected max active torrents 100, got %d", cfg.Limits.MaxActiveTorrents)
	}

	// Test manifest defaults
	if cfg.Manifest.Source != "" {
		t.Errorf("expected empty manifest source, got %s", cfg.Manifest.Source)
	}
	if cfg.Manifest.RefreshInterval != 1*time.Hour {
		t.Errorf("expected refresh interval 1h, got %v", cfg.Manifest.RefreshInterval)
	}

	// Test log defaults
	if cfg.Log.Level != "info" {
		t.Errorf("expected log level info, got %s", cfg.Log.Level)
	}
	if cfg.Log.Format != "json" {
		t.Errorf("expected log format json, got %s", cfg.Log.Format)
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name      string
		cfg       *Config
		wantError bool
	}{
		{
			name:      "valid default config",
			cfg:       DefaultConfig(),
			wantError: false,
		},
		{
			name: "invalid port - too low",
			cfg: &Config{
				Server: ServerConfig{
					MetricsPort: 9090,
					HealthPort:  8080,
				},
				Network: NetworkConfig{
					BindAddress: "0.0.0.0",
					Port:        1023, // Invalid: below 1024
					EnableIPv6:  true,
				},
				DHT: DHTConfig{
					Enabled: true,
					BootstrapPeers: []string{
						"router.bittorrent.com:6881",
					},
				},
				Storage: StorageConfig{
					DataDir:      "./data",
					MaxDiskUsage: "50GB",
					CacheSizeMB:  256,
				},
				Limits: LimitsConfig{
					MaxUploadKBps:     0,
					MaxDownloadKBps:   0,
					MaxConnections:    200,
					MaxActiveTorrents: 100,
				},
				Manifest: ManifestConfig{
					Source:          "",
					RefreshInterval: 1 * time.Hour,
				},
				Log: LogConfig{
					Level:  "info",
					Format: "json",
				},
			},
			wantError: true,
		},
		{
			name: "invalid port - too high",
			cfg: &Config{
				Server: ServerConfig{
					MetricsPort: 9090,
					HealthPort:  8080,
				},
				Network: NetworkConfig{
					BindAddress: "0.0.0.0",
					Port:        65536, // Invalid: above 65535
					EnableIPv6:  true,
				},
				DHT: DHTConfig{
					Enabled: true,
					BootstrapPeers: []string{
						"router.bittorrent.com:6881",
					},
				},
				Storage: StorageConfig{
					DataDir:      "./data",
					MaxDiskUsage: "50GB",
					CacheSizeMB:  256,
				},
				Limits: LimitsConfig{
					MaxUploadKBps:     0,
					MaxDownloadKBps:   0,
					MaxConnections:    200,
					MaxActiveTorrents: 100,
				},
				Manifest: ManifestConfig{
					Source:          "",
					RefreshInterval: 1 * time.Hour,
				},
				Log: LogConfig{
					Level:  "info",
					Format: "json",
				},
			},
			wantError: true,
		},
		{
			name: "invalid log level",
			cfg: &Config{
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
					},
				},
				Storage: StorageConfig{
					DataDir:      "./data",
					MaxDiskUsage: "50GB",
					CacheSizeMB:  256,
				},
				Limits: LimitsConfig{
					MaxUploadKBps:     0,
					MaxDownloadKBps:   0,
					MaxConnections:    200,
					MaxActiveTorrents: 100,
				},
				Manifest: ManifestConfig{
					Source:          "",
					RefreshInterval: 1 * time.Hour,
				},
				Log: LogConfig{
					Level:  "invalid", // Invalid log level
					Format: "json",
				},
			},
			wantError: true,
		},
		{
			name: "invalid log format",
			cfg: &Config{
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
					},
				},
				Storage: StorageConfig{
					DataDir:      "./data",
					MaxDiskUsage: "50GB",
					CacheSizeMB:  256,
				},
				Limits: LimitsConfig{
					MaxUploadKBps:     0,
					MaxDownloadKBps:   0,
					MaxConnections:    200,
					MaxActiveTorrents: 100,
				},
				Manifest: ManifestConfig{
					Source:          "",
					RefreshInterval: 1 * time.Hour,
				},
				Log: LogConfig{
					Level:  "info",
					Format: "invalid", // Invalid log format
				},
			},
			wantError: true,
		},
		{
			name: "empty data directory",
			cfg: &Config{
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
					},
				},
				Storage: StorageConfig{
					DataDir:      "", // Empty data dir
					MaxDiskUsage: "50GB",
					CacheSizeMB:  256,
				},
				Limits: LimitsConfig{
					MaxUploadKBps:     0,
					MaxDownloadKBps:   0,
					MaxConnections:    200,
					MaxActiveTorrents: 100,
				},
				Manifest: ManifestConfig{
					Source:          "",
					RefreshInterval: 1 * time.Hour,
				},
				Log: LogConfig{
					Level:  "info",
					Format: "json",
				},
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if (err != nil) != tt.wantError {
				t.Errorf("Validate() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestLoadConfig(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "seeder.yaml")

	configContent := `
server:
  metrics_port: 9091
  health_port: 8081

network:
  bind_address: "127.0.0.1"
  port: 7777
  enable_ipv6: false

dht:
  enabled: true
  bootstrap_peers:
    - "test.node.com:6881"

storage:
  data_dir: "/tmp/test-data"
  max_disk_usage: "10GB"
  cache_size_mb: 128

limits:
  max_upload_kbps: 1024
  max_download_kbps: 2048
  max_connections: 150
  max_active_torrents: 50

manifest:
  source: "https://test.com/manifest.yaml"
  refresh_interval: 30m

log:
  level: "debug"
  format: "console"
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	// Configure viper to use the test config file
	viper.Reset()
	viper.SetConfigFile(configPath)

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	// Verify loaded values
	if cfg.Server.MetricsPort != 9091 {
		t.Errorf("expected metrics port 9091, got %d", cfg.Server.MetricsPort)
	}
	if cfg.Server.HealthPort != 8081 {
		t.Errorf("expected health port 8081, got %d", cfg.Server.HealthPort)
	}
	if cfg.Network.BindAddress != "127.0.0.1" {
		t.Errorf("expected bind address 127.0.0.1, got %s", cfg.Network.BindAddress)
	}
	if cfg.Network.Port != 7777 {
		t.Errorf("expected port 7777, got %d", cfg.Network.Port)
	}
	if cfg.Network.EnableIPv6 {
		t.Error("expected IPv6 to be disabled")
	}
	if len(cfg.DHT.BootstrapPeers) != 1 {
		t.Errorf("expected 1 bootstrap peer, got %d", len(cfg.DHT.BootstrapPeers))
	}
	if cfg.Storage.DataDir != "/tmp/test-data" {
		t.Errorf("expected data dir /tmp/test-data, got %s", cfg.Storage.DataDir)
	}
	if cfg.Storage.MaxDiskUsage != "10GB" {
		t.Errorf("expected max disk usage 10GB, got %s", cfg.Storage.MaxDiskUsage)
	}
	if cfg.Storage.CacheSizeMB != 128 {
		t.Errorf("expected cache size 128MB, got %d", cfg.Storage.CacheSizeMB)
	}
	if cfg.Limits.MaxUploadKBps != 1024 {
		t.Errorf("expected max upload 1024 KB/s, got %d", cfg.Limits.MaxUploadKBps)
	}
	if cfg.Limits.MaxDownloadKBps != 2048 {
		t.Errorf("expected max download 2048 KB/s, got %d", cfg.Limits.MaxDownloadKBps)
	}
	if cfg.Limits.MaxConnections != 150 {
		t.Errorf("expected max connections 150, got %d", cfg.Limits.MaxConnections)
	}
	if cfg.Limits.MaxActiveTorrents != 50 {
		t.Errorf("expected max active torrents 50, got %d", cfg.Limits.MaxActiveTorrents)
	}
	if cfg.Manifest.Source != "https://test.com/manifest.yaml" {
		t.Errorf("expected manifest source https://test.com/manifest.yaml, got %s", cfg.Manifest.Source)
	}
	if cfg.Manifest.RefreshInterval != 30*time.Minute {
		t.Errorf("expected refresh interval 30m, got %v", cfg.Manifest.RefreshInterval)
	}
	if cfg.Log.Level != "debug" {
		t.Errorf("expected log level debug, got %s", cfg.Log.Level)
	}
	if cfg.Log.Format != "console" {
		t.Errorf("expected log format console, got %s", cfg.Log.Format)
	}
}

func TestLoadConfigNonExistent(t *testing.T) {
	// Reset viper and configure it to look for a non-existent file
	viper.Reset()
	viper.SetConfigFile("/nonexistent/config.yaml")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() should return default config on missing file, got error: %v", err)
	}

	// Should return default config
	if cfg.Server.MetricsPort != 9090 {
		t.Errorf("expected default metrics port 9090, got %d", cfg.Server.MetricsPort)
	}
	if cfg.Network.BindAddress != "0.0.0.0" {
		t.Errorf("expected default bind address 0.0.0.0, got %s", cfg.Network.BindAddress)
	}
	if cfg.Network.Port != 6881 {
		t.Errorf("expected default port 6881, got %d", cfg.Network.Port)
	}
}
