package daemon

import (
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/libreseed/libreseed/pkg/storage"
	"gopkg.in/yaml.v3"
)

// PackageInfo represents metadata about a single package managed by the daemon.
// This is stored in packages.yaml and tracks local package state.
type PackageInfo struct {
	// PackageID is the unique identifier (SHA-256 hash of package file)
	PackageID string `yaml:"package_id"`

	// Name is the human-readable package name
	Name string `yaml:"name"`

	// Version follows semantic versioning (e.g., "1.0.0")
	Version string `yaml:"version"`

	// Description provides a human-readable summary
	Description string `yaml:"description"`

	// FilePath is the absolute path to the package file in storage
	FilePath string `yaml:"file_path"`

	// FileHash is the SHA-256 hash of the package file (hex-encoded)
	FileHash string `yaml:"file_hash"`

	// FileSize is the size of the package file in bytes
	FileSize int64 `yaml:"file_size"`

	// CreatedAt is when this package was added to the daemon
	CreatedAt time.Time `yaml:"created_at"`

	// CreatorFingerprint is the creator's public key fingerprint
	CreatorFingerprint string `yaml:"creator_fingerprint"`

	// ManifestSignature is the hex-encoded signature of the package manifest
	ManifestSignature string `yaml:"manifest_signature"`

	// AnnouncedToDHT indicates if this package has been announced to the DHT
	AnnouncedToDHT bool `yaml:"announced_to_dht"`

	// LastAnnounced is the last time this package was announced to the DHT
	LastAnnounced time.Time `yaml:"last_announced,omitempty"`
}

// PackageManager manages the local package database and metadata.
// It provides thread-safe operations for adding, removing, and querying packages.
type PackageManager struct {
	// packages is the in-memory map of package_id -> PackageInfo
	packages map[string]*PackageInfo

	// storageDir is the directory where package files are stored
	storageDir string

	// metaFile is the path to packages.yaml
	metaFile string

	// mu protects concurrent access to the packages map
	mu sync.RWMutex
}

// NewPackageManager creates a new PackageManager instance.
//
// Parameters:
//   - storageDir: directory where package files are stored
//   - metaFile: path to packages.yaml metadata file
//
// Returns a new PackageManager instance ready to use.
func NewPackageManager(storageDir, metaFile string) *PackageManager {
	return &PackageManager{
		packages:   make(map[string]*PackageInfo),
		storageDir: storageDir,
		metaFile:   metaFile,
	}
}

// LoadState loads package metadata from packages.yaml.
// If the file doesn't exist, it initializes an empty state.
// This should be called during daemon startup.
//
// Returns error if the file exists but cannot be parsed.
func (pm *PackageManager) LoadState() error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// If metadata file doesn't exist, start with empty state
	if !storage.FileExists(pm.metaFile) {
		pm.packages = make(map[string]*PackageInfo)
		return nil
	}

	// Read YAML file
	data, err := os.ReadFile(pm.metaFile)
	if err != nil {
		return fmt.Errorf("failed to read packages metadata: %w", err)
	}

	// Parse YAML into slice of packages
	var packageList []*PackageInfo
	if err := yaml.Unmarshal(data, &packageList); err != nil {
		return fmt.Errorf("failed to parse packages metadata: %w", err)
	}

	// Build map from slice
	pm.packages = make(map[string]*PackageInfo)
	for _, pkg := range packageList {
		pm.packages[pkg.PackageID] = pkg
	}

	return nil
}

// SaveState saves the current package metadata to packages.yaml atomically.
// This should be called after any modification to the package database.
//
// Returns error if the write fails.
func (pm *PackageManager) SaveState() error {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	// Convert map to slice for YAML serialization
	packageList := make([]*PackageInfo, 0, len(pm.packages))
	for _, pkg := range pm.packages {
		packageList = append(packageList, pkg)
	}

	// Marshal to YAML
	data, err := yaml.Marshal(packageList)
	if err != nil {
		return fmt.Errorf("failed to marshal packages metadata: %w", err)
	}

	// Write atomically using storage utility
	if err := storage.AtomicWriteFile(pm.metaFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write packages metadata: %w", err)
	}

	return nil
}

// AddPackage adds a new package to the database and persists the change.
// If a package with the same ID already exists, it returns an error.
//
// Parameters:
//   - info: complete package metadata
//
// Returns error if the package already exists or save fails.
func (pm *PackageManager) AddPackage(info *PackageInfo) error {
	if err := pm.validatePackageInfo(info); err != nil {
		return fmt.Errorf("invalid package info: %w", err)
	}

	pm.mu.Lock()
	defer pm.mu.Unlock()

	// Check for duplicate
	if _, exists := pm.packages[info.PackageID]; exists {
		return fmt.Errorf("package with ID %s already exists", info.PackageID)
	}

	// Add to map
	pm.packages[info.PackageID] = info

	// Save state immediately
	pm.mu.Unlock() // Unlock before SaveState (which will acquire RLock)
	err := pm.SaveState()
	pm.mu.Lock() // Re-lock to maintain defer unlock

	return err
}

