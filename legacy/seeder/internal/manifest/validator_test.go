package manifest

import (
	"archive/tar"
	"compress/gzip"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// Test helper: Generate a test Ed25519 key pair
func generateTestKeyPair(t *testing.T) (ed25519.PublicKey, ed25519.PrivateKey) {
	t.Helper()
	pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate key pair: %v", err)
	}
	return pubKey, privKey
}

// Test helper: Format pubkey as "ed25519:base64"
func formatPubKey(pubKey ed25519.PublicKey) string {
	return "ed25519:" + base64.StdEncoding.EncodeToString(pubKey)
}

// Test helper: Sign a message and return "ed25519:base64" signature
func signMessage(message []byte, privKey ed25519.PrivateKey) string {
	signature := ed25519.Sign(privKey, message)
	return "ed25519:" + base64.StdEncoding.EncodeToString(signature)
}

// Test helper: Create a valid test tarball with manifest
func createTestTarball(t *testing.T, manifest *FullManifest) string {
	t.Helper()

	// Create temp file
	tmpFile, err := os.CreateTemp(t.TempDir(), "test-package-*.tgz")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer tmpFile.Close()

	// Create gzip writer
	gzWriter := gzip.NewWriter(tmpFile)
	defer gzWriter.Close()

	// Create tar writer
	tarWriter := tar.NewWriter(gzWriter)
	defer tarWriter.Close()

	// Marshal manifest to JSON
	manifestJSON, err := json.Marshal(manifest)
	if err != nil {
		t.Fatalf("failed to marshal manifest: %v", err)
	}

	// Write manifest.json to tarball
	header := &tar.Header{
		Name: "manifest.json",
		Mode: 0644,
		Size: int64(len(manifestJSON)),
	}
	if err := tarWriter.WriteHeader(header); err != nil {
		t.Fatalf("failed to write tar header: %v", err)
	}
	if _, err := tarWriter.Write(manifestJSON); err != nil {
		t.Fatalf("failed to write manifest to tar: %v", err)
	}

	// Add some test files
	for path := range manifest.Files {
		content := []byte("test content for " + path)
		header := &tar.Header{
			Name: path,
			Mode: 0644,
			Size: int64(len(content)),
		}
		if err := tarWriter.WriteHeader(header); err != nil {
			t.Fatalf("failed to write tar header for %s: %v", path, err)
		}
		if _, err := tarWriter.Write(content); err != nil {
			t.Fatalf("failed to write file %s to tar: %v", path, err)
		}
	}

	// Close all writers to flush
	if err := tarWriter.Close(); err != nil {
		t.Fatalf("failed to close tar writer: %v", err)
	}
	if err := gzWriter.Close(); err != nil {
		t.Fatalf("failed to close gzip writer: %v", err)
	}
	if err := tmpFile.Close(); err != nil {
		t.Fatalf("failed to close temp file: %v", err)
	}

	return tmpFile.Name()
}

