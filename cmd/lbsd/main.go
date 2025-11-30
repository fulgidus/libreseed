package main

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/libreseed/libreseed/pkg/daemon"
)

var version = "dev" // Set via ldflags during build

func main() {
	// Parse command-line flags
	configPath := getDefaultConfigPath()
	showVersion := false
	showHelp := false

	for i := 1; i < len(os.Args); i++ {
		switch os.Args[i] {
		case "--config":
			if i+1 < len(os.Args) {
				configPath = os.Args[i+1]
				i++
			} else {
				fmt.Fprintf(os.Stderr, "Error: --config requires a path argument\n")
				os.Exit(1)
			}
		case "--version", "-v":
			showVersion = true
		case "--help", "-h":
			showHelp = true
		default:
			fmt.Fprintf(os.Stderr, "Unknown option: %s\n", os.Args[i])
			printUsage()
			os.Exit(1)
		}
	}

	if showVersion {
		fmt.Printf("lbsd version %s\n", version)
		os.Exit(0)
	}

	if showHelp {
		printUsage()
		os.Exit(0)
	}

	// Load or create configuration
	config, err := loadOrCreateConfig(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Create daemon instance
	d, err := daemon.New(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to create daemon: %v\n", err)
		os.Exit(1)
	}

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Create shutdown channel for HTTP-initiated shutdown
	shutdownChan := make(chan struct{})

	// Write PID file BEFORE starting daemon to avoid race condition
	// (includes listen address for client discovery)
	if err := writePIDFile(config.ListenAddr); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to write PID file: %v\n", err)
	}

	// Start daemon
	if err := d.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to start daemon: %v\n", err)
		// Remove PID file if daemon fails to start
		removePIDFile()
		os.Exit(1)
	}

	fmt.Printf("LibreSeed Daemon started\n")
	fmt.Printf("HTTP API: %s\n", config.ListenAddr)
	fmt.Printf("DHT Port: %d\n", config.DHTPort)
	fmt.Printf("Storage: %s\n", config.StorageDir)
	fmt.Println("\nPress Ctrl+C to stop")

	// Monitor daemon status in background to detect HTTP-initiated shutdown
	go func() {
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				// Check if daemon was stopped (e.g., via HTTP /shutdown endpoint)
				if d.GetState().Status == "stopped" || d.GetState().Status == "stopping" {
					close(shutdownChan)
					return
				}
			case <-sigChan:
				// Signal received, let main goroutine handle it
				return
			}
		}
	}()

	// Wait for shutdown signal (either OS signal or HTTP-initiated)
	select {
	case <-sigChan:
		fmt.Println("\nShutting down...")
	case <-shutdownChan:
		fmt.Println("\nShutdown requested via API...")
	}

	// Stop daemon
	if err := d.Stop(); err != nil {
		fmt.Fprintf(os.Stderr, "Error during shutdown: %v\n", err)
		// Best effort: try to remove PID file even on error
		removePIDFile()
		os.Exit(1)
	}

	// Remove PID file on clean shutdown
	removePIDFile()

	fmt.Println("Daemon stopped")
}

func loadOrCreateConfig(configPath string) (*daemon.DaemonConfig, error) {
	// Expand home directory if needed
	if len(configPath) >= 2 && configPath[:2] == "~/" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		configPath = filepath.Join(home, configPath[2:])
	}

	// Start with defaults (Tier 1: Defaults)
	config := daemon.DefaultConfig()

	// Try to load config file if it exists (Tier 2: File overrides)
	if _, err := os.Stat(configPath); err == nil {
		// File exists, try to load it
		fileConfig, err := daemon.LoadConfig(configPath)
		if err != nil {
			return nil, fmt.Errorf("config file exists but failed to load: %w", err)
		}
		config = fileConfig
		fmt.Printf("Loaded configuration from: %s\n", configPath)
	} else if !os.IsNotExist(err) {
		// Some other error besides "not exist"
		return nil, fmt.Errorf("failed to check config file: %w", err)
	}
	// If file doesn't exist, silently use defaults (no creation)

	// Apply environment variable overrides (Tier 3: Env vars override everything)
	if err := config.LoadFromEnv(); err != nil {
		return nil, fmt.Errorf("failed to load environment variables: %w", err)
	}

	// Validate final configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return config, nil
}

func getDefaultConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".local/share/libreseed/config.yaml"
	}
	return filepath.Join(home, ".local/share/libreseed", "config.yaml")
}

func getDefaultPIDPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".local/share/libreseed/lbsd.pid"
	}
	return filepath.Join(home, ".local/share/libreseed", "lbsd.pid")
}

func writePIDFile(listenAddr string) error {
	pidPath := getDefaultPIDPath()

	// Ensure directory exists
	dir := filepath.Dir(pidPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create PID directory: %w", err)
	}

	// Write PID and listen address in format: PID:ADDRESS
	pid := os.Getpid()
	content := fmt.Sprintf("%d:%s\n", pid, listenAddr)

	if err := os.WriteFile(pidPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write PID file: %w", err)
	}

	return nil
}

func removePIDFile() {
	pidPath := getDefaultPIDPath()
	// Best effort removal, ignore errors
	_ = os.Remove(pidPath)
}

func printUsage() {
	fmt.Println("lbsd - LibreSeed Daemon")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  lbsd [options]")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  --config PATH    Path to configuration file (default: ~/.local/share/libreseed/config.yaml)")
	fmt.Println("  --version, -v    Show version information")
	fmt.Println("  --help, -h       Show this help message")
	fmt.Println()
	fmt.Println("Note: Use 'lbs' CLI for daemon management (start, stop, status, restart, stats)")
	fmt.Println()
}
