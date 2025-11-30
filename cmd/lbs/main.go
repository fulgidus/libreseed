package main

import (
	"fmt"
	"os"
)

var version = "dev" // Set via ldflags during build

// getAPIAddr returns the daemon API address from env var or default
func getAPIAddr() string {
	// Check environment variable first
	if addr := os.Getenv("LIBRESEED_LISTEN_ADDR"); addr != "" {
		return "http://" + addr
	}
	// TODO: Could read from config file if needed
	// Default to localhost:8080
	return "http://localhost:8080"
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]
	args := os.Args[2:]

	switch command {
	case "start":
		if err := startCommand(args); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "stop":
		if err := stopCommand(args); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "status":
		if err := statusCommand(args); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "restart":
		if err := restartCommand(args); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "stats":
		if err := statsCommand(args); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "help", "-h", "--help":
		printUsage()
	case "version", "--version", "-v":
		fmt.Printf("lbs version %s\n", version)
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("lbs - LibreSeed CLI")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  lbs start [--config PATH]  Start the daemon")
	fmt.Println("  lbs stop                   Stop the running daemon")
	fmt.Println("  lbs status                 Show daemon status")
	fmt.Println("  lbs restart                Restart the daemon")
	fmt.Println("  lbs stats                  Show daemon statistics")
	fmt.Println("  lbs version                Show version information")
	fmt.Println("  lbs help                   Show this help message")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  --config PATH    Path to configuration file (default: ~/.libreseed/config.yaml)")
	fmt.Println()
}