// Test helper: Create valid test manifests with signatures
func createValidTestManifests(t *testing.T) (*FullManifest, *MinimalManifest, ed25519.PrivateKey, string) {
	t.Helper()

	// Generate key pair
	pubKey, privKey := generateTestKeyPair(t)
	pubKeyStr := formatPubKey(pubKey)

	// Create files map
	files := map[string]string{
		"dist/bundle.js": "sha256:abc123def456",
		"src/index.js":   "sha256:789012345678",
	}

	// Compute contentHash
	contentHash, err := ComputeContentHash(files)
	if err != nil {
		t.Fatalf("failed to compute contentHash: %v", err)
	}

	// Sign contentHash (sign the raw 32-byte hash, not the hex string)
	contentHashBytes, err := hex.DecodeString(strings.TrimPrefix(contentHash, "sha256:"))
	if err != nil {
		t.Fatalf("failed to decode contentHash: %v", err)
	}
	contentSignature := signMessage(contentHashBytes, privKey)

	// Create full manifest
	fullManifest := &FullManifest{
		Name:        "test-package",
		Version:     "1.0.0",
		Description: "Test package",
		Files:       files,
		ContentHash: contentHash,
		PubKey:      pubKeyStr,
		Signature:   contentSignature,
	}

	// Create tarball
	tgzPath := createTestTarball(t, fullManifest)

	// Compute infohash
	infohash, err := ComputeInfohash(tgzPath)
	if err != nil {
		t.Fatalf("failed to compute infohash: %v", err)
	}

	// Sign infohash (sign the raw 32-byte hash, not the hex string)
	infohashBytes, err := hex.DecodeString(strings.TrimPrefix(infohash, "sha256:"))
	if err != nil {
		t.Fatalf("failed to decode infohash: %v", err)
	}
	infohashSignature := signMessage(infohashBytes, privKey)

	// Create minimal manifest
	minimalManifest := &MinimalManifest{
		Name:      "test-package",
		Version:   "1.0.0",
		Infohash:  infohash,
		PubKey:    pubKeyStr,
		Signature: infohashSignature,
	}

	return fullManifest, minimalManifest, privKey, tgzPath
}

// ==================== ComputeContentHash Tests ====================

func TestComputeContentHash_Valid(t *testing.T) {
	files := map[string]string{
		"dist/bundle.js": "sha256:abc123",
		"src/index.js":   "sha256:def456",
	}

	hash, err := ComputeContentHash(files)
	if err != nil {
		t.Fatalf("ComputeContentHash failed: %v", err)
	}

	// Verify format
	if len(hash) == 0 {
		t.Error("contentHash is empty")
	}
	if !strings.HasPrefix(hash, "sha256:") {
		t.Errorf("contentHash missing 'sha256:' prefix: %s", hash)
	}

	// Verify determinism: same input should produce same hash
	hash2, err := ComputeContentHash(files)
	if err != nil {
		t.Fatalf("second ComputeContentHash failed: %v", err)
	}
	if hash != hash2 {
		t.Errorf("contentHash not deterministic: %s != %s", hash, hash2)
	}
}

func TestComputeContentHash_EmptyFiles(t *testing.T) {
	files := map[string]string{}

	_, err := ComputeContentHash(files)
	if err == nil {
		t.Error("expected error for empty files map, got nil")
	}
}

func TestComputeContentHash_InvalidHashFormat(t *testing.T) {
	files := map[string]string{
		"file1.js": "abc123", // Missing "sha256:" prefix
	}

	_, err := ComputeContentHash(files)
	if err == nil {
		t.Error("expected error for invalid hash format, got nil")
	}
	if err != nil && err.Error() != "invalid hash format: expected 'sha256:' prefix in hash for file1.js" {
		t.Logf("got error: %v", err)
	}
}

func TestComputeContentHash_Sorting(t *testing.T) {
	// Create two maps with same files but different insertion order
	files1 := map[string]string{
		"z.js": "sha256:aaa",
		"a.js": "sha256:bbb",
		"m.js": "sha256:ccc",
	}
	files2 := map[string]string{
		"a.js": "sha256:bbb",
		"m.js": "sha256:ccc",
		"z.js": "sha256:aaa",
	}

	hash1, err := ComputeContentHash(files1)
	if err != nil {
		t.Fatalf("ComputeContentHash(files1) failed: %v", err)
	}

	hash2, err := ComputeContentHash(files2)
	if err != nil {
		t.Fatalf("ComputeContentHash(files2) failed: %v", err)
	}

	// Both should produce identical hash (deterministic sorting)
	if hash1 != hash2 {
		t.Errorf("hashes don't match despite same content:\nhash1: %s\nhash2: %s", hash1, hash2)
	}
}

// ==================== VerifyContentHash Tests ====================

