package daemon

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

// MaintainerInfo represents a registered maintainer in the system.
// Maintainers are trusted parties who can co-sign packages.
type MaintainerInfo struct {
	// Fingerprint is the unique identifier (last 8 bytes of public key hash, hex-encoded)
	Fingerprint string `yaml:"fingerprint"`

	// Name is the human-readable maintainer name or organization
	Name string `yaml:"name"`

	// PublicKey is the Ed25519 public key (hex-encoded, 64 characters)
	PublicKey string `yaml:"public_key"`

	// Email is an optional contact email for the maintainer
	Email string `yaml:"email,omitempty"`

	// RegisteredAt is when this maintainer was registered
	RegisteredAt time.Time `yaml:"registered_at"`

	// Active indicates if the maintainer is currently active
	Active bool `yaml:"active"`

	// PackagesSigned is the count of packages co-signed by this maintainer
	PackagesSigned int `yaml:"packages_signed"`

	// LastSignedAt is when this maintainer last signed a package
	LastSignedAt time.Time `yaml:"last_signed_at,omitempty"`
}

// PendingSignature represents a package awaiting maintainer co-signature.
type PendingSignature struct {
	// PackageID is the package awaiting signature
	PackageID string `yaml:"package_id"`

	// PackageName is the human-readable package name
	PackageName string `yaml:"package_name"`

	// PackageVersion is the package version
	PackageVersion string `yaml:"package_version"`

	// CreatorFingerprint is the creator's public key fingerprint
	CreatorFingerprint string `yaml:"creator_fingerprint"`

	// ManifestHash is the hash of the manifest to be signed (hex-encoded)
	ManifestHash string `yaml:"manifest_hash"`

	// CreatedAt is when this pending signature was created
	CreatedAt time.Time `yaml:"created_at"`

	// ExpiresAt is when this pending signature request expires
	ExpiresAt time.Time `yaml:"expires_at"`
}

// MaintainerRegistry manages registered maintainers and pending signatures.
// It provides thread-safe operations with YAML file persistence.
type MaintainerRegistry struct {
	mu          sync.RWMutex
	filePath    string
	Maintainers []MaintainerInfo   `yaml:"maintainers"`
	Pending     []PendingSignature `yaml:"pending_signatures"`
}

var (
	ErrMaintainerNotFound      = errors.New("maintainer not found")
	ErrMaintainerAlreadyExists = errors.New("maintainer already registered")
	ErrMaintainerInactive      = errors.New("maintainer is inactive")
	ErrPendingNotFound         = errors.New("pending signature not found")
	ErrPendingExpired          = errors.New("pending signature has expired")
	ErrInvalidFingerprint      = errors.New("invalid fingerprint format")
	ErrInvalidPublicKey        = errors.New("invalid public key format")
)

