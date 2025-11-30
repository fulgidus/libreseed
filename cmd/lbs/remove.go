package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// removeResponse represents the API response from DELETE/POST /packages/remove
type removeResponse struct {
	Status    string `json:"status"`
	PackageID string `json:"package_id"`
	Message   string `json:"message"`
}

// removeCommand removes a package from the daemon.
// Usage: lbs remove <package_id>
func removeCommand(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: lbs remove <package_id>")
	}

	packageID := args[0]

	// Build API endpoint
	apiAddr := getAPIAddr()
	url := fmt.Sprintf("%s/packages/remove?package_id=%s", apiAddr, packageID)

	// Make DELETE request
	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect to daemon: %w (is daemon running?)", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	// Check status code
	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("package not found: %s", packageID)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("daemon returned error: %s\nResponse: %s", resp.Status, string(body))
	}

	// Parse JSON response
	var removeResp removeResponse
	if err := json.Unmarshal(body, &removeResp); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	// Display success message
	fmt.Printf("✓ Package removed successfully\n")
	fmt.Printf("  Package ID: %s\n", removeResp.PackageID)
	fmt.Printf("  Status: %s\n", removeResp.Message)

	return nil
}

// removeCommandWithJSON is an alternative implementation using JSON body (POST method)
// This is kept for reference but not used by default
func removeCommandWithJSON(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: lbs remove <package_id>")
	}

	packageID := args[0]

	// Build API endpoint
	apiAddr := getAPIAddr()
	url := fmt.Sprintf("%s/packages/remove", apiAddr)

	// Create JSON request body
	reqBody := map[string]string{
		"package_id": packageID,
	}
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to create request body: %w", err)
	}

	// Make POST request
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to connect to daemon: %w (is daemon running?)", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	// Check status code
	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("package not found: %s", packageID)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("daemon returned error: %s\nResponse: %s", resp.Status, string(body))
	}

	// Parse JSON response
	var removeResp removeResponse
	if err := json.Unmarshal(body, &removeResp); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	// Display success message
	fmt.Printf("✓ Package removed successfully\n")
	fmt.Printf("  Package ID: %s\n", removeResp.PackageID)
	fmt.Printf("  Status: %s\n", removeResp.Message)

	return nil
}
