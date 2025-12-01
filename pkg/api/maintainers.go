package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/libreseed/libreseed/pkg/daemon"
)

// MaintainerHandlers manages HTTP handlers for maintainer operations
type MaintainerHandlers struct {
	registry       *daemon.MaintainerRegistry
	packageManager *daemon.PackageManager
}

// NewMaintainerHandlers creates a new MaintainerHandlers instance
func NewMaintainerHandlers(registry *daemon.MaintainerRegistry, pm *daemon.PackageManager) *MaintainerHandlers {
	return &MaintainerHandlers{
		registry:       registry,
		packageManager: pm,
	}
}

// MaintainerResponse represents a maintainer in API responses
type MaintainerResponse struct {
	Fingerprint    string    `json:"fingerprint"`
	Name           string    `json:"name"`
	PublicKey      string    `json:"public_key"`
	Email          string    `json:"email,omitempty"`
	RegisteredAt   time.Time `json:"registered_at"`
	Active         bool      `json:"active"`
	PackagesSigned int       `json:"packages_signed"`
	LastSignedAt   time.Time `json:"last_signed_at,omitempty"`
}

// RegisterMaintainerRequest represents the request to register a new maintainer
type RegisterMaintainerRequest struct {
	Fingerprint string `json:"fingerprint"`
	Name        string `json:"name"`
	PublicKey   string `json:"public_key"`
	Email       string `json:"email,omitempty"`
}

// PendingSignatureResponse represents a pending signature in API responses
type PendingSignatureResponse struct {
	PackageID          string    `json:"package_id"`
	PackageName        string    `json:"package_name"`
	PackageVersion     string    `json:"package_version"`
	CreatorFingerprint string    `json:"creator_fingerprint"`
	ManifestHash       string    `json:"manifest_hash"`
	CreatedAt          time.Time `json:"created_at"`
	ExpiresAt          time.Time `json:"expires_at"`
}

// SubmitSignatureRequest represents the request to submit a maintainer signature
type SubmitSignatureRequest struct {
	MaintainerFingerprint string `json:"maintainer_fingerprint"`
	Signature             string `json:"signature"`
}

// HandleList handles GET /api/v1/maintainers
func (h *MaintainerHandlers) HandleList() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			WriteError(w, r, BadRequest("method not allowed"))
			return
		}

		// Get query parameters for filtering
		activeOnly := r.URL.Query().Get("active") == "true"

		// Get maintainers
		var maintainers []daemon.MaintainerInfo
		if activeOnly {
			maintainers = h.registry.ListActive()
		} else {
			maintainers = h.registry.List()
		}

		// Get pagination parameters
		params := ParsePagination(r)

		// Calculate pagination
		totalItems := len(maintainers)
		startIdx := (params.Page - 1) * params.PerPage
		endIdx := startIdx + params.PerPage

		if startIdx >= totalItems {
			startIdx = totalItems
		}
		if endIdx > totalItems {
			endIdx = totalItems
		}

		// Get page slice
		var pageMaintainers []daemon.MaintainerInfo
		if startIdx < totalItems {
			pageMaintainers = maintainers[startIdx:endIdx]
		} else {
			pageMaintainers = []daemon.MaintainerInfo{}
		}

		// Convert to response format
		response := make([]MaintainerResponse, len(pageMaintainers))
		for i, m := range pageMaintainers {
			response[i] = convertMaintainerToResponse(&m)
		}

		// Calculate metadata
		meta := CalculateMeta(params.Page, params.PerPage, totalItems)

		WriteSuccessWithMeta(w, response, meta)
	}
}

// HandleGet handles GET /api/v1/maintainers/{fingerprint}
func (h *MaintainerHandlers) HandleGet() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			WriteError(w, r, BadRequest("method not allowed"))
			return
		}

		// Extract fingerprint from URL path
		fingerprint := extractMaintainerFingerprint(r.URL.Path)
		if fingerprint == "" {
			WriteError(w, r, BadRequest("missing maintainer fingerprint"))
			return
		}

		// Get maintainer
		maintainer, err := h.registry.Get(fingerprint)
		if err != nil {
			if err == daemon.ErrMaintainerNotFound {
				WriteError(w, r, NotFound("maintainer"))
				return
			}
			WriteError(w, r, InternalServerError(err.Error()))
			return
		}

		response := convertMaintainerToResponse(maintainer)
		WriteSuccess(w, response)
	}
}

