package daemon

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/libreseed/libreseed/pkg/crypto"
	pkgtype "github.com/libreseed/libreseed/pkg/package"
)

// createMultipartRequest creates a multipart form request with a file upload
func createMultipartRequest(packageBytes []byte, filename string) (*http.Request, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return nil, err
	}

	if _, err := part.Write(packageBytes); err != nil {
		return nil, err
	}

	if err := writer.Close(); err != nil {
		return nil, err
	}

	req := httptest.NewRequest("POST", "/packages/add", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	return req, nil
}

// TestDualSignatureIntegration_EndToEnd tests the complete workflow:
// 1. Generate creator and maintainer keypairs
// 2. Create a package with dual signatures
// 3. Upload via HTTP API
// 4. Verify dual-signature validation works
// 5. Verify database persistence
func TestDualSignatureIntegration_EndToEnd(t *testing.T) {
	// Step 1: Generate keypairs
	creatorPub, creatorPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("Failed to generate creator keypair: %v", err)
	}

	maintainerPub, maintainerPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("Failed to generate maintainer keypair: %v", err)
	}

	// Convert to crypto.PublicKey (correct API)
	creatorPublicKey := crypto.PublicKey{
		Algorithm: "ed25519",
		KeyBytes:  creatorPub,
	}
	maintainerPublicKey := crypto.PublicKey{
		Algorithm: "ed25519",
		KeyBytes:  maintainerPub,
	}

	// Step 2: Create package manifest (correct fields)
	manifest := pkgtype.Manifest{
		PackageName:      "test-integration-package",
		Version:          "1.0.0",
		Description:      "Integration test package with dual signatures",
		CreatorPubKey:    creatorPublicKey,
		MaintainerPubKey: maintainerPublicKey,
		ContentHash:      strings.Repeat("a", 64), // Valid 64-char hex SHA-256
		ContentList: []pkgtype.FileEntry{
			{Path: "test.txt", Hash: strings.Repeat("b", 64), Size: 100},
		},
		CreatedAt: time.Now(),
	}

	// Step 3: Serialize manifest for signing
	manifestBytes, err := pkgtype.SerializeManifest(&manifest)
	if err != nil {
		t.Fatalf("Failed to serialize manifest: %v", err)
	}

	// Step 4: Create creator signature (correct API)
	creatorSignature, err := crypto.Sign(creatorPriv, creatorPublicKey, manifestBytes)
	if err != nil {
		t.Fatalf("Failed to create creator signature: %v", err)
	}

	// Step 5: Create maintainer signature (correct API)
	maintainerSignature, err := crypto.Sign(maintainerPriv, maintainerPublicKey, manifestBytes)
	if err != nil {
		t.Fatalf("Failed to create maintainer signature: %v", err)
	}

	// Step 6: Create complete package (correct fields)
	pkg := &pkgtype.Package{
		PackageID:                   strings.Repeat("c", 64), // Valid 64-char hex package ID
		FormatVersion:               "1.0",
		Manifest:                    manifest,
		ManifestSignature:           *creatorSignature,
		MaintainerManifestSignature: *maintainerSignature,
		FilePath:                    "",
		SizeBytes:                   1024,
	}

	// Step 7: Verify dual signature locally before upload (correct API)
	err = crypto.VerifyDualSignature(
		manifestBytes,
		creatorPublicKey,
		creatorSignature,
		maintainerPublicKey,
		maintainerSignature,
	)
	if err != nil {
		t.Fatalf("Dual signature verification failed before upload: %v", err)
	}

	// Step 8: Write package to temporary file
	tmpDir := t.TempDir()
	pkgPath := filepath.Join(tmpDir, "test-package.lspkg")

	err = pkgtype.WritePackageToFile(pkg, pkgPath)
	if err != nil {
		t.Fatalf("Failed to write package file: %v", err)
	}

	// Step 9: Read package back from file
	loadedPkg, err := pkgtype.LoadPackageFromFile(pkgPath)
	if err != nil {
		t.Fatalf("Failed to load package from file: %v", err)
	}

	// Verify loaded package has correct signatures
	if !bytes.Equal(loadedPkg.ManifestSignature.SignedBy.KeyBytes, creatorPub) {
		t.Error("Creator public key mismatch after load")
	}
	if !bytes.Equal(loadedPkg.MaintainerManifestSignature.SignedBy.KeyBytes, maintainerPub) {
		t.Error("Maintainer public key mismatch after load")
	}

	// Step 10: Setup test HTTP server with Daemon (correct API)
	daemonDir := filepath.Join(tmpDir, "daemon")
	config := &DaemonConfig{
		StorageDir:        daemonDir,
		ListenAddr:        "127.0.0.1:0",
		MaxConnections:    10,
		EnableDHT:         false,
		DHTPort:           6881,
		DHTBootstrapNodes: []string{},
		AnnounceInterval:  5 * time.Minute,
		LogLevel:          "info",
	}
	daemon, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create daemon: %v", err)
	}
	defer daemon.Stop()

	// Step 11: Prepare HTTP POST request with multipart form
	pkgBytes, err := os.ReadFile(pkgPath)
	if err != nil {
		t.Fatalf("Failed to read package file: %v", err)
	}

	req, err := createMultipartRequest(pkgBytes, filepath.Base(pkgPath))
	if err != nil {
		t.Fatalf("Failed to create multipart request: %v", err)
	}

	// Step 12: Execute request (use daemon method)
	rr := httptest.NewRecorder()
	daemon.handlePackageAdd(rr, req)

	// Step 13: Verify HTTP response
	if rr.Code != http.StatusCreated {
		body, _ := io.ReadAll(rr.Body)
		t.Fatalf("Expected status 201 Created, got %d. Body: %s", rr.Code, string(body))
	}

	var response map[string]interface{}
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Step 14: Verify response contains expected fields
	if response["status"] != "success" {
		t.Errorf("Expected status 'success', got '%v'", response["status"])
	}

	// Step 15: Verify database persistence
	packages := daemon.packageManager.ListPackages()
	if len(packages) != 1 {
		t.Fatalf("Expected 1 package in database, got %d", len(packages))
	}

	dbPkg := packages[0]
	if dbPkg.Name != "test-integration-package" {
		t.Errorf("Expected package name 'test-integration-package', got '%s'", dbPkg.Name)
	}
	if dbPkg.Version != "1.0.0" {
		t.Errorf("Expected version '1.0.0', got '%s'", dbPkg.Version)
	}

	// Step 16: Verify creator signature stored correctly
	expectedCreatorSig := hex.EncodeToString(creatorSignature.SignedData)
	if dbPkg.ManifestSignature != expectedCreatorSig {
		t.Errorf("Creator signature mismatch in database")
	}

	// Step 17: Verify maintainer signature stored correctly
	expectedMaintainerSig := hex.EncodeToString(maintainerSignature.SignedData)
	if dbPkg.MaintainerManifestSignature != expectedMaintainerSig {
		t.Errorf("Maintainer signature mismatch in database")
	}

	t.Log("✅ End-to-end dual-signature integration test passed")
}

