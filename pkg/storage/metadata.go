// Package storage provides utilities for YAML serialization and file operations
// used throughout the LibreSeed project.
//
// This package implements helper functions for working with YAML-formatted data,
// including manifests, minimal descriptions, and configuration files. All file
// operations use atomic writes to prevent partial file corruption.
//
// YAML Conventions:
//   - All YAML files use 2-space indentation
//   - Field names follow snake_case convention
//   - Arrays and maps are formatted for readability
//   - Empty fields are omitted by default
//
// Error Handling:
//   - All functions return descriptive errors with context
//   - File operations include path information in errors
//   - YAML parsing errors include line and column numbers when available
package storage

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// MarshalYAML serializes a Go value to YAML format.
//
// The function uses yaml.v3 with default settings, which provides:
//   - 2-space indentation
//   - Omission of empty fields
//   - Proper handling of anchors and aliases
//   - UTF-8 encoding
//
// Example:
//
//	type Config struct {
//	    Name    string `yaml:"name"`
//	    Version string `yaml:"version"`
//	}
//	cfg := Config{Name: "libreseed", Version: "1.0.0"}
//	data, err := MarshalYAML(cfg)
//
// Parameters:
//   - v: The value to serialize (can be any Go type)
//
// Returns:
//   - []byte: The YAML-encoded data
//   - error: Any error encountered during marshaling
func MarshalYAML(v interface{}) ([]byte, error) {
	data, err := yaml.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal YAML: %w", err)
	}
	return data, nil
}

// UnmarshalYAML deserializes YAML data into a Go value.
//
// The function validates the YAML syntax and populates the provided value.
// The target value must be a pointer to allow modification.
//
// Example:
//
//	type Config struct {
//	    Name    string `yaml:"name"`
//	    Version string `yaml:"version"`
//	}
//	var cfg Config
//	err := UnmarshalYAML(yamlData, &cfg)
//
// Parameters:
//   - data: The YAML-encoded data to parse
//   - v: Pointer to the target value (must be a pointer)
//
// Returns:
//   - error: Any error encountered during unmarshaling
func UnmarshalYAML(data []byte, v interface{}) error {
	if err := yaml.Unmarshal(data, v); err != nil {
		return fmt.Errorf("failed to unmarshal YAML: %w", err)
	}
	return nil
}

// SaveYAMLFile writes a Go value to a YAML file using atomic operations.
//
// This function ensures data integrity by:
//  1. Creating a temporary file in the same directory
//  2. Writing the YAML data to the temporary file
//  3. Syncing the data to disk
//  4. Renaming the temporary file to the target path (atomic operation)
//
// This approach prevents partial file corruption in case of crashes or
// power failures. The temporary file is automatically cleaned up on error.
//
// Example:
//
//	cfg := Config{Name: "libreseed", Version: "1.0.0"}
//	err := SaveYAMLFile("/etc/libreseed/config.yaml", cfg)
//
// Parameters:
//   - path: The target file path (directories must exist)
//   - v: The value to serialize and save
//
// Returns:
//   - error: Any error encountered during file operations or marshaling
func SaveYAMLFile(path string, v interface{}) error {
	// Marshal the value to YAML
	data, err := MarshalYAML(v)
	if err != nil {
		return fmt.Errorf("failed to save YAML file %q: %w", path, err)
	}

	// Get the directory for the temporary file
	dir := filepath.Dir(path)

	// Ensure the directory exists
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory for %q: %w", path, err)
	}

	// Create a temporary file in the same directory as the target
	// This ensures the rename operation is atomic (same filesystem)
	tmpFile, err := os.CreateTemp(dir, ".tmp-*.yaml")
	if err != nil {
		return fmt.Errorf("failed to create temporary file for %q: %w", path, err)
	}
	tmpPath := tmpFile.Name()

	// Ensure cleanup on error
	defer func() {
		if err != nil {
			tmpFile.Close()
			os.Remove(tmpPath)
		}
	}()

	// Write the YAML data to the temporary file
	if _, err = tmpFile.Write(data); err != nil {
		return fmt.Errorf("failed to write data to temporary file %q: %w", tmpPath, err)
	}

	// Sync to ensure data is written to disk
	if err = tmpFile.Sync(); err != nil {
		return fmt.Errorf("failed to sync temporary file %q: %w", tmpPath, err)
	}

	// Close the temporary file before renaming
	if err = tmpFile.Close(); err != nil {
		return fmt.Errorf("failed to close temporary file %q: %w", tmpPath, err)
	}

	// Atomically replace the target file with the temporary file
	if err = os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("failed to rename temporary file %q to %q: %w", tmpPath, path, err)
	}

	return nil
}

// LoadYAMLFile reads and parses a YAML file into a Go value.
//
// This function validates that the file exists and is readable before
// attempting to parse the YAML content. The target value must be a
// pointer to allow modification.
//
// Example:
//
//	var cfg Config
//	err := LoadYAMLFile("/etc/libreseed/config.yaml", &cfg)
//
// Parameters:
//   - path: The file path to read
//   - v: Pointer to the target value (must be a pointer)
//
// Returns:
//   - error: Any error encountered during file operations or unmarshaling
func LoadYAMLFile(path string, v interface{}) error {
	// Check if file exists
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("YAML file %q does not exist", path)
		}
		return fmt.Errorf("failed to access YAML file %q: %w", path, err)
	}

	// Read the file contents
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read YAML file %q: %w", path, err)
	}

	// Unmarshal the YAML data
	if err := UnmarshalYAML(data, v); err != nil {
		return fmt.Errorf("failed to parse YAML file %q: %w", path, err)
	}

	return nil
}

// ValidateYAML checks if the provided data is valid YAML syntax.
//
// This function attempts to parse the YAML data without storing the result.
// It's useful for validating YAML input before processing.
//
// Example:
//
//	yamlData := []byte("name: libreseed\nversion: 1.0.0")
//	if err := ValidateYAML(yamlData); err != nil {
//	    log.Printf("Invalid YAML: %v", err)
//	}
//
// Parameters:
//   - data: The YAML data to validate
//
// Returns:
//   - error: Any syntax error found in the YAML data, or nil if valid
func ValidateYAML(data []byte) error {
	var temp interface{}
	if err := yaml.Unmarshal(data, &temp); err != nil {
		return fmt.Errorf("invalid YAML syntax: %w", err)
	}
	return nil
}

// YAMLToString converts a Go value to a pretty-printed YAML string.
//
// This function is primarily intended for debugging and logging purposes.
// It formats the YAML with proper indentation and includes a trailing newline.
//
// Example:
//
//	cfg := Config{Name: "libreseed", Version: "1.0.0"}
//	yamlStr, err := YAMLToString(cfg)
//	fmt.Println(yamlStr)
//	// Output:
//	// name: libreseed
//	// version: 1.0.0
//
// Parameters:
//   - v: The value to convert to YAML string
//
// Returns:
//   - string: The pretty-printed YAML string
//   - error: Any error encountered during marshaling
func YAMLToString(v interface{}) (string, error) {
	data, err := MarshalYAML(v)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
