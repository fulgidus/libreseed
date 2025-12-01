package api

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/libreseed/libreseed/pkg/daemon"
	"github.com/libreseed/libreseed/pkg/storage"
)

// PackageHandlers manages HTTP handlers for package operations
type PackageHandlers struct {
	packageManager *daemon.PackageManager
}

// NewPackageHandlers creates a new PackageHandlers instance
func NewPackageHandlers(pm *daemon.PackageManager) *PackageHandlers {
	return &PackageHandlers{
		packageManager: pm,
	}
}

// PackageResponse represents a package in API responses
type PackageResponse struct {
	PackageID             string    `json:"package_id"`
	Name                  string    `json:"name"`
	Version               string    `json:"version"`
	Description           string    `json:"description"`
	FileHash              string    `json:"file_hash"`
	FileSize              int64     `json:"file_size"`
	CreatedAt             time.Time `json:"created_at"`
	CreatorFingerprint    string    `json:"creator_fingerprint"`
	MaintainerFingerprint string    `json:"maintainer_fingerprint,omitempty"`
	AnnouncedToDHT        bool      `json:"announced_to_dht"`
	LastAnnounced         time.Time `json:"last_announced,omitempty"`
}

// PackageListResponse represents the list packages response
type PackageListResponse struct {
	Packages []PackageResponse `json:"packages"`
}

// AddPackageRequest represents the add package request
type AddPackageRequest struct {
	Name                        string `json:"name"`
	Version                     string `json:"version"`
	Description                 string `json:"description"`
	CreatorFingerprint          string `json:"creator_fingerprint"`
	ManifestSignature           string `json:"manifest_signature"`
	MaintainerFingerprint       string `json:"maintainer_fingerprint,omitempty"`
	MaintainerManifestSignature string `json:"maintainer_manifest_signature,omitempty"`
}

// HandleList handles GET /api/v1/packages
func (h *PackageHandlers) HandleList() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			WriteError(w, r, BadRequest("method not allowed"))
			return
		}

		// Get pagination parameters
		params := ParsePagination(r)

		// Get all packages
		allPackages := h.packageManager.ListPackages()

		// Calculate pagination
		totalItems := len(allPackages)
		startIdx := (params.Page - 1) * params.PerPage
		endIdx := startIdx + params.PerPage

		if startIdx >= totalItems {
			startIdx = totalItems
		}
		if endIdx > totalItems {
			endIdx = totalItems
		}

		// Get page slice
		var pagePackages []*daemon.PackageInfo
		if startIdx < totalItems {
			pagePackages = allPackages[startIdx:endIdx]
		} else {
			pagePackages = []*daemon.PackageInfo{}
		}

		// Convert to response format
		response := make([]PackageResponse, len(pagePackages))
		for i, pkg := range pagePackages {
			response[i] = convertPackageToResponse(pkg)
		}

		// Calculate metadata
		meta := CalculateMeta(params.Page, params.PerPage, totalItems)

		// Write response with direct array (not wrapped in PackageListResponse)
		WriteSuccessWithMeta(w, response, meta)
	}
}

// HandleGet handles GET /api/v1/packages/{id}
func (h *PackageHandlers) HandleGet() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			WriteError(w, r, BadRequest("method not allowed"))
			return
		}

		// Extract package ID from URL path
		packageID := extractPackageID(r.URL.Path)
		if packageID == "" {
			WriteError(w, r, BadRequest("missing package ID"))
			return
		}

		// Get package
		pkg, exists := h.packageManager.GetPackage(packageID)
		if !exists {
			WriteError(w, r, NotFound("package"))
			return
		}

		// Convert to response format
		response := convertPackageToResponse(pkg)

		WriteSuccess(w, response)
	}
}

