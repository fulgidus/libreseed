package daemon

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
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
	packagetypes "github.com/libreseed/libreseed/pkg/package"
)

// createTestPackageFile creates a valid .lspkg file for testing
func createTestPackageFile(t *testing.T) ([]byte, *packagetypes.Package) {
	t.Helper()

	// Create temporary keys directory
	tempDir := t.TempDir()
	keysDir := filepath.Join(tempDir, "keys")
	if err := os.MkdirAll(keysDir, 0755); err != nil {
		t.Fatalf("failed to create keys directory: %v", err)
	}

	// Initialize key managers for creator and maintainer
	creatorKeyManager, err := crypto.NewKeyManager(filepath.Join(keysDir, "creator"))
	if err != nil {
		t.Fatalf("failed to create creator key manager: %v", err)
	}
	if err := creatorKeyManager.EnsureKeysExist(); err != nil {
		t.Fatalf("failed to ensure creator keys: %v", err)
	}

	maintainerKeyManager, err := crypto.NewKeyManager(filepath.Join(keysDir, "maintainer"))
	if err != nil {
		t.Fatalf("failed to create maintainer key manager: %v", err)
	}
	if err := maintainerKeyManager.EnsureKeysExist(); err != nil {
		t.Fatalf("failed to ensure maintainer keys: %v", err)
	}

	// Create manifest
	manifest := &packagetypes.Manifest{
		PackageName: "test-package",
		Version:     "1.0.0",
		Description: "Test package for unit tests",
		ContentHash: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		CreatorPubKey: crypto.PublicKey{
			Algorithm: "ed25519",
			KeyBytes:  creatorKeyManager.PublicKey(),
		},
		MaintainerPubKey: crypto.PublicKey{
			Algorithm: "ed25519",
			KeyBytes:  maintainerKeyManager.PublicKey(),
		},
		ContentList: []packagetypes.FileEntry{
			{
				Path: "README.md",
				Hash: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855", // SHA-256 of empty string
				Size: 0,
				Mode: 0644,
			},
		},
		CreatedAt: time.Now(),
	}

	// Serialize manifest for signing
	manifestData, err := packagetypes.SerializeManifest(manifest)
	if err != nil {
		t.Fatalf("failed to serialize manifest: %v", err)
	}

	// Sign with creator key
	creatorSig, err := crypto.Sign(
		creatorKeyManager.PrivateKey(),
		manifest.CreatorPubKey,
		manifestData,
	)
	if err != nil {
		t.Fatalf("failed to sign with creator key: %v", err)
	}

	// Sign with maintainer key
	maintainerSig, err := crypto.Sign(
		maintainerKeyManager.PrivateKey(),
		manifest.MaintainerPubKey,
		manifestData,
	)
	if err != nil {
		t.Fatalf("failed to sign with maintainer key: %v", err)
	}

	// Create package with placeholder PackageID (will be recomputed after serialization)
	pkg := &packagetypes.Package{
		PackageID:                   strings.Repeat("0", 64), // Placeholder: will be computed from serialized data
		FormatVersion:               "1.0",
		Manifest:                    *manifest,
		ManifestSignature:           *creatorSig,
		MaintainerManifestSignature: *maintainerSig,
		SizeBytes:                   1024,
	}

	// Serialize to bytes
	pkgData, err := packagetypes.SerializePackage(pkg)
	if err != nil {
		t.Fatalf("failed to serialize package: %v", err)
	}

	// Compute PackageID from serialized data
	pkg.PackageID = pkg.ComputePackageID(pkgData)
	pkg.SizeBytes = int64(len(pkgData))

	// Re-serialize with correct PackageID and SizeBytes
	pkgData, err = packagetypes.SerializePackage(pkg)
	if err != nil {
		t.Fatalf("failed to re-serialize package with computed PackageID: %v", err)
	}

	return pkgData, pkg
}

// createInvalidPackageFile creates an invalid .lspkg file for testing
func createInvalidPackageFile() []byte {
	return []byte("INVALID_YAML_DATA\n!!@@##$$\n")
}

