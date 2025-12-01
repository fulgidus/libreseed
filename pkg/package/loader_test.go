// Package package provides serialization and deserialization for LibreSeed packages.
package packagetypes

import (
	"crypto/ed25519"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/libreseed/libreseed/pkg/crypto"
)

// TestLoadPackageFromFile_ValidPackage tests loading a valid .lspkg file from disk.
func TestLoadPackageFromFile_ValidPackage(t *testing.T) {
	// Create a test package file
	testFile, pkg := createTestPackageFile(t)
	defer os.Remove(testFile)

	// Load the package from disk
	loadedPkg, err := LoadPackageFromFile(testFile)
	if err != nil {
		t.Fatalf("LoadPackageFromFile failed: %v", err)
	}

	// Verify loaded package matches original
	if loadedPkg.Manifest.PackageName != pkg.Manifest.PackageName {
		t.Errorf("PackageName mismatch: got %s, want %s", loadedPkg.Manifest.PackageName, pkg.Manifest.PackageName)
	}
	if loadedPkg.Manifest.Version != pkg.Manifest.Version {
		t.Errorf("Version mismatch: got %s, want %s", loadedPkg.Manifest.Version, pkg.Manifest.Version)
	}
	if loadedPkg.FilePath != testFile {
		t.Errorf("FilePath mismatch: got %s, want %s", loadedPkg.FilePath, testFile)
	}
	if loadedPkg.PackageID == "" {
		t.Error("PackageID should not be empty after loading")
	}
	if loadedPkg.SizeBytes <= 0 {
		t.Error("SizeBytes should be positive after loading")
	}
}

// TestLoadPackageFromFile_NonExistentFile tests loading a non-existent file.
func TestLoadPackageFromFile_NonExistentFile(t *testing.T) {
	_, err := LoadPackageFromFile("/nonexistent/path/to/package.lspkg")
	if err == nil {
		t.Fatal("Expected error for non-existent file, got nil")
	}
}

// TestLoadPackageFromFile_InvalidYAML tests loading a file with malformed YAML.
func TestLoadPackageFromFile_InvalidYAML(t *testing.T) {
	// Create a temporary file with invalid YAML
	tmpFile := filepath.Join(t.TempDir(), "invalid.lspkg")
	invalidYAML := []byte("this is not valid YAML: {[broken")
	if err := os.WriteFile(tmpFile, invalidYAML, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer os.Remove(tmpFile)

	// Attempt to load the invalid file
	_, err := LoadPackageFromFile(tmpFile)
	if err == nil {
		t.Fatal("Expected error for invalid YAML, got nil")
	}
}

// TestLoadPackageFromBytes_Success tests loading a package from memory.
func TestLoadPackageFromBytes_Success(t *testing.T) {
	// Create a test package
	pkg := createTestPackage(t)

	// Serialize to bytes
	data, err := SerializePackage(pkg)
	if err != nil {
		t.Fatalf("SerializePackage failed: %v", err)
	}

	// Load from bytes
	loadedPkg, err := LoadPackageFromBytes(data)
	if err != nil {
		t.Fatalf("LoadPackageFromBytes failed: %v", err)
	}

	// Verify loaded package matches original
	if loadedPkg.Manifest.PackageName != pkg.Manifest.PackageName {
		t.Errorf("PackageName mismatch: got %s, want %s", loadedPkg.Manifest.PackageName, pkg.Manifest.PackageName)
	}
	if loadedPkg.Manifest.Version != pkg.Manifest.Version {
		t.Errorf("Version mismatch: got %s, want %s", loadedPkg.Manifest.Version, pkg.Manifest.Version)
	}
}

// TestLoadPackageFromBytes_InvalidData tests loading invalid byte data.
func TestLoadPackageFromBytes_InvalidData(t *testing.T) {
	invalidData := []byte("not valid YAML at all")
	_, err := LoadPackageFromBytes(invalidData)
	if err == nil {
		t.Fatal("Expected error for invalid data, got nil")
	}
}

// TestSerializePackage_RoundTrip tests that serialization and deserialization preserve data.
func TestSerializePackage_RoundTrip(t *testing.T) {
	// Create original package
	original := createTestPackage(t)

	// Serialize
	data, err := SerializePackage(original)
	if err != nil {
		t.Fatalf("SerializePackage failed: %v", err)
	}

	// Deserialize
	loaded, err := LoadPackageFromBytes(data)
	if err != nil {
		t.Fatalf("LoadPackageFromBytes failed: %v", err)
	}

	// Verify round-trip consistency
	if loaded.Manifest.PackageName != original.Manifest.PackageName {
		t.Errorf("PackageName mismatch after round-trip: got %s, want %s",
			loaded.Manifest.PackageName, original.Manifest.PackageName)
	}
	if loaded.Manifest.Version != original.Manifest.Version {
		t.Errorf("Version mismatch after round-trip: got %s, want %s",
			loaded.Manifest.Version, original.Manifest.Version)
	}
	if loaded.Manifest.ContentHash != original.Manifest.ContentHash {
		t.Errorf("ContentHash mismatch after round-trip: got %s, want %s",
			loaded.Manifest.ContentHash, original.Manifest.ContentHash)
	}
	if len(loaded.ManifestSignature.SignedData) != len(original.ManifestSignature.SignedData) {
		t.Errorf("ManifestSignature length mismatch after round-trip: got %d, want %d",
			len(loaded.ManifestSignature.SignedData), len(original.ManifestSignature.SignedData))
	}
	if len(loaded.MaintainerManifestSignature.SignedData) != len(original.MaintainerManifestSignature.SignedData) {
		t.Errorf("MaintainerManifestSignature length mismatch after round-trip: got %d, want %d",
			len(loaded.MaintainerManifestSignature.SignedData), len(original.MaintainerManifestSignature.SignedData))
	}
}

// TestSerializeManifest_Success tests manifest serialization.
func TestSerializeManifest_Success(t *testing.T) {
	pkg := createTestPackage(t)

	// Serialize manifest
	data, err := SerializeManifest(&pkg.Manifest)
	if err != nil {
		t.Fatalf("SerializeManifest failed: %v", err)
	}

	if len(data) == 0 {
		t.Error("SerializeManifest returned empty data")
	}
}

// TestWritePackageToFile_Success tests writing a package to disk.
func TestWritePackageToFile_Success(t *testing.T) {
	// Create a test package
	pkg := createTestPackage(t)

	// Write to temporary file
	tmpFile := filepath.Join(t.TempDir(), "test_package.lspkg")
	err := WritePackageToFile(pkg, tmpFile)
	if err != nil {
		t.Fatalf("WritePackageToFile failed: %v", err)
	}
	defer os.Remove(tmpFile)

	// Verify file exists
	if _, err := os.Stat(tmpFile); os.IsNotExist(err) {
		t.Error("Package file was not created")
	}

	// Verify package metadata was updated
	if pkg.FilePath != tmpFile {
		t.Errorf("FilePath not updated: got %s, want %s", pkg.FilePath, tmpFile)
	}
	if pkg.PackageID == "" {
		t.Error("PackageID should not be empty after writing")
	}
	if pkg.SizeBytes <= 0 {
		t.Error("SizeBytes should be positive after writing")
	}

	// Load the file back and verify
	loadedPkg, err := LoadPackageFromFile(tmpFile)
	if err != nil {
		t.Fatalf("LoadPackageFromFile failed: %v", err)
	}

	if loadedPkg.Manifest.PackageName != pkg.Manifest.PackageName {
		t.Errorf("PackageName mismatch after write: got %s, want %s",
			loadedPkg.Manifest.PackageName, pkg.Manifest.PackageName)
	}
}

// TestSerializePackage_InvalidPackage tests serialization of an invalid package.
func TestSerializePackage_InvalidPackage(t *testing.T) {
	// Create an invalid package (missing required fields)
	pkg := &Package{
		PackageID:     "abc123", // Invalid: not 64 chars
		FormatVersion: "1.1",
		Manifest: Manifest{
			PackageName: "", // Invalid: empty
		},
		SizeBytes: 100,
	}

	// Attempt to serialize
	_, err := SerializePackage(pkg)
	if err == nil {
		t.Fatal("Expected error for invalid package, got nil")
	}
}

// =============================================================================
// Test Helpers
// =============================================================================

// generateTestKeypair generates a test Ed25519 keypair for use in tests.
func generateTestKeypair(t *testing.T) (ed25519.PrivateKey, *crypto.PublicKey, error) {
	t.Helper()

	// Generate Ed25519 keypair
	pubKeyRaw, privKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		return nil, nil, err
	}

	// Wrap public key in crypto.PublicKey
	pubKey, err := crypto.NewPublicKey(pubKeyRaw)
	if err != nil {
		return nil, nil, err
	}

	return privKey, pubKey, nil
}

