package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"

	"github.com/fulgidus/libreseed/seeder/internal/config"
	"github.com/fulgidus/libreseed/seeder/internal/torrent"
)

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

	// TODO Week 2-3: Initialize DHT (integrated in engine)
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

	// Stop torrent engine
	if err := engine.Stop(shutdownCtx); err != nil {
		logger.Error("Error stopping torrent engine", zap.Error(err))
	}

	// TODO: Shutdown additional components
	// - Close DHT (integrated in engine)
	// - Flush metrics

	logger.Info("Seeder service stopped")
	return nil
}