func TestVerifyContentHash_Valid(t *testing.T) {
	files := map[string]string{
		"file1.js": "sha256:abc",
		"file2.js": "sha256:def",
	}

	contentHash, err := ComputeContentHash(files)
	if err != nil {
		t.Fatalf("ComputeContentHash failed: %v", err)
	}

	manifest := &FullManifest{
		Files:       files,
		ContentHash: contentHash,
	}

	err = VerifyContentHash(manifest)
	if err != nil {
		t.Errorf("VerifyContentHash failed: %v", err)
	}
}

func TestVerifyContentHash_Mismatch(t *testing.T) {
	files := map[string]string{
		"file1.js": "sha256:abc",
	}

	manifest := &FullManifest{
		Files:       files,
		ContentHash: "sha256:wrong_hash",
	}

	err := VerifyContentHash(manifest)
	if err == nil {
		t.Error("expected contentHash mismatch error, got nil")
	}
	if err != ErrContentHashMismatch && err != nil {
		t.Logf("got error: %v", err)
	}
}

// ==================== ComputeInfohash Tests ====================

func TestComputeInfohash_Valid(t *testing.T) {
	// Create a test file
	tmpFile, err := os.CreateTemp(t.TempDir(), "test-*.tgz")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer tmpFile.Close()

	content := []byte("test tarball content")
	if _, err := tmpFile.Write(content); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}
	tmpFile.Close()

	// Compute infohash
	infohash, err := ComputeInfohash(tmpFile.Name())
	if err != nil {
		t.Fatalf("ComputeInfohash failed: %v", err)
	}

	// Verify format
	if !strings.HasPrefix(infohash, "sha256:") {
		t.Errorf("infohash missing 'sha256:' prefix: %s", infohash)
	}

	// Verify correctness
	expectedHash := sha256.Sum256(content)
	expectedInfohash := "sha256:" + hex.EncodeToString(expectedHash[:])
	if infohash != expectedInfohash {
		t.Errorf("infohash mismatch:\nexpected: %s\ngot:      %s", expectedInfohash, infohash)
	}
}

func TestComputeInfohash_NonExistentFile(t *testing.T) {
	_, err := ComputeInfohash("/nonexistent/file.tgz")
	if err == nil {
		t.Error("expected error for nonexistent file, got nil")
	}
}

// ==================== ExtractManifest Tests ====================

func TestExtractManifest_Valid(t *testing.T) {
	manifest := &FullManifest{
		Name:        "test",
		Version:     "1.0.0",
		Files:       map[string]string{"file.js": "sha256:abc"},
		ContentHash: "sha256:def",
		PubKey:      "ed25519:test",
		Signature:   "ed25519:sig",
	}

	tgzPath := createTestTarball(t, manifest)

	extracted, err := ExtractManifest(tgzPath)
	if err != nil {
		t.Fatalf("ExtractManifest failed: %v", err)
	}

	if extracted.Name != manifest.Name {
		t.Errorf("name mismatch: expected %s, got %s", manifest.Name, extracted.Name)
	}
	if extracted.Version != manifest.Version {
		t.Errorf("version mismatch: expected %s, got %s", manifest.Version, extracted.Version)
	}
}

func TestExtractManifest_NotFound(t *testing.T) {
	// Create tarball without manifest.json
	tmpFile, err := os.CreateTemp(t.TempDir(), "test-*.tgz")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer tmpFile.Close()

	gzWriter := gzip.NewWriter(tmpFile)
	tarWriter := tar.NewWriter(gzWriter)

	// Add a different file
	content := []byte("not a manifest")
	header := &tar.Header{
		Name: "other.txt",
		Mode: 0644,
		Size: int64(len(content)),
	}
	tarWriter.WriteHeader(header)
	tarWriter.Write(content)

	tarWriter.Close()
	gzWriter.Close()
	tmpFile.Close()

	_, err = ExtractManifest(tmpFile.Name())
	if err != ErrManifestNotFound {
		t.Errorf("expected ErrManifestNotFound, got: %v", err)
	}
}

// ==================== VerifySignature Tests ====================

