// Package cli provides command-line interface commands for the seeder.
package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"github.com/fulgidus/libreseed/seeder/internal/config"
	"github.com/fulgidus/libreseed/seeder/internal/manifest"
	"github.com/fulgidus/libreseed/seeder/internal/torrent"
)

var (
	packageTgz      string
	minimalManifest string
	packageTimeout  time.Duration
)

// addPackageCmd represents the add-package command for LibreSeed packages
var addPackageCmd = &cobra.Command{
	Use:   "add-package",
	Short: "Add a LibreSeed package to the seeder",
	Long: `Add a validated LibreSeed package to the seeder.

This command performs comprehensive validation of the dual-manifest architecture:
  1. Validates the minimal manifest structure
  2. Extracts and validates the full manifest from the .tgz
  3. Verifies all cryptographic signatures
  4. Validates content integrity
  5. Adds the package to the seeder if all checks pass

Examples:
  # Add a LibreSeed package
  seeder add-package --package hello-world@1.0.0.tgz --manifest hello-world@1.0.0.minimal.json`,
	RunE: runAddPackage,
}

func init() {
	rootCmd.AddCommand(addPackageCmd)

	addPackageCmd.Flags().StringVarP(&packageTgz, "package", "p", "", "Path to the .tgz package file (required)")
	addPackageCmd.Flags().StringVarP(&minimalManifest, "manifest", "m", "", "Path to the minimal manifest JSON file (required)")
	addPackageCmd.Flags().DurationVarP(&packageTimeout, "timeout", "t", 60*time.Second, "Timeout for adding package")

	addPackageCmd.MarkFlagRequired("package")
	addPackageCmd.MarkFlagRequired("manifest")
}

func runAddPackage(cmd *cobra.Command, args []string) error {
	logger.Info("Starting LibreSeed package validation",
		zap.String("package", packageTgz),
		zap.String("manifest", minimalManifest))

	// 1. Load and parse the minimal manifest
	minimalManifestData, err := os.ReadFile(minimalManifest)
	if err != nil {
		return fmt.Errorf("failed to read minimal manifest: %w", err)
	}

	var minimal manifest.MinimalManifest
	if err := json.Unmarshal(minimalManifestData, &minimal); err != nil {
		return fmt.Errorf("failed to parse minimal manifest: %w", err)
	}

	logger.Info("Parsed minimal manifest",
		zap.String("name", minimal.Name),
		zap.String("version", minimal.Version))

	// 2. Validate the package (dual-manifest validation)
	logger.Info("Validating package integrity and signatures...")
	fullManifest, err := manifest.ValidatePackage(packageTgz, &minimal)
	if err != nil {
		return fmt.Errorf("package validation failed: %w", err)
	}

	logger.Info("✓ Package validation successful",
		zap.String("package", fmt.Sprintf("%s@%s", minimal.Name, minimal.Version)),
		zap.Int("files", len(fullManifest.Files)),
		zap.String("description", fullManifest.Description))

	// 3. Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// 4. Create torrent engine
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

	// 5. Start the engine
	logger.Info("Starting torrent engine...")
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

	// 6. Create a context with timeout for the add operation
	addCtx, addCancel := context.WithTimeout(ctx, packageTimeout)
	defer addCancel()

	// 7. Add the validated package as a torrent
	// This will create a .torrent structure from the .tgz and add it to the engine
	logger.Info("Adding package to seeder...", zap.String("path", packageTgz))
	handle, err := engine.AddPackage(addCtx, packageTgz)
	if err != nil {
		return fmt.Errorf("failed to add package to seeder: %w", err)
	}

	// 8. Print success information
	fmt.Println()
	fmt.Println("✓ Successfully added LibreSeed package:")
	fmt.Printf("  Package:     %s@%s\n", minimal.Name, minimal.Version)
	fmt.Printf("  Description: %s\n", fullManifest.Description)
	fmt.Printf("  Files:       %d\n", len(fullManifest.Files))
	fmt.Printf("  Name:        %s\n", handle.Name)
	fmt.Printf("  InfoHash:    %s\n", handle.InfoHash)
	fmt.Printf("  Infohash:    %s (manifest)\n", minimal.Infohash)
	fmt.Printf("  Publisher:   %s...\n", minimal.PubKey[:20])
	fmt.Printf("  Added At:    %s\n", handle.AddedAt.Format(time.RFC3339))
	fmt.Println()
	fmt.Println("Package is now being seeded and announced to the DHT.")

	return nil
}
