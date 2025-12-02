package api

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/libreseed/libreseed/pkg/daemon"
)

// setupTestPackageManager creates a test package manager with temp storage
func setupTestPackageManager(t *testing.T) (*daemon.PackageManager, string) {
	t.Helper()

	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "libreseed-pkg-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Create storage directory
	storageDir := filepath.Join(tmpDir, "storage")
	if err := os.MkdirAll(storageDir, 0755); err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to create storage dir: %v", err)
	}

	// Create metadata file path
	metaFile := filepath.Join(tmpDir, "packages.yaml")

	// Create package manager
	pm := daemon.NewPackageManager(storageDir, metaFile)

	return pm, tmpDir
}

// createTestPackage creates a test package file
func createTestPackage(t *testing.T) string {
	t.Helper()

	tmpFile, err := os.CreateTemp("", "test-package-*.tar.gz")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer tmpFile.Close()

	// Write some test data
	testData := []byte("This is a test package file content")
	if _, err := tmpFile.Write(testData); err != nil {
		os.Remove(tmpFile.Name())
		t.Fatalf("Failed to write test data: %v", err)
	}

	return tmpFile.Name()
}

// createMultipartRequest creates a multipart/form-data request with a file
func createMultipartRequest(t *testing.T, filePath, fieldName string) (*http.Request, string) {
	t.Helper()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Open test file
	file, err := os.Open(filePath)
	if err != nil {
		t.Fatalf("Failed to open test file: %v", err)
	}
	defer file.Close()

	// Create form file
	part, err := writer.CreateFormFile(fieldName, filepath.Base(filePath))
	if err != nil {
		t.Fatalf("Failed to create form file: %v", err)
	}

	// Copy file content
	if _, err := io.Copy(part, file); err != nil {
		t.Fatalf("Failed to copy file content: %v", err)
	}

	// Add metadata JSON field
	metadata := map[string]string{
		"name":                          "test-package",
		"version":                       "1.0.0",
		"description":                   "Test package description",
		"creator_fingerprint":           "dddddddddddddddd",
		"manifest_signature":            "dddddddddddddddddddddddddddddddd",
		"maintainer_fingerprint":        "eeeeeeeeeeeeeeee",
		"maintainer_manifest_signature": "eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
	}
	metadataJSON, _ := json.Marshal(metadata)
	writer.WriteField("metadata", string(metadataJSON))

	writer.Close()

	// Create request
	req := httptest.NewRequest(http.MethodPost, "/api/v1/packages", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	return req, writer.Boundary()
}

func TestNewPackageHandlers(t *testing.T) {
	pm, tmpDir := setupTestPackageManager(t)
	defer os.RemoveAll(tmpDir)

	handlers := NewPackageHandlers(pm)

	if handlers == nil {
		t.Fatal("Expected handlers to be non-nil")
	}

	if handlers.packageManager != pm {
		t.Error("Expected packageManager to be set correctly")
	}
}

func TestHandleList(t *testing.T) {
	pm, tmpDir := setupTestPackageManager(t)
	defer os.RemoveAll(tmpDir)

	handlers := NewPackageHandlers(pm)

	// Add test packages
	pkg1 := &daemon.PackageInfo{
		PackageID:                   "a1b2c3d4e5f6789012345678901234567890123456789012345678901234abcd",
		Name:                        "test-package-1",
		Version:                     "1.0.0",
		Description:                 "Test package 1",
		FilePath:                    filepath.Join(tmpDir, "pkg1.tar.gz"),
		FileHash:                    "1111111111111111111111111111111111111111111111111111111111111111",
		FileSize:                    1024,
		CreatedAt:                   time.Now(),
		CreatorFingerprint:          "1111111111111111",
		ManifestSignature:           "11111111111111111111111111111111",
		MaintainerFingerprint:       "2222222222222222",
		MaintainerManifestSignature: "22222222222222222222222222222222",
		AnnouncedToDHT:              true,
	}
	pkg2 := &daemon.PackageInfo{
		PackageID:                   "b2c3d4e5f6a7890123456789012345678901234567890123456789012345bcde",
		Name:                        "test-package-2",
		Version:                     "2.0.0",
		Description:                 "Test package 2",
		FilePath:                    filepath.Join(tmpDir, "pkg2.tar.gz"),
		FileHash:                    "2222222222222222222222222222222222222222222222222222222222222222",
		FileSize:                    2048,
		CreatedAt:                   time.Now(),
		CreatorFingerprint:          "3333333333333333",
		ManifestSignature:           "33333333333333333333333333333333",
		MaintainerFingerprint:       "4444444444444444",
		MaintainerManifestSignature: "44444444444444444444444444444444",
		AnnouncedToDHT:              false,
	}

	if err := pm.AddPackage(pkg1); err != nil {
		t.Fatalf("Failed to add test package 1: %v", err)
	}
	if err := pm.AddPackage(pkg2); err != nil {
		t.Fatalf("Failed to add test package 2: %v", err)
	}

	tests := []struct {
		name           string
		url            string
		expectedStatus int
		expectedCount  int
	}{
		{
			name:           "List all packages",
			url:            "/api/v1/packages",
			expectedStatus: http.StatusOK,
			expectedCount:  2,
		},
		{
			name:           "List with pagination",
			url:            "/api/v1/packages?page=1&per_page=1",
			expectedStatus: http.StatusOK,
			expectedCount:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			w := httptest.NewRecorder()

			handler := handlers.HandleList()
			handler.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.expectedStatus == http.StatusOK {
				var response map[string]interface{}
				if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}

				data, ok := response["data"].([]interface{})
				if !ok {
					t.Fatal("Expected 'data' field in response")
				}

				if len(data) != tt.expectedCount {
					t.Errorf("Expected %d packages, got %d", tt.expectedCount, len(data))
				}
			}
		})
	}
}