// TestDualSignatureIntegration_InvalidMaintainerSignature tests rejection of invalid maintainer signature
func TestDualSignatureIntegration_InvalidMaintainerSignature(t *testing.T) {
	// Generate keypairs
	creatorPub, creatorPriv, _ := ed25519.GenerateKey(rand.Reader)
	maintainerPub, _, _ := ed25519.GenerateKey(rand.Reader)
	_, maintainerPrivInvalid, _ := ed25519.GenerateKey(rand.Reader) // Wrong key

	creatorPublicKey := crypto.PublicKey{
		Algorithm: "ed25519",
		KeyBytes:  creatorPub,
	}
	maintainerPublicKey := crypto.PublicKey{
		Algorithm: "ed25519",
		KeyBytes:  maintainerPub,
	}

	tmpDir := t.TempDir()

	// Create valid manifest
	manifest := pkgtype.Manifest{
		PackageName:      "test-invalid-maintainer",
		Version:          "1.0.0",
		Description:      "Test package with invalid maintainer signature",
		CreatorPubKey:    creatorPublicKey,
		MaintainerPubKey: maintainerPublicKey,
		ContentHash:      strings.Repeat("a", 64), // Valid 64-char hex SHA-256
		ContentList: []pkgtype.FileEntry{
			{Path: "test.txt", Hash: strings.Repeat("b", 64), Size: 100},
		},
		CreatedAt: time.Now(),
	}

	// Serialize manifest
	manifestBytes, _ := pkgtype.SerializeManifest(&manifest)

	// Creator signs correctly
	creatorSignature, _ := crypto.Sign(creatorPriv, creatorPublicKey, manifestBytes)

	// Maintainer signs with WRONG key (simulating attack/error)
	maintainerSignatureInvalid, _ := crypto.Sign(maintainerPrivInvalid, maintainerPublicKey, manifestBytes)

	// Create package with mismatched maintainer signature
	pkg := &pkgtype.Package{
		PackageID:                   strings.Repeat("d", 64), // Valid 64-char hex package ID
		FormatVersion:               "1.0",
		Manifest:                    manifest,
		ManifestSignature:           *creatorSignature,
		MaintainerManifestSignature: *maintainerSignatureInvalid, // INVALID
		FilePath:                    "",
		SizeBytes:                   1024,
	}

	// Write package to file
	pkgPath := filepath.Join(tmpDir, "invalid-maintainer.lspkg")
	err := pkgtype.WritePackageToFile(pkg, pkgPath)
	if err != nil {
		t.Fatalf("Failed to write package: %v", err)
	}

	// Setup Daemon
	daemonDir := filepath.Join(tmpDir, "daemon")
	config := &DaemonConfig{
		StorageDir:        daemonDir,
		ListenAddr:        "127.0.0.1:0",
		MaxConnections:    10,
		EnableDHT:         false,
		DHTPort:           6881,
		DHTBootstrapNodes: []string{},
		AnnounceInterval:  5 * time.Minute,
		LogLevel:          "info",
	}
	daemon, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create daemon: %v", err)
	}
	defer daemon.Stop()

	// Send request with multipart form
	pkgBytes, _ := os.ReadFile(pkgPath)
	req, err := createMultipartRequest(pkgBytes, "invalid-maintainer.lspkg")
	if err != nil {
		t.Fatalf("Failed to create multipart request: %v", err)
	}
	rr := httptest.NewRecorder()
	daemon.handlePackageAdd(rr, req)

	// Verify rejection (handler returns 401 Unauthorized for signature failures)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401 for invalid maintainer signature, got %d", rr.Code)
	}

	bodyStr := rr.Body.String()
	// Check for signature-related error (support both English and Italian messages)
	if !contains(bodyStr, "Signature verification failed") &&
		!contains(bodyStr, "verifica firma") &&
		!contains(bodyStr, "signature") {
		t.Errorf("Expected error message about signature verification, got: %s", bodyStr)
	}

	// Verify NOT stored in database
	packages := daemon.packageManager.ListPackages()
	if len(packages) != 0 {
		t.Errorf("Expected 0 packages in database after rejection, got %d", len(packages))
	}

	t.Log("✅ Invalid maintainer signature correctly rejected")
}