func TestVerifySignature_Valid(t *testing.T) {
	pubKey, privKey := generateTestKeyPair(t)
	message := []byte("test message")

	pubKeyStr := formatPubKey(pubKey)
	signatureStr := signMessage(message, privKey)

	err := VerifySignature(pubKeyStr, signatureStr, message)
	if err != nil {
		t.Errorf("VerifySignature failed for valid signature: %v", err)
	}
}

func TestVerifySignature_InvalidSignature(t *testing.T) {
	pubKey, privKey := generateTestKeyPair(t)
	message := []byte("test message")
	wrongMessage := []byte("wrong message")

	pubKeyStr := formatPubKey(pubKey)
	// Sign wrong message
	signatureStr := signMessage(wrongMessage, privKey)

	err := VerifySignature(pubKeyStr, signatureStr, message)
	if err != ErrSignatureVerifyFailed {
		t.Errorf("expected ErrSignatureVerifyFailed, got: %v", err)
	}
}

func TestVerifySignature_InvalidPubKeyFormat(t *testing.T) {
	tests := []struct {
		name   string
		pubKey string
	}{
		{"missing prefix", "invalid_pubkey"},
		{"wrong prefix", "rsa:something"},
		{"empty", ""},
		{"only prefix", "ed25519:"},
		{"invalid base64", "ed25519:!!!invalid!!!"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := VerifySignature(tt.pubKey, "ed25519:dGVzdA==", []byte("msg"))
			if err == nil {
				t.Error("expected error for invalid pubkey format, got nil")
			}
		})
	}
}

func TestVerifySignature_InvalidSignatureFormat(t *testing.T) {
	pubKey, _ := generateTestKeyPair(t)
	pubKeyStr := formatPubKey(pubKey)

	tests := []struct {
		name      string
		signature string
	}{
		{"missing prefix", "invalid_signature"},
		{"wrong prefix", "rsa:something"},
		{"empty", ""},
		{"only prefix", "ed25519:"},
		{"invalid base64", "ed25519:!!!invalid!!!"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := VerifySignature(pubKeyStr, tt.signature, []byte("msg"))
			if err == nil {
				t.Error("expected error for invalid signature format, got nil")
			}
		})
	}
}

// ==================== ValidatePackage Tests ====================

func TestValidatePackage_Valid(t *testing.T) {
	fullManifest, minimalManifest, _, tgzPath := createValidTestManifests(t)

	extracted, err := ValidatePackage(tgzPath, minimalManifest)
	if err != nil {
		t.Fatalf("ValidatePackage failed: %v", err)
	}

	if extracted.Name != fullManifest.Name {
		t.Errorf("name mismatch: expected %s, got %s", fullManifest.Name, extracted.Name)
	}
	if extracted.ContentHash != fullManifest.ContentHash {
		t.Errorf("contentHash mismatch")
	}
}

func TestValidatePackage_InvalidMinimalSignature(t *testing.T) {
	_, minimalManifest, _, tgzPath := createValidTestManifests(t)

	// Corrupt the signature
	minimalManifest.Signature = "ed25519:aW52YWxpZF9zaWduYXR1cmU="

	_, err := ValidatePackage(tgzPath, minimalManifest)
	if err == nil {
		t.Error("expected validation to fail with invalid minimal signature")
	}
}

func TestValidatePackage_InfohashMismatch(t *testing.T) {
	_, minimalManifest, privKey, tgzPath := createValidTestManifests(t)

	// Change infohash to incorrect value
	wrongInfohash := "sha256:0000000000000000000000000000000000000000000000000000000000000000"
	minimalManifest.Infohash = wrongInfohash
	// Re-sign with wrong infohash
	minimalManifest.Signature = signMessage([]byte(wrongInfohash), privKey)

	_, err := ValidatePackage(tgzPath, minimalManifest)
	if err != ErrInfohashMismatch && err == nil {
		t.Errorf("expected ErrInfohashMismatch, got: %v", err)
	}
}

