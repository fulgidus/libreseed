// Package crypto provides key management utilities for LibreSeed.
package crypto

import (
	"crypto/ed25519"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
)

// KeyManager handles Ed25519 keypair generation, storage, and loading.
// Keys are stored in plaintext hex format in the user's data directory.
//
// Security Model:
//   - Private key stored with 0600 permissions (owner read/write only)
//   - Public key stored with 0644 permissions (owner rw, others read)
//   - Keys auto-generated on first use (first `lbs add` command)
//   - Key rotation not yet supported (future enhancement)
type KeyManager struct {
	// keysDir is the directory where keys are stored
	// Default: ~/.local/share/libreseed/keys/
	keysDir string

	// privateKey is the Ed25519 private key (64 bytes)
	privateKey ed25519.PrivateKey

	// publicKey is the Ed25519 public key (32 bytes)
	publicKey ed25519.PublicKey
}

const (
	// DefaultKeysDir is the default directory for storing keys relative to user data dir
	DefaultKeysDir = "libreseed/keys"

	// PrivateKeyFilename is the filename for the private key
	PrivateKeyFilename = "private.key"

	// PublicKeyFilename is the filename for the public key
	PublicKeyFilename = "public.key"

	// PrivateKeyPerm is the file permission for private key (owner read/write only)
	PrivateKeyPerm = 0600

	// PublicKeyPerm is the file permission for public key (owner rw, others read)
	PublicKeyPerm = 0644
)

// NewKeyManager creates a new KeyManager instance.
//
// If keysDir is empty, it uses the default location:
//   - Linux/Unix: ~/.local/share/libreseed/keys/
//   - Windows: %LOCALAPPDATA%\libreseed\keys\
//
// The manager does NOT automatically load keys; call EnsureKeysExist() to load or generate.
func NewKeyManager(keysDir string) (*KeyManager, error) {
	if keysDir == "" {
		// Use system-appropriate user data directory
		userDataDir, err := os.UserConfigDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get user config dir: %w", err)
		}
		keysDir = filepath.Join(userDataDir, "..", "share", DefaultKeysDir)
	}

	// Clean the path
	keysDir = filepath.Clean(keysDir)

	return &KeyManager{
		keysDir: keysDir,
	}, nil
}

// EnsureKeysExist ensures that keypair exists, generating if necessary.
// This is the main entry point for key management:
//   - If keys exist on disk, loads them
//   - If keys don't exist, generates new keypair and saves to disk
//
// Returns error if generation or loading fails.
func (km *KeyManager) EnsureKeysExist() error {
	privateKeyPath := filepath.Join(km.keysDir, PrivateKeyFilename)
	publicKeyPath := filepath.Join(km.keysDir, PublicKeyFilename)

	// Check if keys already exist
	privateExists := fileExists(privateKeyPath)
	publicExists := fileExists(publicKeyPath)

	// Case 1: Both keys exist - load them
	if privateExists && publicExists {
		return km.LoadKeys()
	}

	// Case 2: Only one key exists - inconsistent state, regenerate
	if privateExists || publicExists {
		fmt.Printf("Warning: Incomplete keypair found in %s, regenerating...\n", km.keysDir)
		// Remove partial keys
		os.Remove(privateKeyPath)
		os.Remove(publicKeyPath)
	}

	// Case 3: No keys exist - generate new keypair
	fmt.Printf("Generating new Ed25519 keypair in %s...\n", km.keysDir)
	return km.GenerateAndSaveKeypair()
}