// TestHandlePackageAdd_InvalidMethod tests that non-POST methods return 405
func TestHandlePackageAdd_InvalidMethod(t *testing.T) {
	methods := []string{http.MethodGet, http.MethodPut, http.MethodDelete, http.MethodPatch}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			// Create daemon with real PackageManager
			tempDir := t.TempDir()
			packagesDir := filepath.Join(tempDir, "packages")
			os.MkdirAll(packagesDir, 0755)

			pm := NewPackageManager(packagesDir, filepath.Join(tempDir, "packages.yaml"))

			config := &DaemonConfig{
				StorageDir: tempDir,
				ListenAddr: "127.0.0.1:0",
				EnableDHT:  false,
			}
			d := &Daemon{
				config:         config,
				state:          NewDaemonState(),
				stats:          NewDaemonStatistics(),
				packageManager: pm,
			}

			// Create request
			req := httptest.NewRequest(method, "/packages/add", nil)
			w := httptest.NewRecorder()

			// Call handler
			d.handlePackageAdd(w, req)

			// Verify response
			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("expected status %d, got %d", http.StatusMethodNotAllowed, w.Code)
			}
		})
	}
}

// TestHandlePackageAdd_MalformedMultipartForm tests handling of malformed multipart data
func TestHandlePackageAdd_MalformedMultipartForm(t *testing.T) {
	tempDir := t.TempDir()
	packagesDir := filepath.Join(tempDir, "packages")
	os.MkdirAll(packagesDir, 0755)

	pm := NewPackageManager(packagesDir, filepath.Join(tempDir, "packages.yaml"))

	config := &DaemonConfig{
		StorageDir: tempDir,
		ListenAddr: "127.0.0.1:0",
		EnableDHT:  false,
	}
	d := &Daemon{
		config:         config,
		state:          NewDaemonState(),
		stats:          NewDaemonStatistics(),
		packageManager: pm,
	}

	// Create request with invalid multipart data
	req := httptest.NewRequest(http.MethodPost, "/packages/add", strings.NewReader("NOT_MULTIPART_DATA"))
	req.Header.Set("Content-Type", "multipart/form-data; boundary=----INVALID")
	w := httptest.NewRecorder()

	// Call handler
	d.handlePackageAdd(w, req)

	// Verify response
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
	if !strings.Contains(w.Body.String(), "Failed to parse form") {
		t.Errorf("expected error message about parsing form, got: %s", w.Body.String())
	}
}

// TestHandlePackageAdd_MissingFileField tests handling when file field is missing
func TestHandlePackageAdd_MissingFileField(t *testing.T) {
	tempDir := t.TempDir()
	packagesDir := filepath.Join(tempDir, "packages")
	os.MkdirAll(packagesDir, 0755)

	pm := NewPackageManager(packagesDir, filepath.Join(tempDir, "packages.yaml"))

	config := &DaemonConfig{
		StorageDir: tempDir,
		ListenAddr: "127.0.0.1:0",
		EnableDHT:  false,
	}
	d := &Daemon{
		config:         config,
		state:          NewDaemonState(),
		stats:          NewDaemonStatistics(),
		packageManager: pm,
	}

	// Create multipart form without file field
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	writer.WriteField("other_field", "some_value")
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/packages/add", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()

	// Call handler
	d.handlePackageAdd(w, req)

	// Verify response
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
	if !strings.Contains(w.Body.String(), "Failed to get file") {
		t.Errorf("expected error about missing file, got: %s", w.Body.String())
	}
}

// TestHandlePackageAdd_InvalidLspkgFormat tests handling of invalid .lspkg file
func TestHandlePackageAdd_InvalidLspkgFormat(t *testing.T) {
	tempDir := t.TempDir()
	packagesDir := filepath.Join(tempDir, "packages")
	os.MkdirAll(packagesDir, 0755)

	pm := NewPackageManager(packagesDir, filepath.Join(tempDir, "packages.yaml"))

	config := &DaemonConfig{
		StorageDir: tempDir,
		ListenAddr: "127.0.0.1:0",
		EnableDHT:  false,
	}
	d := &Daemon{
		config:         config,
		state:          NewDaemonState(),
		stats:          NewDaemonStatistics(),
		packageManager: pm,
	}

	// Create multipart form with invalid .lspkg data
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, _ := writer.CreateFormFile("file", "invalid.lspkg")
	part.Write(createInvalidPackageFile())
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/packages/add", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()

	// Call handler
	d.handlePackageAdd(w, req)

	// Verify response
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
	if !strings.Contains(w.Body.String(), "Failed to parse .lspkg file") {
		t.Errorf("expected error about parsing .lspkg, got: %s", w.Body.String())
	}
}

