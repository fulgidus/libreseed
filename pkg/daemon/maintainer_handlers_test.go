package daemon

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/libreseed/libreseed/pkg/crypto"
)

// createTestDaemonWithMaintainerRegistry creates a Daemon with PackageManager and MaintainerRegistry for testing
func createTestDaemonWithMaintainerRegistry(t *testing.T) (*Daemon, string) {
	t.Helper()

	tempDir := t.TempDir()
	packagesDir := filepath.Join(tempDir, "packages")
	if err := os.MkdirAll(packagesDir, 0755); err != nil {
		t.Fatalf("failed to create packages directory: %v", err)
	}

	pm := NewPackageManager(packagesDir, filepath.Join(tempDir, "packages.yaml"))

	maintainersFile := filepath.Join(tempDir, "maintainers.yaml")
	mr, err := NewMaintainerRegistry(maintainersFile)
	if err != nil {
		t.Fatalf("failed to create maintainer registry: %v", err)
	}

	config := &DaemonConfig{
		StorageDir: tempDir,
		ListenAddr: "127.0.0.1:0",
		EnableDHT:  false,
	}

	d := &Daemon{
		config:             config,
		state:              NewDaemonState(),
		stats:              NewDaemonStatistics(),
		packageManager:     pm,
		maintainerRegistry: mr,
	}

	return d, tempDir
}

// addTestPackage adds a package directly to the PackageManager for testing, bypassing validation.
// This should only be used in tests to set up test state.
func addTestPackage(pm *PackageManager, pkg *PackageInfo) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.packages[pkg.PackageID] = pkg
}

// createTestMaintainerKeys creates a key pair and returns hex-encoded public key and fingerprint
func createTestMaintainerKeys(t *testing.T) (string, string, *crypto.KeyManager) {
	t.Helper()

	tempDir := t.TempDir()
	keysDir := filepath.Join(tempDir, "keys")
	if err := os.MkdirAll(keysDir, 0755); err != nil {
		t.Fatalf("failed to create keys directory: %v", err)
	}

	km, err := crypto.NewKeyManager(keysDir)
	if err != nil {
		t.Fatalf("failed to create key manager: %v", err)
	}
	if err := km.EnsureKeysExist(); err != nil {
		t.Fatalf("failed to ensure keys: %v", err)
	}

	pubKeyHex := hex.EncodeToString(km.PublicKey())
	pubKey, err := crypto.NewPublicKey(km.PublicKey())
	if err != nil {
		t.Fatalf("failed to create public key: %v", err)
	}
	fingerprint := pubKey.Fingerprint()

	return pubKeyHex, fingerprint, km
}

// ==================== handleMaintainerList Tests ====================

