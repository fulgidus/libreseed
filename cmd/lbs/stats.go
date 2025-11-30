package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type statsResponse struct {
	BytesUploaded    uint64  `json:"bytes_uploaded"`
	BytesDownloaded  uint64  `json:"bytes_downloaded"`
	PackagesSeeded   int     `json:"packages_seeded"`
	PeersConnected   int     `json:"peers_connected"`
	UploadRate       float64 `json:"upload_rate"`
	DownloadRate     float64 `json:"download_rate"`
	PeakUploadRate   float64 `json:"peak_upload_rate"`
	PeakDownloadRate float64 `json:"peak_download_rate"`
}

func statsCommand(_ []string) error {
	// Get API address from PID file or fall back to env
	apiAddr := getDaemonAddr()

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// Fetch stats from daemon
	resp, err := client.Get(apiAddr + "/stats")
	if err != nil {
		return fmt.Errorf("failed to connect to daemon: %w (is the daemon running?)", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("daemon returned error: %s (status: %d)", string(body), resp.StatusCode)
	}

	// Parse response
	var stats statsResponse
	if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	// Display statistics
	printStats(&stats)
	return nil
}

func printStats(stats *statsResponse) {
	fmt.Println("Libreseed Daemon Statistics")
	fmt.Println("============================")
	fmt.Println()

	fmt.Println("Transfer Statistics:")
	fmt.Printf("  Uploaded:   %s\n", formatBytes(stats.BytesUploaded))
	fmt.Printf("  Downloaded: %s\n", formatBytes(stats.BytesDownloaded))
	fmt.Println()

	fmt.Println("Current Rates:")
	fmt.Printf("  Upload:   %s/s\n", formatBytes(uint64(stats.UploadRate)))
	fmt.Printf("  Download: %s/s\n", formatBytes(uint64(stats.DownloadRate)))
	fmt.Println()

	fmt.Println("Peak Rates:")
	fmt.Printf("  Upload:   %s/s\n", formatBytes(uint64(stats.PeakUploadRate)))
	fmt.Printf("  Download: %s/s\n", formatBytes(uint64(stats.PeakDownloadRate)))
	fmt.Println()

	fmt.Println("Activity:")
	fmt.Printf("  Packages Seeded:  %d\n", stats.PackagesSeeded)
	fmt.Printf("  Peers Connected:  %d\n", stats.PeersConnected)
	fmt.Println()
}

func formatBytes(bytes uint64) string {
	const (
		KB = 1024
		MB = 1024 * KB
		GB = 1024 * MB
		TB = 1024 * GB
	)

	switch {
	case bytes >= TB:
		return fmt.Sprintf("%.2f TB", float64(bytes)/float64(TB))
	case bytes >= GB:
		return fmt.Sprintf("%.2f GB", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.2f MB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.2f KB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}