// TestHandlePackageAdd_DHTDisabled tests successful package add with DHT disabled
func TestHandlePackageAdd_DHTDisabled(t *testing.T) {
	tempDir := t.TempDir()
	packagesDir := filepath.Join(tempDir, "packages")
	os.MkdirAll(packagesDir, 0755)

	pm := NewPackageManager(packagesDir, filepath.Join(tempDir, "packages.yaml"))

	config := &DaemonConfig{
		StorageDir: tempDir,
		ListenAddr: "127.0.0.1:0",
		EnableDHT:  false,
	}
	d := &Daemon{
		config:         config,
		state:          NewDaemonState(),
		stats:          NewDaemonStatistics(),
		packageManager: pm,
	}

	// Create valid package
	pkgData, pkg := createTestPackageFile(t)

	// Create multipart form
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, _ := writer.CreateFormFile("file", "test.lspkg")
	part.Write(pkgData)
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/packages/add", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()

	// Call handler
	d.handlePackageAdd(w, req)

	// Verify response
	if w.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d", http.StatusCreated, w.Code)
		t.Logf("Response body: %s", w.Body.String())
	}

	// Parse response
	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Verify response fields
	if response["status"] != "success" {
		t.Errorf("expected status 'success', got %v", response["status"])
	}
	if response["package_id"] != pkg.PackageID {
		t.Errorf("expected package_id %s, got %v", pkg.PackageID, response["package_id"])
	}
	if response["verified"] != true {
		t.Errorf("expected verified=true, got %v", response["verified"])
	}

	// Verify state updates
	stateSnapshot := d.state.Snapshot()
	if stateSnapshot.ActivePackages != 1 {
		t.Errorf("expected ActivePackages=1, got %d", stateSnapshot.ActivePackages)
	}

	statsSnapshot := d.stats.Snapshot()
	if statsSnapshot.TotalPackagesSeeded != 1 {
		t.Errorf("expected TotalPackagesSeeded=1, got %d", statsSnapshot.TotalPackagesSeeded)
	}
}

// TestHandlePackageAdd_DHTEnabled tests successful package add with DHT enabled
func TestHandlePackageAdd_DHTEnabled(t *testing.T) {
	tempDir := t.TempDir()
	packagesDir := filepath.Join(tempDir, "packages")
	os.MkdirAll(packagesDir, 0755)

	pm := NewPackageManager(packagesDir, filepath.Join(tempDir, "packages.yaml"))

	config := &DaemonConfig{
		StorageDir: tempDir,
		ListenAddr: "127.0.0.1:0",
		EnableDHT:  false, // TODO: Refactor to use interfaces for proper DHT testing
	}
	d := &Daemon{
		config:         config,
		state:          NewDaemonState(),
		stats:          NewDaemonStatistics(),
		packageManager: pm,
		announcer:      nil,
		dhtClient:      nil,
	}

	// Create valid package
	pkgData, _ := createTestPackageFile(t)

	// Create multipart form
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, _ := writer.CreateFormFile("file", "test.lspkg")
	part.Write(pkgData)
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/packages/add", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()

	// Call handler
	d.handlePackageAdd(w, req)

	// Verify response
	if w.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d", http.StatusCreated, w.Code)
		t.Logf("Response body: %s", w.Body.String())
	}

	// Note: DHT testing disabled - TODO: refactor to use interfaces for proper DHT mocking
}

// TestHandlePackageList_InvalidMethod tests that non-GET methods return 405
func TestHandlePackageList_InvalidMethod(t *testing.T) {
	methods := []string{http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			tempDir := t.TempDir()
			packagesDir := filepath.Join(tempDir, "packages")
			os.MkdirAll(packagesDir, 0755)

			pm := NewPackageManager(packagesDir, filepath.Join(tempDir, "packages.yaml"))

			config := &DaemonConfig{
				StorageDir: tempDir,
				ListenAddr: "127.0.0.1:0",
				EnableDHT:  false,
			}
			d := &Daemon{
				config:         config,
				state:          NewDaemonState(),
				stats:          NewDaemonStatistics(),
				packageManager: pm,
			}

			req := httptest.NewRequest(method, "/packages/list", nil)
			w := httptest.NewRecorder()

			d.handlePackageList(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("expected status %d, got %d", http.StatusMethodNotAllowed, w.Code)
			}
		})
	}
}

