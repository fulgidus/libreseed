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
	torrentFile string
	magnetLink  string
	infoHashHex string
	addTimeout  time.Duration
)

// addCmd represents the add command
var addCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a torrent to the seeder",
	Long: `Add a torrent to the seeder from a .torrent file, magnet link, or info hash.

Examples:
  # Add from a .torrent file
  seeder add --file /path/to/file.torrent

  # Add from a magnet link
  seeder add --magnet "magnet:?xt=urn:btih:..."

  # Add from an info hash (hex)
  seeder add --infohash "abc123..."`,
	RunE: runAdd,
}

func init() {
	rootCmd.AddCommand(addCmd)

	addCmd.Flags().StringVarP(&torrentFile, "file", "f", "", "Path to .torrent file")
	addCmd.Flags().StringVarP(&magnetLink, "magnet", "m", "", "Magnet link")
	addCmd.Flags().StringVarP(&infoHashHex, "infohash", "i", "", "Info hash (hex)")
	addCmd.Flags().DurationVarP(&addTimeout, "timeout", "t", 30*time.Second, "Timeout for adding torrent")
}

func runAdd(cmd *cobra.Command, args []string) error {
	// Validate that exactly one source is provided
	sources := 0
	if torrentFile != "" {
		sources++
	}
	if magnetLink != "" {
		sources++
	}
	if infoHashHex != "" {
		sources++
	}

	if sources == 0 {
		return fmt.Errorf("must specify one of --file, --magnet, or --infohash")
	}
	if sources > 1 {
		return fmt.Errorf("specify only one of --file, --magnet, or --infohash")
	}

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

	// Create a context with timeout for the add operation
	addCtx, addCancel := context.WithTimeout(ctx, addTimeout)
	defer addCancel()

	// Add the torrent based on the source type
	var handle *torrent.TorrentHandle
	switch {
	case torrentFile != "":
		logger.Info("Adding torrent from file", zap.String("path", torrentFile))
		handle, err = engine.AddTorrentFromFile(addCtx, torrentFile)
	case magnetLink != "":
		logger.Info("Adding torrent from magnet link")
		handle, err = engine.AddTorrentFromMagnet(addCtx, magnetLink)
	case infoHashHex != "":
		logger.Info("Adding torrent from info hash", zap.String("infohash", infoHashHex))
		handle, err = engine.AddTorrentFromInfoHash(addCtx, infoHashHex)
	}

	if err != nil {
		return fmt.Errorf("failed to add torrent: %w", err)
	}

	// Print success information
	fmt.Printf("Successfully added torrent:\n")
	fmt.Printf("  Name:      %s\n", handle.Name)
	fmt.Printf("  InfoHash:  %s\n", handle.InfoHash)
	fmt.Printf("  Added At:  %s\n", handle.AddedAt.Format(time.RFC3339))

	return nil
}
