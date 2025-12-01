package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestGenerateKey tests API key generation
func TestGenerateKey(t *testing.T) {
	store := setupTestKeyStore(t)
	defer cleanupTestKeyStore(t, store)

	tests := []struct {
		name    string
		keyName string
		level   string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid read key",
			keyName: "test-read",
			level:   LevelRead,
			wantErr: false,
		},
		{
			name:    "valid write key",
			keyName: "test-write",
			level:   LevelWrite,
			wantErr: false,
		},
		{
			name:    "valid admin key",
			keyName: "test-admin",
			level:   LevelAdmin,
			wantErr: false,
		},
		{
			name:    "invalid level",
			keyName: "test-invalid",
			level:   "superuser",
			wantErr: true,
			errMsg:  "invalid permission level",
		},
		{
			name:    "empty name",
			keyName: "",
			level:   LevelRead,
			wantErr: true,
			errMsg:  "key name cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plaintextKey, key, err := store.GenerateKey(tt.keyName, tt.level)

			if tt.wantErr {
				if err == nil {
					t.Errorf("GenerateKey() expected error containing %q, got nil", tt.errMsg)
				} else if tt.errMsg != "" && err.Error() != tt.errMsg {
					t.Errorf("GenerateKey() error = %q, want %q", err.Error(), tt.errMsg)
				}
				return
			}

			if err != nil {
				t.Fatalf("GenerateKey() unexpected error: %v", err)
			}

			// Verify key format
			if len(plaintextKey) != 68 { // "lbs_" + 64 hex chars = 68 total
				t.Errorf("Generated key length = %d, want 68", len(plaintextKey))
			}
			if plaintextKey[:4] != "lbs_" {
				t.Errorf("Generated key prefix = %q, want %q", plaintextKey[:4], "lbs_")
			}

			// Verify stored key properties
			if key.Name != tt.keyName {
				t.Errorf("Key.Name = %q, want %q", key.Name, tt.keyName)
			}
			if key.Level != tt.level {
				t.Errorf("Key.Level = %q, want %q", key.Level, tt.level)
			}
			if key.KeyHash == "" {
				t.Error("Key.KeyHash is empty")
			}
			if key.Revoked {
				t.Error("Newly generated key should not be revoked")
			}
		})
	}
}

// TestGenerateKeyUniqueness tests that multiple generated keys are unique
func TestGenerateKeyUniqueness(t *testing.T) {
	store := setupTestKeyStore(t)
	defer cleanupTestKeyStore(t, store)

	keys := make(map[string]bool)
	hashes := make(map[string]bool)

	for i := 0; i < 10; i++ {
		plaintextKey, key, err := store.GenerateKey("test-key", LevelRead)
		if err != nil {
			t.Fatalf("GenerateKey() error: %v", err)
		}

		if keys[plaintextKey] {
			t.Errorf("Generated duplicate key: %s", plaintextKey)
		}
		keys[plaintextKey] = true

		if hashes[key.KeyHash] {
			t.Errorf("Generated duplicate hash: %s", key.KeyHash)
		}
		hashes[key.KeyHash] = true
	}
}

// TestValidateKey tests API key validation
func TestValidateKey(t *testing.T) {
	store := setupTestKeyStore(t)
	defer cleanupTestKeyStore(t, store)

	// Generate a valid key
	validPlaintextKey, _, err := store.GenerateKey("test-key", LevelRead)
	if err != nil {
		t.Fatalf("GenerateKey() error: %v", err)
	}

	tests := []struct {
		name    string
		key     string
		wantNil bool
	}{
		{
			name:    "valid key",
			key:     validPlaintextKey,
			wantNil: false,
		},
		{
			name:    "invalid key",
			key:     "lbs_" + string(make([]byte, 64)),
			wantNil: true,
		},
		{
			name:    "malformed key - too short",
			key:     "lbs_abc",
			wantNil: true,
		},
		{
			name:    "malformed key - wrong prefix",
			key:     "xyz_" + string(make([]byte, 64)),
			wantNil: true,
		},
		{
			name:    "empty key",
			key:     "",
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, err := store.ValidateKey(tt.key)
			if tt.wantNil {
				if key != nil || err == nil {
					t.Errorf("ValidateKey() = %v, %v; want nil, error", key, err)
				}
			} else {
				if key == nil || err != nil {
					t.Errorf("ValidateKey() = %v, %v; want valid key, nil error", key, err)
				}
			}
		})
	}
}