// HandleRegister handles POST /api/v1/maintainers
func (h *MaintainerHandlers) HandleRegister() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			WriteError(w, r, BadRequest("method not allowed"))
			return
		}

		// Parse request body
		var req RegisterMaintainerRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			WriteError(w, r, BadRequest("invalid JSON request body"))
			return
		}

		// Validate required fields
		if req.Fingerprint == "" {
			WriteError(w, r, BadRequest("fingerprint is required"))
			return
		}
		if req.Name == "" {
			WriteError(w, r, BadRequest("name is required"))
			return
		}
		if req.PublicKey == "" {
			WriteError(w, r, BadRequest("public_key is required"))
			return
		}

		// Register maintainer
		maintainer, err := h.registry.Register(req.Fingerprint, req.Name, req.PublicKey, req.Email)
		if err != nil {
			switch err {
			case daemon.ErrMaintainerAlreadyExists:
				WriteError(w, r, Conflict("maintainer with this fingerprint already exists"))
			case daemon.ErrInvalidFingerprint:
				WriteError(w, r, BadRequest("fingerprint must be 16 hexadecimal characters"))
			case daemon.ErrInvalidPublicKey:
				WriteError(w, r, BadRequest("public_key must be 64 hexadecimal characters"))
			default:
				WriteError(w, r, InternalServerError(err.Error()))
			}
			return
		}

		response := convertMaintainerToResponse(maintainer)
		WriteCreated(w, response)
	}
}

// HandleActivate handles POST /api/v1/maintainers/{fingerprint}/activate
func (h *MaintainerHandlers) HandleActivate() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			WriteError(w, r, BadRequest("method not allowed"))
			return
		}

		fingerprint := extractMaintainerFingerprintFromAction(r.URL.Path, "/activate")
		if fingerprint == "" {
			WriteError(w, r, BadRequest("missing maintainer fingerprint"))
			return
		}

		if err := h.registry.Activate(fingerprint); err != nil {
			if err == daemon.ErrMaintainerNotFound {
				WriteError(w, r, NotFound("maintainer"))
				return
			}
			WriteError(w, r, InternalServerError(err.Error()))
			return
		}

		// Get updated maintainer
		maintainer, _ := h.registry.Get(fingerprint)
		response := convertMaintainerToResponse(maintainer)
		WriteSuccess(w, response)
	}
}

// HandleDeactivate handles POST /api/v1/maintainers/{fingerprint}/deactivate
func (h *MaintainerHandlers) HandleDeactivate() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			WriteError(w, r, BadRequest("method not allowed"))
			return
		}

		fingerprint := extractMaintainerFingerprintFromAction(r.URL.Path, "/deactivate")
		if fingerprint == "" {
			WriteError(w, r, BadRequest("missing maintainer fingerprint"))
			return
		}

		if err := h.registry.Deactivate(fingerprint); err != nil {
			if err == daemon.ErrMaintainerNotFound {
				WriteError(w, r, NotFound("maintainer"))
				return
			}
			WriteError(w, r, InternalServerError(err.Error()))
			return
		}

		// Get updated maintainer
		maintainer, _ := h.registry.Get(fingerprint)
		response := convertMaintainerToResponse(maintainer)
		WriteSuccess(w, response)
	}
}

// HandleListPending handles GET /api/v1/maintainers/pending
func (h *MaintainerHandlers) HandleListPending() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			WriteError(w, r, BadRequest("method not allowed"))
			return
		}

		// Get all pending signatures (already filters expired)
		pending := h.registry.ListPending()

		// Get pagination parameters
		params := ParsePagination(r)

		// Calculate pagination
		totalItems := len(pending)
		startIdx := (params.Page - 1) * params.PerPage
		endIdx := startIdx + params.PerPage

		if startIdx >= totalItems {
			startIdx = totalItems
		}
		if endIdx > totalItems {
			endIdx = totalItems
		}

		// Get page slice
		var pagePending []daemon.PendingSignature
		if startIdx < totalItems {
			pagePending = pending[startIdx:endIdx]
		} else {
			pagePending = []daemon.PendingSignature{}
		}

		// Convert to response format
		response := make([]PendingSignatureResponse, len(pagePending))
		for i, p := range pagePending {
			response[i] = convertPendingToResponse(&p)
		}

		// Calculate metadata
		meta := CalculateMeta(params.Page, params.PerPage, totalItems)

		WriteSuccessWithMeta(w, response, meta)
	}
}

