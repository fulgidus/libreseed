package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"text/tabwriter"
	"time"
)

// PackageInfo represents package metadata for display.
// Matches the structure from pkg/daemon/package_manager.go
// JSON tags match the API's PascalCase field names
type PackageInfo struct {
	PackageID                   string    `json:"PackageID"`
	Name                        string    `json:"Name"`
	Version                     string    `json:"Version"`
	Description                 string    `json:"Description"`
	FilePath                    string    `json:"FilePath"`
	FileHash                    string    `json:"FileHash"`
	FileSize                    int64     `json:"FileSize"`
	CreatedAt                   time.Time `json:"CreatedAt"`
	CreatorFingerprint          string    `json:"CreatorFingerprint"`
	MaintainerFingerprint       string    `json:"MaintainerFingerprint"`
	ManifestSignature           string    `json:"ManifestSignature"`
	MaintainerManifestSignature string    `json:"MaintainerManifestSignature"`
	AnnouncedToDHT              bool      `json:"AnnouncedToDHT"`
	LastAnnounced               time.Time `json:"LastAnnounced"`
}

// listResponse represents the API response from GET /packages/list
type listResponse struct {
	Status   string        `json:"status"`
	Count    int           `json:"count"`
	Packages []PackageInfo `json:"packages"`
}

// listCommand lists all packages from the daemon.
// Usage: lbs list
func listCommand(args []string) error {
	if len(args) > 0 {
		return fmt.Errorf("list command does not accept arguments")
	}

	// Build API endpoint
	apiAddr := getAPIAddr()
	url := fmt.Sprintf("%s/packages/list", apiAddr)

	// Make GET request
	resp, err := http.Get(url)
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
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("daemon returned error: %s\nResponse: %s", resp.Status, string(body))
	}

	// Parse JSON response
	var listResp listResponse
	if err := json.Unmarshal(body, &listResp); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	// Display packages
	if listResp.Count == 0 {
		fmt.Println("No packages found.")
		fmt.Println("\nUse 'lbs add <file> <name> <version> [description]' to add a package.")
		return nil
	}

	fmt.Printf("Found %d package(s):\n\n", listResp.Count)

	// Create tabwriter for aligned output
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	defer w.Flush()

	// Print header
	fmt.Fprintln(w, "NAME\tVERSION\tDESCRIPTION\tCREATED\tFILE HASH")
	fmt.Fprintln(w, "----\t-------\t-----------\t-------\t---------")

	// Print each package
	for _, pkg := range listResp.Packages {
		// Format created time
		createdStr := pkg.CreatedAt.Format("2006-01-02 15:04")

		// Truncate description if too long
		desc := pkg.Description
		if len(desc) > 40 {
			desc = desc[:37] + "..."
		}
		if desc == "" {
			desc = "-"
		}

		// Truncate file hash for display
		hashShort := pkg.FileHash
		if len(hashShort) > 16 {
			hashShort = hashShort[:16] + "..."
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			pkg.Name,
			pkg.Version,
			desc,
			createdStr,
			hashShort,
		)
	}

	// Print detailed info section
	fmt.Println("\n--- Package Details ---")
	for i, pkg := range listResp.Packages {
		fmt.Printf("\n[%d] %s v%s\n", i+1, pkg.Name, pkg.Version)
		fmt.Printf("    Package ID:  %s\n", pkg.PackageID)
		fmt.Printf("    Description: %s\n", pkg.Description)
		fmt.Printf("    File Path:   %s\n", pkg.FilePath)
		fmt.Printf("    File Hash:   %s\n", pkg.FileHash)
		fmt.Printf("    File Size:   %d bytes\n", pkg.FileSize)
		fmt.Printf("    Creator:     %s\n", pkg.CreatorFingerprint)

		// Display maintainer if different from creator
		if pkg.MaintainerFingerprint != "" && pkg.MaintainerFingerprint != pkg.CreatorFingerprint {
			fmt.Printf("    Maintainer:  %s\n", pkg.MaintainerFingerprint)
		}

		fmt.Printf("    Created At:  %s\n", pkg.CreatedAt.Format("2006-01-02 15:04:05 MST"))

		if pkg.AnnouncedToDHT {
			fmt.Printf("    DHT Status:  Announced (Last: %s)\n", pkg.LastAnnounced.Format("2006-01-02 15:04:05"))
		} else {
			fmt.Printf("    DHT Status:  Not announced\n")
		}
	}

	return nil
}