// TestRevokeKey tests key revocation
func TestRevokeKey(t *testing.T) {
	store := setupTestKeyStore(t)
	defer cleanupTestKeyStore(t, store)

	plaintextKey, keyObj, err := store.GenerateKey("test-key", LevelRead)
	if err != nil {
		t.Fatalf("GenerateKey() error: %v", err)
	}

	// Verify key works before revocation
	validKeyObj, err := store.ValidateKey(plaintextKey)
	if validKeyObj == nil || err != nil {
		t.Fatal("Key should be valid before revocation")
	}

	// Revoke the key
	err = store.RevokeKey(keyObj.ID)
	if err != nil {
		t.Fatalf("RevokeKey() error: %v", err)
	}

	// Verify key no longer validates
	validKeyObj, err = store.ValidateKey(plaintextKey)
	if validKeyObj != nil || err == nil {
		t.Error("Key should be invalid after revocation")
	}

	// Verify revocation persists
	keys := store.ListKeys()
	found := false
	for _, k := range keys {
		if k.ID == keyObj.ID {
			found = true
			if !k.Revoked {
				t.Error("Key should be marked as revoked")
			}
		}
	}
	if !found {
		t.Error("Revoked key should still be in the store")
	}
}

// TestDeleteKey tests key deletion
func TestDeleteKey(t *testing.T) {
	store := setupTestKeyStore(t)
	defer cleanupTestKeyStore(t, store)

	_, keyObj, err := store.GenerateKey("test-key", LevelRead)
	if err != nil {
		t.Fatalf("GenerateKey() error: %v", err)
	}

	// Delete the key
	err = store.DeleteKey(keyObj.ID)
	if err != nil {
		t.Fatalf("DeleteKey() error: %v", err)
	}

	// Verify key is completely removed
	keys := store.ListKeys()
	for _, k := range keys {
		if k.ID == keyObj.ID {
			t.Error("Deleted key should not be in the store")
		}
	}
}

// TestHasPermission tests permission hierarchy
func TestHasPermission(t *testing.T) {
	tests := []struct {
		name     string
		keyLevel string
		required string
		want     bool
	}{
		// Admin permissions
		{
			name:     "admin can read",
			keyLevel: LevelAdmin,
			required: LevelRead,
			want:     true,
		},
		{
			name:     "admin can write",
			keyLevel: LevelAdmin,
			required: LevelWrite,
			want:     true,
		},
		{
			name:     "admin can admin",
			keyLevel: LevelAdmin,
			required: LevelAdmin,
			want:     true,
		},
		// Write permissions
		{
			name:     "write can read",
			keyLevel: LevelWrite,
			required: LevelRead,
			want:     true,
		},
		{
			name:     "write can write",
			keyLevel: LevelWrite,
			required: LevelWrite,
			want:     true,
		},
		{
			name:     "write cannot admin",
			keyLevel: LevelWrite,
			required: LevelAdmin,
			want:     false,
		},
		// Read permissions
		{
			name:     "read can read",
			keyLevel: LevelRead,
			required: LevelRead,
			want:     true,
		},
		{
			name:     "read cannot write",
			keyLevel: LevelRead,
			required: LevelWrite,
			want:     false,
		},
		{
			name:     "read cannot admin",
			keyLevel: LevelRead,
			required: LevelAdmin,
			want:     false,
		},
		// Invalid levels
		{
			name:     "invalid key level",
			keyLevel: "invalid",
			required: LevelRead,
			want:     false,
		},
		{
			name:     "invalid required level",
			keyLevel: LevelAdmin,
			required: "invalid",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := HasPermission(tt.keyLevel, tt.required)
			if got != tt.want {
				t.Errorf("HasPermission(%q, %q) = %v, want %v", tt.keyLevel, tt.required, got, tt.want)
			}
		})
	}
}