// TestHandlePackageList_EmptyList tests listing when no packages exist
func TestHandlePackageList_EmptyList(t *testing.T) {
	tempDir := t.TempDir()
	packagesDir := filepath.Join(tempDir, "packages")
	os.MkdirAll(packagesDir, 0755)

	pm := NewPackageManager(packagesDir, filepath.Join(tempDir, "packages.yaml"))

	config := &DaemonConfig{
		StorageDir: tempDir,
		ListenAddr: "127.0.0.1:0",
		EnableDHT:  false,
	}
	d := &Daemon{
		config:         config,
		state:          NewDaemonState(),
		stats:          NewDaemonStatistics(),
		packageManager: pm,
	}

	req := httptest.NewRequest(http.MethodGet, "/packages/list", nil)
	w := httptest.NewRecorder()

	d.handlePackageList(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response["status"] != "success" {
		t.Errorf("expected status 'success', got %v", response["status"])
	}
	if response["count"] != float64(0) {
		t.Errorf("expected count=0, got %v", response["count"])
	}
}

// TestHandlePackageList_MultiplePackages tests listing multiple packages
func TestHandlePackageList_MultiplePackages(t *testing.T) {
	tempDir := t.TempDir()
	packagesDir := filepath.Join(tempDir, "packages")
	os.MkdirAll(packagesDir, 0755)

	pm := NewPackageManager(packagesDir, filepath.Join(tempDir, "packages.yaml"))

	// Create three test packages with complete metadata
	for i := 1; i <= 3; i++ {
		pkgBytes, pkg := createTestPackageFile(t)

		// Compute file hash
		fileHash := sha256.Sum256(pkgBytes)
		fileHashHex := hex.EncodeToString(fileHash[:])

		// Create unique package file
		pkgFileName := fmt.Sprintf("test-package-%d.lspkg", i)
		pkgFilePath := filepath.Join(packagesDir, pkgFileName)
		if err := os.WriteFile(pkgFilePath, pkgBytes, 0644); err != nil {
			t.Fatalf("failed to write package file %d: %v", i, err)
		}

		// Override package name and version to make them unique
		pkg.Manifest.PackageName = fmt.Sprintf("package-%d", i)
		pkg.Manifest.Version = fmt.Sprintf("%d.0.0", i)

		// Add complete PackageInfo
		pm.AddPackage(&PackageInfo{
			PackageID:                   pkg.PackageID,
			Name:                        pkg.Manifest.PackageName,
			Version:                     pkg.Manifest.Version,
			Description:                 pkg.Manifest.Description,
			FilePath:                    pkgFilePath,
			FileHash:                    fileHashHex,
			FileSize:                    int64(len(pkgBytes)),
			CreatedAt:                   time.Now(),
			CreatorFingerprint:          pkg.Manifest.CreatorPubKey.Fingerprint(),
			ManifestSignature:           hex.EncodeToString(pkg.ManifestSignature.SignedData),
			MaintainerFingerprint:       pkg.Manifest.MaintainerPubKey.Fingerprint(),
			MaintainerManifestSignature: hex.EncodeToString(pkg.MaintainerManifestSignature.SignedData),
		})
	}

	config := &DaemonConfig{
		StorageDir: tempDir,
		ListenAddr: "127.0.0.1:0",
		EnableDHT:  false,
	}
	d := &Daemon{
		config:         config,
		state:          NewDaemonState(),
		stats:          NewDaemonStatistics(),
		packageManager: pm,
	}

	req := httptest.NewRequest(http.MethodGet, "/packages/list", nil)
	w := httptest.NewRecorder()

	d.handlePackageList(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response["count"] != float64(3) {
		t.Errorf("expected count=3, got %v", response["count"])
	}

	packages, ok := response["packages"].([]interface{})
	if !ok || len(packages) != 3 {
		t.Errorf("expected 3 packages in response, got %v", response["packages"])
	}
}

// TestHandlePackageRemove_InvalidMethod tests that invalid methods return 405
func TestHandlePackageRemove_InvalidMethod(t *testing.T) {
	methods := []string{http.MethodGet, http.MethodPut, http.MethodPatch}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			tempDir := t.TempDir()
			packagesDir := filepath.Join(tempDir, "packages")
			os.MkdirAll(packagesDir, 0755)

			pm := NewPackageManager(packagesDir, filepath.Join(tempDir, "packages.yaml"))

			config := &DaemonConfig{
				StorageDir: tempDir,
				ListenAddr: "127.0.0.1:0",
				EnableDHT:  false,
			}
			d := &Daemon{
				config:         config,
				state:          NewDaemonState(),
				stats:          NewDaemonStatistics(),
				packageManager: pm,
			}

			req := httptest.NewRequest(method, "/packages/remove?package_id=test", nil)
			w := httptest.NewRecorder()

			d.handlePackageRemove(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("expected status %d, got %d", http.StatusMethodNotAllowed, w.Code)
			}
		})
	}
}

