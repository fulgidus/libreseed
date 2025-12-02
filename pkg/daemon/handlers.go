package daemon

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/anacrolix/torrent/metainfo"
	"github.com/libreseed/libreseed/pkg/crypto"
	packagetypes "github.com/libreseed/libreseed/pkg/package"
)

// handlePackageAdd handles package addition requests.
// POST /packages/add
// Multipart form data:
// - file: the .lspkg package file (YAML with dual signatures)
//
// The package file must contain:
// - Manifest with creator and maintainer public keys
// - ManifestSignature (creator's signature)
// - MaintainerManifestSignature (maintainer's signature)
//
// Both signatures are verified before accepting the package.
func (d *Daemon) handlePackageAdd(w http.ResponseWriter, r *http.Request) {
	log.Println("=== handlePackageAdd CALLED ===")

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse multipart form (limit to 500MB)
	if err := r.ParseMultipartForm(500 << 20); err != nil {
		http.Error(w, fmt.Sprintf("Failed to parse form: %v", err), http.StatusBadRequest)
		return
	}

	// Extract .lspkg file
	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get file: %v", err), http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Read entire file into memory for parsing
	fileData, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to read file: %v", err), http.StatusInternalServerError)
		return
	}

	// Parse .lspkg file structure
	pkg, err := packagetypes.LoadPackageFromBytes(fileData)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to parse .lspkg file: %v", err), http.StatusBadRequest)
		return
	}

	// Serialize manifest for signature verification
	manifestData, err := packagetypes.SerializeManifest(&pkg.Manifest)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to serialize manifest: %v", err), http.StatusInternalServerError)
		return
	}

	// Verify dual signatures
	err = crypto.VerifyDualSignature(
		manifestData,
		pkg.Manifest.CreatorPubKey,
		&pkg.ManifestSignature,
		pkg.Manifest.MaintainerPubKey,
		&pkg.MaintainerManifestSignature,
	)
	if err != nil {
		http.Error(w, fmt.Sprintf("Signature verification failed: %v", err), http.StatusUnauthorized)
		return
	}

	log.Printf("✓ Dual signature verification passed for package %s v%s\n", pkg.Manifest.PackageName, pkg.Manifest.Version)

	// Compute creator and maintainer fingerprints
	creatorFingerprint := pkg.Manifest.CreatorPubKey.Fingerprint()
	maintainerFingerprint := pkg.Manifest.MaintainerPubKey.Fingerprint()

	// Create PackageInfo from parsed package
	packageInfo := &PackageInfo{
		PackageID:                   pkg.PackageID,
		Name:                        pkg.Manifest.PackageName,
		Version:                     pkg.Manifest.Version,
		Description:                 pkg.Manifest.Description,
		FilePath:                    "", // Will be set after file copy
		FileHash:                    pkg.Manifest.ContentHash,
		FileSize:                    pkg.SizeBytes,
		CreatedAt:                   time.Now(),
		CreatorFingerprint:          creatorFingerprint,
		ManifestSignature:           hex.EncodeToString(pkg.ManifestSignature.SignedData),
		MaintainerFingerprint:       maintainerFingerprint,
		MaintainerManifestSignature: hex.EncodeToString(pkg.MaintainerManifestSignature.SignedData),
		AnnouncedToDHT:              false,
	}

	// Save .lspkg file to packages directory
	destPath := filepath.Join(d.packageManager.GetStorageDir(), header.Filename)
	if err := os.WriteFile(destPath, fileData, 0644); err != nil {
		http.Error(w, fmt.Sprintf("Failed to save package file: %v", err), http.StatusInternalServerError)
		return
	}

	// Update FilePath in packageInfo
	packageInfo.FilePath = destPath

	// Save metadata via packageManager
	if err := d.packageManager.AddPackage(packageInfo); err != nil {
		os.Remove(destPath) // Clean up on failure
		http.Error(w, fmt.Sprintf("Failed to save metadata: %v", err), http.StatusInternalServerError)
		return
	}

	// Announce to DHT if enabled
	log.Printf("DHT check - EnableDHT=%v, dhtClient=%v, announcer=%v\n", d.config.EnableDHT, d.dhtClient != nil, d.announcer != nil)
	if d.config.EnableDHT && d.dhtClient != nil && d.announcer != nil {
		log.Printf("Attempting DHT announcement for package %s (ID: %s)\n", packageInfo.Name, packageInfo.PackageID)
		// Convert package ID (SHA-256 hex) to DHT InfoHash (first 20 bytes)
		infoHashBytes, err := hex.DecodeString(packageInfo.PackageID[:40])
		if err == nil && len(infoHashBytes) >= 20 {
			var infoHash metainfo.Hash
			copy(infoHash[:], infoHashBytes[:20])

			// Add package to DHT announcer with dual signature fingerprints
			d.announcer.AddPackage(infoHash, packageInfo.Name, creatorFingerprint, maintainerFingerprint)
			log.Printf("Called d.announcer.AddPackage for %s with InfoHash %x (Creator: %s, Maintainer: %s)\n",
				packageInfo.Name, infoHash, creatorFingerprint, maintainerFingerprint)

			// Update announcement status in package manager
			if err := d.packageManager.UpdateAnnouncementStatus(packageInfo.PackageID, true); err != nil {
				log.Printf("Warning: Failed to update announcement status: %v\n", err)
			} else {
				log.Printf("Successfully updated announcement status for package %s\n", packageInfo.PackageID)
			}

			log.Printf("Package %s announced to DHT with InfoHash %x\n", packageInfo.Name, pkg.Manifest.ContentHash)
		} else {
			log.Printf("Warning: Failed to convert package ID to InfoHash: %v\n", err)
		}
	} else {
		log.Printf("DHT announcement skipped - one or more conditions not met\n")
	}

	// Update daemon state
	d.state.mu.Lock()
	d.state.ActivePackages++
	d.state.mu.Unlock()

	d.stats.mu.Lock()
	d.stats.TotalPackagesSeeded++
	d.stats.mu.Unlock()

	// Return success response with both fingerprints
	response := map[string]interface{}{
		"status":                 "success",
		"package_id":             packageInfo.PackageID,
		"creator_fingerprint":    creatorFingerprint,
		"maintainer_fingerprint": maintainerFingerprint,
		"file_hash":              pkg.Manifest.ContentHash,
		"filename":               header.Filename,
		"verified":               true,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// handlePackageList handles package listing requests.
// GET /packages/list
func (d *Daemon) handlePackageList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	packages := d.packageManager.ListPackages()

	response := map[string]interface{}{
		"status":   "success",
		"count":    len(packages),
		"packages": packages,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handlePackageRemove handles package removal requests.
// DELETE /packages/remove?package_id=<id>
// or POST /packages/remove with JSON body: {"package_id": "<id>"}
func (d *Daemon) handlePackageRemove(w http.ResponseWriter, r *http.Request) {
	var packageID string

	switch r.Method {
	case http.MethodDelete:
		// Extract package_id from query parameters
		packageID = r.URL.Query().Get("package_id")
	case http.MethodPost:
		// Extract package_id from JSON body
		var req struct {
			PackageID string `json:"package_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, fmt.Sprintf("Failed to parse request body: %v", err), http.StatusBadRequest)
			return
		}
		packageID = req.PackageID
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if packageID == "" {
		http.Error(w, "package_id is required", http.StatusBadRequest)
		return
	}

	// Get package info before removal (to delete file)
	packageInfo, exists := d.packageManager.GetPackage(packageID)
	if !exists {
		http.Error(w, "Package not found", http.StatusNotFound)
		return
	}

	// Remove from DHT if enabled
	if d.config.EnableDHT && d.dhtClient != nil && d.announcer != nil {
		// Convert package ID to DHT InfoHash
		infoHashBytes, err := hex.DecodeString(packageID[:40])
		if err == nil && len(infoHashBytes) >= 20 {
			var infoHash metainfo.Hash
			copy(infoHash[:], infoHashBytes[:20])

			// Remove package from DHT announcer
			d.announcer.RemovePackage(infoHash)
			fmt.Printf("Package %s removed from DHT announcements (InfoHash %x)\n", packageInfo.Name, infoHash)
		} else {
			fmt.Printf("Warning: Failed to convert package ID to InfoHash for DHT removal: %v\n", err)
		}
	}

	// Remove from package manager (this also deletes the file)
	if err := d.packageManager.RemovePackage(packageID); err != nil {
		http.Error(w, fmt.Sprintf("Failed to remove package: %v", err), http.StatusInternalServerError)
		return
	}

	// Update daemon state
	d.state.mu.Lock()
	if d.state.ActivePackages > 0 {
		d.state.ActivePackages--
	}
	d.state.mu.Unlock()

	// Return success response
	response := map[string]interface{}{
		"status":     "success",
		"package_id": packageID,
		"message":    "Package removed successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// =============================================================================
// Maintainer Management Handlers
// =============================================================================

// handleMaintainerList handles listing all registered maintainers.
// GET /maintainers
func (d *Daemon) handleMaintainerList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	maintainers := d.maintainerRegistry.List()

	// Convert to response format
	var result []map[string]interface{}
	for _, m := range maintainers {
		result = append(result, map[string]interface{}{
			"fingerprint":     m.Fingerprint,
			"name":            m.Name,
			"email":           m.Email,
			"active":          m.Active,
			"registered_at":   m.RegisteredAt,
			"packages_signed": m.PackagesSigned,
			"last_signed_at":  m.LastSignedAt,
		})
	}

	response := map[string]interface{}{
		"status":      "success",
		"count":       len(result),
		"maintainers": result,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleMaintainerGet handles getting a specific maintainer by fingerprint.
// GET /maintainers/{fingerprint}
func (d *Daemon) handleMaintainerGet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract fingerprint from URL path: /maintainers/{fingerprint}
	fingerprint := r.URL.Path[len("/maintainers/"):]
	if fingerprint == "" {
		http.Error(w, "Fingerprint is required", http.StatusBadRequest)
		return
	}

	maintainer, err := d.maintainerRegistry.Get(fingerprint)
	if err != nil {
		http.Error(w, fmt.Sprintf("Maintainer not found: %v", err), http.StatusNotFound)
		return
	}

	response := map[string]interface{}{
		"status": "success",
		"maintainer": map[string]interface{}{
			"fingerprint":     maintainer.Fingerprint,
			"name":            maintainer.Name,
			"email":           maintainer.Email,
			"active":          maintainer.Active,
			"registered_at":   maintainer.RegisteredAt,
			"packages_signed": maintainer.PackagesSigned,
			"last_signed_at":  maintainer.LastSignedAt,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleMaintainerRegister handles registering a new maintainer.
// POST /maintainers
// JSON body: {"public_key": "<base64-encoded>", "name": "...", "email": "..."}
func (d *Daemon) handleMaintainerRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		PublicKey string `json:"public_key"`
		Name      string `json:"name"`
		Email     string `json:"email"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Failed to parse request body: %v", err), http.StatusBadRequest)
		return
	}

	if req.PublicKey == "" {
		http.Error(w, "public_key is required", http.StatusBadRequest)
		return
	}

	// Decode the public key hex string to get bytes, then create PublicKey to compute fingerprint
	pubKeyBytes, err := hex.DecodeString(req.PublicKey)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid public key format (expected hex): %v", err), http.StatusBadRequest)
		return
	}

	pubKey, err := crypto.NewPublicKey(pubKeyBytes)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid public key: %v", err), http.StatusBadRequest)
		return
	}

	// Compute fingerprint from the public key
	fingerprint := pubKey.Fingerprint()

	// Register the maintainer (stores public key as hex string)
	maintainer, err := d.maintainerRegistry.Register(fingerprint, req.Name, req.PublicKey, req.Email)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to register maintainer: %v", err), http.StatusConflict)
		return
	}

	response := map[string]interface{}{
		"status":  "success",
		"message": "Maintainer registered successfully",
		"maintainer": map[string]interface{}{
			"fingerprint":   maintainer.Fingerprint,
			"name":          maintainer.Name,
			"email":         maintainer.Email,
			"active":        maintainer.Active,
			"registered_at": maintainer.RegisteredAt,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// handleMaintainerActivate handles activating a maintainer.
// POST /maintainers/activate/{fingerprint}
func (d *Daemon) handleMaintainerActivate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract fingerprint from URL path: /maintainers/activate/{fingerprint}
	fingerprint := r.URL.Path[len("/maintainers/activate/"):]
	if fingerprint == "" {
		http.Error(w, "Fingerprint is required", http.StatusBadRequest)
		return
	}

	if err := d.maintainerRegistry.Activate(fingerprint); err != nil {
		http.Error(w, fmt.Sprintf("Failed to activate maintainer: %v", err), http.StatusNotFound)
		return
	}

	response := map[string]interface{}{
		"status":      "success",
		"message":     "Maintainer activated successfully",
		"fingerprint": fingerprint,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleMaintainerDeactivate handles deactivating a maintainer.
// POST /maintainers/deactivate/{fingerprint}
func (d *Daemon) handleMaintainerDeactivate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract fingerprint from URL path: /maintainers/deactivate/{fingerprint}
	fingerprint := r.URL.Path[len("/maintainers/deactivate/"):]
	if fingerprint == "" {
		http.Error(w, "Fingerprint is required", http.StatusBadRequest)
		return
	}

	if err := d.maintainerRegistry.Deactivate(fingerprint); err != nil {
		http.Error(w, fmt.Sprintf("Failed to deactivate maintainer: %v", err), http.StatusNotFound)
		return
	}

	response := map[string]interface{}{
		"status":      "success",
		"message":     "Maintainer deactivated successfully",
		"fingerprint": fingerprint,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handlePendingSignatures handles listing packages awaiting maintainer signature.
// GET /signatures/pending
func (d *Daemon) handlePendingSignatures(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get all packages and filter those without maintainer signature
	allPackages := d.packageManager.ListPackages()
	var pending []map[string]interface{}

	for _, pkg := range allPackages {
		// A package is pending if it has no maintainer signature
		if pkg.MaintainerManifestSignature == "" {
			pending = append(pending, map[string]interface{}{
				"package_id":          pkg.PackageID,
				"name":                pkg.Name,
				"version":             pkg.Version,
				"creator_fingerprint": pkg.CreatorFingerprint,
				"created_at":          pkg.CreatedAt,
			})
		}
	}

	response := map[string]interface{}{
		"status":           "success",
		"count":            len(pending),
		"pending_packages": pending,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handlePackageSign handles maintainer signature submission for a package.
// POST /packages/sign/{package_id}
// JSON body: {"maintainer_fingerprint": "...", "signature": "<hex-encoded>"}
func (d *Daemon) handlePackageSign(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract package_id from URL path: /packages/sign/{package_id}
	packageID := r.URL.Path[len("/packages/sign/"):]
	if packageID == "" {
		http.Error(w, "Package ID is required", http.StatusBadRequest)
		return
	}

	var req struct {
		MaintainerFingerprint string `json:"maintainer_fingerprint"`
		Signature             string `json:"signature"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Failed to parse request body: %v", err), http.StatusBadRequest)
		return
	}

	if req.MaintainerFingerprint == "" || req.Signature == "" {
		http.Error(w, "maintainer_fingerprint and signature are required", http.StatusBadRequest)
		return
	}

	// Verify maintainer exists and is active
	maintainer, err := d.maintainerRegistry.Get(req.MaintainerFingerprint)
	if err != nil {
		http.Error(w, fmt.Sprintf("Maintainer not found: %v", err), http.StatusNotFound)
		return
	}

	if !maintainer.Active {
		http.Error(w, "Maintainer is not active", http.StatusForbidden)
		return
	}

	// Get the package
	pkg, exists := d.packageManager.GetPackage(packageID)
	if !exists {
		http.Error(w, "Package not found", http.StatusNotFound)
		return
	}

	// Load the package file to get manifest for verification
	pkgData, err := packagetypes.LoadPackageFromFile(pkg.FilePath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load package file: %v", err), http.StatusInternalServerError)
		return
	}

	// Serialize manifest for signature verification
	manifestData, err := packagetypes.SerializeManifest(&pkgData.Manifest)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to serialize manifest: %v", err), http.StatusInternalServerError)
		return
	}

	// Decode the provided signature
	signatureBytes, err := hex.DecodeString(req.Signature)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid signature format: %v", err), http.StatusBadRequest)
		return
	}

	// Create signature struct
	signature := &crypto.Signature{
		SignedData: signatureBytes,
	}

	// Decode the maintainer's public key from hex string
	pubKeyBytes, err := hex.DecodeString(maintainer.PublicKey)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid maintainer public key format: %v", err), http.StatusInternalServerError)
		return
	}

	pubKey, err := crypto.NewPublicKey(pubKeyBytes)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid maintainer public key: %v", err), http.StatusInternalServerError)
		return
	}

	// Verify the signature using the maintainer's public key
	if err := crypto.Verify(*pubKey, manifestData, signature); err != nil {
		http.Error(w, fmt.Sprintf("Signature verification failed: %v", err), http.StatusUnauthorized)
		return
	}

	// Update package with maintainer signature
	if err := d.packageManager.UpdateMaintainerSignature(packageID, req.MaintainerFingerprint, req.Signature); err != nil {
		http.Error(w, fmt.Sprintf("Failed to update package signature: %v", err), http.StatusInternalServerError)
		return
	}

	// Increment maintainer's signature count
	if err := d.maintainerRegistry.IncrementSignCount(req.MaintainerFingerprint); err != nil {
		log.Printf("Warning: Failed to increment signature count: %v", err)
	}

	response := map[string]interface{}{
		"status":                 "success",
		"message":                "Package signed successfully",
		"package_id":             packageID,
		"maintainer_fingerprint": req.MaintainerFingerprint,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
