package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

func startCommand(args []string) error {
	// Parse flags
	configPath := ""
	for i := 0; i < len(args); i++ {
		if args[i] == "--config" && i+1 < len(args) {
			configPath = args[i+1]
			i++
		}
	}

	// Check if daemon is already running
	if isRunning() {
		return fmt.Errorf("daemon is already running (PID file exists: %s)", getDefaultPIDPath())
	}

	// Find lbsd binary
	lbsdPath, err := findLbsdBinary()
	if err != nil {
		return fmt.Errorf("failed to find lbsd binary: %w", err)
	}

	// Prepare command arguments
	cmdArgs := []string{}
	if configPath != "" {
		cmdArgs = append(cmdArgs, "--config", configPath)
	}

	// Start lbsd as background process
	cmd := exec.Command(lbsdPath, cmdArgs...)

	// Pass environment variables to daemon
	cmd.Env = os.Environ()

	// Detach from parent process (platform-specific)
	setProcAttributes(cmd)

	// Redirect stdout/stderr to log file
	logPath := getLogPath()

	// Ensure log directory exists
	logDir := filepath.Dir(logPath)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}
	defer logFile.Close()

	cmd.Stdout = logFile
	cmd.Stderr = logFile

	// Start the daemon
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start daemon: %w", err)
	}

	fmt.Println("Daemon started successfully")
	fmt.Printf("Process started with PID: %d\n", cmd.Process.Pid)
	fmt.Printf("Logs: %s\n", logPath)

	// Wait for daemon to initialize and write its PID file
	// Try multiple times with delays to handle slow initialization
	maxAttempts := 6
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		time.Sleep(500 * time.Millisecond)

		if isRunning() {
			fmt.Println("Daemon initialization verified")
			fmt.Println("Use 'lbs status' to check daemon status")
			return nil
		}

		if attempt < maxAttempts {
			fmt.Printf("Waiting for daemon initialization... (%d/%d)\n", attempt, maxAttempts)
		}
	}

	return fmt.Errorf("daemon failed to initialize within %d seconds (check logs: %s)", maxAttempts/2, logPath)

}

func findLbsdBinary() (string, error) {
	// First, check if lbsd is in PATH
	if path, err := exec.LookPath("lbsd"); err == nil {
		return path, nil
	}

	// Check if lbsd is in the same directory as lbs
	lbsPath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("failed to get executable path: %w", err)
	}

	lbsdPath := filepath.Join(filepath.Dir(lbsPath), "lbsd")
	if _, err := os.Stat(lbsdPath); err == nil {
		return lbsdPath, nil
	}

	return "", fmt.Errorf("lbsd binary not found in PATH or alongside lbs")
}

func getDefaultConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".local/share/libreseed/config.yaml"
	}
	return filepath.Join(home, ".local", "share", "libreseed", "config.yaml")
}

func getDefaultPIDPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".local/share/libreseed/lbsd.pid"
	}
	return filepath.Join(home, ".local", "share", "libreseed", "lbsd.pid")
}

func getLogPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".local/share/libreseed/daemon.log"
	}
	return filepath.Join(home, ".local", "share", "libreseed", "daemon.log")
}

func isRunning() bool {
	pidPath := getDefaultPIDPath()
	if _, err := os.Stat(pidPath); os.IsNotExist(err) {
		return false
	}

	// Read PID file and parse PID:ADDRESS format
	data, err := os.ReadFile(pidPath)
	if err != nil {
		// Can't read PID file, consider it stale
		removePIDFile()
		return false
	}

	content := strings.TrimSpace(string(data))

	// Parse PID from either "PID" or "PID:ADDRESS" format
	var pidStr string
	if strings.Contains(content, ":") {
		// New format: PID:ADDRESS
		parts := strings.SplitN(content, ":", 2)
		pidStr = parts[0]
	} else {
		// Old format: just PID
		pidStr = content
	}

	pid, err := strconv.Atoi(pidStr)
	if err != nil || pid <= 0 {
		// Invalid PID format, remove stale file
		removePIDFile()
		return false
	}

	// Check if process exists by sending signal 0
	process, err := os.FindProcess(pid)
	if err != nil {
		// Process doesn't exist on this system
		removePIDFile()
		return false
	}

	// Try to send signal 0 (no-op) to check if process is alive
	err = process.Signal(syscall.Signal(0))
	if err != nil {
		// Process is not running or we don't have permission
		removePIDFile()
		return false
	}

	// Additional verification: check if the process is actually lbsd
	// Read /proc/<pid>/exe to verify process name (Linux-specific)
	exePath := fmt.Sprintf("/proc/%d/exe", pid)
	if target, err := os.Readlink(exePath); err == nil {
		baseName := filepath.Base(target)
		// Check if the executable name contains "lbsd"
		if baseName != "lbsd" {
			// PID file exists but process is not lbsd, remove stale file
			removePIDFile()
			return false
		}
	}
	// Note: If /proc check fails (non-Linux or permission issues),
	// we still trust the PID file if signal 0 succeeded

	return true
}

func removePIDFile() {
	pidPath := getDefaultPIDPath()
	os.Remove(pidPath) // Ignore errors - best effort cleanup
}

// getDaemonAddr returns the daemon's listen address from the PID file.
// Returns the address stored in the PID file (new format: PID:ADDRESS),
// or falls back to the LIBRESEED_LISTEN_ADDR environment variable if:
// - PID file doesn't exist (daemon not running)
// - PID file uses old format (PID only)
// - Reading PID file fails
func getDaemonAddr() string {
	pidPath := getDefaultPIDPath()

	// Try to read address from PID file
	data, err := os.ReadFile(pidPath)
	if err != nil {
		// PID file doesn't exist or can't be read, fall back to env var
		return getAPIAddrFromEnv()
	}

	content := strings.TrimSpace(string(data))

	// Check if new format (PID:ADDRESS)
	if strings.Contains(content, ":") {
		parts := strings.SplitN(content, ":", 2)
		if len(parts) == 2 && parts[1] != "" {
			// Return the address portion
			return "http://" + parts[1]
		}
	}

	// Old format or invalid format, fall back to env var
	return getAPIAddrFromEnv()
}

// getAPIAddrFromEnv returns the API address from environment variable or default
func getAPIAddrFromEnv() string {
	addr := os.Getenv("LIBRESEED_LISTEN_ADDR")
	if addr == "" {
		addr = "localhost:8080" // Default
	}
	return "http://" + addr
}
