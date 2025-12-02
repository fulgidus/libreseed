// Package cli provides command-line interface commands for the seeder.
package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"github.com/fulgidus/libreseed/seeder/internal/config"
	"github.com/fulgidus/libreseed/seeder/internal/torrent"
)

var (
	statusInfoHash string
	statusTimeout  time.Duration
)

// statusCmd represents the status command
var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show seeder status",
	Long: `Show the current status of the seeder engine or a specific torrent.

Examples:
  # Show engine status
  seeder status

  # Show status of a specific torrent
  seeder status --infohash "abc123..."`,
	RunE: runStatus,
}

func init() {
	rootCmd.AddCommand(statusCmd)

	statusCmd.Flags().StringVarP(&statusInfoHash, "infohash", "i", "", "Show status of a specific torrent by info hash")
	statusCmd.Flags().DurationVarP(&statusTimeout, "timeout", "t", 30*time.Second, "Timeout for operation")
}

func runStatus(cmd *cobra.Command, args []string) error {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Create engine
	engine := torrent.NewEngine(cfg, logger)

	// Setup context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle interrupt signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		logger.Info("Received interrupt signal, cancelling...")
		cancel()
	}()

	// Start the engine
	if err := engine.Start(ctx); err != nil {
		return fmt.Errorf("failed to start engine: %w", err)
	}

	// Ensure engine is stopped on exit
	defer func() {
		stopCtx, stopCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer stopCancel()
		if err := engine.Stop(stopCtx); err != nil {
			logger.Error("Failed to stop engine", zap.Error(err))
		}
	}()

	// If a specific torrent is requested, show its status
	if statusInfoHash != "" {
		return showTorrentStatus(engine, statusInfoHash)
	}

	// Otherwise, show engine status
	return showEngineStatus(engine)
}

func showEngineStatus(engine *torrent.Engine) error {
	stats := engine.Stats()

	fmt.Println("=== Seeder Engine Status ===")
	fmt.Printf("  State:                %s\n", stats.State.String())
	if !stats.StartedAt.IsZero() {
		fmt.Printf("  Started At:           %s\n", stats.StartedAt.Format(time.RFC3339))
		fmt.Printf("  Uptime:               %s\n", time.Since(stats.StartedAt).Round(time.Second))
	}
	fmt.Println()
	fmt.Println("=== Torrent Summary ===")
	fmt.Printf("  Total Torrents:       %d\n", stats.TotalTorrents)
	fmt.Printf("  Active Torrents:      %d\n", stats.ActiveTorrents)
	fmt.Printf("  Seeding:              %d\n", stats.SeedingTorrents)
	fmt.Printf("  Downloading:          %d\n", stats.DownloadingTorrents)
	fmt.Println()
	fmt.Println("=== Transfer Statistics ===")
	fmt.Printf("  Total Uploaded:       %s\n", formatBytes(stats.TotalBytesUploaded))
	fmt.Printf("  Total Downloaded:     %s\n", formatBytes(stats.TotalBytesDownloaded))
	fmt.Printf("  Connected Peers:      %d\n", stats.TotalConnectedPeers)

	return nil
}

func showTorrentStatus(engine *torrent.Engine, infoHash string) error {
	handle, err := engine.GetTorrent(infoHash)
	if err != nil {
		return fmt.Errorf("torrent not found: %w", err)
	}

	stats := handle.Stats()

	fmt.Println("=== Torrent Status ===")
	fmt.Printf("  Name:                 %s\n", stats.Name)
	fmt.Printf("  Info Hash:            %s\n", stats.InfoHash)
	fmt.Printf("  State:                %s\n", stats.State.String())
	fmt.Printf("  Added At:             %s\n", stats.AddedAt.Format(time.RFC3339))
	fmt.Println()
	fmt.Println("=== Progress ===")
	fmt.Printf("  Progress:             %.2f%%\n", stats.Progress)
	fmt.Printf("  Completed:            %s / %s\n", formatBytes(stats.BytesCompleted), formatBytes(stats.TotalBytes))
	fmt.Printf("  Pieces:               %d / %d\n", stats.PiecesCompleted, stats.NumPieces)
	if stats.PieceLength > 0 {
		fmt.Printf("  Piece Size:           %s\n", formatBytes(stats.PieceLength))
	}
	fmt.Println()
	fmt.Println("=== Transfer Statistics ===")
	fmt.Printf("  Uploaded:             %s\n", formatBytes(stats.BytesUploaded))
	fmt.Printf("  Downloaded:           %s\n", formatBytes(stats.BytesDownloaded))
	fmt.Println()
	fmt.Println("=== Peers ===")
	fmt.Printf("  Connected:            %d\n", stats.ConnectedPeers)
	fmt.Printf("  Total Known:          %d\n", stats.TotalPeers)

	return nil
}