// TestDualSignatureIntegration_MissingMaintainerSignature tests rejection when maintainer signature is missing
func TestDualSignatureIntegration_MissingMaintainerSignature(t *testing.T) {
	// Generate creator keypair only
	creatorPub, creatorPriv, _ := ed25519.GenerateKey(rand.Reader)
	maintainerPub, _, _ := ed25519.GenerateKey(rand.Reader)

	creatorPublicKey := crypto.PublicKey{
		Algorithm: "ed25519",
		KeyBytes:  creatorPub,
	}
	maintainerPublicKey := crypto.PublicKey{
		Algorithm: "ed25519",
		KeyBytes:  maintainerPub,
	}

	// Create manifest
	manifest := pkgtype.Manifest{
		PackageName:      "test-missing-maintainer",
		Version:          "1.0.0",
		Description:      "Test missing maintainer signature",
		CreatorPubKey:    creatorPublicKey,
		MaintainerPubKey: maintainerPublicKey,
		ContentHash:      strings.Repeat("a", 64), // Valid 64-char hex SHA-256
		ContentList: []pkgtype.FileEntry{
			{Path: "test.txt", Hash: strings.Repeat("b", 64), Size: 100},
		},
		CreatedAt: time.Now(),
	}

	manifestBytes, _ := pkgtype.SerializeManifest(&manifest)

	// Create creator signature
	creatorSignature, _ := crypto.Sign(creatorPriv, creatorPublicKey, manifestBytes)

	// Create package WITHOUT maintainer signature (empty signature)
	emptySignature := crypto.Signature{
		Algorithm:  "",
		SignedBy:   crypto.PublicKey{},
		SignedData: nil,
		SignedAt:   time.Time{},
	}

	pkg := &pkgtype.Package{
		PackageID:                   strings.Repeat("e", 64), // Valid 64-char hex package ID
		FormatVersion:               "1.0",
		Manifest:                    manifest,
		ManifestSignature:           *creatorSignature,
		MaintainerManifestSignature: emptySignature,
		FilePath:                    "",
		SizeBytes:                   1024,
	}

	// Write to file
	tmpDir := t.TempDir()
	pkgPath := filepath.Join(tmpDir, "missing-maintainer.lspkg")
	pkgtype.WritePackageToFile(pkg, pkgPath)

	// Setup Daemon
	daemonDir := filepath.Join(tmpDir, "daemon")
	config := &DaemonConfig{
		StorageDir:        daemonDir,
		ListenAddr:        "127.0.0.1:0",
		MaxConnections:    10,
		EnableDHT:         false,
		DHTPort:           6881,
		DHTBootstrapNodes: []string{},
		AnnounceInterval:  5 * time.Minute,
		LogLevel:          "info",
	}
	daemon, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create daemon: %v", err)
	}
	defer daemon.Stop()

	// Send request with multipart form
	pkgBytes, _ := os.ReadFile(pkgPath)
	req, err := createMultipartRequest(pkgBytes, "missing-maintainer.lspkg")
	if err != nil {
		t.Fatalf("Failed to create multipart request: %v", err)
	}
	rr := httptest.NewRecorder()
	daemon.handlePackageAdd(rr, req)

	// Verify rejection
	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for missing maintainer signature, got %d", rr.Code)
	}

	// Verify NOT stored in database
	packages := daemon.packageManager.ListPackages()
	if len(packages) != 0 {
		t.Errorf("Expected 0 packages in database after rejection, got %d", len(packages))
	}

	t.Log("✅ Missing maintainer signature correctly rejected")
}

