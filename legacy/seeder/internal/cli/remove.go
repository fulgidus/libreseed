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
	removeInfoHash   string
	removeDeleteData bool
	removeTimeout    time.Duration
)

// removeCmd represents the remove command
var removeCmd = &cobra.Command{
	Use:   "remove",
	Short: "Remove a torrent from the seeder",
	Long: `Remove a torrent from the seeder by its info hash.

Examples:
  # Remove a torrent (keep data files)
  seeder remove --infohash "abc123..."

  # Remove a torrent and delete data files
  seeder remove --infohash "abc123..." --delete-data`,
	RunE: runRemove,
}

func init() {
	rootCmd.AddCommand(removeCmd)

	removeCmd.Flags().StringVarP(&removeInfoHash, "infohash", "i", "", "Info hash of the torrent to remove (required)")
	removeCmd.Flags().BoolVarP(&removeDeleteData, "delete-data", "d", false, "Delete downloaded data files")
	removeCmd.Flags().DurationVarP(&removeTimeout, "timeout", "t", 30*time.Second, "Timeout for operation")

	_ = removeCmd.MarkFlagRequired("infohash")
}

func runRemove(cmd *cobra.Command, args []string) error {
	// Validate info hash is provided
	if removeInfoHash == "" {
		return fmt.Errorf("--infohash is required")
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

	// Create a context with timeout for the remove operation
	removeCtx, removeCancel := context.WithTimeout(ctx, removeTimeout)
	defer removeCancel()

	// Remove the torrent
	logger.Info("Removing torrent",
		zap.String("infohash", removeInfoHash),
		zap.Bool("delete_data", removeDeleteData),
	)

	err = engine.RemoveTorrent(removeCtx, removeInfoHash, removeDeleteData)
	if err != nil {
		return fmt.Errorf("failed to remove torrent: %w", err)
	}

	// Print success information
	fmt.Printf("Successfully removed torrent:\n")
	fmt.Printf("  InfoHash:    %s\n", removeInfoHash)
	fmt.Printf("  DeleteData:  %t\n", removeDeleteData)

	return nil
}