// RemovePackage removes a package from the database and deletes the package file.
// This operation is permanent and cannot be undone.
//
// Parameters:
//   - packageID: the package ID to remove
//
// Returns error if the package doesn't exist or deletion fails.
func (pm *PackageManager) RemovePackage(packageID string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// Check if package exists
	pkg, exists := pm.packages[packageID]
	if !exists {
		return fmt.Errorf("package with ID %s not found", packageID)
	}

	// Delete the package file
	if err := storage.SafeRemove(pkg.FilePath); err != nil {
		return fmt.Errorf("failed to delete package file: %w", err)
	}

	// Remove from map
	delete(pm.packages, packageID)

	// Save state immediately
	pm.mu.Unlock()
	err := pm.SaveState()
	pm.mu.Lock()

	return err
}

// GetPackage retrieves package metadata by ID.
//
// Parameters:
//   - packageID: the package ID to retrieve
//
// Returns the package info and true if found, or nil and false if not found.
func (pm *PackageManager) GetPackage(packageID string) (*PackageInfo, bool) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	pkg, exists := pm.packages[packageID]
	return pkg, exists
}

// ListPackages returns a list of all packages in the database.
// The returned slice is a copy and can be safely modified by the caller.
//
// Returns a slice of all package metadata.
func (pm *PackageManager) ListPackages() []*PackageInfo {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	// Create a copy of the slice to avoid race conditions
	packageList := make([]*PackageInfo, 0, len(pm.packages))
	for _, pkg := range pm.packages {
		packageList = append(packageList, pkg)
	}

	return packageList
}

// PackageExists checks if a package with the given ID exists.
//
// Parameters:
//   - packageID: the package ID to check
//
// Returns true if the package exists, false otherwise.
func (pm *PackageManager) PackageExists(packageID string) bool {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	_, exists := pm.packages[packageID]
	return exists
}

// UpdateAnnouncementStatus updates the DHT announcement status for a package.
//
// Parameters:
//   - packageID: the package ID to update
//   - announced: whether the package has been announced
//
// Returns error if the package doesn't exist or save fails.
func (pm *PackageManager) UpdateAnnouncementStatus(packageID string, announced bool) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	pkg, exists := pm.packages[packageID]
	if !exists {
		return fmt.Errorf("package with ID %s not found", packageID)
	}

	pkg.AnnouncedToDHT = announced
	if announced {
		pkg.LastAnnounced = time.Now()
	}

	pm.mu.Unlock()
	err := pm.SaveState()
	pm.mu.Lock()

	return err
}

// GetStorageDir returns the package storage directory path.
func (pm *PackageManager) GetStorageDir() string {
	return pm.storageDir
}

// GetMetaFile returns the metadata file path.
func (pm *PackageManager) GetMetaFile() string {
	return pm.metaFile
}

// Count returns the total number of packages in the database.
func (pm *PackageManager) Count() int {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return len(pm.packages)
}

// validatePackageInfo validates that PackageInfo contains all required fields.
func (pm *PackageManager) validatePackageInfo(info *PackageInfo) error {
	if info == nil {
		return fmt.Errorf("package info is nil")
	}

	if info.PackageID == "" {
		return fmt.Errorf("package_id is required")
	}

	// Validate package ID format (must be 64-character hex string)
	if len(info.PackageID) != 64 {
		return fmt.Errorf("package_id must be 64-character hex string")
	}
	if _, err := hex.DecodeString(info.PackageID); err != nil {
		return fmt.Errorf("package_id must be valid hex: %w", err)
	}

	if info.Name == "" {
		return fmt.Errorf("name is required")
	}

	if info.Version == "" {
		return fmt.Errorf("version is required")
	}

	if info.Description == "" {
		return fmt.Errorf("description is required")
	}

	if info.FilePath == "" {
		return fmt.Errorf("file_path is required")
	}

	// Validate file path is absolute
	if !filepath.IsAbs(info.FilePath) {
		return fmt.Errorf("file_path must be absolute path")
	}

	if info.FileHash == "" {
		return fmt.Errorf("file_hash is required")
	}

	// Validate file hash format (must be 64-character hex string)
	if len(info.FileHash) != 64 {
		return fmt.Errorf("file_hash must be 64-character hex string")
	}
	if _, err := hex.DecodeString(info.FileHash); err != nil {
		return fmt.Errorf("file_hash must be valid hex: %w", err)
	}

	if info.FileSize <= 0 {
		return fmt.Errorf("file_size must be positive")
	}

	if info.CreatedAt.IsZero() {
		return fmt.Errorf("created_at timestamp is required")
	}

	if info.CreatorFingerprint == "" {
		return fmt.Errorf("creator_fingerprint is required")
	}

	// Validate creator fingerprint format (must be 16-character hex string)
	if len(info.CreatorFingerprint) != 16 {
		return fmt.Errorf("creator_fingerprint must be 16-character hex string")
	}
	if _, err := hex.DecodeString(info.CreatorFingerprint); err != nil {
		return fmt.Errorf("creator_fingerprint must be valid hex: %w", err)
	}

	if info.ManifestSignature == "" {
		return fmt.Errorf("manifest_signature is required")
	}

	// Validate manifest signature format (must be hex-encoded)
	if _, err := hex.DecodeString(info.ManifestSignature); err != nil {
		return fmt.Errorf("manifest_signature must be valid hex: %w", err)
	}

	return nil
}
