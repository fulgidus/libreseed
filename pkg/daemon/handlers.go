package daemon

import (
	"crypto/sha256"
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
)

// handlePackageAdd handles package addition requests.
// POST /packages/add
// Multipart form data:
// - file: the package file
// - name: package name
// - version: package version
// - description: package description (optional)
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

	// Extract file
	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get file: %v", err), http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Extract metadata
	name := r.FormValue("name")
	version := r.FormValue("version")
	description := r.FormValue("description")

	if name == "" || version == "" {
		http.Error(w, "name and version are required", http.StatusBadRequest)
		return
	}

	// Compute file hash
	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		http.Error(w, fmt.Sprintf("Failed to compute hash: %v", err), http.StatusInternalServerError)
		return
	}
	fileHash := hex.EncodeToString(hasher.Sum(nil))

	// Reset file pointer for subsequent read
	if _, err := file.Seek(0, 0); err != nil {
		http.Error(w, fmt.Sprintf("Failed to reset file: %v", err), http.StatusInternalServerError)
		return
	}

	// Create manifest data string for signing
	manifestData := fmt.Sprintf("%s:%s:%s:%s", name, version, description, fileHash)

	// Get public key for signing
	pubKey, err := d.keyManager.PublicKeyCrypto()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get public key: %v", err), http.StatusInternalServerError)
		return
	}

	// Sign the manifest data using crypto.Sign()
	signature, err := crypto.Sign(d.keyManager.PrivateKey(), *pubKey, []byte(manifestData))
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to sign manifest: %v", err), http.StatusInternalServerError)
		return
	}

	// Create PackageInfo with actual file size from uploaded file
	packageInfo := &PackageInfo{
		PackageID:          fileHash, // Using file hash as package ID
		Name:               name,
		Version:            version,
		Description:        description,
		FilePath:           "", // Will be set after file copy
		FileHash:           fileHash,
		FileSize:           header.Size, // Actual file size from multipart header
		CreatedAt:          time.Now(),
		CreatorFingerprint: d.keyManager.Fingerprint(),
		ManifestSignature:  hex.EncodeToString(signature.Bytes()),
		AnnouncedToDHT:     false,
	}

	// Copy file to packages directory
	destPath := filepath.Join(d.packageManager.GetStorageDir(), header.Filename)
	destFile, err := os.Create(destPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create destination file: %v", err), http.StatusInternalServerError)
		return
	}
	defer destFile.Close()

	if _, err := io.Copy(destFile, file); err != nil {
		os.Remove(destPath) // Clean up on failure
		http.Error(w, fmt.Sprintf("Failed to copy file: %v", err), http.StatusInternalServerError)
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

			// Add package to DHT announcer
			d.announcer.AddPackage(infoHash, packageInfo.Name)
			log.Printf("Called d.announcer.AddPackage for %s with InfoHash %x\n", packageInfo.Name, infoHash)

			// Update announcement status in package manager
			if err := d.packageManager.UpdateAnnouncementStatus(packageInfo.PackageID, true); err != nil {
				log.Printf("Warning: Failed to update announcement status: %v\n", err)
			} else {
				log.Printf("Successfully updated announcement status for package %s\n", packageInfo.PackageID)
			}

			log.Printf("Package %s announced to DHT with InfoHash %x\n", packageInfo.Name, infoHash)
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

	// Return success response
	response := map[string]interface{}{
		"status":      "success",
		"package_id":  packageInfo.PackageID,
		"fingerprint": packageInfo.CreatorFingerprint,
		"file_hash":   fileHash,
		"filename":    header.Filename,
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

	// Remove from package manager
	if err := d.packageManager.RemovePackage(packageID); err != nil {
		http.Error(w, fmt.Sprintf("Failed to remove package: %v", err), http.StatusInternalServerError)
		return
	}

	// Delete file from packages directory
	filePath := packageInfo.FilePath
	if err := os.Remove(filePath); err != nil {
		// Log warning but don't fail the request
		// The package metadata is already removed
		fmt.Printf("Warning: Failed to delete package file %s: %v\n", filePath, err)
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