// TestAuthenticationMiddleware tests the authentication middleware
func TestAuthenticationMiddleware(t *testing.T) {
	store := setupTestKeyStore(t)
	defer cleanupTestKeyStore(t, store)

	// Generate keys with different levels
	readKey, _, _ := store.GenerateKey("read-key", LevelRead)
	writeKey, _, _ := store.GenerateKey("write-key", LevelWrite)
	adminKey, _, _ := store.GenerateKey("admin-key", LevelAdmin)
	revokedKey, revokedKeyObj, _ := store.GenerateKey("revoked-key", LevelRead)
	store.RevokeKey(revokedKeyObj.ID)

	tests := []struct {
		name           string
		authHeader     string
		requiredLevel  string
		wantStatusCode int
		checkContext   bool
	}{
		{
			name:           "valid read key for read endpoint",
			authHeader:     "Bearer " + readKey,
			requiredLevel:  LevelRead,
			wantStatusCode: http.StatusOK,
			checkContext:   true,
		},
		{
			name:           "valid write key for read endpoint",
			authHeader:     "Bearer " + writeKey,
			requiredLevel:  LevelRead,
			wantStatusCode: http.StatusOK,
			checkContext:   true,
		},
		{
			name:           "valid admin key for read endpoint",
			authHeader:     "Bearer " + adminKey,
			requiredLevel:  LevelRead,
			wantStatusCode: http.StatusOK,
			checkContext:   true,
		},
		{
			name:           "read key for write endpoint - forbidden",
			authHeader:     "Bearer " + readKey,
			requiredLevel:  LevelWrite,
			wantStatusCode: http.StatusForbidden,
			checkContext:   false,
		},
		{
			name:           "write key for admin endpoint - forbidden",
			authHeader:     "Bearer " + writeKey,
			requiredLevel:  LevelAdmin,
			wantStatusCode: http.StatusForbidden,
			checkContext:   false,
		},
		{
			name:           "revoked key - unauthorized",
			authHeader:     "Bearer " + revokedKey,
			requiredLevel:  LevelRead,
			wantStatusCode: http.StatusUnauthorized,
			checkContext:   false,
		},
		{
			name:           "invalid key - unauthorized",
			authHeader:     "Bearer lbs_" + string(make([]byte, 64)),
			requiredLevel:  LevelRead,
			wantStatusCode: http.StatusUnauthorized,
			checkContext:   false,
		},
		{
			name:           "malformed header - unauthorized",
			authHeader:     "Basic " + readKey,
			requiredLevel:  LevelRead,
			wantStatusCode: http.StatusUnauthorized,
			checkContext:   false,
		},
		{
			name:           "missing header - unauthorized",
			authHeader:     "",
			requiredLevel:  LevelRead,
			wantStatusCode: http.StatusUnauthorized,
			checkContext:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test handler that checks for key in context
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.checkContext {
					key := GetAPIKeyFromContext(r.Context())
					if key == nil {
						t.Error("Expected API key in context, got nil")
					}
				}
				w.WriteHeader(http.StatusOK)
			})

			// Wrap with authentication middleware
			middleware := AuthenticationMiddleware(store, tt.requiredLevel)
			wrappedHandler := middleware(handler)

			// Create test request
			req := httptest.NewRequest("GET", "/test", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			rec := httptest.NewRecorder()

			// Execute request
			wrappedHandler.ServeHTTP(rec, req)

			// Check status code
			if rec.Code != tt.wantStatusCode {
				t.Errorf("Status code = %d, want %d", rec.Code, tt.wantStatusCode)
			}
		})
	}
}

// TestContextInjection tests context injection and retrieval
func TestContextInjection(t *testing.T) {
	key := &APIKey{
		ID:    "test-id",
		Name:  "test-key",
		Level: LevelRead,
	}

	ctx := context.Background()
	ctx = WithAPIKey(ctx, key)

	retrievedKey := GetAPIKeyFromContext(ctx)
	if retrievedKey == nil {
		t.Fatal("GetAPIKeyFromContext() returned nil")
	}

	if retrievedKey.ID != key.ID {
		t.Errorf("Retrieved key ID = %q, want %q", retrievedKey.ID, key.ID)
	}
	if retrievedKey.Name != key.Name {
		t.Errorf("Retrieved key Name = %q, want %q", retrievedKey.Name, key.Name)
	}
	if retrievedKey.Level != key.Level {
		t.Errorf("Retrieved key Level = %q, want %q", retrievedKey.Level, key.Level)
	}
}