// HandleSubmitSignature handles POST /api/v1/packages/{id}/sign
func (h *MaintainerHandlers) HandleSubmitSignature() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			WriteError(w, r, BadRequest("method not allowed"))
			return
		}

		// Extract package ID from URL path
		packageID := extractPackageIDFromSign(r.URL.Path)
		if packageID == "" {
			WriteError(w, r, BadRequest("missing package ID"))
			return
		}

		// Parse request body
		var req SubmitSignatureRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			WriteError(w, r, BadRequest("invalid JSON request body"))
			return
		}

		// Validate required fields
		if req.MaintainerFingerprint == "" {
			WriteError(w, r, BadRequest("maintainer_fingerprint is required"))
			return
		}
		if req.Signature == "" {
			WriteError(w, r, BadRequest("signature is required"))
			return
		}

		// Check if package exists
		_, exists := h.packageManager.GetPackage(packageID)
		if !exists {
			WriteError(w, r, NotFound("package"))
			return
		}

		// Check if maintainer exists and is active
		maintainer, err := h.registry.Get(req.MaintainerFingerprint)
		if err != nil {
			if err == daemon.ErrMaintainerNotFound {
				WriteError(w, r, NotFound("maintainer"))
				return
			}
			WriteError(w, r, InternalServerError(err.Error()))
			return
		}

		if !maintainer.Active {
			WriteError(w, r, BadRequest("maintainer is inactive"))
			return
		}

		// Check if there's a pending signature for this package
		pending, err := h.registry.GetPending(packageID)
		if err != nil {
			if err == daemon.ErrPendingNotFound {
				WriteError(w, r, NotFound("no pending signature request for this package"))
				return
			}
			if err == daemon.ErrPendingExpired {
				WriteError(w, r, BadRequest("pending signature request has expired"))
				return
			}
			WriteError(w, r, InternalServerError(err.Error()))
			return
		}

		// TODO: Verify the signature against the manifest hash using maintainer's public key
		// This will be implemented as part of T030 (cryptographic verification)
		// For now, we just record the signature

		// Update package with maintainer signature
		if err := h.packageManager.UpdateMaintainerSignature(packageID, req.MaintainerFingerprint, req.Signature); err != nil {
			WriteError(w, r, InternalServerError(err.Error()))
			return
		}

		// Increment maintainer's sign count
		if err := h.registry.IncrementSignCount(req.MaintainerFingerprint); err != nil {
			// Log but don't fail - the signature was recorded
			_ = err
		}

		// Remove from pending list
		if err := h.registry.RemovePending(packageID); err != nil {
			// Log but don't fail - the signature was recorded
			_ = err
		}

		// Return success response
		response := struct {
			Message          string    `json:"message"`
			PackageID        string    `json:"package_id"`
			MaintainerSigned bool      `json:"maintainer_signed"`
			SignedBy         string    `json:"signed_by"`
			SignedAt         time.Time `json:"signed_at"`
			ManifestHash     string    `json:"manifest_hash"`
		}{
			Message:          "package co-signed successfully",
			PackageID:        packageID,
			MaintainerSigned: true,
			SignedBy:         req.MaintainerFingerprint,
			SignedAt:         time.Now(),
			ManifestHash:     pending.ManifestHash,
		}

		WriteSuccess(w, response)
	}
}

// Helper functions

// convertMaintainerToResponse converts daemon.MaintainerInfo to MaintainerResponse
func convertMaintainerToResponse(m *daemon.MaintainerInfo) MaintainerResponse {
	return MaintainerResponse{
		Fingerprint:    m.Fingerprint,
		Name:           m.Name,
		PublicKey:      m.PublicKey,
		Email:          m.Email,
		RegisteredAt:   m.RegisteredAt,
		Active:         m.Active,
		PackagesSigned: m.PackagesSigned,
		LastSignedAt:   m.LastSignedAt,
	}
}

