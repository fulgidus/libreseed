// Package manifest provides types and validation for LibreSeed package manifests.
package manifest

import (
	"archive/tar"
	"compress/gzip"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
)

// Validation errors
var (
	ErrInvalidHashFormat      = fmt.Errorf("invalid hash format")
	ErrInvalidPubkeyFormat    = fmt.Errorf("invalid pubkey format")
	ErrInvalidSignatureFormat = fmt.Errorf("invalid signature format")
	ErrContentHashMismatch    = fmt.Errorf("contentHash mismatch")
	ErrInfohashMismatch       = fmt.Errorf("infohash mismatch")
	ErrSignatureVerifyFailed  = fmt.Errorf("signature verification failed")
	ErrPubkeyMismatch         = fmt.Errorf("pubkey mismatch between manifests")
	ErrManifestNotFound       = fmt.Errorf("manifest.json not found in tarball")
)

// ComputeContentHash calculates the contentHash from a map of file hashes.
// Algorithm: SHA256(sorted concatenation of "sha256:hash" strings)
//
// Example:
//
//	files := {
//	  "dist/bundle.js": "sha256:abc123",
//	  "src/index.js":   "sha256:def456"
//	}
//	→ sorted: ["dist/bundle.js", "src/index.js"]
//	→ concat: "sha256:abc123sha256:def456"
//	→ hash:   SHA256(concat) = "sha256:final_hash"
func ComputeContentHash(files map[string]string) (string, error) {
	if len(files) == 0 {
		return "", fmt.Errorf("files map is empty")
	}

	// 1. Sort file paths alphabetically
	paths := make([]string, 0, len(files))
	for path := range files {
		paths = append(paths, path)
	}
	sort.Strings(paths)

	// 2. Concatenate hashes in sorted order
	var hashConcat strings.Builder
	for _, path := range paths {
		hashValue := files[path]
		if !strings.HasPrefix(hashValue, "sha256:") {
			return "", fmt.Errorf("%w: expected 'sha256:' prefix in hash for %s", ErrInvalidHashFormat, path)
		}
		hashConcat.WriteString(hashValue)
	}

	// 3. Hash the concatenation
	concatenated := hashConcat.String()
	hash := sha256.Sum256([]byte(concatenated))
	contentHash := "sha256:" + hex.EncodeToString(hash[:])

	return contentHash, nil
}

// VerifyContentHash validates that the contentHash in the manifest matches
// the computed hash from the files map.
func VerifyContentHash(manifest *FullManifest) error {
	computed, err := ComputeContentHash(manifest.Files)
	if err != nil {
		return fmt.Errorf("failed to compute contentHash: %w", err)
	}

	if computed != manifest.ContentHash {
		return fmt.Errorf("%w: expected %s, got %s", ErrContentHashMismatch, computed, manifest.ContentHash)
	}

	return nil
}

// ComputeInfohash calculates the SHA256 hash of a .tgz file.
// This is the value that gets signed in the MinimalManifest.
func ComputeInfohash(tgzPath string) (string, error) {
	file, err := os.Open(tgzPath)
	if err != nil {
		return "", fmt.Errorf("failed to open tarball: %w", err)
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", fmt.Errorf("failed to hash tarball: %w", err)
	}

	infohash := "sha256:" + hex.EncodeToString(hash.Sum(nil))
	return infohash, nil
}

// ExtractManifest extracts the manifest.json from a .tgz file.
// Returns the parsed FullManifest.
func ExtractManifest(tgzPath string) (*FullManifest, error) {
	file, err := os.Open(tgzPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open tarball: %w", err)
	}
	defer file.Close()

	// Decompress gzip
	gzReader, err := gzip.NewReader(file)
	if err != nil {
		return nil, fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzReader.Close()

	// Read tar archive
	tarReader := tar.NewReader(gzReader)

	// Search for manifest.json
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read tar entry: %w", err)
		}

		// Check if this is manifest.json at the root
		// Accept both "manifest.json" and "./manifest.json"
		name := strings.TrimPrefix(header.Name, "./")
		if name == "manifest.json" {
			// Parse manifest
			var manifest FullManifest
			decoder := json.NewDecoder(tarReader)
			if err := decoder.Decode(&manifest); err != nil {
				return nil, fmt.Errorf("failed to parse manifest.json: %w", err)
			}
			return &manifest, nil
		}
	}

	return nil, ErrManifestNotFound
}