// NewMaintainerRegistry creates a new MaintainerRegistry instance.
// It loads existing state from file if it exists.
//
// Parameters:
//   - filePath: path to the maintainers.yaml file
//
// Returns error if directory creation or file loading fails.
func NewMaintainerRegistry(filePath string) (*MaintainerRegistry, error) {
	registry := &MaintainerRegistry{
		filePath:    filePath,
		Maintainers: []MaintainerInfo{},
		Pending:     []PendingSignature{},
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	// Load existing state if file exists
	if _, err := os.Stat(filePath); err == nil {
		if err := registry.load(); err != nil {
			return nil, fmt.Errorf("failed to load maintainer registry: %w", err)
		}
	}

	return registry, nil
}

// Register adds a new maintainer to the registry.
//
// Parameters:
//   - fingerprint: unique identifier (16-character hex string)
//   - name: human-readable maintainer name
//   - publicKey: Ed25519 public key (64-character hex string)
//   - email: optional contact email
//
// Returns error if maintainer already exists or validation fails.
func (r *MaintainerRegistry) Register(fingerprint, name, publicKey, email string) (*MaintainerInfo, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Validate fingerprint format (16-character hex)
	if len(fingerprint) != 16 {
		return nil, ErrInvalidFingerprint
	}

	// Validate public key format (64-character hex for Ed25519)
	if len(publicKey) != 64 {
		return nil, ErrInvalidPublicKey
	}

	// Validate name is not empty
	if name == "" {
		return nil, errors.New("maintainer name cannot be empty")
	}

	// Check if maintainer already exists
	for _, m := range r.Maintainers {
		if m.Fingerprint == fingerprint {
			return nil, ErrMaintainerAlreadyExists
		}
	}

	// Create new maintainer
	maintainer := MaintainerInfo{
		Fingerprint:    fingerprint,
		Name:           name,
		PublicKey:      publicKey,
		Email:          email,
		RegisteredAt:   time.Now(),
		Active:         true,
		PackagesSigned: 0,
	}

	// Add to registry
	r.Maintainers = append(r.Maintainers, maintainer)

	// Persist to disk
	if err := r.save(); err != nil {
		// Rollback on save failure
		r.Maintainers = r.Maintainers[:len(r.Maintainers)-1]
		return nil, fmt.Errorf("failed to save maintainer: %w", err)
	}

	return &maintainer, nil
}

// Get retrieves a maintainer by fingerprint.
//
// Parameters:
//   - fingerprint: the maintainer's fingerprint
//
// Returns the maintainer info or error if not found.
func (r *MaintainerRegistry) Get(fingerprint string) (*MaintainerInfo, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for i := range r.Maintainers {
		if r.Maintainers[i].Fingerprint == fingerprint {
			return &r.Maintainers[i], nil
		}
	}

	return nil, ErrMaintainerNotFound
}

// List returns all registered maintainers.
// Returns a copy to prevent external modification.
func (r *MaintainerRegistry) List() []MaintainerInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Return a copy
	maintainers := make([]MaintainerInfo, len(r.Maintainers))
	copy(maintainers, r.Maintainers)
	return maintainers
}

// ListActive returns only active maintainers.
func (r *MaintainerRegistry) ListActive() []MaintainerInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var active []MaintainerInfo
	for _, m := range r.Maintainers {
		if m.Active {
			active = append(active, m)
		}
	}
	return active
}

// Deactivate marks a maintainer as inactive.
//
// Parameters:
//   - fingerprint: the maintainer's fingerprint
//
// Returns error if maintainer not found or save fails.
func (r *MaintainerRegistry) Deactivate(fingerprint string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i := range r.Maintainers {
		if r.Maintainers[i].Fingerprint == fingerprint {
			r.Maintainers[i].Active = false
			return r.save()
		}
	}

	return ErrMaintainerNotFound
}

// Activate marks a maintainer as active.
//
// Parameters:
//   - fingerprint: the maintainer's fingerprint
//
// Returns error if maintainer not found or save fails.
func (r *MaintainerRegistry) Activate(fingerprint string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i := range r.Maintainers {
		if r.Maintainers[i].Fingerprint == fingerprint {
			r.Maintainers[i].Active = true
			return r.save()
		}
	}

	return ErrMaintainerNotFound
}

// IncrementSignCount increments the packages signed count and updates last signed time.
//
// Parameters:
//   - fingerprint: the maintainer's fingerprint
//
// Returns error if maintainer not found or save fails.
func (r *MaintainerRegistry) IncrementSignCount(fingerprint string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i := range r.Maintainers {
		if r.Maintainers[i].Fingerprint == fingerprint {
			r.Maintainers[i].PackagesSigned++
			r.Maintainers[i].LastSignedAt = time.Now()
			return r.save()
		}
	}

	return ErrMaintainerNotFound
}

