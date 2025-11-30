package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
)

// addCommand handles package addition via daemon API
func addCommand(args []string) error {
	if len(args) < 3 {
		return fmt.Errorf("usage: lbs add <file> <name> <version> [description]")
	}

	filePath := args[0]
	name := args[1]
	version := args[2]
	description := ""
	if len(args) > 3 {
		description = args[3]
	}

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("file not found: %s", filePath)
	}

	// Open file
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Create multipart form
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add file part
	part, err := writer.CreateFormFile("file", filepath.Base(filePath))
	if err != nil {
		return fmt.Errorf("failed to create form file: %w", err)
	}
	if _, err := io.Copy(part, file); err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}

	// Add metadata fields
	if err := writer.WriteField("name", name); err != nil {
		return fmt.Errorf("failed to write name field: %w", err)
	}
	if err := writer.WriteField("version", version); err != nil {
		return fmt.Errorf("failed to write version field: %w", err)
	}
	if description != "" {
		if err := writer.WriteField("description", description); err != nil {
			return fmt.Errorf("failed to write description field: %w", err)
		}
	}

	if err := writer.Close(); err != nil {
		return fmt.Errorf("failed to close multipart writer: %w", err)
	}

	// Send request to daemon
	apiAddr := getAPIAddr()
	url := apiAddr + "/packages/add"

	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w (is daemon running?)", err)
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	// Check status code
	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	// Parse and display response
	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	fmt.Println("✓ Package added successfully")
	fmt.Printf("  Package ID:  %s\n", result["package_id"])
	fmt.Printf("  Fingerprint: %s\n", result["fingerprint"])
	fmt.Printf("  File Hash:   %s\n", result["file_hash"])

	return nil
}