// convertPendingToResponse converts daemon.PendingSignature to PendingSignatureResponse
func convertPendingToResponse(p *daemon.PendingSignature) PendingSignatureResponse {
	return PendingSignatureResponse{
		PackageID:          p.PackageID,
		PackageName:        p.PackageName,
		PackageVersion:     p.PackageVersion,
		CreatorFingerprint: p.CreatorFingerprint,
		ManifestHash:       p.ManifestHash,
		CreatedAt:          p.CreatedAt,
		ExpiresAt:          p.ExpiresAt,
	}
}

// extractMaintainerFingerprint extracts fingerprint from /api/v1/maintainers/{fingerprint}
func extractMaintainerFingerprint(path string) string {
	prefix := "/api/v1/maintainers/"
	if !strings.HasPrefix(path, prefix) {
		return ""
	}

	fingerprint := strings.TrimPrefix(path, prefix)
	fingerprint = strings.TrimSuffix(fingerprint, "/")

	// Check for sub-paths
	if strings.Contains(fingerprint, "/") {
		return ""
	}

	// Don't return "pending" as a fingerprint
	if fingerprint == "pending" {
		return ""
	}

	return fingerprint
}

// extractMaintainerFingerprintFromAction extracts fingerprint from /api/v1/maintainers/{fingerprint}/{action}
func extractMaintainerFingerprintFromAction(path, action string) string {
	prefix := "/api/v1/maintainers/"
	if !strings.HasPrefix(path, prefix) || !strings.HasSuffix(path, action) {
		return ""
	}

	fingerprint := strings.TrimPrefix(path, prefix)
	fingerprint = strings.TrimSuffix(fingerprint, action)
	fingerprint = strings.TrimSuffix(fingerprint, "/")

	return fingerprint
}

// extractPackageIDFromSign extracts package ID from /api/v1/packages/{id}/sign
func extractPackageIDFromSign(path string) string {
	prefix := "/api/v1/packages/"
	suffix := "/sign"

	if !strings.HasPrefix(path, prefix) || !strings.HasSuffix(path, suffix) {
		return ""
	}

	packageID := strings.TrimPrefix(path, prefix)
	packageID = strings.TrimSuffix(packageID, suffix)

	return packageID
}

// RegisterMaintainerRoutes registers maintainer management routes on the router
func (router *Router) RegisterMaintainerRoutes(handlers *MaintainerHandlers) {
	// List maintainers (GET /api/v1/maintainers)
	router.Handle("/api/v1/maintainers", handlers.HandleList())

	// Register maintainer (POST /api/v1/maintainers)
	router.Handle("/api/v1/maintainers", handlers.HandleRegister())

	// List pending signatures (GET /api/v1/maintainers/pending)
	router.Handle("/api/v1/maintainers/pending", handlers.HandleListPending())

	// Routes with fingerprint parameter
	router.Handle("/api/v1/maintainers/", func(w http.ResponseWriter, req *http.Request) {
		path := req.URL.Path

		// Route to list or register handlers for base path
		if path == "/api/v1/maintainers/" || path == "/api/v1/maintainers" {
			if req.Method == http.MethodGet {
				handlers.HandleList()(w, req)
			} else if req.Method == http.MethodPost {
				handlers.HandleRegister()(w, req)
			} else {
				WriteError(w, req, BadRequest("method not allowed"))
			}
			return
		}

		// List pending signatures
		if path == "/api/v1/maintainers/pending" || path == "/api/v1/maintainers/pending/" {
			handlers.HandleListPending()(w, req)
			return
		}

		// Activate maintainer
		if strings.HasSuffix(path, "/activate") {
			handlers.HandleActivate()(w, req)
			return
		}

		// Deactivate maintainer
		if strings.HasSuffix(path, "/deactivate") {
			handlers.HandleDeactivate()(w, req)
			return
		}

		// Get maintainer by fingerprint
		if req.Method == http.MethodGet {
			handlers.HandleGet()(w, req)
			return
		}

		WriteError(w, req, BadRequest("method not allowed"))
	})

	// Sign package (POST /api/v1/packages/{id}/sign)
	// Note: This is registered as part of package routes in router.go
}