func TestHandleGet(t *testing.T) {
	pm, tmpDir := setupTestPackageManager(t)
	defer os.RemoveAll(tmpDir)

	handlers := NewPackageHandlers(pm)

	// Add test package
	pkg := &daemon.PackageInfo{
		PackageID:                   "c3d4e5f6a7b8901234567890123456789012345678901234567890123456cdef",
		Name:                        "test-package",
		Version:                     "1.0.0",
		Description:                 "Test package",
		FilePath:                    filepath.Join(tmpDir, "pkg.tar.gz"),
		FileHash:                    "3333333333333333333333333333333333333333333333333333333333333333",
		FileSize:                    1024,
		CreatedAt:                   time.Now(),
		CreatorFingerprint:          "5555555555555555",
		ManifestSignature:           "55555555555555555555555555555555",
		MaintainerFingerprint:       "6666666666666666",
		MaintainerManifestSignature: "66666666666666666666666666666666",
		AnnouncedToDHT:              true,
	}

	if err := pm.AddPackage(pkg); err != nil {
		t.Fatalf("Failed to add test package: %v", err)
	}

	tests := []struct {
		name           string
		packageID      string
		expectedStatus int
	}{
		{
			name:           "Get existing package",
			packageID:      "c3d4e5f6a7b8901234567890123456789012345678901234567890123456cdef",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Get non-existent package",
			packageID:      "nonexistent",
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/packages/"+tt.packageID, nil)
			w := httptest.NewRecorder()

			handler := handlers.HandleGet()
			handler.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.expectedStatus == http.StatusOK {
				var response map[string]interface{}
				if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}

				data, ok := response["data"].(map[string]interface{})
				if !ok {
					t.Fatal("Expected 'data' field in response")
				}

				if data["package_id"] != tt.packageID {
					t.Errorf("Expected package_id %s, got %v", tt.packageID, data["package_id"])
				}
			}
		})
	}
}

func TestHandleAdd(t *testing.T) {
	pm, tmpDir := setupTestPackageManager(t)
	defer os.RemoveAll(tmpDir)

	handlers := NewPackageHandlers(pm)

	// Create test package file
	testFile := createTestPackage(t)
	defer os.Remove(testFile)

	tests := []struct {
		name           string
		createRequest  func() *http.Request
		expectedStatus int
	}{
		{
			name: "Add valid package",
			createRequest: func() *http.Request {
				req, _ := createMultipartRequest(t, testFile, "package")
				return req
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "Add without file",
			createRequest: func() *http.Request {
				return httptest.NewRequest(http.MethodPost, "/api/v1/packages", nil)
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := tt.createRequest()
			w := httptest.NewRecorder()

			handler := handlers.HandleAdd()
			handler.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
				t.Logf("Response body: %s", w.Body.String())
			}
		})
	}
}