// HandleAdd handles POST /api/v1/packages (multipart upload)
func (h *PackageHandlers) HandleAdd() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			WriteError(w, r, BadRequest("method not allowed"))
			return
		}

		// Parse multipart form (max 500 MB)
		if err := r.ParseMultipartForm(500 << 20); err != nil {
			WriteError(w, r, BadRequest(fmt.Sprintf("failed to parse multipart form: %v", err)))
			return
		}

		// Get package file
		file, fileHeader, err := r.FormFile("package")
		if err != nil {
			WriteError(w, r, BadRequest("missing 'package' file in multipart form"))
			return
		}
		defer file.Close()

		// Get metadata JSON
		metadataJSON := r.FormValue("metadata")
		if metadataJSON == "" {
			WriteError(w, r, BadRequest("missing 'metadata' field in multipart form"))
			return
		}

		// Parse metadata
		var req AddPackageRequest
		if err := json.Unmarshal([]byte(metadataJSON), &req); err != nil {
			WriteError(w, r, BadRequest(fmt.Sprintf("invalid metadata JSON: %v", err)))
			return
		}

		// Validate required fields
		if req.Name == "" || req.Version == "" || req.Description == "" {
			WriteError(w, r, BadRequest("name, version, and description are required"))
			return
		}

		if req.CreatorFingerprint == "" || req.ManifestSignature == "" {
			WriteError(w, r, BadRequest("creator_fingerprint and manifest_signature are required"))
			return
		}

		// Save package file to storage
		packagePath := filepath.Join(h.packageManager.GetStorageDir(), fileHeader.Filename)
		packageID, fileHash, fileSize, err := h.savePackageFile(file, packagePath)
		if err != nil {
			WriteError(w, r, InternalServerError(fmt.Sprintf("failed to save package file: %v", err)))
			return
		}

		// Create package info
		packageInfo := &daemon.PackageInfo{
			PackageID:                   packageID,
			Name:                        req.Name,
			Version:                     req.Version,
			Description:                 req.Description,
			FilePath:                    packagePath,
			FileHash:                    fileHash,
			FileSize:                    fileSize,
			CreatedAt:                   time.Now(),
			CreatorFingerprint:          req.CreatorFingerprint,
			ManifestSignature:           req.ManifestSignature,
			MaintainerFingerprint:       req.MaintainerFingerprint,
			MaintainerManifestSignature: req.MaintainerManifestSignature,
			AnnouncedToDHT:              false,
		}

		// Add package to manager
		if err := h.packageManager.AddPackage(packageInfo); err != nil {
			// Clean up file if add fails
			os.Remove(packagePath)
			WriteError(w, r, InternalServerError(fmt.Sprintf("failed to add package: %v", err)))
			return
		}

		// Return created package
		response := convertPackageToResponse(packageInfo)
		WriteCreated(w, response)
	}
}

// HandleDelete handles DELETE /api/v1/packages/{id}
func (h *PackageHandlers) HandleDelete() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			WriteError(w, r, BadRequest("method not allowed"))
			return
		}

		// Extract package ID from URL path
		packageID := extractPackageID(r.URL.Path)
		if packageID == "" {
			WriteError(w, r, BadRequest("missing package ID"))
			return
		}

		// Check if package exists
		if !h.packageManager.PackageExists(packageID) {
			WriteError(w, r, NotFound("package"))
			return
		}

		// Remove package
		if err := h.packageManager.RemovePackage(packageID); err != nil {
			WriteError(w, r, InternalServerError(fmt.Sprintf("failed to remove package: %v", err)))
			return
		}

		WriteNoContent(w)
	}
}

// HandleRestart handles POST /api/v1/packages/{id}/restart
func (h *PackageHandlers) HandleRestart() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			WriteError(w, r, BadRequest("method not allowed"))
			return
		}

		// Extract package ID from URL path
		packageID := extractPackageIDFromRestart(r.URL.Path)
		if packageID == "" {
			WriteError(w, r, BadRequest("missing package ID"))
			return
		}

		// Check if package exists
		pkg, exists := h.packageManager.GetPackage(packageID)
		if !exists {
			WriteError(w, r, NotFound("package"))
			return
		}

		// TODO: Implement actual restart logic (T031 - Seeder Integration)
		// For now, just mark as announced to DHT
		if err := h.packageManager.UpdateAnnouncementStatus(packageID, true); err != nil {
			WriteError(w, r, InternalServerError(fmt.Sprintf("failed to restart seeding: %v", err)))
			return
		}

		response := convertPackageToResponse(pkg)
		WriteSuccess(w, response)
	}
}