// VerifySignature verifies an Ed25519 signature.
//
// Parameters:
//   - pubkeyStr: Format "ed25519:base64string"
//   - signatureStr: Format "ed25519:base64string"
//   - message: The data that was signed
func VerifySignature(pubkeyStr, signatureStr string, message []byte) error {
	// Parse pubkey
	if !strings.HasPrefix(pubkeyStr, "ed25519:") {
		return fmt.Errorf("%w: expected 'ed25519:' prefix in pubkey", ErrInvalidPubkeyFormat)
	}
	pubkeyB64 := strings.TrimPrefix(pubkeyStr, "ed25519:")
	pubkeyBytes, err := base64.StdEncoding.DecodeString(pubkeyB64)
	if err != nil {
		return fmt.Errorf("%w: failed to decode pubkey: %v", ErrInvalidPubkeyFormat, err)
	}
	if len(pubkeyBytes) != ed25519.PublicKeySize {
		return fmt.Errorf("%w: invalid pubkey length %d, expected %d", ErrInvalidPubkeyFormat, len(pubkeyBytes), ed25519.PublicKeySize)
	}
	pubkey := ed25519.PublicKey(pubkeyBytes)

	// Parse signature
	if !strings.HasPrefix(signatureStr, "ed25519:") {
		return fmt.Errorf("%w: expected 'ed25519:' prefix in signature", ErrInvalidSignatureFormat)
	}
	signatureB64 := strings.TrimPrefix(signatureStr, "ed25519:")
	signature, err := base64.StdEncoding.DecodeString(signatureB64)
	if err != nil {
		return fmt.Errorf("%w: failed to decode signature: %v", ErrInvalidSignatureFormat, err)
	}
	if len(signature) != ed25519.SignatureSize {
		return fmt.Errorf("%w: invalid signature length %d, expected %d", ErrInvalidSignatureFormat, len(signature), ed25519.SignatureSize)
	}

	// Verify signature
	if !ed25519.Verify(pubkey, message, signature) {
		return ErrSignatureVerifyFailed
	}

	return nil
}

// VerifyFullManifestSignature verifies the signature in a FullManifest.
// The signature covers the contentHash field.
// The signature is over the raw 32-byte hash, not the "sha256:..." string.
func VerifyFullManifestSignature(manifest *FullManifest) error {
	// Extract raw hash bytes from "sha256:hexstring" format
	if !strings.HasPrefix(manifest.ContentHash, "sha256:") {
		return fmt.Errorf("%w: expected 'sha256:' prefix in contentHash", ErrInvalidHashFormat)
	}

	hashHex := strings.TrimPrefix(manifest.ContentHash, "sha256:")
	hashBytes, err := hex.DecodeString(hashHex)
	if err != nil {
		return fmt.Errorf("%w: failed to decode contentHash hex: %v", ErrInvalidHashFormat, err)
	}
	if len(hashBytes) != 32 {
		return fmt.Errorf("%w: invalid hash length %d, expected 32", ErrInvalidHashFormat, len(hashBytes))
	}

	// The signature signs the raw 32-byte hash
	return VerifySignature(manifest.PubKey, manifest.Signature, hashBytes)
}

// VerifyMinimalManifestSignature verifies the signature in a MinimalManifest.
// The signature covers the infohash field.
// The signature is over the raw 32-byte hash, not the "sha256:..." string.
func VerifyMinimalManifestSignature(manifest *MinimalManifest) error {
	// Extract raw hash bytes from "sha256:hexstring" format
	if !strings.HasPrefix(manifest.Infohash, "sha256:") {
		return fmt.Errorf("%w: expected 'sha256:' prefix in infohash", ErrInvalidHashFormat)
	}

	hashHex := strings.TrimPrefix(manifest.Infohash, "sha256:")
	hashBytes, err := hex.DecodeString(hashHex)
	if err != nil {
		return fmt.Errorf("%w: failed to decode infohash hex: %v", ErrInvalidHashFormat, err)
	}
	if len(hashBytes) != 32 {
		return fmt.Errorf("%w: invalid hash length %d, expected 32", ErrInvalidHashFormat, len(hashBytes))
	}

	// The signature signs the raw 32-byte hash
	return VerifySignature(manifest.PubKey, manifest.Signature, hashBytes)
}

