// Package package provides core data structures for LibreSeed packages.
package packagetypes

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/libreseed/libreseed/pkg/crypto"
)

// Manifest represents the complete package metadata and content description.
// This is the INNER signed structure that describes all package contents.
//
// Design Decision: The manifest is signed by the creator to ensure
// content integrity. Any modification to the package content will
// invalidate the manifest signature.
type Manifest struct {
	// PackageName is the human-readable package identifier (e.g., "my-library")
	PackageName string `yaml:"package_name" json:"package_name"`

	// Version follows semantic versioning (e.g., "1.2.3", "2.0.0-beta.1")
	Version string `yaml:"version" json:"version"`

	// Description provides a human-readable summary of the package purpose
	Description string `yaml:"description" json:"description"`

	// CreatorPubKey is the Ed25519 public key of the package creator
	// This must match the key used to sign the manifest
	CreatorPubKey crypto.PublicKey `yaml:"creator_pubkey" json:"creator_pubkey"`

	// ContentHash is the SHA-256 hash of all package content files
	// This ensures tamper-proof content integrity
	ContentHash string `yaml:"content_hash" json:"content_hash"`

	// ContentList describes all files included in the package
	ContentList []FileEntry `yaml:"content_list" json:"content_list"`

	// CreatedAt records when the package was created (ISO 8601 timestamp)
	CreatedAt time.Time `yaml:"created_at" json:"created_at"`

	// Metadata stores optional additional package information
	Metadata map[string]string `yaml:"metadata,omitempty" json:"metadata,omitempty"`
}

// FileEntry describes a single file within the package content.
type FileEntry struct {
	// Path is the relative path within the package (e.g., "src/main.go")
	Path string `yaml:"path" json:"path"`

	// Hash is the SHA-256 hash of the file content
	Hash string `yaml:"hash" json:"hash"`

	// Size is the file size in bytes
	Size int64 `yaml:"size" json:"size"`

	// Mode is the Unix file permission bits (e.g., 0644, 0755)
	Mode uint32 `yaml:"mode" json:"mode"`
}

// Package represents a complete LibreSeed package with all metadata.
// This is the top-level structure that ties together the manifest,
// signatures, and physical package file.
type Package struct {
	// PackageID is the SHA-256 hash of the complete .lspkg file
	// This provides a globally unique, content-addressed identifier
	PackageID string `yaml:"package_id" json:"package_id"`

	// FormatVersion specifies the package format version (currently "1.0")
	FormatVersion string `yaml:"format_version" json:"format_version"`

	// Manifest contains the package metadata and content description
	Manifest Manifest `yaml:"manifest" json:"manifest"`

	// ManifestSignature is the Ed25519 signature over (Manifest + ContentHash)
	// This is the INNER signature that proves manifest authenticity
	ManifestSignature crypto.Signature `yaml:"manifest_signature" json:"manifest_signature"`

	// FilePath is the absolute path to the .lspkg file on disk
	// This is NOT serialized (local information only)
	FilePath string `yaml:"-" json:"-"`

	// SizeBytes is the total size of the .lspkg file
	SizeBytes int64 `yaml:"size_bytes" json:"size_bytes"`
}

// Validate checks that the Manifest contains all required fields and valid data.
func (m *Manifest) Validate() error {
	if m.PackageName == "" {
		return fmt.Errorf("manifest: package_name is required")
	}
	if m.Version == "" {
		return fmt.Errorf("manifest: version is required")
	}
	if m.Description == "" {
		return fmt.Errorf("manifest: description is required")
	}
	if m.CreatorPubKey.Algorithm == "" {
		return fmt.Errorf("manifest: creator_pubkey is required")
	}
	if m.ContentHash == "" {
		return fmt.Errorf("manifest: content_hash is required")
	}
	if len(m.ContentList) == 0 {
		return fmt.Errorf("manifest: content_list must contain at least one file")
	}
	if m.CreatedAt.IsZero() {
		return fmt.Errorf("manifest: created_at timestamp is required")
	}

	// Validate content hash format (must be hex-encoded SHA-256)
	if len(m.ContentHash) != 64 {
		return fmt.Errorf("manifest: content_hash must be 64-character hex string")
	}
	if _, err := hex.DecodeString(m.ContentHash); err != nil {
		return fmt.Errorf("manifest: content_hash must be valid hex: %w", err)
	}

	// Validate each file entry
	for i, entry := range m.ContentList {
		if err := entry.Validate(); err != nil {
			return fmt.Errorf("manifest: content_list[%d]: %w", i, err)
		}
	}

	return nil
}

// Validate checks that the FileEntry contains valid data.
func (f *FileEntry) Validate() error {
	if f.Path == "" {
		return fmt.Errorf("file entry: path is required")
	}
	if f.Hash == "" {
		return fmt.Errorf("file entry: hash is required")
	}
	if len(f.Hash) != 64 {
		return fmt.Errorf("file entry: hash must be 64-character hex string")
	}
	if _, err := hex.DecodeString(f.Hash); err != nil {
		return fmt.Errorf("file entry: hash must be valid hex: %w", err)
	}
	if f.Size < 0 {
		return fmt.Errorf("file entry: size must be non-negative")
	}
	return nil
}

// ComputePackageID computes the SHA-256 hash of the complete .lspkg file.
// This provides a unique, content-addressed identifier for the package.
func (p *Package) ComputePackageID(fileContent []byte) string {
	hash := sha256.Sum256(fileContent)
	return hex.EncodeToString(hash[:])
}

// Validate checks that the Package contains all required fields and valid data.
func (p *Package) Validate() error {
	if p.PackageID == "" {
		return fmt.Errorf("package: package_id is required")
	}
	if len(p.PackageID) != 64 {
		return fmt.Errorf("package: package_id must be 64-character hex string")
	}
	if _, err := hex.DecodeString(p.PackageID); err != nil {
		return fmt.Errorf("package: package_id must be valid hex: %w", err)
	}
	if p.FormatVersion != "1.0" {
		return fmt.Errorf("package: unsupported format_version: %s (expected 1.0)", p.FormatVersion)
	}
	if err := p.Manifest.Validate(); err != nil {
		return fmt.Errorf("package: invalid manifest: %w", err)
	}
	if len(p.ManifestSignature.SignedData) == 0 {
		return fmt.Errorf("package: manifest_signature is required")
	}
	if p.SizeBytes <= 0 {
		return fmt.Errorf("package: size_bytes must be positive")
	}
	return nil
}

// Fingerprint returns the truncated package ID (first 16 characters) for display.
func (p *Package) Fingerprint() string {
	if len(p.PackageID) >= 16 {
		return p.PackageID[:16]
	}
	return p.PackageID
}

// FullName returns the package identifier with version (e.g., "my-library@1.2.3").
func (p *Package) FullName() string {
	return fmt.Sprintf("%s@%s", p.Manifest.PackageName, p.Manifest.Version)
}
