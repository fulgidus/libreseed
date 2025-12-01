package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/libreseed/libreseed/pkg/api"
)

// apikeyGenerateCommand generates a new API key
func apikeyGenerateCommand(args []string) error {
	fs := flag.NewFlagSet("apikey generate", flag.ExitOnError)
	name := fs.String("name", "", "Key name (required)")
	level := fs.String("level", api.LevelRead, "Permission level (read, write, admin)")

	if err := fs.Parse(args); err != nil {
		return err
	}

	if *name == "" {
		return fmt.Errorf("--name is required")
	}

	// Validate level
	validLevels := []string{api.LevelRead, api.LevelWrite, api.LevelAdmin}
	levelValid := false
	for _, l := range validLevels {
		if *level == l {
			levelValid = true
			break
		}
	}
	if !levelValid {
		return fmt.Errorf("invalid level %q, must be one of: read, write, admin", *level)
	}

	// Make API request
	url := fmt.Sprintf("%s/api/v1/admin/keys", getAPIAddr())
	reqBody := strings.NewReader(fmt.Sprintf(`{"name":"%s","level":"%s"}`, *name, *level))

	req, err := http.NewRequest("POST", url, reqBody)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Get admin key from environment
	adminKey := os.Getenv("LIBRESEED_ADMIN_KEY")
	if adminKey != "" {
		req.Header.Set("Authorization", "Bearer "+adminKey)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect to daemon: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API error (%d): %s", resp.StatusCode, string(body))
	}

	// Parse response
	var result struct {
		PlaintextKey string     `json:"plaintext_key"`
		Key          api.APIKey `json:"key"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	// Display the key
	fmt.Println("API Key Generated Successfully")
	fmt.Println("==============================")
	fmt.Printf("Name:       %s\n", result.Key.Name)
	fmt.Printf("Level:      %s\n", result.Key.Level)
	fmt.Printf("ID:         %s\n", result.Key.ID)
	fmt.Println()
	fmt.Println("PLAINTEXT KEY (save this, it won't be shown again):")
	fmt.Println(result.PlaintextKey)
	fmt.Println()
	fmt.Println("Export as environment variable:")
	fmt.Printf("  export LIBRESEED_API_KEY='%s'\n", result.PlaintextKey)

	return nil
}

// apikeyListCommand lists all API keys
func apikeyListCommand(args []string) error {
	fs := flag.NewFlagSet("apikey list", flag.ExitOnError)
	if err := fs.Parse(args); err != nil {
		return err
	}

	url := fmt.Sprintf("%s/api/v1/admin/keys", getAPIAddr())

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Get admin key from environment
	adminKey := os.Getenv("LIBRESEED_ADMIN_KEY")
	if adminKey != "" {
		req.Header.Set("Authorization", "Bearer "+adminKey)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect to daemon: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API error (%d): %s", resp.StatusCode, string(body))
	}

	// Parse response
	var keys []api.APIKey
	if err := json.Unmarshal(body, &keys); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if len(keys) == 0 {
		fmt.Println("No API keys found")
		return nil
	}

	// Display keys in table format
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tNAME\tLEVEL\tCREATED\tLAST USED\tREVOKED")
	fmt.Fprintln(w, "--\t----\t-----\t-------\t---------\t-------")

	for _, key := range keys {
		created := key.CreatedAt.Format("2006-01-02")
		lastUsed := "never"
		if !key.LastUsed.IsZero() {
			lastUsed = key.LastUsed.Format("2006-01-02 15:04")
		}
		revoked := "no"
		if key.Revoked {
			revoked = "YES"
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
			key.ID[:8], key.Name, key.Level, created, lastUsed, revoked)
	}

	w.Flush()
	return nil
}

// apikeyRevokeCommand revokes an API key
func apikeyRevokeCommand(args []string) error {
	fs := flag.NewFlagSet("apikey revoke", flag.ExitOnError)
	if err := fs.Parse(args); err != nil {
		return err
	}

	if len(fs.Args()) < 1 {
		return fmt.Errorf("key ID required: lbs apikey revoke <key_id>")
	}

	keyID := fs.Args()[0]

	url := fmt.Sprintf("%s/api/v1/admin/keys/%s/revoke", getAPIAddr(), keyID)

	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Get admin key from environment
	adminKey := os.Getenv("LIBRESEED_ADMIN_KEY")
	if adminKey != "" {
		req.Header.Set("Authorization", "Bearer "+adminKey)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect to daemon: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API error (%d): %s", resp.StatusCode, string(body))
	}

	fmt.Printf("API key %s revoked successfully\n", keyID)
	return nil
}

// apikeyDeleteCommand deletes an API key
func apikeyDeleteCommand(args []string) error {
	fs := flag.NewFlagSet("apikey delete", flag.ExitOnError)
	if err := fs.Parse(args); err != nil {
		return err
	}

	if len(fs.Args()) < 1 {
		return fmt.Errorf("key ID required: lbs apikey delete <key_id>")
	}

	keyID := fs.Args()[0]

	url := fmt.Sprintf("%s/api/v1/admin/keys/%s", getAPIAddr(), keyID)

	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Get admin key from environment
	adminKey := os.Getenv("LIBRESEED_ADMIN_KEY")
	if adminKey != "" {
		req.Header.Set("Authorization", "Bearer "+adminKey)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect to daemon: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API error (%d): %s", resp.StatusCode, string(body))
	}

	fmt.Printf("API key %s deleted successfully\n", keyID)
	return nil
}

// apikeyCommand handles the apikey subcommands
func apikeyCommand(args []string) error {
	if len(args) < 1 {
		printApikeyUsage()
		return fmt.Errorf("subcommand required")
	}

	subcommand := args[0]
	subargs := args[1:]

	switch subcommand {
	case "generate", "gen", "create":
		return apikeyGenerateCommand(subargs)
	case "list", "ls":
		return apikeyListCommand(subargs)
	case "revoke":
		return apikeyRevokeCommand(subargs)
	case "delete", "del", "rm":
		return apikeyDeleteCommand(subargs)
	case "help", "-h", "--help":
		printApikeyUsage()
		return nil
	default:
		printApikeyUsage()
		return fmt.Errorf("unknown subcommand: %s", subcommand)
	}
}

func printApikeyUsage() {
	fmt.Println("lbs apikey - Manage API keys")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  lbs apikey generate --name NAME [--level LEVEL]  Generate a new API key")
	fmt.Println("  lbs apikey list                                  List all API keys")
	fmt.Println("  lbs apikey revoke <key_id>                       Revoke an API key")
	fmt.Println("  lbs apikey delete <key_id>                       Delete an API key")
	fmt.Println()
	fmt.Println("Options for 'generate':")
	fmt.Println("  --name NAME     Key name (required)")
	fmt.Println("  --level LEVEL   Permission level: read, write, admin (default: read)")
	fmt.Println()
	fmt.Println("Environment Variables:")
	fmt.Println("  LIBRESEED_ADMIN_KEY   Admin API key for authentication")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  # Generate an admin key")
	fmt.Println("  lbs apikey generate --name my-admin-key --level admin")
	fmt.Println()
	fmt.Println("  # List all keys")
	fmt.Println("  export LIBRESEED_ADMIN_KEY='lbs_...'")
	fmt.Println("  lbs apikey list")
	fmt.Println()
	fmt.Println("  # Revoke a key")
	fmt.Println("  lbs apikey revoke abc12345")
	fmt.Println()
}