// TestContextWithoutKey tests context retrieval when no key is present
func TestContextWithoutKey(t *testing.T) {
	ctx := context.Background()
	key := GetAPIKeyFromContext(ctx)
	if key != nil {
		t.Errorf("GetAPIKeyFromContext() = %v, want nil", key)
	}
}

// TestUpdateLastUsed tests that LastUsed timestamp is updated
func TestUpdateLastUsed(t *testing.T) {
	store := setupTestKeyStore(t)
	defer cleanupTestKeyStore(t, store)

	_, keyObj, err := store.GenerateKey("test-key", LevelRead)
	if err != nil {
		t.Fatalf("GenerateKey() error: %v", err)
	}

	originalLastUsed := keyObj.LastUsed

	// Wait a bit to ensure timestamp differs
	time.Sleep(10 * time.Millisecond)

	// Update last used
	err = store.UpdateLastUsed(keyObj.ID)
	if err != nil {
		t.Fatalf("UpdateLastUsed() error: %v", err)
	}

	// Retrieve updated key
	keys := store.ListKeys()
	var updatedKey *APIKey
	for _, k := range keys {
		if k.ID == keyObj.ID {
			updatedKey = &k
			break
		}
	}

	if updatedKey == nil {
		t.Fatal("Could not find updated key")
	}

	if !updatedKey.LastUsed.After(originalLastUsed) {
		t.Errorf("LastUsed not updated: original=%v, updated=%v", originalLastUsed, updatedKey.LastUsed)
	}
}

// TestPersistence tests that keys persist across store reloads
func TestPersistence(t *testing.T) {
	store1 := setupTestKeyStore(t)
	defer cleanupTestKeyStore(t, store1)

	// Generate keys in first store
	plaintextKey1, keyObj1, err := store1.GenerateKey("test-key-1", LevelRead)
	if err != nil {
		t.Fatalf("GenerateKey() error: %v", err)
	}

	plaintextKey2, _, err := store1.GenerateKey("test-key-2", LevelWrite)
	if err != nil {
		t.Fatalf("GenerateKey() error: %v", err)
	}

	// Create second store using same file path
	store2, err := NewAPIKeyStore(store1.filePath)
	if err != nil {
		t.Fatalf("NewAPIKeyStore() error: %v", err)
	}

	// Verify keys exist in second store
	validKey1, err := store2.ValidateKey(plaintextKey1)
	if err != nil || validKey1 == nil {
		t.Error("First key should be valid in second store")
	}
	validKey2, err := store2.ValidateKey(plaintextKey2)
	if err != nil || validKey2 == nil {
		t.Error("Second key should be valid in second store")
	}

	// Revoke a key in second store
	err = store2.RevokeKey(keyObj1.ID)
	if err != nil {
		t.Fatalf("RevokeKey() error: %v", err)
	}

	// Create third store and verify revocation persisted
	store3, err := NewAPIKeyStore(store1.filePath)
	if err != nil {
		t.Fatalf("NewAPIKeyStore() error: %v", err)
	}

	revokedKey, err := store3.ValidateKey(plaintextKey1)
	if err == nil && revokedKey != nil {
		t.Error("Revoked key should be invalid in third store")
	}
}

// Helper functions

func setupTestKeyStore(t *testing.T) *APIKeyStore {
	t.Helper()

	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "libreseed-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	filePath := filepath.Join(tmpDir, "api-keys.yaml")

	store, err := NewAPIKeyStore(filePath)
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("NewAPIKeyStore() error: %v", err)
	}

	return store
}

func cleanupTestKeyStore(t *testing.T, store *APIKeyStore) {
	t.Helper()

	// Remove the entire temp directory
	tmpDir := filepath.Dir(store.filePath)
	if err := os.RemoveAll(tmpDir); err != nil {
		t.Errorf("Failed to cleanup temp dir: %v", err)
	}
}
