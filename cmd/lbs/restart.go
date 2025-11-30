package main

import (
	"fmt"
)

func restartCommand(args []string) error {
	// Check if daemon is running
	if !isRunning() {
		fmt.Println("Daemon is not running, starting it...")
		return startCommand(args)
	}

	fmt.Println("Stopping daemon...")
	if err := stopCommand(args); err != nil {
		return fmt.Errorf("failed to stop daemon: %w", err)
	}

	fmt.Println()
	fmt.Println("Starting daemon...")
	if err := startCommand(args); err != nil {
		return fmt.Errorf("failed to start daemon: %w", err)
	}

	return nil
}
