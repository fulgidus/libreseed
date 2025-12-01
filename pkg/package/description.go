// Package packagetypes provides core data structures for LibreSeed packages.
package packagetypes

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"

	"github.com/libreseed/libreseed/pkg/crypto"
)

// MinimalDescription is the OUTER signed structure distributed via DHT.
// It provides lightweight package discovery without requiring the full package download.
//
// Design Decision: The minimal description is separately signed to enable
// trustless distribution through untrusted DHT nodes. Recipients can verify
// the description signature before deciding to download the full package.
type MinimalDescription struct {
	// PackageID is the SHA-256 hash of the complete .lspkg file
	// This links the description to the full package
	PackageID string `yaml:"package_id" json:"package_id"`

	// CreatorPubKey is the Ed25519 public key of the package creator
	// This MUST match the public key in the manifest (verified during package verification)
	CreatorPubKey crypto.PublicKey `yaml:"creator_pubkey" json:"creator_pubkey"`

	// MaintainerPubKey is the Ed25519 public key of the package maintainer
	// This key provides the second signature in the dual-signature trust system
	// May be the same as CreatorPubKey if creator is also maintainer
	MaintainerPubKey crypto.PublicKey `yaml:"maintainer_pubkey" json:"maintainer_pubkey"`

	// Name is the human-readable package name (matches manifest.package_name)
	Name string `yaml:"name" json:"name"`

	// Version follows semantic versioning (matches manifest.version)
	Version string `yaml:"version" json:"version"`

	// ShortDescription provides a brief summary for discovery (max 200 characters)
	ShortDescription string `yaml:"short_description" json:"short_description"`

	// DHTKey is the computed key for DHT storage (SHA-1 of creator_pubkey || name || "libreseed")
	// This enables predictable discovery queries
	DHTKey string `yaml:"dht_key" json:"dht_key"`

	// TorrentInfoHash is the BitTorrent info hash for peer discovery (optional)
	TorrentInfoHash string `yaml:"torrent_info_hash,omitempty" json:"torrent_info_hash,omitempty"`
}

// Validate checks that the MinimalDescription contains all required fields and valid data.
func (d *MinimalDescription) Validate() error {
	if d.PackageID == "" {
		return fmt.Errorf("minimal description: package_id is required")
	}
	if len(d.PackageID) != 64 {
		return fmt.Errorf("minimal description: package_id must be 64-character hex string")
	}
	if _, err := hex.DecodeString(d.PackageID); err != nil {
		return fmt.Errorf("minimal description: package_id must be valid hex: %w", err)
	}
	if d.CreatorPubKey.Algorithm == "" {
		return fmt.Errorf("minimal description: creator_pubkey is required")
	}
	if d.MaintainerPubKey.Algorithm == "" {
		return fmt.Errorf("minimal description: maintainer_pubkey is required")
	}
	if d.Name == "" {
		return fmt.Errorf("minimal description: name is required")
	}
	if d.Version == "" {
		return fmt.Errorf("minimal description: version is required")
	}
	if d.ShortDescription == "" {
		return fmt.Errorf("minimal description: short_description is required")
	}
	if len(d.ShortDescription) > 200 {
		return fmt.Errorf("minimal description: short_description must be ≤200 characters (got %d)", len(d.ShortDescription))
	}
	if d.DHTKey == "" {
		return fmt.Errorf("minimal description: dht_key is required")
	}
	if len(d.DHTKey) != 40 {
		return fmt.Errorf("minimal description: dht_key must be 40-character hex string (SHA-1)")
	}
	if _, err := hex.DecodeString(d.DHTKey); err != nil {
		return fmt.Errorf("minimal description: dht_key must be valid hex: %w", err)
	}

	// If torrent_info_hash is provided, validate it
	if d.TorrentInfoHash != "" {
		if len(d.TorrentInfoHash) != 40 {
			return fmt.Errorf("minimal description: torrent_info_hash must be 40-character hex string (SHA-1)")
		}
		if _, err := hex.DecodeString(d.TorrentInfoHash); err != nil {
			return fmt.Errorf("minimal description: torrent_info_hash must be valid hex: %w", err)
		}
	}

	return nil
}

// ComputeDHTKey generates the DHT storage key for this package.
// Formula: SHA-1(creator_pubkey_bytes || package_name || "libreseed")
//
// This enables deterministic discovery: given a creator's public key and
// package name, anyone can compute the DHT key and query for the package.
func (d *MinimalDescription) ComputeDHTKey() (string, error) {
	if len(d.CreatorPubKey.KeyBytes) == 0 {
		return "", fmt.Errorf("creator_pubkey.key_bytes is required for DHT key computation")
	}
	if d.Name == "" {
		return "", fmt.Errorf("name is required for DHT key computation")
	}

	// Compute SHA-1(key_bytes || name || "libreseed")
	hasher := sha1.New()
	hasher.Write(d.CreatorPubKey.KeyBytes)
	hasher.Write([]byte(d.Name))
	hasher.Write([]byte("libreseed"))
	dhtKey := hex.EncodeToString(hasher.Sum(nil))

	return dhtKey, nil
}

// UpdateDHTKey computes and updates the DHTKey field.
// This should be called before serializing the description for DHT storage.
func (d *MinimalDescription) UpdateDHTKey() error {
	dhtKey, err := d.ComputeDHTKey()
	if err != nil {
		return fmt.Errorf("failed to compute DHT key: %w", err)
	}
	d.DHTKey = dhtKey
	return nil
}

// Fingerprint returns the truncated package ID (first 16 characters) for display.
func (d *MinimalDescription) Fingerprint() string {
	if len(d.PackageID) >= 16 {
		return d.PackageID[:16]
	}
	return d.PackageID
}

// FullName returns the package identifier with version (e.g., "my-library@1.2.3").
func (d *MinimalDescription) FullName() string {
	return fmt.Sprintf("%s@%s", d.Name, d.Version)
}

// CreatorFingerprint returns the truncated creator public key fingerprint for display.
func (d *MinimalDescription) CreatorFingerprint() string {
	return d.CreatorPubKey.Fingerprint()
}

// MaintainerFingerprint returns the truncated maintainer public key fingerprint for display.
func (d *MinimalDescription) MaintainerFingerprint() string {
	return d.MaintainerPubKey.Fingerprint()
}

// IsMaintainerSameAsCreator checks if the maintainer is the same as the creator.
// Returns true if both public keys are identical (same fingerprint).
func (d *MinimalDescription) IsMaintainerSameAsCreator() bool {
	return d.MaintainerPubKey.Fingerprint() == d.CreatorPubKey.Fingerprint()
}