// AddPending adds a package to the pending signatures list.
//
// Parameters:
//   - packageID: unique package identifier
//   - packageName: human-readable package name
//   - packageVersion: package version
//   - creatorFingerprint: creator's public key fingerprint
//   - manifestHash: hash of manifest to be signed
//   - ttl: time-to-live for the pending request
//
// Returns error if save fails.
func (r *MaintainerRegistry) AddPending(packageID, packageName, packageVersion, creatorFingerprint, manifestHash string, ttl time.Duration) (*PendingSignature, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Remove any existing pending signature for this package
	for i := len(r.Pending) - 1; i >= 0; i-- {
		if r.Pending[i].PackageID == packageID {
			r.Pending = append(r.Pending[:i], r.Pending[i+1:]...)
		}
	}

	now := time.Now()
	pending := PendingSignature{
		PackageID:          packageID,
		PackageName:        packageName,
		PackageVersion:     packageVersion,
		CreatorFingerprint: creatorFingerprint,
		ManifestHash:       manifestHash,
		CreatedAt:          now,
		ExpiresAt:          now.Add(ttl),
	}

	r.Pending = append(r.Pending, pending)

	if err := r.save(); err != nil {
		// Rollback on save failure
		r.Pending = r.Pending[:len(r.Pending)-1]
		return nil, fmt.Errorf("failed to save pending signature: %w", err)
	}

	return &pending, nil
}

// GetPending retrieves a pending signature by package ID.
//
// Parameters:
//   - packageID: the package ID
//
// Returns the pending signature or error if not found or expired.
func (r *MaintainerRegistry) GetPending(packageID string) (*PendingSignature, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for i := range r.Pending {
		if r.Pending[i].PackageID == packageID {
			// Check if expired
			if time.Now().After(r.Pending[i].ExpiresAt) {
				return nil, ErrPendingExpired
			}
			return &r.Pending[i], nil
		}
	}

	return nil, ErrPendingNotFound
}

// ListPending returns all non-expired pending signatures.
func (r *MaintainerRegistry) ListPending() []PendingSignature {
	r.mu.RLock()
	defer r.mu.RUnlock()

	now := time.Now()
	var pending []PendingSignature
	for _, p := range r.Pending {
		if now.Before(p.ExpiresAt) {
			pending = append(pending, p)
		}
	}
	return pending
}

// RemovePending removes a pending signature by package ID.
//
// Parameters:
//   - packageID: the package ID
//
// Returns error if not found or save fails.
func (r *MaintainerRegistry) RemovePending(packageID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i := range r.Pending {
		if r.Pending[i].PackageID == packageID {
			r.Pending = append(r.Pending[:i], r.Pending[i+1:]...)
			return r.save()
		}
	}

	return ErrPendingNotFound
}

// CleanupExpired removes all expired pending signatures.
// Returns the number of entries removed.
func (r *MaintainerRegistry) CleanupExpired() (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	removed := 0
	newPending := make([]PendingSignature, 0, len(r.Pending))

	for _, p := range r.Pending {
		if now.Before(p.ExpiresAt) {
			newPending = append(newPending, p)
		} else {
			removed++
		}
	}

	if removed > 0 {
		r.Pending = newPending
		if err := r.save(); err != nil {
			return 0, fmt.Errorf("failed to save after cleanup: %w", err)
		}
	}

	return removed, nil
}

// Count returns the number of registered maintainers.
func (r *MaintainerRegistry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.Maintainers)
}

// PendingCount returns the number of pending signatures.
func (r *MaintainerRegistry) PendingCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.Pending)
}

// load reads the registry from disk.
func (r *MaintainerRegistry) load() error {
	data, err := os.ReadFile(r.filePath)
	if err != nil {
		return err
	}

	return yaml.Unmarshal(data, r)
}

// save writes the registry to disk.
func (r *MaintainerRegistry) save() error {
	data, err := yaml.Marshal(r)
	if err != nil {
		return err
	}

	return os.WriteFile(r.filePath, data, 0600)
}
