package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	anacrolixtorrent "github.com/anacrolix/torrent"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"

	"github.com/fulgidus/libreseed/seeder/internal/config"
	"github.com/fulgidus/libreseed/seeder/internal/dht"
	"github.com/fulgidus/libreseed/seeder/internal/torrent"
	"github.com/fulgidus/libreseed/seeder/internal/watcher"
)

// zapLoggerAdapter adapts zap.Logger to the dht.Logger interface.
type zapLoggerAdapter struct {
	logger *zap.Logger
}

// newZapLoggerAdapter creates a new zap logger adapter.
func newZapLoggerAdapter(logger *zap.Logger) dht.Logger {
	return &zapLoggerAdapter{logger: logger}
}

// Debugf implements dht.Logger.
func (a *zapLoggerAdapter) Debugf(format string, args ...interface{}) {
	a.logger.Debug(fmt.Sprintf(format, args...))
}

// Infof implements dht.Logger.
func (a *zapLoggerAdapter) Infof(format string, args ...interface{}) {
	a.logger.Info(fmt.Sprintf(format, args...))
}

// Warnf implements dht.Logger.
func (a *zapLoggerAdapter) Warnf(format string, args ...interface{}) {
	a.logger.Warn(fmt.Sprintf(format, args...))
}

// Errorf implements dht.Logger.
func (a *zapLoggerAdapter) Errorf(format string, args ...interface{}) {
	a.logger.Error(fmt.Sprintf(format, args...))
}

// startCmd represents the start command
var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the seeder service",
	Long: `Start the LibreSeed seeder service to begin seeding content.

The seeder will:
  1. Load configuration from file and environment
  2. Initialize DHT for peer discovery
  3. Load content manifests
  4. Start seeding torrents
  5. Serve metrics and health endpoints`,
	RunE: runStart,
}

func init() {
	rootCmd.AddCommand(startCmd)

	// Start-specific flags (no defaults - will use config defaults from Viper)
	startCmd.Flags().String("data-dir", "", "data directory for torrents and state")
	startCmd.Flags().Int("port", 0, "port for BitTorrent protocol")
	startCmd.Flags().String("bind", "", "bind address for services")

	// Bind flags to viper - only set if user explicitly provides them
	// This allows Viper defaults to work when flags are not provided
	_ = viper.BindPFlag("storage.data_dir", startCmd.Flags().Lookup("data-dir"))
	_ = viper.BindPFlag("network.port", startCmd.Flags().Lookup("port"))
	_ = viper.BindPFlag("network.bind_address", startCmd.Flags().Lookup("bind"))
}

func runStart(cmd *cobra.Command, args []string) error {
	// Get the full configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	logger.Info("Starting LibreSeed Seeder",
		zap.String("version", Version),
		zap.String("data_dir", cfg.Storage.DataDir),
		zap.Int("port", cfg.Network.Port),
	)

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup signal handling
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	// Initialize the torrent engine
	logger.Info("Initializing torrent engine",
		zap.String("bind_address", cfg.Network.BindAddress),
		zap.Int("port", cfg.Network.Port),
	)

	engine := torrent.NewEngine(cfg, logger)

	// Start the torrent engine
	if err := engine.Start(ctx); err != nil {
		return fmt.Errorf("failed to start torrent engine: %w", err)
	}

	// Initialize DHT manager if enabled
	var dhtManager *dht.Manager
	if cfg.DHT.Enabled {
		logger.Info("Initializing DHT manager (using torrent engine's DHT)")

		// Get DHT server from torrent engine's client
		client := engine.Client()
		if client == nil {
			logger.Error("Failed to get torrent client from engine")
			logger.Warn("Continuing without DHT - peer discovery will be limited")
		} else {
			// Get DHT servers from the client
			dhtServers := client.DhtServers()
			if len(dhtServers) == 0 {
				logger.Error("No DHT servers available from torrent client")
				logger.Warn("Continuing without DHT - peer discovery will be limited")
			} else {
				// Extract the anacrolix DHT server from the wrapper
				wrapper, ok := dhtServers[0].(anacrolixtorrent.AnacrolixDhtServerWrapper)
				if !ok {
					logger.Error("DHT server is not of expected type")
					logger.Warn("Continuing without DHT - peer discovery will be limited")
				} else {
					// Create DHT configuration (without Port and BootstrapNodes)
					dhtCfg := dht.ManagerConfig{
						DisableReannounce: false,
						Logger:            newZapLoggerAdapter(logger),
					}

					// Create DHT manager with the existing DHT server
					var err error
					dhtManager, err = dht.NewManager(wrapper.Server, dhtCfg)
					if err != nil {
						logger.Error("Failed to create DHT manager", zap.Error(err))
						logger.Warn("Continuing without DHT - peer discovery will be limited")
						dhtManager = nil
					} else {
						// Start DHT manager (re-announce loop only)
						if err := dhtManager.Start(); err != nil {
							// Non-fatal error - continue without DHT
							logger.Error("Failed to start DHT manager", zap.Error(err))
							logger.Warn("Continuing without DHT - peer discovery will be limited")
							dhtManager = nil
						} else {
							logger.Info("DHT manager started successfully (reusing torrent engine's DHT)")
						}
					}
				}
			}
		}
	} else {
		logger.Info("DHT disabled in configuration")
	}

	// Initialize file watcher if watch directory is configured
	var fileWatcher *watcher.Watcher
	if cfg.Manifest.WatchDir != "" {
		logger.Info("Initializing file watcher",
			zap.String("watch_dir", cfg.Manifest.WatchDir))

		var err error
		fileWatcher, err = watcher.NewWatcher(cfg, logger, engine)
		if err != nil {
			logger.Error("Failed to create file watcher", zap.Error(err))
			logger.Warn("Continuing without file watcher")
		} else {
			if err := fileWatcher.Start(ctx); err != nil {
				logger.Error("Failed to start file watcher", zap.Error(err))
				logger.Warn("Continuing without file watcher")
			} else {
				logger.Info("File watcher started successfully")
			}
		}
	} else {
		logger.Info("File watcher disabled (no watch directory configured)")
	}

	// TODO Week 3-4: Load manifests
	// TODO Week 6: Start metrics server

	logger.Info("Seeder service started successfully")
	logger.Info("Press Ctrl+C to stop")

	// Wait for shutdown signal
	select {
	case sig := <-sigCh:
		logger.Info("Received shutdown signal", zap.String("signal", sig.String()))
	case <-ctx.Done():
		logger.Info("Context cancelled")
	}

	// Graceful shutdown
	logger.Info("Shutting down seeder service...")

	// Create shutdown context with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// Stop file watcher if running
	if fileWatcher != nil {
		logger.Info("Stopping file watcher...")
		if err := fileWatcher.Stop(); err != nil {
			logger.Error("Error stopping file watcher", zap.Error(err))
		} else {
			logger.Info("File watcher stopped successfully")
		}
	}

	// Stop DHT manager if running
	if dhtManager != nil {
		logger.Info("Stopping DHT manager...")
		if err := dhtManager.Stop(); err != nil {
			logger.Error("Error stopping DHT manager", zap.Error(err))
		} else {
			logger.Info("DHT manager stopped successfully")
		}
	}

	// Stop torrent engine
	if err := engine.Stop(shutdownCtx); err != nil {
		logger.Error("Error stopping torrent engine", zap.Error(err))
	}

	// TODO Week 6: Flush metrics

	logger.Info("Seeder service stopped")
	return nil
}