// createTestPackage creates a minimal valid package for testing.
func createTestPackage(t *testing.T) *Package {
	t.Helper()

	// Generate test keypairs (creator and maintainer)
	creatorPrivKey, creatorPubKey, err := generateTestKeypair(t)
	if err != nil {
		t.Fatalf("Failed to generate creator keypair: %v", err)
	}

	maintainerPrivKey, maintainerPubKey, err := generateTestKeypair(t)
	if err != nil {
		t.Fatalf("Failed to generate maintainer keypair: %v", err)
	}

	// Create a valid manifest
	manifest := Manifest{
		PackageName:      "test-package",
		Version:          "1.0.0",
		Description:      "A test package for unit tests",
		CreatorPubKey:    *creatorPubKey,
		MaintainerPubKey: *maintainerPubKey,
		ContentHash:      strings.Repeat("a", 64), // 64-char hex string
		ContentList: []FileEntry{
			{
				Path: "test.txt",
				Hash: strings.Repeat("b", 64), // 64-char hex string
				Size: 1024,
				Mode: 0644,
			},
		},
		CreatedAt: time.Now().UTC(),
	}

	// Serialize manifest for signing
	manifestData, err := SerializeManifest(&manifest)
	if err != nil {
		t.Fatalf("Failed to serialize manifest: %v", err)
	}

	// Sign manifest with creator key
	creatorSig, err := crypto.Sign(creatorPrivKey, *creatorPubKey, manifestData)
	if err != nil {
		t.Fatalf("Failed to create creator signature: %v", err)
	}

	// Sign manifest with maintainer key
	maintainerSig, err := crypto.Sign(maintainerPrivKey, *maintainerPubKey, manifestData)
	if err != nil {
		t.Fatalf("Failed to create maintainer signature: %v", err)
	}

	// Create package with both signatures
	pkg := &Package{
		PackageID:                   strings.Repeat("c", 64), // 64-char hex string
		FormatVersion:               "1.1",
		Manifest:                    manifest,
		ManifestSignature:           *creatorSig,
		MaintainerManifestSignature: *maintainerSig,
		SizeBytes:                   2048,
	}

	return pkg
}

// createTestPackageFile creates a test .lspkg file on disk and returns the path and package.
func createTestPackageFile(t *testing.T) (string, *Package) {
	t.Helper()

	// Create a test package
	pkg := createTestPackage(t)

	// Write to temporary file
	tmpFile := filepath.Join(t.TempDir(), "test_package.lspkg")
	err := WritePackageToFile(pkg, tmpFile)
	if err != nil {
		t.Fatalf("Failed to write test package file: %v", err)
	}

	return tmpFile, pkg
}