// ValidatePackage performs complete validation of a package.
// This is the main validation flow orchestrator.
//
// Validation steps:
//  1. Verify MinimalManifest signature (infohash signature)
//  2. Compute actual infohash of .tgz file
//  3. Verify infohash matches MinimalManifest.Infohash
//  4. Extract FullManifest from .tgz
//  5. Verify FullManifest signature (contentHash signature)
//  6. Verify contentHash matches computed hash
//  7. Verify pubkeys match between both manifests
//
// Returns the extracted FullManifest if validation succeeds.
func ValidatePackage(tgzPath string, minimalManifest *MinimalManifest) (*FullManifest, error) {
	// Step 1: Verify minimal manifest signature
	if err := VerifyMinimalManifestSignature(minimalManifest); err != nil {
		return nil, fmt.Errorf("minimal manifest signature verification failed: %w", err)
	}

	// Step 2: Compute infohash
	computedInfohash, err := ComputeInfohash(tgzPath)
	if err != nil {
		return nil, fmt.Errorf("failed to compute infohash: %w", err)
	}

	// Step 3: Verify infohash matches
	if computedInfohash != minimalManifest.Infohash {
		return nil, fmt.Errorf("%w: expected %s, got %s", ErrInfohashMismatch, minimalManifest.Infohash, computedInfohash)
	}

	// Step 4: Extract full manifest
	fullManifest, err := ExtractManifest(tgzPath)
	if err != nil {
		return nil, fmt.Errorf("failed to extract manifest: %w", err)
	}

	// Step 5: Verify full manifest signature
	if err := VerifyFullManifestSignature(fullManifest); err != nil {
		return nil, fmt.Errorf("full manifest signature verification failed: %w", err)
	}

	// Step 6: Verify contentHash
	if err := VerifyContentHash(fullManifest); err != nil {
		return nil, fmt.Errorf("contentHash verification failed: %w", err)
	}

	// Step 7: Verify pubkeys match
	if fullManifest.PubKey != minimalManifest.PubKey {
		return nil, fmt.Errorf("%w: full=%s, minimal=%s", ErrPubkeyMismatch, fullManifest.PubKey, minimalManifest.PubKey)
	}

	// All validations passed
	return fullManifest, nil
}

// ValidateMinimalManifest performs basic field validation on a MinimalManifest.
func ValidateMinimalManifest(m *MinimalManifest) error {
	if m.Name == "" {
		return fmt.Errorf("name is empty")
	}
	if m.Version == "" {
		return fmt.Errorf("version is empty")
	}
	if m.Infohash == "" {
		return fmt.Errorf("infohash is empty")
	}
	if !strings.HasPrefix(m.Infohash, "sha256:") {
		return fmt.Errorf("%w: infohash missing 'sha256:' prefix", ErrInvalidHashFormat)
	}
	if m.PubKey == "" {
		return fmt.Errorf("pubkey is empty")
	}
	if !strings.HasPrefix(m.PubKey, "ed25519:") {
		return fmt.Errorf("%w: pubkey missing 'ed25519:' prefix", ErrInvalidPubkeyFormat)
	}
	if m.Signature == "" {
		return fmt.Errorf("signature is empty")
	}
	if !strings.HasPrefix(m.Signature, "ed25519:") {
		return fmt.Errorf("%w: signature missing 'ed25519:' prefix", ErrInvalidSignatureFormat)
	}
	return nil
}

// ValidateFullManifest performs basic field validation on a FullManifest.
func ValidateFullManifest(m *FullManifest) error {
	if m.Name == "" {
		return fmt.Errorf("name is empty")
	}
	if m.Version == "" {
		return fmt.Errorf("version is empty")
	}
	if len(m.Files) == 0 {
		return fmt.Errorf("files map is empty")
	}
	for path, hash := range m.Files {
		if !strings.HasPrefix(hash, "sha256:") {
			return fmt.Errorf("%w: file hash missing 'sha256:' prefix for %s", ErrInvalidHashFormat, path)
		}
	}
	if m.ContentHash == "" {
		return fmt.Errorf("contentHash is empty")
	}
	if !strings.HasPrefix(m.ContentHash, "sha256:") {
		return fmt.Errorf("%w: contentHash missing 'sha256:' prefix", ErrInvalidHashFormat)
	}
	if m.PubKey == "" {
		return fmt.Errorf("pubkey is empty")
	}
	if !strings.HasPrefix(m.PubKey, "ed25519:") {
		return fmt.Errorf("%w: pubkey missing 'ed25519:' prefix", ErrInvalidPubkeyFormat)
	}
	if m.Signature == "" {
		return fmt.Errorf("signature is empty")
	}
	if !strings.HasPrefix(m.Signature, "ed25519:") {
		return fmt.Errorf("%w: signature missing 'ed25519:' prefix", ErrInvalidSignatureFormat)
	}
	return nil
}
