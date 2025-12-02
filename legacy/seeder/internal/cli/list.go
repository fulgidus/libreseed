// Package cli provides command-line interface commands for the seeder.
package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"github.com/fulgidus/libreseed/seeder/internal/config"
	"github.com/fulgidus/libreseed/seeder/internal/torrent"
)

var (
	listVerbose bool
	listTimeout time.Duration
)

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all torrents in the seeder",
	Long: `List all torrents currently managed by the seeder.

Examples:
  # List all torrents
  seeder list

  # List with verbose output
  seeder list --verbose`,
	RunE: runList,
}

func init() {
	rootCmd.AddCommand(listCmd)

	listCmd.Flags().BoolVarP(&listVerbose, "verbose", "v", false, "Show verbose output with more details")
	listCmd.Flags().DurationVarP(&listTimeout, "timeout", "t", 30*time.Second, "Timeout for operation")
}

func runList(cmd *cobra.Command, args []string) error {
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

	// Get all torrents
	torrents := engine.ListTorrents()

	if len(torrents) == 0 {
		fmt.Println("No torrents found.")
		return nil
	}

	// Create a tabwriter for aligned output
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	if listVerbose {
		// Verbose output with more columns
		fmt.Fprintln(w, "INFOHASH\tNAME\tSTATE\tPROGRESS\tPEERS\tUPLOADED\tDOWNLOADED\tADDED")
		fmt.Fprintln(w, "--------\t----\t-----\t--------\t-----\t--------\t----------\t-----")

		for _, handle := range torrents {
			stats := handle.Stats()
			fmt.Fprintf(w, "%s\t%s\t%s\t%.1f%%\t%d/%d\t%s\t%s\t%s\n",
				truncateString(stats.InfoHash, 16),
				truncateString(stats.Name, 30),
				stats.State.String(),
				stats.Progress,
				stats.ConnectedPeers,
				stats.TotalPeers,
				formatBytes(stats.BytesUploaded),
				formatBytes(stats.BytesDownloaded),
				stats.AddedAt.Format("2006-01-02 15:04"),
			)
		}
	} else {
		// Simple output
		fmt.Fprintln(w, "INFOHASH\tNAME\tSTATE\tPROGRESS")
		fmt.Fprintln(w, "--------\t----\t-----\t--------")

		for _, handle := range torrents {
			stats := handle.Stats()
			fmt.Fprintf(w, "%s\t%s\t%s\t%.1f%%\n",
				truncateString(stats.InfoHash, 16),
				truncateString(stats.Name, 40),
				stats.State.String(),
				stats.Progress,
			)
		}
	}

	w.Flush()

	fmt.Printf("\nTotal: %d torrent(s)\n", len(torrents))

	return nil
}

// truncateString truncates a string to maxLen characters and adds "..." if truncated
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

// formatBytes formats bytes into a human-readable string
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
