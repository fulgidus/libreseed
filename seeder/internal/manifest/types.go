// Package manifest provides types and validation for LibreSeed package manifests.
// It implements the dual-manifest signature model as specified in DESIGN_DECISIONS.md.
package manifest

// FullManifest represents the complete manifest inside the .tgz package.
// This manifest contains all file hashes and is signed by the publisher.
// It is used to verify package contents integrity.
//
// Location: Inside the .tgz at the root (manifest.json)
// Signature covers: contentHash (hash of all file hashes)
type FullManifest struct {
	// Name is the package name (max 64 bytes).
	Name string `json:"name"`

	// Version is the semantic version string (max 32 bytes).
	Version string `json:"version"`

	// Description is an optional package description.
	Description string `json:"description,omitempty"`

	// Author is the optional package author.
	Author string `json:"author,omitempty"`

	// Files maps relative file paths to their SHA256 hashes.
	// Format: path → "sha256:hexstring"
	// Example: {"dist/bundle.js": "sha256:abc123..."}
	Files map[string]string `json:"files"`

	// ContentHash is the SHA256 hash of the sorted concatenation of file hashes.
	// This is the value that gets signed by the publisher.
	// Format: "sha256:hexstring"
	ContentHash string `json:"contentHash"`

	// PubKey is the publisher's Ed25519 public key.
	// Format: "ed25519:base64string"
	PubKey string `json:"pubkey"`

	// Signature is the Ed25519 signature of the contentHash.
	// Format: "ed25519:base64string"
	Signature string `json:"signature"`
}

// MinimalManifest represents the lightweight manifest used for DHT announcements.
// This manifest is signed separately and announced to the DHT by seeders.
//
// Location: Separate file (e.g., hello-world@1.0.0.minimal.json)
// Signature covers: infohash (hash of the .tgz file)
type MinimalManifest struct {
	// Name is the package name (max 64 bytes).
	Name string `json:"name"`

	// Version is the semantic version string (max 32 bytes).
	Version string `json:"version"`

	// Infohash is the SHA256 hash of the entire .tgz file.
	// Format: "sha256:hexstring"
	Infohash string `json:"infohash"`

	// PubKey is the publisher's Ed25519 public key.
	// Format: "ed25519:base64string"
	// Must match the PubKey in the FullManifest.
	PubKey string `json:"pubkey"`

	// Signature is the Ed25519 signature of the infohash.
	// Format: "ed25519:base64string"
	Signature string `json:"signature"`
}
