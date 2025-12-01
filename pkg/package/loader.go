// Package package provides serialization and deserialization for LibreSeed packages.
package packagetypes

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// LoadPackageFromFile reads and parses a .lspkg file from disk.
// It performs structural validation but does NOT verify cryptographic signatures.
// Use crypto.VerifyDualSignature() after loading to validate signatures.
func LoadPackageFromFile(filePath string) (*Package, error) {
	// Read file contents
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read package file: %w", err)
	}

	// Parse YAML structure
	pkg, err := LoadPackageFromBytes(data)
	if err != nil {
		return nil, err
	}

	// Store file path reference
	pkg.FilePath = filePath

	// Compute package ID from file contents
	pkg.PackageID = pkg.ComputePackageID(data)

	// Store actual file size
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat package file: %w", err)
	}
	pkg.SizeBytes = fileInfo.Size()

	return pkg, nil
}

// LoadPackageFromBytes parses a .lspkg file from memory.
// It performs structural validation but does NOT verify cryptographic signatures.
// Use crypto.VerifyDualSignature() after loading to validate signatures.
func LoadPackageFromBytes(data []byte) (*Package, error) {
	var pkg Package

	// Parse YAML
	if err := yaml.Unmarshal(data, &pkg); err != nil {
		return nil, fmt.Errorf("failed to parse package YAML: %w", err)
	}

	// Perform structural validation
	if err := pkg.Validate(); err != nil {
		return nil, fmt.Errorf("package validation failed: %w", err)
	}

	return &pkg, nil
}

// SerializePackage converts a Package structure into YAML bytes.
// This is used when creating new packages or re-serializing existing ones.
func SerializePackage(pkg *Package) ([]byte, error) {
	// Validate before serialization
	if err := pkg.Validate(); err != nil {
		return nil, fmt.Errorf("cannot serialize invalid package: %w", err)
	}

	// Marshal to YAML
	data, err := yaml.Marshal(pkg)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal package to YAML: %w", err)
	}

	return data, nil
}

// SerializeManifest converts a Manifest structure into YAML bytes.
// This is used for signature verification (signing canonical manifest representation).
func SerializeManifest(manifest *Manifest) ([]byte, error) {
	// Validate before serialization
	if err := manifest.Validate(); err != nil {
		return nil, fmt.Errorf("cannot serialize invalid manifest: %w", err)
	}

	// Marshal to YAML
	data, err := yaml.Marshal(manifest)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal manifest to YAML: %w", err)
	}

	return data, nil
}

// WritePackageToFile serializes and writes a Package to disk as a .lspkg file.
func WritePackageToFile(pkg *Package, filePath string) error {
	// Serialize package
	data, err := SerializePackage(pkg)
	if err != nil {
		return err
	}

	// Write to disk with read-only permissions for regular users
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write package file: %w", err)
	}

	// Update package metadata
	pkg.FilePath = filePath
	pkg.PackageID = pkg.ComputePackageID(data)
	pkg.SizeBytes = int64(len(data))

	return nil
}