// GenerateAndSaveKeypair generates a new Ed25519 keypair and saves it to disk.
//
// Process:
//  1. Generate Ed25519 keypair (32-byte public, 64-byte private)
//  2. Encode keys as hex strings
//  3. Create keys directory with 0755 permissions
//  4. Write private key with 0600 permissions
//  5. Write public key with 0644 permissions
//
// Returns error if generation, directory creation, or file writing fails.
func (km *KeyManager) GenerateAndSaveKeypair() error {
	// Generate Ed25519 keypair
	publicKey, privateKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		return fmt.Errorf("failed to generate keypair: %w", err)
	}

	// Store in manager
	km.privateKey = privateKey
	km.publicKey = publicKey

	// Ensure keys directory exists
	if err := os.MkdirAll(km.keysDir, 0755); err != nil {
		return fmt.Errorf("failed to create keys directory: %w", err)
	}

	// Encode keys as hex
	privateHex := hex.EncodeToString(privateKey)
	publicHex := hex.EncodeToString(publicKey)

	// Write private key
	privateKeyPath := filepath.Join(km.keysDir, PrivateKeyFilename)
	if err := os.WriteFile(privateKeyPath, []byte(privateHex), PrivateKeyPerm); err != nil {
		return fmt.Errorf("failed to write private key: %w", err)
	}

	// Write public key
	publicKeyPath := filepath.Join(km.keysDir, PublicKeyFilename)
	if err := os.WriteFile(publicKeyPath, []byte(publicHex), PublicKeyPerm); err != nil {
		return fmt.Errorf("failed to write public key: %w", err)
	}

	fmt.Printf("✓ Keypair generated successfully\n")
	fmt.Printf("  Public key fingerprint: %s\n", km.Fingerprint())
	fmt.Printf("  Private key: %s\n", privateKeyPath)
	fmt.Printf("  Public key:  %s\n", publicKeyPath)

	return nil
}

// LoadKeys loads an existing keypair from disk.
//
// Process:
//  1. Read private and public key files
//  2. Decode hex strings to bytes
//  3. Validate key sizes (32 bytes public, 64 bytes private)
//  4. Verify that public key matches private key
//
// Returns error if files don't exist, decoding fails, or keys are invalid.
func (km *KeyManager) LoadKeys() error {
	privateKeyPath := filepath.Join(km.keysDir, PrivateKeyFilename)
	publicKeyPath := filepath.Join(km.keysDir, PublicKeyFilename)

	// Read private key
	privateHex, err := os.ReadFile(privateKeyPath)
	if err != nil {
		return fmt.Errorf("failed to read private key: %w", err)
	}

	// Read public key
	publicHex, err := os.ReadFile(publicKeyPath)
	if err != nil {
		return fmt.Errorf("failed to read public key: %w", err)
	}

	// Decode private key
	privateKey, err := hex.DecodeString(string(privateHex))
	if err != nil {
		return fmt.Errorf("failed to decode private key: %w", err)
	}

	// Decode public key
	publicKey, err := hex.DecodeString(string(publicHex))
	if err != nil {
		return fmt.Errorf("failed to decode public key: %w", err)
	}

	// Validate sizes
	if len(privateKey) != ed25519.PrivateKeySize {
		return fmt.Errorf("invalid private key size: expected %d bytes, got %d", ed25519.PrivateKeySize, len(privateKey))
	}
	if len(publicKey) != ed25519.PublicKeySize {
		return fmt.Errorf("invalid public key size: expected %d bytes, got %d", ed25519.PublicKeySize, len(publicKey))
	}

	// Verify key pair consistency
	// Ed25519 private key contains public key in last 32 bytes
	derivedPublicKey := privateKey[32:]
	if !bytesEqual(derivedPublicKey, publicKey) {
		return fmt.Errorf("public key does not match private key")
	}

	// Store in manager
	km.privateKey = ed25519.PrivateKey(privateKey)
	km.publicKey = ed25519.PublicKey(publicKey)

	return nil
}

// PrivateKey returns the loaded private key.
// Returns nil if keys haven't been loaded or generated yet.
func (km *KeyManager) PrivateKey() ed25519.PrivateKey {
	return km.privateKey
}

// PublicKey returns the loaded public key.
// Returns nil if keys haven't been loaded or generated yet.
func (km *KeyManager) PublicKey() ed25519.PublicKey {
	return km.publicKey
}

// PublicKeyCrypto returns the loaded public key as crypto.PublicKey.
// Returns error if keys haven't been loaded or generated yet.
func (km *KeyManager) PublicKeyCrypto() (*PublicKey, error) {
	if km.publicKey == nil {
		return nil, fmt.Errorf("keys not loaded")
	}
	return NewPublicKey(km.publicKey)
}

// Fingerprint returns the SHA-256 fingerprint of the public key (first 8 bytes as hex).
// Returns empty string if keys haven't been loaded yet.
func (km *KeyManager) Fingerprint() string {
	if km.publicKey == nil {
		return ""
	}
	pubKey, err := NewPublicKey(km.publicKey)
	if err != nil {
		return ""
	}
	return pubKey.Fingerprint()
}

// KeysDir returns the directory where keys are stored.
func (km *KeyManager) KeysDir() string {
	return km.keysDir
}

// fileExists checks if a file exists and is not a directory.
func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

// bytesEqual compares two byte slices for equality.
func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