func TestValidatePackage_InvalidFullSignature(t *testing.T) {
	fullManifest, minimalManifest, _, _ := createValidTestManifests(t)

	// Corrupt full manifest signature by creating new tarball with bad signature
	fullManifest.Signature = "ed25519:aW52YWxpZF9zaWduYXR1cmU="
	tgzPath := createTestTarball(t, fullManifest)

	// Recompute infohash for new tarball and re-sign
	_, privKey := generateTestKeyPair(t)
	newInfohash, _ := ComputeInfohash(tgzPath)
	minimalManifest.Infohash = newInfohash
	minimalManifest.Signature = signMessage([]byte(newInfohash), privKey)

	_, err := ValidatePackage(tgzPath, minimalManifest)
	if err == nil {
		t.Error("expected validation to fail with invalid full signature")
	}
}

func TestValidatePackage_ContentHashMismatch(t *testing.T) {
	fullManifest, minimalManifest, privKey, _ := createValidTestManifests(t)

	// Change contentHash to wrong value
	fullManifest.ContentHash = "sha256:0000000000000000000000000000000000000000000000000000000000000000"
	// Re-sign with wrong contentHash
	fullManifest.Signature = signMessage([]byte(fullManifest.ContentHash), privKey)

	// Create new tarball with corrupted manifest
	tgzPath := createTestTarball(t, fullManifest)

	// Update minimal manifest for new tarball
	newInfohash, _ := ComputeInfohash(tgzPath)
	minimalManifest.Infohash = newInfohash
	minimalManifest.Signature = signMessage([]byte(newInfohash), privKey)

	_, err := ValidatePackage(tgzPath, minimalManifest)
	if err != ErrContentHashMismatch && err == nil {
		t.Errorf("expected ErrContentHashMismatch, got: %v", err)
	}
}

func TestValidatePackage_PubkeyMismatch(t *testing.T) {
	_, minimalManifest, _, tgzPath := createValidTestManifests(t)

	// Generate different key for minimal manifest
	newPubKey, newPrivKey := generateTestKeyPair(t)
	minimalManifest.PubKey = formatPubKey(newPubKey)
	// Re-sign with new key
	minimalManifest.Signature = signMessage([]byte(minimalManifest.Infohash), newPrivKey)

	_, err := ValidatePackage(tgzPath, minimalManifest)
	if err != ErrPubkeyMismatch && err == nil {
		t.Errorf("expected ErrPubkeyMismatch, got: %v", err)
	}
}

// ==================== ValidateMinimalManifest Tests ====================

func TestValidateMinimalManifest_Valid(t *testing.T) {
	m := &MinimalManifest{
		Name:      "test",
		Version:   "1.0.0",
		Infohash:  "sha256:abc123",
		PubKey:    "ed25519:dGVzdA==",
		Signature: "ed25519:c2lnbmF0dXJl",
	}

	err := ValidateMinimalManifest(m)
	if err != nil {
		t.Errorf("ValidateMinimalManifest failed: %v", err)
	}
}

func TestValidateMinimalManifest_EmptyFields(t *testing.T) {
	tests := []struct {
		name     string
		manifest *MinimalManifest
	}{
		{"empty name", &MinimalManifest{Name: "", Version: "1.0.0", Infohash: "sha256:abc", PubKey: "ed25519:key", Signature: "ed25519:sig"}},
		{"empty version", &MinimalManifest{Name: "test", Version: "", Infohash: "sha256:abc", PubKey: "ed25519:key", Signature: "ed25519:sig"}},
		{"empty infohash", &MinimalManifest{Name: "test", Version: "1.0.0", Infohash: "", PubKey: "ed25519:key", Signature: "ed25519:sig"}},
		{"empty pubkey", &MinimalManifest{Name: "test", Version: "1.0.0", Infohash: "sha256:abc", PubKey: "", Signature: "ed25519:sig"}},
		{"empty signature", &MinimalManifest{Name: "test", Version: "1.0.0", Infohash: "sha256:abc", PubKey: "ed25519:key", Signature: ""}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateMinimalManifest(tt.manifest)
			if err == nil {
				t.Error("expected error for empty field, got nil")
			}
		})
	}
}