func TestHandleDelete(t *testing.T) {
	pm, tmpDir := setupTestPackageManager(t)
	defer os.RemoveAll(tmpDir)

	handlers := NewPackageHandlers(pm)

	// Add test package
	pkg := &daemon.PackageInfo{
		PackageID:                   "d4e5f6a7b8c9012345678901234567890123456789012345678901234567def0",
		Name:                        "test-package",
		Version:                     "1.0.0",
		Description:                 "Test package for deletion",
		FilePath:                    filepath.Join(tmpDir, "pkg-delete.tar.gz"),
		FileHash:                    "4444444444444444444444444444444444444444444444444444444444444444",
		FileSize:                    1024,
		CreatedAt:                   time.Now(),
		CreatorFingerprint:          "7777777777777777",
		ManifestSignature:           "77777777777777777777777777777777",
		MaintainerFingerprint:       "8888888888888888",
		MaintainerManifestSignature: "88888888888888888888888888888888",
		AnnouncedToDHT:              false,
	}

	if err := pm.AddPackage(pkg); err != nil {
		t.Fatalf("Failed to add test package: %v", err)
	}

	tests := []struct {
		name           string
		packageID      string
		expectedStatus int
	}{
		{
			name:           "Delete existing package",
			packageID:      "d4e5f6a7b8c9012345678901234567890123456789012345678901234567def0",
			expectedStatus: http.StatusNoContent,
		},
		{
			name:           "Delete non-existent package",
			packageID:      "nonexistent",
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodDelete, "/api/v1/packages/"+tt.packageID, nil)
			w := httptest.NewRecorder()

			handler := handlers.HandleDelete()
			handler.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestHandleRestart(t *testing.T) {
	pm, tmpDir := setupTestPackageManager(t)
	defer os.RemoveAll(tmpDir)

	handlers := NewPackageHandlers(pm)

	// Add test package
	pkg := &daemon.PackageInfo{
		PackageID:                   "e5f6a7b8c9d0123456789012345678901234567890123456789012345678ef01",
		Name:                        "test-package",
		Version:                     "1.0.0",
		Description:                 "Test package for restart",
		FilePath:                    filepath.Join(tmpDir, "pkg-restart.tar.gz"),
		FileHash:                    "5555555555555555555555555555555555555555555555555555555555555555",
		FileSize:                    1024,
		CreatedAt:                   time.Now(),
		CreatorFingerprint:          "9999999999999999",
		ManifestSignature:           "99999999999999999999999999999999",
		MaintainerFingerprint:       "aaaaaaaaaaaaaaaa",
		MaintainerManifestSignature: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		AnnouncedToDHT:              false,
	}

	if err := pm.AddPackage(pkg); err != nil {
		t.Fatalf("Failed to add test package: %v", err)
	}

	tests := []struct {
		name           string
		packageID      string
		expectedStatus int
	}{
		{
			name:           "Restart existing package",
			packageID:      "e5f6a7b8c9d0123456789012345678901234567890123456789012345678ef01",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Restart non-existent package",
			packageID:      "nonexistent",
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/packages/"+tt.packageID+"/restart", nil)
			w := httptest.NewRecorder()

			handler := handlers.HandleRestart()
			handler.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestExtractPackageID(t *testing.T) {
	tests := []struct {
		name      string
		path      string
		expected  string
		shouldErr bool
	}{
		{
			name:     "Valid package ID",
			path:     "/api/v1/packages/test-pkg-123",
			expected: "test-pkg-123",
		},
		{
			name:     "Invalid prefix",
			path:     "/invalid/packages/test-pkg-123",
			expected: "",
		},
		{
			name:     "Empty path",
			path:     "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractPackageID(tt.path)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestExtractPackageIDFromRestart(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{
			name:     "Valid restart path",
			path:     "/api/v1/packages/test-pkg-123/restart",
			expected: "test-pkg-123",
		},
		{
			name:     "Invalid prefix",
			path:     "/invalid/packages/test-pkg-123/restart",
			expected: "",
		},
		{
			name:     "Missing restart suffix",
			path:     "/api/v1/packages/test-pkg-123",
			expected: "",
		},
		{
			name:     "Empty path",
			path:     "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractPackageIDFromRestart(tt.path)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestConvertPackageToResponse(t *testing.T) {
	now := time.Now()
	pkg := &daemon.PackageInfo{
		PackageID:                   "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		Name:                        "test-package",
		Version:                     "1.0.0",
		Description:                 "test description",
		FileHash:                    "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
		FileSize:                    1024,
		FilePath:                    "/test/path",
		CreatorFingerprint:          "bbbbbbbbbbbbbbbb",
		ManifestSignature:           "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		MaintainerFingerprint:       "cccccccccccccccc",
		MaintainerManifestSignature: "cccccccccccccccccccccccccccccccc",
		CreatedAt:                   now,
		AnnouncedToDHT:              true,
	}

	response := convertPackageToResponse(pkg)

	if response.PackageID != pkg.PackageID {
		t.Errorf("Expected PackageID %s, got %s", pkg.PackageID, response.PackageID)
	}

	if response.Name != pkg.Name {
		t.Errorf("Expected Name %s, got %s", pkg.Name, response.Name)
	}

	if response.Version != pkg.Version {
		t.Errorf("Expected Version %s, got %s", pkg.Version, response.Version)
	}

	if response.FileHash != pkg.FileHash {
		t.Errorf("Expected FileHash %s, got %s", pkg.FileHash, response.FileHash)
	}

	if response.FileSize != pkg.FileSize {
		t.Errorf("Expected FileSize %d, got %d", pkg.FileSize, response.FileSize)
	}

	if response.AnnouncedToDHT != pkg.AnnouncedToDHT {
		t.Errorf("Expected AnnouncedToDHT %v, got %v", pkg.AnnouncedToDHT, response.AnnouncedToDHT)
	}

	if response.CreatedAt.Unix() != pkg.CreatedAt.Unix() {
		t.Errorf("Expected CreatedAt %v, got %v", pkg.CreatedAt, response.CreatedAt)
	}
}