// TestHandlePackageRemove_DELETEWithQueryParameter tests successful removal via DELETE
func TestHandlePackageRemove_DELETEWithQueryParameter(t *testing.T) {
	tempDir := t.TempDir()
	packagesDir := filepath.Join(tempDir, "packages")
	os.MkdirAll(packagesDir, 0755)

	pm := NewPackageManager(packagesDir, filepath.Join(tempDir, "packages.yaml"))

	// Create a complete test package with all required fields
	pkgBytes, pkg := createTestPackageFile(t)
	fileHash := sha256.Sum256(pkgBytes)

	// Write package file to disk
	testFilePath := filepath.Join(packagesDir, pkg.PackageID+".lspkg")
	if err := os.WriteFile(testFilePath, pkgBytes, 0644); err != nil {
		t.Fatalf("failed to write package file: %v", err)
	}

	// Add complete package info to manager
	pm.AddPackage(&PackageInfo{
		PackageID:                   pkg.PackageID,
		Name:                        pkg.Manifest.PackageName,
		Version:                     pkg.Manifest.Version,
		Description:                 pkg.Manifest.Description,
		FilePath:                    testFilePath,
		FileHash:                    hex.EncodeToString(fileHash[:]),
		FileSize:                    int64(len(pkgBytes)),
		CreatedAt:                   pkg.Manifest.CreatedAt,
		CreatorFingerprint:          pkg.Manifest.CreatorPubKey.Fingerprint(),
		ManifestSignature:           hex.EncodeToString(pkg.ManifestSignature.SignedData),
		MaintainerFingerprint:       pkg.Manifest.MaintainerPubKey.Fingerprint(),
		MaintainerManifestSignature: hex.EncodeToString(pkg.MaintainerManifestSignature.SignedData),
	})

	config := &DaemonConfig{
		StorageDir: tempDir,
		ListenAddr: "127.0.0.1:0",
		EnableDHT:  false,
	}
	d := &Daemon{
		config:         config,
		state:          NewDaemonState(),
		stats:          NewDaemonStatistics(),
		packageManager: pm,
		announcer:      nil, // DHT disabled
		dhtClient:      nil, // DHT disabled
	}

	d.state.mu.Lock()
	d.state.ActivePackages = 1
	d.state.mu.Unlock()

	req := httptest.NewRequest(http.MethodDelete, "/packages/remove?package_id="+pkg.PackageID, nil)
	w := httptest.NewRecorder()

	d.handlePackageRemove(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
		t.Logf("Response body: %s", w.Body.String())
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response["status"] != "success" {
		t.Errorf("expected status 'success', got %v", response["status"])
	}

	// Verify state update
	stateSnapshot := d.state.Snapshot()
	if stateSnapshot.ActivePackages != 0 {
		t.Errorf("expected ActivePackages=0, got %d", stateSnapshot.ActivePackages)
	}

	// Verify file was deleted
	if _, err := os.Stat(testFilePath); !os.IsNotExist(err) {
		t.Error("expected package file to be deleted")
	}
}

// TestHandlePackageRemove_POSTWithJSONBody tests successful removal via POST
func TestHandlePackageRemove_POSTWithJSONBody(t *testing.T) {
	tempDir := t.TempDir()
	packagesDir := filepath.Join(tempDir, "packages")
	os.MkdirAll(packagesDir, 0755)

	pm := NewPackageManager(packagesDir, filepath.Join(tempDir, "packages.yaml"))

	// Create a complete test package with all required fields
	pkgBytes, pkg := createTestPackageFile(t)
	fileHash := sha256.Sum256(pkgBytes)

	// Write package file to disk
	testFilePath := filepath.Join(packagesDir, pkg.PackageID+".lspkg")
	if err := os.WriteFile(testFilePath, pkgBytes, 0644); err != nil {
		t.Fatalf("failed to write package file: %v", err)
	}

	// Add complete package info to manager
	pm.AddPackage(&PackageInfo{
		PackageID:                   pkg.PackageID,
		Name:                        pkg.Manifest.PackageName,
		Version:                     pkg.Manifest.Version,
		Description:                 pkg.Manifest.Description,
		FilePath:                    testFilePath,
		FileHash:                    hex.EncodeToString(fileHash[:]),
		FileSize:                    int64(len(pkgBytes)),
		CreatedAt:                   pkg.Manifest.CreatedAt,
		CreatorFingerprint:          pkg.Manifest.CreatorPubKey.Fingerprint(),
		ManifestSignature:           hex.EncodeToString(pkg.ManifestSignature.SignedData),
		MaintainerFingerprint:       pkg.Manifest.MaintainerPubKey.Fingerprint(),
		MaintainerManifestSignature: hex.EncodeToString(pkg.MaintainerManifestSignature.SignedData),
	})

	config := &DaemonConfig{
		StorageDir: tempDir,
		ListenAddr: "127.0.0.1:0",
		EnableDHT:  false,
	}
	d := &Daemon{
		config:         config,
		state:          NewDaemonState(),
		stats:          NewDaemonStatistics(),
		packageManager: pm,
	}

	d.state.mu.Lock()
	d.state.ActivePackages = 1
	d.state.mu.Unlock()

	// Create JSON body
	body := map[string]string{"package_id": pkg.PackageID}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/packages/remove", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	d.handlePackageRemove(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

// TestHandlePackageRemove_MissingPackageID tests that missing package_id returns 400
func TestHandlePackageRemove_MissingPackageID(t *testing.T) {
	tests := []struct {
		name   string
		method string
		url    string
		body   io.Reader
	}{
		{
			name:   "DELETE without query parameter",
			method: http.MethodDelete,
			url:    "/packages/remove",
			body:   nil,
		},
		{
			name:   "POST with empty JSON",
			method: http.MethodPost,
			url:    "/packages/remove",
			body:   bytes.NewReader([]byte(`{}`)),
		},
		{
			name:   "POST with missing package_id field",
			method: http.MethodPost,
			url:    "/packages/remove",
			body:   bytes.NewReader([]byte(`{"other_field": "value"}`)),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			packagesDir := filepath.Join(tempDir, "packages")
			os.MkdirAll(packagesDir, 0755)

			pm := NewPackageManager(packagesDir, filepath.Join(tempDir, "packages.yaml"))

			config := &DaemonConfig{
				StorageDir: tempDir,
				ListenAddr: "127.0.0.1:0",
				EnableDHT:  false,
			}
			d := &Daemon{
				config:         config,
				state:          NewDaemonState(),
				stats:          NewDaemonStatistics(),
				packageManager: pm,
			}

			req := httptest.NewRequest(tt.method, tt.url, tt.body)
			if tt.body != nil {
				req.Header.Set("Content-Type", "application/json")
			}
			w := httptest.NewRecorder()

			d.handlePackageRemove(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
			}
			if !strings.Contains(w.Body.String(), "package_id is required") {
				t.Errorf("expected error about missing package_id, got: %s", w.Body.String())
			}
		})
	}
}

// TestHandlePackageRemove_PackageNotFound tests that non-existent package returns 404
func TestHandlePackageRemove_PackageNotFound(t *testing.T) {
	tempDir := t.TempDir()
	packagesDir := filepath.Join(tempDir, "packages")
	os.MkdirAll(packagesDir, 0755)

	pm := NewPackageManager(packagesDir, filepath.Join(tempDir, "packages.yaml"))

	config := &DaemonConfig{
		StorageDir: tempDir,
		ListenAddr: "127.0.0.1:0",
		EnableDHT:  false,
	}
	d := &Daemon{
		config:         config,
		state:          NewDaemonState(),
		stats:          NewDaemonStatistics(),
		packageManager: pm,
	}

	req := httptest.NewRequest(http.MethodDelete, "/packages/remove?package_id=nonexistent", nil)
	w := httptest.NewRecorder()

	d.handlePackageRemove(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
	if !strings.Contains(w.Body.String(), "Package not found") {
		t.Errorf("expected 'Package not found' error, got: %s", w.Body.String())
	}
}

// TestHandlePackageRemove_DHTRemoval tests removal with DHT enabled
func TestHandlePackageRemove_DHTRemoval(t *testing.T) {
	tempDir := t.TempDir()
	packagesDir := filepath.Join(tempDir, "packages")
	os.MkdirAll(packagesDir, 0755)

	pm := NewPackageManager(packagesDir, filepath.Join(tempDir, "packages.yaml"))

	// Create a proper test package
	pkgBytes, pkg := createTestPackageFile(t)

	// Write the package to a file
	testFilePath := filepath.Join(packagesDir, "test.lspkg")
	if err := os.WriteFile(testFilePath, pkgBytes, 0644); err != nil {
		t.Fatalf("failed to write test package: %v", err)
	}

	packageID := pkg.PackageID

	// Compute file hash
	fileHash := sha256.Sum256(pkgBytes)
	fileHashHex := hex.EncodeToString(fileHash[:])

	// Compute fingerprints
	creatorFingerprint := pkg.Manifest.CreatorPubKey.Fingerprint()
	maintainerFingerprint := pkg.Manifest.MaintainerPubKey.Fingerprint()

	// Get signatures as hex strings
	manifestSigHex := hex.EncodeToString(pkg.ManifestSignature.SignedData)
	maintainerSigHex := hex.EncodeToString(pkg.MaintainerManifestSignature.SignedData)

	err := pm.AddPackage(&PackageInfo{
		PackageID:                   packageID,
		Name:                        pkg.Manifest.PackageName,
		Version:                     pkg.Manifest.Version,
		Description:                 pkg.Manifest.Description,
		FilePath:                    testFilePath,
		FileHash:                    fileHashHex,
		FileSize:                    int64(len(pkgBytes)),
		CreatedAt:                   time.Now(),
		CreatorFingerprint:          creatorFingerprint,
		ManifestSignature:           manifestSigHex,
		MaintainerFingerprint:       maintainerFingerprint,
		MaintainerManifestSignature: maintainerSigHex,
	})
	if err != nil {
		t.Fatalf("failed to add package: %v", err)
	}

	config := &DaemonConfig{
		StorageDir: tempDir,
		ListenAddr: "127.0.0.1:0",
		EnableDHT:  false, // TODO: Refactor to use interfaces for proper DHT testing
	}
	d := &Daemon{
		config:         config,
		state:          NewDaemonState(),
		stats:          NewDaemonStatistics(),
		packageManager: pm,
		announcer:      nil,
		dhtClient:      nil,
	}

	d.state.mu.Lock()
	d.state.ActivePackages = 1
	d.state.mu.Unlock()

	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/packages/remove?package_id=%s", packageID), nil)
	w := httptest.NewRecorder()

	d.handlePackageRemove(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	// Note: DHT testing disabled - TODO: refactor to use interfaces for proper DHT mocking
}

// TestHandlePackageRemove_InvalidJSON tests that malformed JSON returns 400
func TestHandlePackageRemove_InvalidJSON(t *testing.T) {
	tempDir := t.TempDir()
	packagesDir := filepath.Join(tempDir, "packages")
	os.MkdirAll(packagesDir, 0755)

	pm := NewPackageManager(packagesDir, filepath.Join(tempDir, "packages.yaml"))

	config := &DaemonConfig{
		StorageDir: tempDir,
		ListenAddr: "127.0.0.1:0",
		EnableDHT:  false,
	}
	d := &Daemon{
		config:         config,
		state:          NewDaemonState(),
		stats:          NewDaemonStatistics(),
		packageManager: pm,
	}

	req := httptest.NewRequest(http.MethodPost, "/packages/remove", bytes.NewReader([]byte(`{INVALID_JSON`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	d.handlePackageRemove(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
	if !strings.Contains(w.Body.String(), "Failed to parse request body") {
		t.Errorf("expected error about parsing JSON, got: %s", w.Body.String())
	}
}