// Helper functions

// convertPackageToResponse converts daemon.PackageInfo to PackageResponse
func convertPackageToResponse(pkg *daemon.PackageInfo) PackageResponse {
	return PackageResponse{
		PackageID:             pkg.PackageID,
		Name:                  pkg.Name,
		Version:               pkg.Version,
		Description:           pkg.Description,
		FileHash:              pkg.FileHash,
		FileSize:              pkg.FileSize,
		CreatedAt:             pkg.CreatedAt,
		CreatorFingerprint:    pkg.CreatorFingerprint,
		MaintainerFingerprint: pkg.MaintainerFingerprint,
		AnnouncedToDHT:        pkg.AnnouncedToDHT,
		LastAnnounced:         pkg.LastAnnounced,
	}
}

// extractPackageID extracts package ID from /api/v1/packages/{id}
func extractPackageID(path string) string {
	// Remove /api/v1/packages/ prefix
	prefix := "/api/v1/packages/"
	if !strings.HasPrefix(path, prefix) {
		return ""
	}

	packageID := strings.TrimPrefix(path, prefix)
	packageID = strings.TrimSuffix(packageID, "/")

	// Validate it's not a sub-path
	if strings.Contains(packageID, "/") {
		return ""
	}

	return packageID
}

// extractPackageIDFromRestart extracts package ID from /api/v1/packages/{id}/restart
func extractPackageIDFromRestart(path string) string {
	// Remove /api/v1/packages/ prefix and /restart suffix
	prefix := "/api/v1/packages/"
	suffix := "/restart"

	if !strings.HasPrefix(path, prefix) || !strings.HasSuffix(path, suffix) {
		return ""
	}

	packageID := strings.TrimPrefix(path, prefix)
	packageID = strings.TrimSuffix(packageID, suffix)

	return packageID
}

// savePackageFile saves uploaded file and calculates hash
func (h *PackageHandlers) savePackageFile(file multipart.File, destPath string) (packageID, fileHash string, fileSize int64, err error) {
	// Create destination file
	destFile, err := os.Create(destPath)
	if err != nil {
		return "", "", 0, fmt.Errorf("failed to create file: %w", err)
	}
	defer destFile.Close()

	// Copy file content
	written, err := io.Copy(destFile, file)
	if err != nil {
		os.Remove(destPath)
		return "", "", 0, fmt.Errorf("failed to copy file: %w", err)
	}

	fileSize = written

	// Calculate SHA-256 hash
	hash, err := storage.ComputeFileHash(destPath)
	if err != nil {
		os.Remove(destPath)
		return "", "", 0, fmt.Errorf("failed to calculate hash: %w", err)
	}

	// Convert hash to hex string
	hashStr := hex.EncodeToString(hash)

	// Package ID is the same as file hash
	packageID = hashStr
	fileHash = hashStr

	return packageID, fileHash, fileSize, nil
}

// RegisterPackageRoutes registers package management routes on the router
func (r *Router) RegisterPackageRoutes(handlers *PackageHandlers) {
	// List packages (GET /api/v1/packages)
	r.Handle("/api/v1/packages", handlers.HandleList())

	// Add package (POST /api/v1/packages)
	r.Handle("/api/v1/packages", handlers.HandleAdd())

	// Get package details (GET /api/v1/packages/{id})
	// Note: This uses a pattern match, actual routing done in handler
	r.Handle("/api/v1/packages/", func(w http.ResponseWriter, req *http.Request) {
		path := req.URL.Path

		// Route based on path pattern
		if path == "/api/v1/packages/" || path == "/api/v1/packages" {
			// List packages
			handlers.HandleList()(w, req)
			return
		}

		if strings.HasSuffix(path, "/restart") {
			// Restart seeding
			handlers.HandleRestart()(w, req)
			return
		}

		// Otherwise, it's a get/delete operation
		switch req.Method {
		case http.MethodGet:
			handlers.HandleGet()(w, req)
		case http.MethodDelete:
			handlers.HandleDelete()(w, req)
		default:
			WriteError(w, req, BadRequest("method not allowed"))
		}
	})
}