func TestHandleMaintainerList_Success_Empty(t *testing.T) {
	d, _ := createTestDaemonWithMaintainerRegistry(t)

	req := httptest.NewRequest(http.MethodGet, "/maintainers", nil)
	w := httptest.NewRecorder()

	d.handleMaintainerList(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if response["status"] != "success" {
		t.Errorf("expected status 'success', got %v", response["status"])
	}

	count, ok := response["count"].(float64)
	if !ok || int(count) != 0 {
		t.Errorf("expected count 0, got %v", response["count"])
	}
}

func TestHandleMaintainerList_Success_WithMaintainers(t *testing.T) {
	d, _ := createTestDaemonWithMaintainerRegistry(t)

	// Register a maintainer first
	pubKeyHex, fingerprint, _ := createTestMaintainerKeys(t)
	_, err := d.maintainerRegistry.Register(fingerprint, "Test Maintainer", pubKeyHex, "test@example.com")
	if err != nil {
		t.Fatalf("failed to register maintainer: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/maintainers", nil)
	w := httptest.NewRecorder()

	d.handleMaintainerList(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if response["status"] != "success" {
		t.Errorf("expected status 'success', got %v", response["status"])
	}

	count, ok := response["count"].(float64)
	if !ok || int(count) != 1 {
		t.Errorf("expected count 1, got %v", response["count"])
	}

	maintainers, ok := response["maintainers"].([]interface{})
	if !ok || len(maintainers) != 1 {
		t.Errorf("expected 1 maintainer, got %v", len(maintainers))
	}
}

func TestHandleMaintainerList_InvalidMethod(t *testing.T) {
	d, _ := createTestDaemonWithMaintainerRegistry(t)

	methods := []string{http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/maintainers", nil)
			w := httptest.NewRecorder()

			d.handleMaintainerList(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("expected status %d, got %d", http.StatusMethodNotAllowed, w.Code)
			}
		})
	}
}

// ==================== handleMaintainerGet Tests ====================

func TestHandleMaintainerGet_Success(t *testing.T) {
	d, _ := createTestDaemonWithMaintainerRegistry(t)

	// Register a maintainer first
	pubKeyHex, fingerprint, _ := createTestMaintainerKeys(t)
	_, err := d.maintainerRegistry.Register(fingerprint, "Test Maintainer", pubKeyHex, "test@example.com")
	if err != nil {
		t.Fatalf("failed to register maintainer: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/maintainers/"+fingerprint, nil)
	w := httptest.NewRecorder()

	d.handleMaintainerGet(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if response["status"] != "success" {
		t.Errorf("expected status 'success', got %v", response["status"])
	}

	maintainer, ok := response["maintainer"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected maintainer object in response")
	}

	if maintainer["fingerprint"] != fingerprint {
		t.Errorf("expected fingerprint %s, got %v", fingerprint, maintainer["fingerprint"])
	}

	if maintainer["name"] != "Test Maintainer" {
		t.Errorf("expected name 'Test Maintainer', got %v", maintainer["name"])
	}
}

func TestHandleMaintainerGet_NotFound(t *testing.T) {
	d, _ := createTestDaemonWithMaintainerRegistry(t)

	req := httptest.NewRequest(http.MethodGet, "/maintainers/nonexistent1234", nil)
	w := httptest.NewRecorder()

	d.handleMaintainerGet(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestHandleMaintainerGet_EmptyFingerprint(t *testing.T) {
	d, _ := createTestDaemonWithMaintainerRegistry(t)

	req := httptest.NewRequest(http.MethodGet, "/maintainers/", nil)
	w := httptest.NewRecorder()

	d.handleMaintainerGet(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestHandleMaintainerGet_InvalidMethod(t *testing.T) {
	d, _ := createTestDaemonWithMaintainerRegistry(t)

	req := httptest.NewRequest(http.MethodPost, "/maintainers/somefingerprint", nil)
	w := httptest.NewRecorder()

	d.handleMaintainerGet(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status %d, got %d", http.StatusMethodNotAllowed, w.Code)
	}
}

// ==================== handleMaintainerRegister Tests ====================

func TestHandleMaintainerRegister_Success(t *testing.T) {
	d, _ := createTestDaemonWithMaintainerRegistry(t)

	pubKeyHex, _, _ := createTestMaintainerKeys(t)

	reqBody := map[string]string{
		"public_key": pubKeyHex,
		"name":       "New Maintainer",
		"email":      "new@example.com",
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/maintainers", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	d.handleMaintainerRegister(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d: %s", http.StatusCreated, w.Code, w.Body.String())
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if response["status"] != "success" {
		t.Errorf("expected status 'success', got %v", response["status"])
	}

	maintainer, ok := response["maintainer"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected maintainer object in response")
	}

	if maintainer["name"] != "New Maintainer" {
		t.Errorf("expected name 'New Maintainer', got %v", maintainer["name"])
	}
}

func TestHandleMaintainerRegister_MissingPublicKey(t *testing.T) {
	d, _ := createTestDaemonWithMaintainerRegistry(t)

	reqBody := map[string]string{
		"name":  "New Maintainer",
		"email": "new@example.com",
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/maintainers", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	d.handleMaintainerRegister(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestHandleMaintainerRegister_InvalidPublicKey(t *testing.T) {
	d, _ := createTestDaemonWithMaintainerRegistry(t)

	reqBody := map[string]string{
		"public_key": "INVALID_HEX_KEY",
		"name":       "New Maintainer",
		"email":      "new@example.com",
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/maintainers", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	d.handleMaintainerRegister(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d: %s", http.StatusBadRequest, w.Code, w.Body.String())
	}
}

func TestHandleMaintainerRegister_DuplicateMaintainer(t *testing.T) {
	d, _ := createTestDaemonWithMaintainerRegistry(t)

	pubKeyHex, fingerprint, _ := createTestMaintainerKeys(t)

	// Register the first time
	_, err := d.maintainerRegistry.Register(fingerprint, "First Maintainer", pubKeyHex, "first@example.com")
	if err != nil {
		t.Fatalf("failed to register first maintainer: %v", err)
	}

	// Try to register again
	reqBody := map[string]string{
		"public_key": pubKeyHex,
		"name":       "Duplicate Maintainer",
		"email":      "duplicate@example.com",
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/maintainers", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	d.handleMaintainerRegister(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("expected status %d, got %d", http.StatusConflict, w.Code)
	}
}

func TestHandleMaintainerRegister_InvalidMethod(t *testing.T) {
	d, _ := createTestDaemonWithMaintainerRegistry(t)

	req := httptest.NewRequest(http.MethodGet, "/maintainers", nil)
	w := httptest.NewRecorder()

	d.handleMaintainerRegister(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status %d, got %d", http.StatusMethodNotAllowed, w.Code)
	}
}

func TestHandleMaintainerRegister_InvalidJSON(t *testing.T) {
	d, _ := createTestDaemonWithMaintainerRegistry(t)

	req := httptest.NewRequest(http.MethodPost, "/maintainers", bytes.NewReader([]byte("NOT_JSON")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	d.handleMaintainerRegister(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

// ==================== handleMaintainerActivate Tests ====================

func TestHandleMaintainerActivate_Success(t *testing.T) {
	d, _ := createTestDaemonWithMaintainerRegistry(t)

	pubKeyHex, fingerprint, _ := createTestMaintainerKeys(t)
	_, err := d.maintainerRegistry.Register(fingerprint, "Test Maintainer", pubKeyHex, "test@example.com")
	if err != nil {
		t.Fatalf("failed to register maintainer: %v", err)
	}

	// Deactivate first
	if err := d.maintainerRegistry.Deactivate(fingerprint); err != nil {
		t.Fatalf("failed to deactivate maintainer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/maintainers/activate/"+fingerprint, nil)
	w := httptest.NewRecorder()

	d.handleMaintainerActivate(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if response["status"] != "success" {
		t.Errorf("expected status 'success', got %v", response["status"])
	}

	if response["fingerprint"] != fingerprint {
		t.Errorf("expected fingerprint %s, got %v", fingerprint, response["fingerprint"])
	}

	// Verify maintainer is active
	m, err := d.maintainerRegistry.Get(fingerprint)
	if err != nil {
		t.Fatalf("failed to get maintainer: %v", err)
	}
	if !m.Active {
		t.Error("expected maintainer to be active")
	}
}

func TestHandleMaintainerActivate_NotFound(t *testing.T) {
	d, _ := createTestDaemonWithMaintainerRegistry(t)

	req := httptest.NewRequest(http.MethodPost, "/maintainers/activate/nonexistent", nil)
	w := httptest.NewRecorder()

	d.handleMaintainerActivate(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestHandleMaintainerActivate_EmptyFingerprint(t *testing.T) {
	d, _ := createTestDaemonWithMaintainerRegistry(t)

	req := httptest.NewRequest(http.MethodPost, "/maintainers/activate/", nil)
	w := httptest.NewRecorder()

	d.handleMaintainerActivate(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestHandleMaintainerActivate_InvalidMethod(t *testing.T) {
	d, _ := createTestDaemonWithMaintainerRegistry(t)

	req := httptest.NewRequest(http.MethodGet, "/maintainers/activate/somefingerprint", nil)
	w := httptest.NewRecorder()

	d.handleMaintainerActivate(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status %d, got %d", http.StatusMethodNotAllowed, w.Code)
	}
}

// ==================== handleMaintainerDeactivate Tests ====================

func TestHandleMaintainerDeactivate_Success(t *testing.T) {
	d, _ := createTestDaemonWithMaintainerRegistry(t)

	pubKeyHex, fingerprint, _ := createTestMaintainerKeys(t)
	_, err := d.maintainerRegistry.Register(fingerprint, "Test Maintainer", pubKeyHex, "test@example.com")
	if err != nil {
		t.Fatalf("failed to register maintainer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/maintainers/deactivate/"+fingerprint, nil)
	w := httptest.NewRecorder()

	d.handleMaintainerDeactivate(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if response["status"] != "success" {
		t.Errorf("expected status 'success', got %v", response["status"])
	}

	// Verify maintainer is inactive
	m, err := d.maintainerRegistry.Get(fingerprint)
	if err != nil {
		t.Fatalf("failed to get maintainer: %v", err)
	}
	if m.Active {
		t.Error("expected maintainer to be inactive")
	}
}

func TestHandleMaintainerDeactivate_NotFound(t *testing.T) {
	d, _ := createTestDaemonWithMaintainerRegistry(t)

	req := httptest.NewRequest(http.MethodPost, "/maintainers/deactivate/nonexistent", nil)
	w := httptest.NewRecorder()

	d.handleMaintainerDeactivate(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestHandleMaintainerDeactivate_EmptyFingerprint(t *testing.T) {
	d, _ := createTestDaemonWithMaintainerRegistry(t)

	req := httptest.NewRequest(http.MethodPost, "/maintainers/deactivate/", nil)
	w := httptest.NewRecorder()

	d.handleMaintainerDeactivate(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestHandleMaintainerDeactivate_InvalidMethod(t *testing.T) {
	d, _ := createTestDaemonWithMaintainerRegistry(t)

	req := httptest.NewRequest(http.MethodGet, "/maintainers/deactivate/somefingerprint", nil)
	w := httptest.NewRecorder()

	d.handleMaintainerDeactivate(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status %d, got %d", http.StatusMethodNotAllowed, w.Code)
	}
}

// ==================== handlePendingSignatures Tests ====================

func TestHandlePendingSignatures_Success_Empty(t *testing.T) {
	d, _ := createTestDaemonWithMaintainerRegistry(t)

	req := httptest.NewRequest(http.MethodGet, "/signatures/pending", nil)
	w := httptest.NewRecorder()

	d.handlePendingSignatures(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if response["status"] != "success" {
		t.Errorf("expected status 'success', got %v", response["status"])
	}

	count, ok := response["count"].(float64)
	if !ok || int(count) != 0 {
		t.Errorf("expected count 0, got %v", response["count"])
	}
}

func TestHandlePendingSignatures_Success_WithPendingPackages(t *testing.T) {
	d, tempDir := createTestDaemonWithMaintainerRegistry(t)

	// Add a package without maintainer signature
	pkg := &PackageInfo{
		PackageID:                   "abc123def456",
		Name:                        "test-pending",
		Version:                     "1.0.0",
		CreatorFingerprint:          "creator123",
		MaintainerManifestSignature: "", // No maintainer signature
		CreatedAt:                   time.Now(),
		FilePath:                    filepath.Join(tempDir, "packages", "test.lspkg"),
	}
	addTestPackage(d.packageManager, pkg)

	req := httptest.NewRequest(http.MethodGet, "/signatures/pending", nil)
	w := httptest.NewRecorder()

	d.handlePendingSignatures(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if response["status"] != "success" {
		t.Errorf("expected status 'success', got %v", response["status"])
	}

	count, ok := response["count"].(float64)
	if !ok || int(count) != 1 {
		t.Errorf("expected count 1, got %v", response["count"])
	}

	pendingPkgs, ok := response["pending_packages"].([]interface{})
	if !ok || len(pendingPkgs) != 1 {
		t.Errorf("expected 1 pending package, got %v", len(pendingPkgs))
	}
}

func TestHandlePendingSignatures_InvalidMethod(t *testing.T) {
	d, _ := createTestDaemonWithMaintainerRegistry(t)

	methods := []string{http.MethodPost, http.MethodPut, http.MethodDelete}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/signatures/pending", nil)
			w := httptest.NewRecorder()

			d.handlePendingSignatures(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("expected status %d, got %d", http.StatusMethodNotAllowed, w.Code)
			}
		})
	}
}

// ==================== handlePackageSign Tests ====================

func TestHandlePackageSign_InvalidMethod(t *testing.T) {
	d, _ := createTestDaemonWithMaintainerRegistry(t)

	methods := []string{http.MethodGet, http.MethodPut, http.MethodDelete}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/packages/sign/someid", nil)
			w := httptest.NewRecorder()

			d.handlePackageSign(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("expected status %d, got %d", http.StatusMethodNotAllowed, w.Code)
			}
		})
	}
}

func TestHandlePackageSign_MissingPackageID(t *testing.T) {
	d, _ := createTestDaemonWithMaintainerRegistry(t)

	req := httptest.NewRequest(http.MethodPost, "/packages/sign/", bytes.NewReader([]byte(`{}`)))
	w := httptest.NewRecorder()

	d.handlePackageSign(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestHandlePackageSign_InvalidJSON(t *testing.T) {
	d, _ := createTestDaemonWithMaintainerRegistry(t)

	req := httptest.NewRequest(http.MethodPost, "/packages/sign/someid", bytes.NewReader([]byte("NOT_JSON")))
	w := httptest.NewRecorder()

	d.handlePackageSign(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestHandlePackageSign_MissingRequiredFields(t *testing.T) {
	d, _ := createTestDaemonWithMaintainerRegistry(t)

	testCases := []struct {
		name string
		body map[string]string
	}{
		{
			name: "missing_both",
			body: map[string]string{},
		},
		{
			name: "missing_fingerprint",
			body: map[string]string{"signature": "abc123"},
		},
		{
			name: "missing_signature",
			body: map[string]string{"maintainer_fingerprint": "abc123"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			bodyBytes, _ := json.Marshal(tc.body)
			req := httptest.NewRequest(http.MethodPost, "/packages/sign/someid", bytes.NewReader(bodyBytes))
			w := httptest.NewRecorder()

			d.handlePackageSign(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("expected status %d, got %d: %s", http.StatusBadRequest, w.Code, w.Body.String())
			}
		})
	}
}

func TestHandlePackageSign_MaintainerNotFound(t *testing.T) {
	d, _ := createTestDaemonWithMaintainerRegistry(t)

	reqBody := map[string]string{
		"maintainer_fingerprint": "nonexistent",
		"signature":              "abc123",
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/packages/sign/someid", bytes.NewReader(bodyBytes))
	w := httptest.NewRecorder()

	d.handlePackageSign(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestHandlePackageSign_MaintainerInactive(t *testing.T) {
	d, _ := createTestDaemonWithMaintainerRegistry(t)

	// Register and deactivate maintainer
	pubKeyHex, fingerprint, _ := createTestMaintainerKeys(t)
	_, err := d.maintainerRegistry.Register(fingerprint, "Test Maintainer", pubKeyHex, "test@example.com")
	if err != nil {
		t.Fatalf("failed to register maintainer: %v", err)
	}
	if err := d.maintainerRegistry.Deactivate(fingerprint); err != nil {
		t.Fatalf("failed to deactivate maintainer: %v", err)
	}

	reqBody := map[string]string{
		"maintainer_fingerprint": fingerprint,
		"signature":              "abc123",
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/packages/sign/someid", bytes.NewReader(bodyBytes))
	w := httptest.NewRecorder()

	d.handlePackageSign(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected status %d, got %d: %s", http.StatusForbidden, w.Code, w.Body.String())
	}
}

func TestHandlePackageSign_PackageNotFound(t *testing.T) {
	d, _ := createTestDaemonWithMaintainerRegistry(t)

	// Register active maintainer
	pubKeyHex, fingerprint, _ := createTestMaintainerKeys(t)
	_, err := d.maintainerRegistry.Register(fingerprint, "Test Maintainer", pubKeyHex, "test@example.com")
	if err != nil {
		t.Fatalf("failed to register maintainer: %v", err)
	}

	reqBody := map[string]string{
		"maintainer_fingerprint": fingerprint,
		"signature":              "abc123",
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/packages/sign/nonexistent_package", bytes.NewReader(bodyBytes))
	w := httptest.NewRecorder()

	d.handlePackageSign(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d: %s", http.StatusNotFound, w.Code, w.Body.String())
	}
}
