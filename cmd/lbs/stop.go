package main

import (
	"fmt"
	"io"
	"net/http"
	"time"
)

func stopCommand(_ []string) error {
	// Check if daemon is running
	if !isRunning() {
		return fmt.Errorf("daemon is not running")
	}

	// Get API address from env or default
	apiAddr := getAPIAddr()

	// Send shutdown request
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	req, err := http.NewRequest("POST", apiAddr+"/shutdown", nil)
	if err != nil {
		return fmt.Errorf("failed to create shutdown request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect to daemon: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("daemon returned error: %s (status: %d)", string(body), resp.StatusCode)
	}

	fmt.Println("Shutdown request sent successfully")
	fmt.Println("Waiting for daemon to stop...")

	// Wait for daemon to stop (check PID file removal)
	maxWait := 30 * time.Second
	start := time.Now()
	for time.Since(start) < maxWait {
		if !isRunning() {
			fmt.Println("Daemon stopped successfully")
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}

	return fmt.Errorf("daemon did not stop within timeout")
}