// TestDualSignatureIntegration_PackageIDCalculation tests correct Package ID calculation with dual signatures
func TestDualSignatureIntegration_PackageIDCalculation(t *testing.T) {
	// Generate keypairs
	creatorPub, creatorPriv, _ := ed25519.GenerateKey(rand.Reader)
	maintainerPub, maintainerPriv, _ := ed25519.GenerateKey(rand.Reader)

	creatorPublicKey := crypto.PublicKey{
		Algorithm: "ed25519",
		KeyBytes:  creatorPub,
	}
	maintainerPublicKey := crypto.PublicKey{
		Algorithm: "ed25519",
		KeyBytes:  maintainerPub,
	}

	// Create manifest
	manifest := pkgtype.Manifest{
		PackageName:      "test-package-id",
		Version:          "1.0.0",
		Description:      "Test package ID calculation",
		CreatorPubKey:    creatorPublicKey,
		MaintainerPubKey: maintainerPublicKey,
		ContentHash:      strings.Repeat("a", 64), // Valid 64-char hex SHA-256
		ContentList: []pkgtype.FileEntry{
			{Path: "test.txt", Hash: strings.Repeat("b", 64), Size: 100},
		},
		CreatedAt: time.Now(),
	}

	manifestBytes, _ := pkgtype.SerializeManifest(&manifest)

	// Create signatures
	creatorSignature, _ := crypto.Sign(creatorPriv, creatorPublicKey, manifestBytes)
	maintainerSignature, _ := crypto.Sign(maintainerPriv, maintainerPublicKey, manifestBytes)

	// Calculate expected Package ID (SHA256 of manifest bytes)
	hash := sha256.Sum256(manifestBytes)
	expectedPackageID := hex.EncodeToString(hash[:])

	// Create package with calculated PackageID
	pkg := &pkgtype.Package{
		PackageID:                   expectedPackageID,
		FormatVersion:               "1.0",
		Manifest:                    manifest,
		ManifestSignature:           *creatorSignature,
		MaintainerManifestSignature: *maintainerSignature,
		FilePath:                    "",
		SizeBytes:                   1024,
	}

	// Upload package
	tmpDir := t.TempDir()
	pkgPath := filepath.Join(tmpDir, "test-package-id.lspkg")
	pkgtype.WritePackageToFile(pkg, pkgPath)

	// Setup Daemon
	daemonDir := filepath.Join(tmpDir, "daemon")
	config := &DaemonConfig{
		StorageDir:        daemonDir,
		ListenAddr:        "127.0.0.1:0",
		MaxConnections:    10,
		EnableDHT:         false,
		DHTPort:           6881,
		DHTBootstrapNodes: []string{},
		AnnounceInterval:  5 * time.Minute,
		LogLevel:          "info",
	}
	daemon, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create daemon: %v", err)
	}
	defer daemon.Stop()

	pkgBytes, _ := os.ReadFile(pkgPath)
	req, err := createMultipartRequest(pkgBytes, "test-package-id.lspkg")
	if err != nil {
		t.Fatalf("Failed to create multipart request: %v", err)
	}
	rr := httptest.NewRecorder()
	daemon.handlePackageAdd(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("Upload failed with status %d (expected 201 Created). Body: %s", rr.Code, rr.Body.String())
	}

	// Verify Package ID in database
	packages := daemon.packageManager.ListPackages()
	if len(packages) != 1 {
		t.Fatalf("Expected 1 package, got %d", len(packages))
	}

	dbPkg := packages[0]
	if dbPkg.PackageID != expectedPackageID {
		t.Errorf("Package ID mismatch:\nExpected: %s\nGot:      %s",
			expectedPackageID, dbPkg.PackageID)
	}

	t.Log("✅ Package ID calculation verified")
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return bytes.Contains([]byte(s), []byte(substr))
}