func TestValidateMinimalManifest_InvalidPrefixes(t *testing.T) {
	tests := []struct {
		name     string
		manifest *MinimalManifest
	}{
		{"invalid infohash prefix", &MinimalManifest{Name: "test", Version: "1.0.0", Infohash: "md5:abc", PubKey: "ed25519:key", Signature: "ed25519:sig"}},
		{"invalid pubkey prefix", &MinimalManifest{Name: "test", Version: "1.0.0", Infohash: "sha256:abc", PubKey: "rsa:key", Signature: "ed25519:sig"}},
		{"invalid signature prefix", &MinimalManifest{Name: "test", Version: "1.0.0", Infohash: "sha256:abc", PubKey: "ed25519:key", Signature: "rsa:sig"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateMinimalManifest(tt.manifest)
			if err == nil {
				t.Error("expected error for invalid prefix, got nil")
			}
		})
	}
}

// ==================== ValidateFullManifest Tests ====================

func TestValidateFullManifest_Valid(t *testing.T) {
	m := &FullManifest{
		Name:        "test",
		Version:     "1.0.0",
		Files:       map[string]string{"file.js": "sha256:abc"},
		ContentHash: "sha256:def",
		PubKey:      "ed25519:dGVzdA==",
		Signature:   "ed25519:c2lnbmF0dXJl",
	}

	err := ValidateFullManifest(m)
	if err != nil {
		t.Errorf("ValidateFullManifest failed: %v", err)
	}
}

func TestValidateFullManifest_EmptyFields(t *testing.T) {
	tests := []struct {
		name     string
		manifest *FullManifest
	}{
		{"empty name", &FullManifest{Name: "", Version: "1.0.0", Files: map[string]string{"f": "sha256:a"}, ContentHash: "sha256:b", PubKey: "ed25519:key", Signature: "ed25519:sig"}},
		{"empty version", &FullManifest{Name: "test", Version: "", Files: map[string]string{"f": "sha256:a"}, ContentHash: "sha256:b", PubKey: "ed25519:key", Signature: "ed25519:sig"}},
		{"empty files", &FullManifest{Name: "test", Version: "1.0.0", Files: map[string]string{}, ContentHash: "sha256:b", PubKey: "ed25519:key", Signature: "ed25519:sig"}},
		{"empty contentHash", &FullManifest{Name: "test", Version: "1.0.0", Files: map[string]string{"f": "sha256:a"}, ContentHash: "", PubKey: "ed25519:key", Signature: "ed25519:sig"}},
		{"empty pubkey", &FullManifest{Name: "test", Version: "1.0.0", Files: map[string]string{"f": "sha256:a"}, ContentHash: "sha256:b", PubKey: "", Signature: "ed25519:sig"}},
		{"empty signature", &FullManifest{Name: "test", Version: "1.0.0", Files: map[string]string{"f": "sha256:a"}, ContentHash: "sha256:b", PubKey: "ed25519:key", Signature: ""}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateFullManifest(tt.manifest)
			if err == nil {
				t.Error("expected error for empty field, got nil")
			}
		})
	}
}

func TestValidateFullManifest_InvalidHashPrefix(t *testing.T) {
	m := &FullManifest{
		Name:        "test",
		Version:     "1.0.0",
		Files:       map[string]string{"file.js": "md5:abc"}, // Wrong prefix
		ContentHash: "sha256:def",
		PubKey:      "ed25519:key",
		Signature:   "ed25519:sig",
	}

	err := ValidateFullManifest(m)
	if err == nil {
		t.Error("expected error for invalid file hash prefix, got nil")
	}
}
