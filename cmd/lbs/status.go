package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

func statusCommand(_ []string) error {
	// Check if daemon is running via PID file
	if !isRunning() {
		fmt.Println("Daemon Status: STOPPED")
		return nil
	}

	// Get API address from PID file or fall back to env
	apiAddr := getDaemonAddr()

	// Try to connect to daemon API
	client := &http.Client{
		Timeout: 3 * time.Second,
	}

	resp, err := client.Get(apiAddr + "/stats")
	if err != nil {
		fmt.Println("Daemon Status: UNKNOWN (PID exists but cannot connect to API)")
		return fmt.Errorf("failed to connect to daemon API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Println("Daemon Status: ERROR")
		return fmt.Errorf("daemon API returned error status: %d", resp.StatusCode)
	}

	// Parse stats to get basic info
	var stats statsResponse
	if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
		fmt.Println("Daemon Status: RUNNING (but stats unavailable)")
		return fmt.Errorf("failed to parse stats: %w", err)
	}

	// Display status
	fmt.Println("Daemon Status: RUNNING")
	fmt.Println()
	fmt.Println("Quick Stats:")
	fmt.Printf("  Packages Seeded:  %d\n", stats.PackagesSeeded)
	fmt.Printf("  Peers Connected:  %d\n", stats.PeersConnected)
	fmt.Printf("  Upload Rate:      %s/s\n", formatBytes(uint64(stats.UploadRate)))
	fmt.Printf("  Download Rate:    %s/s\n", formatBytes(uint64(stats.DownloadRate)))
	fmt.Println()
	fmt.Println("Use 'lbs stats' for detailed statistics")

	return nil
}
