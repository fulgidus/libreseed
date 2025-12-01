package api

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
	"gopkg.in/yaml.v3"
)

// Permission levels for API keys
const (
	LevelRead  = "read"  // Read-only access (GET endpoints)
	LevelWrite = "write" // Read + write access (GET, POST, DELETE)
	LevelAdmin = "admin" // Full access including key management
)

// APIKey represents an API key with metadata
type APIKey struct {
	ID        string    `yaml:"id"`
	Name      string    `yaml:"name"`
	KeyHash   string    `yaml:"key_hash"` // SHA-256 hash of the actual key
	Level     string    `yaml:"level"`    // read, write, or admin
	CreatedAt time.Time `yaml:"created_at"`
	LastUsed  time.Time `yaml:"last_used,omitempty"`
	Revoked   bool      `yaml:"revoked"`
}

// APIKeyStore manages API keys with file-based persistence
type APIKeyStore struct {
	mu       sync.RWMutex
	filePath string
	Keys     []APIKey `yaml:"keys"`
}

var (
	ErrInvalidKeyFormat = errors.New("invalid API key format")
	ErrKeyNotFound      = errors.New("API key not found")
	ErrKeyRevoked       = errors.New("API key has been revoked")
	ErrInvalidLevel     = errors.New("invalid permission level")
)

// NewAPIKeyStore creates a new API key store
func NewAPIKeyStore(filePath string) (*APIKeyStore, error) {
	store := &APIKeyStore{
		filePath: filePath,
		Keys:     []APIKey{},
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	// Load existing keys if file exists
	if _, err := os.Stat(filePath); err == nil {
		if err := store.load(); err != nil {
			return nil, fmt.Errorf("failed to load keys: %w", err)
		}
	}

	return store, nil
}

// GenerateKey creates a new API key with the given name and permission level
// Returns the plaintext key (which should be shown once to the user)
func (s *APIKeyStore) GenerateKey(name, level string) (string, *APIKey, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Validate name is not empty
	if name == "" {
		return "", nil, errors.New("key name cannot be empty")
	}

	// Validate permission level
	if !isValidLevel(level) {
		return "", nil, ErrInvalidLevel
	}

	// Generate cryptographically secure random key
	keyBytes := make([]byte, 32)
	if _, err := rand.Read(keyBytes); err != nil {
		return "", nil, fmt.Errorf("failed to generate random key: %w", err)
	}

	// Format: lbs_<64-char-hex>
	plaintextKey := "lbs_" + hex.EncodeToString(keyBytes)

	// Hash the key for storage
	hash := sha256.Sum256([]byte(plaintextKey))
	keyHash := hex.EncodeToString(hash[:])

	// Create API key record
	key := APIKey{
		ID:        uuid.New().String(),
		Name:      name,
		KeyHash:   keyHash,
		Level:     level,
		CreatedAt: time.Now(),
		Revoked:   false,
	}

	// Add to store
	s.Keys = append(s.Keys, key)

	// Persist to disk
	if err := s.save(); err != nil {
		return "", nil, fmt.Errorf("failed to save key: %w", err)
	}

	return plaintextKey, &key, nil
}

// ValidateKey validates an API key and returns the key record if valid
func (s *APIKeyStore) ValidateKey(plaintextKey string) (*APIKey, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Validate key format: "lbs_" (4) + 64 hex chars = 68 total
	if len(plaintextKey) != 68 || plaintextKey[:4] != "lbs_" {
		return nil, ErrInvalidKeyFormat
	}

	// Hash the provided key
	hash := sha256.Sum256([]byte(plaintextKey))
	keyHash := hex.EncodeToString(hash[:])

	// Find matching key
	for i := range s.Keys {
		if s.Keys[i].KeyHash == keyHash {
			// Check if revoked
			if s.Keys[i].Revoked {
				return nil, ErrKeyRevoked
			}

			// Return pointer to the key (caller can update LastUsed)
			return &s.Keys[i], nil
		}
	}

	return nil, ErrKeyNotFound
}

// UpdateLastUsed updates the last used timestamp for a key
func (s *APIKeyStore) UpdateLastUsed(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i := range s.Keys {
		if s.Keys[i].ID == id {
			s.Keys[i].LastUsed = time.Now()
			return s.save()
		}
	}

	return ErrKeyNotFound
}

// ListKeys returns all API keys (active and revoked)
func (s *APIKeyStore) ListKeys() []APIKey {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Return a copy to prevent external modification
	keys := make([]APIKey, len(s.Keys))
	copy(keys, s.Keys)
	return keys
}

// RevokeKey marks a key as revoked (soft delete)
func (s *APIKeyStore) RevokeKey(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i := range s.Keys {
		if s.Keys[i].ID == id {
			s.Keys[i].Revoked = true
			return s.save()
		}
	}

	return ErrKeyNotFound
}

// DeleteKey permanently removes a key from storage
func (s *APIKeyStore) DeleteKey(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i := range s.Keys {
		if s.Keys[i].ID == id {
			// Remove from slice
			s.Keys = append(s.Keys[:i], s.Keys[i+1:]...)
			return s.save()
		}
	}

	return ErrKeyNotFound
}

// load reads keys from disk
func (s *APIKeyStore) load() error {
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		return err
	}

	return yaml.Unmarshal(data, s)
}

// save writes keys to disk
func (s *APIKeyStore) save() error {
	data, err := yaml.Marshal(s)
	if err != nil {
		return err
	}

	return os.WriteFile(s.filePath, data, 0600)
}

// isValidLevel checks if a permission level is valid
func isValidLevel(level string) bool {
	return level == LevelRead || level == LevelWrite || level == LevelAdmin
}

// HasPermission checks if a key has the required permission level
func HasPermission(keyLevel, requiredLevel string) bool {
	// Validate inputs
	if !isValidLevel(keyLevel) || !isValidLevel(requiredLevel) {
		return false
	}

	// Admin has all permissions
	if keyLevel == LevelAdmin {
		return true
	}

	// Write has read + write permissions
	if keyLevel == LevelWrite && (requiredLevel == LevelRead || requiredLevel == LevelWrite) {
		return true
	}

	// Read only has read permission
	if keyLevel == LevelRead && requiredLevel == LevelRead {
		return true
	}

	return false
}
