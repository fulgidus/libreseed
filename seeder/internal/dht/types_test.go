package dht

import (
	"encoding/json"
	"testing"
	"time"
)

// TestPublisherSelectionPolicy_String tests the String() method for all policy enum values.
func TestPublisherSelectionPolicy_String(t *testing.T) {
	tests := []struct {
		name   string
		policy PublisherSelectionPolicy
		want   string
	}{
		{"PolicyFirstSeen", PolicyFirstSeen, "first-seen"},
		{"PolicyLatestVersion", PolicyLatestVersion, "latest-version"},
		{"PolicyUserTrust", PolicyUserTrust, "user-trust"},
		{"PolicySeederCount", PolicySeederCount, "seeder-count"},
		{"Unknown policy", PublisherSelectionPolicy(999), "unknown(999)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.policy.String()
			if got != tt.want {
				t.Errorf("PublisherSelectionPolicy.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestMinimalManifest_Validate tests validation logic for MinimalManifest.
func TestMinimalManifest_Validate(t *testing.T) {
	validManifest := MinimalManifest{
		Protocol:  ProtocolVersion,
		Name:      "test-package",
		Version:   "1.0.0",
		Infohash:  "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		Pubkey:    "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=",
		Signature: "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
		Timestamp: time.Now().UnixMilli(),
	}

	tests := []struct {
		name     string
		manifest MinimalManifest
		wantErr  error
	}{
		{
			name:     "valid manifest",
			manifest: validManifest,
			wantErr:  nil,
		},
		{
			name: "empty protocol",
			manifest: MinimalManifest{
				Protocol:  "",
				Name:      "test",
				Version:   "1.0.0",
				Infohash:  validManifest.Infohash,
				Pubkey:    validManifest.Pubkey,
				Signature: validManifest.Signature,
				Timestamp: validManifest.Timestamp,
			},
			wantErr: ErrEmptyProtocol,
		},
		{
			name: "invalid protocol",
			manifest: MinimalManifest{
				Protocol:  "wrong-protocol",
				Name:      "test",
				Version:   "1.0.0",
				Infohash:  validManifest.Infohash,
				Pubkey:    validManifest.Pubkey,
				Signature: validManifest.Signature,
				Timestamp: validManifest.Timestamp,
			},
			wantErr: ErrInvalidProtocol,
		},
		{
			name: "empty name",
			manifest: MinimalManifest{
				Protocol:  ProtocolVersion,
				Name:      "",
				Version:   "1.0.0",
				Infohash:  validManifest.Infohash,
				Pubkey:    validManifest.Pubkey,
				Signature: validManifest.Signature,
				Timestamp: validManifest.Timestamp,
			},
			wantErr: ErrEmptyName,
		},
		{
			name: "name too long",
			manifest: MinimalManifest{
				Protocol:  ProtocolVersion,
				Name:      string(make([]byte, MaxPackageNameLength+1)),
				Version:   "1.0.0",
				Infohash:  validManifest.Infohash,
				Pubkey:    validManifest.Pubkey,
				Signature: validManifest.Signature,
				Timestamp: validManifest.Timestamp,
			},
			wantErr: ErrNameTooLong,
		},
		{
			name: "empty version",
			manifest: MinimalManifest{
				Protocol:  ProtocolVersion,
				Name:      "test",
				Version:   "",
				Infohash:  validManifest.Infohash,
				Pubkey:    validManifest.Pubkey,
				Signature: validManifest.Signature,
				Timestamp: validManifest.Timestamp,
			},
			wantErr: ErrEmptyVersion,
		},
		{
			name: "version too long",
			manifest: MinimalManifest{
				Protocol:  ProtocolVersion,
				Name:      "test",
				Version:   string(make([]byte, MaxVersionLength+1)),
				Infohash:  validManifest.Infohash,
				Pubkey:    validManifest.Pubkey,
				Signature: validManifest.Signature,
				Timestamp: validManifest.Timestamp,
			},
			wantErr: ErrVersionTooLong,
		},
		{
			name: "invalid version format",
			manifest: MinimalManifest{
				Protocol:  ProtocolVersion,
				Name:      "test",
				Version:   "not-a-version",
				Infohash:  validManifest.Infohash,
				Pubkey:    validManifest.Pubkey,
				Signature: validManifest.Signature,
				Timestamp: validManifest.Timestamp,
			},
			wantErr: ErrInvalidVersion,
		},
		{
			name: "empty infohash",
			manifest: MinimalManifest{
				Protocol:  ProtocolVersion,
				Name:      "test",
				Version:   "1.0.0",
				Infohash:  "",
				Pubkey:    validManifest.Pubkey,
				Signature: validManifest.Signature,
				Timestamp: validManifest.Timestamp,
			},
			wantErr: ErrEmptyInfohash,
		},
		{
			name: "invalid infohash length",
			manifest: MinimalManifest{
				Protocol:  ProtocolVersion,
				Name:      "test",
				Version:   "1.0.0",
				Infohash:  "tooshort",
				Pubkey:    validManifest.Pubkey,
				Signature: validManifest.Signature,
				Timestamp: validManifest.Timestamp,
			},
			wantErr: ErrInvalidInfohash,
		},
		{
			name: "invalid infohash characters",
			manifest: MinimalManifest{
				Protocol:  ProtocolVersion,
				Name:      "test",
				Version:   "1.0.0",
				Infohash:  "ZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZ",
				Pubkey:    validManifest.Pubkey,
				Signature: validManifest.Signature,
				Timestamp: validManifest.Timestamp,
			},
			wantErr: ErrInvalidInfohash,
		},
		{
			name: "empty pubkey",
			manifest: MinimalManifest{
				Protocol:  ProtocolVersion,
				Name:      "test",
				Version:   "1.0.0",
				Infohash:  validManifest.Infohash,
				Pubkey:    "",
				Signature: validManifest.Signature,
				Timestamp: validManifest.Timestamp,
			},
			wantErr: ErrEmptyPubkey,
		},
		{
			name: "empty signature",
			manifest: MinimalManifest{
				Protocol:  ProtocolVersion,
				Name:      "test",
				Version:   "1.0.0",
				Infohash:  validManifest.Infohash,
				Pubkey:    validManifest.Pubkey,
				Signature: "",
				Timestamp: validManifest.Timestamp,
			},
			wantErr: ErrEmptySignature,
		},
		{
			name: "zero timestamp",
			manifest: MinimalManifest{
				Protocol:  ProtocolVersion,
				Name:      "test",
				Version:   "1.0.0",
				Infohash:  validManifest.Infohash,
				Pubkey:    validManifest.Pubkey,
				Signature: validManifest.Signature,
				Timestamp: 0,
			},
			wantErr: ErrInvalidTimestamp,
		},
		{
			name: "negative timestamp",
			manifest: MinimalManifest{
				Protocol:  ProtocolVersion,
				Name:      "test",
				Version:   "1.0.0",
				Infohash:  validManifest.Infohash,
				Pubkey:    validManifest.Pubkey,
				Signature: validManifest.Signature,
				Timestamp: -1,
			},
			wantErr: ErrInvalidTimestamp,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.manifest.Validate()
			if tt.wantErr == nil {
				if err != nil {
					t.Errorf("Validate() unexpected error = %v", err)
				}
			} else {
				if err == nil {
					t.Errorf("Validate() expected error %v, got nil", tt.wantErr)
				} else if !isErrorType(err, tt.wantErr) {
					t.Errorf("Validate() error = %v, want error type %v", err, tt.wantErr)
				}
			}
		})
	}
}

// TestMinimalManifest_SigningData tests SigningData() for determinism and correctness.
func TestMinimalManifest_SigningData(t *testing.T) {
	manifest := MinimalManifest{
		Protocol:  ProtocolVersion,
		Name:      "test-package",
		Version:   "1.0.0",
		Infohash:  "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		Pubkey:    "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=",
		Signature: "SHOULD_BE_EXCLUDED",
		Timestamp: 1234567890000,
	}

	// Test determinism: calling SigningData() multiple times should return identical bytes
	data1, err1 := manifest.SigningData()
	if err1 != nil {
		t.Fatalf("SigningData() first call error = %v", err1)
	}

	data2, err2 := manifest.SigningData()
	if err2 != nil {
		t.Fatalf("SigningData() second call error = %v", err2)
	}

	if string(data1) != string(data2) {
		t.Errorf("SigningData() not deterministic: %q != %q", data1, data2)
	}

	// Test that signature field is excluded
	var unmarshaled map[string]interface{}
	if err := json.Unmarshal(data1, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal signing data: %v", err)
	}

	if _, hasSignature := unmarshaled["signature"]; hasSignature {
		t.Error("SigningData() should not include signature field")
	}

	// Test that all other required fields are present
	requiredFields := []string{"protocol", "name", "version", "infohash", "pubkey", "timestamp"}
	for _, field := range requiredFields {
		if _, exists := unmarshaled[field]; !exists {
			t.Errorf("SigningData() missing required field: %s", field)
		}
	}
}

// TestMinimalManifest_IsExpired tests expiration logic.
func TestMinimalManifest_IsExpired(t *testing.T) {
	now := time.Now()
	ttl := 24 * time.Hour

	tests := []struct {
		name      string
		timestamp int64
		ttl       time.Duration
		want      bool
	}{
		{
			name:      "not expired - fresh",
			timestamp: now.UnixMilli(),
			ttl:       ttl,
			want:      false,
		},
		{
			name:      "not expired - within TTL",
			timestamp: now.Add(-12 * time.Hour).UnixMilli(),
			ttl:       ttl,
			want:      false,
		},
		{
			name:      "expired - beyond TTL",
			timestamp: now.Add(-25 * time.Hour).UnixMilli(),
			ttl:       ttl,
			want:      true,
		},
		{
			name: "boundary - exactly at TTL",
			// Subtract 1ms buffer to account for test execution time
			timestamp: now.Add(-ttl).Add(time.Millisecond).UnixMilli(),
			ttl:       ttl,
			want:      false, // Should not be expired at exact boundary
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manifest := MinimalManifest{Timestamp: tt.timestamp}
			got := manifest.IsExpired(tt.ttl)
			if got != tt.want {
				t.Errorf("IsExpired() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestPublisherEntry_Validate tests validation for PublisherEntry.
func TestPublisherEntry_Validate(t *testing.T) {
	validEntry := PublisherEntry{
		Pubkey:        "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=",
		LatestVersion: "1.0.0",
		FirstSeen:     time.Now().UnixMilli(),
		Signature:     "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
	}

	tests := []struct {
		name    string
		entry   PublisherEntry
		wantErr error
	}{
		{
			name:    "valid entry",
			entry:   validEntry,
			wantErr: nil,
		},
		{
			name: "empty pubkey",
			entry: PublisherEntry{
				Pubkey:        "",
				LatestVersion: "1.0.0",
				FirstSeen:     validEntry.FirstSeen,
				Signature:     validEntry.Signature,
			},
			wantErr: ErrEmptyPubkey,
		},
		{
			name: "empty latestVersion",
			entry: PublisherEntry{
				Pubkey:        validEntry.Pubkey,
				LatestVersion: "",
				FirstSeen:     validEntry.FirstSeen,
				Signature:     validEntry.Signature,
			},
			wantErr: ErrEmptyLatestVersion,
		},
		{
			name: "invalid latestVersion format",
			entry: PublisherEntry{
				Pubkey:        validEntry.Pubkey,
				LatestVersion: "not-a-version",
				FirstSeen:     validEntry.FirstSeen,
				Signature:     validEntry.Signature,
			},
			wantErr: ErrInvalidVersion,
		},
		{
			name: "zero firstSeen",
			entry: PublisherEntry{
				Pubkey:        validEntry.Pubkey,
				LatestVersion: "1.0.0",
				FirstSeen:     0,
				Signature:     validEntry.Signature,
			},
			wantErr: ErrInvalidTimestamp,
		},
		{
			name: "empty signature",
			entry: PublisherEntry{
				Pubkey:        validEntry.Pubkey,
				LatestVersion: "1.0.0",
				FirstSeen:     validEntry.FirstSeen,
				Signature:     "",
			},
			wantErr: ErrEmptySignature,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.entry.Validate()
			if tt.wantErr == nil {
				if err != nil {
					t.Errorf("Validate() unexpected error = %v", err)
				}
			} else {
				if err == nil {
					t.Errorf("Validate() expected error %v, got nil", tt.wantErr)
				} else if !isErrorType(err, tt.wantErr) {
					t.Errorf("Validate() error = %v, want error type %v", err, tt.wantErr)
				}
			}
		})
	}
}

// TestNameIndex_Validate tests validation for NameIndex.
func TestNameIndex_Validate(t *testing.T) {
	validIndex := NameIndex{
		Name:         "test-package",
		Protocol:     ProtocolVersion,
		IndexVersion: IndexFormatVersion,
		Publishers: []PublisherEntry{
			{
				Pubkey:        "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=",
				LatestVersion: "1.0.0",
				FirstSeen:     time.Now().UnixMilli(),
				Signature:     "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
			},
		},
		Timestamp: time.Now().UnixMilli(),
	}

	tests := []struct {
		name    string
		index   NameIndex
		wantErr error
	}{
		{
			name:    "valid index",
			index:   validIndex,
			wantErr: nil,
		},
		{
			name: "empty name",
			index: NameIndex{
				Name:         "",
				Protocol:     ProtocolVersion,
				IndexVersion: IndexFormatVersion,
				Publishers:   validIndex.Publishers,
				Timestamp:    validIndex.Timestamp,
			},
			wantErr: ErrEmptyName,
		},
		{
			name: "empty indexVersion",
			index: NameIndex{
				Name:         "test",
				Protocol:     ProtocolVersion,
				IndexVersion: "",
				Publishers:   validIndex.Publishers,
				Timestamp:    validIndex.Timestamp,
			},
			wantErr: ErrEmptyIndexVersion,
		},
		{
			name: "empty publishers",
			index: NameIndex{
				Name:         "test",
				Protocol:     ProtocolVersion,
				IndexVersion: IndexFormatVersion,
				Publishers:   []PublisherEntry{},
				Timestamp:    validIndex.Timestamp,
			},
			wantErr: ErrEmptyPublishers,
		},
		{
			name: "zero timestamp",
			index: NameIndex{
				Name:         "test",
				Protocol:     ProtocolVersion,
				IndexVersion: IndexFormatVersion,
				Publishers:   validIndex.Publishers,
				Timestamp:    0,
			},
			wantErr: ErrInvalidTimestamp,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.index.Validate()
			if tt.wantErr == nil {
				if err != nil {
					t.Errorf("Validate() unexpected error = %v", err)
				}
			} else {
				if err == nil {
					t.Errorf("Validate() expected error %v, got nil", tt.wantErr)
				} else if !isErrorType(err, tt.wantErr) {
					t.Errorf("Validate() error = %v, want error type %v", err, tt.wantErr)
				}
			}
		})
	}
}

// TestNameIndex_FindPublisher tests the FindPublisher helper method.
func TestNameIndex_FindPublisher(t *testing.T) {
	index := NameIndex{
		Publishers: []PublisherEntry{
			{Pubkey: "pubkey1", LatestVersion: "1.0.0"},
			{Pubkey: "pubkey2", LatestVersion: "2.0.0"},
			{Pubkey: "pubkey3", LatestVersion: "3.0.0"},
		},
	}

	tests := []struct {
		name      string
		pubkey    string
		wantFound bool
		wantIndex int
	}{
		{"found first", "pubkey1", true, 0},
		{"found middle", "pubkey2", true, 1},
		{"found last", "pubkey3", true, 2},
		{"not found", "nonexistent", false, -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry := index.FindPublisher(tt.pubkey)
			found := entry != nil
			if found != tt.wantFound {
				t.Errorf("FindPublisher() found = %v, want %v", found, tt.wantFound)
			}
			if found {
				if entry.Pubkey != tt.pubkey {
					t.Errorf("FindPublisher() returned wrong entry: got pubkey %q, want %q", entry.Pubkey, tt.pubkey)
				}
			} else if entry != nil {
				t.Errorf("FindPublisher() returned non-nil entry for not found case")
			}
		})
	}
}

// TestAnnounceVersion_Validate tests validation for AnnounceVersion.
func TestAnnounceVersion_Validate(t *testing.T) {
	validVersion := AnnounceVersion{
		Version:     "1.0.0",
		ManifestKey: "0123456789abcdef01234567",
		Timestamp:   time.Now().UnixMilli(),
	}

	tests := []struct {
		name    string
		version AnnounceVersion
		wantErr error
	}{
		{
			name:    "valid version",
			version: validVersion,
			wantErr: nil,
		},
		{
			name: "empty version",
			version: AnnounceVersion{
				Version:     "",
				ManifestKey: validVersion.ManifestKey,
			},
			wantErr: ErrEmptyVersion,
		},
		{
			name: "invalid version format",
			version: AnnounceVersion{
				Version:     "not-a-version",
				ManifestKey: validVersion.ManifestKey,
			},
			wantErr: ErrInvalidVersion,
		},
		{
			name: "empty manifestKey",
			version: AnnounceVersion{
				Version:     "1.0.0",
				ManifestKey: "",
			},
			wantErr: ErrEmptyManifestKey,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.version.Validate()
			if tt.wantErr == nil {
				if err != nil {
					t.Errorf("Validate() unexpected error = %v", err)
				}
			} else {
				if err == nil {
					t.Errorf("Validate() expected error %v, got nil", tt.wantErr)
				} else if !isErrorType(err, tt.wantErr) {
					t.Errorf("Validate() error = %v, want error type %v", err, tt.wantErr)
				}
			}
		})
	}
}

// TestAnnouncePackage_Validate tests validation for AnnouncePackage.
func TestAnnouncePackage_Validate(t *testing.T) {
	validPackage := AnnouncePackage{
		Name:          "test-package",
		LatestVersion: "1.0.0",
		Versions: []AnnounceVersion{
			{Version: "1.0.0", ManifestKey: "key1", Timestamp: time.Now().UnixMilli()},
		},
	}

	tests := []struct {
		name    string
		pkg     AnnouncePackage
		wantErr error
	}{
		{
			name:    "valid package",
			pkg:     validPackage,
			wantErr: nil,
		},
		{
			name: "empty name",
			pkg: AnnouncePackage{
				Name:     "",
				Versions: validPackage.Versions,
			},
			wantErr: ErrEmptyName,
		},
		{
			name: "empty versions",
			pkg: AnnouncePackage{
				Name:          "test",
				LatestVersion: "1.0.0",
				Versions:      []AnnounceVersion{},
			},
			wantErr: ErrEmptyVersions,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.pkg.Validate()
			if tt.wantErr == nil {
				if err != nil {
					t.Errorf("Validate() unexpected error = %v", err)
				}
			} else {
				if err == nil {
					t.Errorf("Validate() expected error %v, got nil", tt.wantErr)
				} else if !isErrorType(err, tt.wantErr) {
					t.Errorf("Validate() error = %v, want error type %v", err, tt.wantErr)
				}
			}
		})
	}
}

// TestAnnouncePackage_FindVersion tests the FindVersion helper method.
func TestAnnouncePackage_FindVersion(t *testing.T) {
	pkg := AnnouncePackage{
		Versions: []AnnounceVersion{
			{Version: "1.0.0", ManifestKey: "key1"},
			{Version: "2.0.0", ManifestKey: "key2"},
			{Version: "3.0.0", ManifestKey: "key3"},
		},
	}

	tests := []struct {
		name      string
		version   string
		wantFound bool
	}{
		{"found first", "1.0.0", true},
		{"found middle", "2.0.0", true},
		{"found last", "3.0.0", true},
		{"not found", "4.0.0", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ver := pkg.FindVersion(tt.version)
			found := ver != nil
			if found != tt.wantFound {
				t.Errorf("FindVersion() found = %v, want %v", found, tt.wantFound)
			}
			if found {
				if ver.Version != tt.version {
					t.Errorf("FindVersion() returned wrong version: got %q, want %q", ver.Version, tt.version)
				}
			} else if ver != nil {
				t.Errorf("FindVersion() returned non-nil version for not found case")
			}
		})
	}
}

// TestAnnounce_Validate tests validation for Announce.
func TestAnnounce_Validate(t *testing.T) {
	validAnnounce := Announce{
		Protocol:        ProtocolVersion,
		AnnounceVersion: AnnounceFormatVersion,
		Pubkey:          "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=",
		Packages: []AnnouncePackage{
			{
				Name:          "test-package",
				LatestVersion: "1.0.0",
				Versions: []AnnounceVersion{
					{Version: "1.0.0", ManifestKey: "key1", Timestamp: time.Now().UnixMilli()},
				},
			},
		},
		Signature: "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
		Timestamp: time.Now().UnixMilli(),
	}

	tests := []struct {
		name     string
		announce Announce
		wantErr  error
	}{
		{
			name:     "valid announce",
			announce: validAnnounce,
			wantErr:  nil,
		},
		{
			name: "empty announceVersion",
			announce: Announce{
				Protocol:        ProtocolVersion,
				AnnounceVersion: "",
				Pubkey:          validAnnounce.Pubkey,
				Packages:        validAnnounce.Packages,
				Signature:       validAnnounce.Signature,
				Timestamp:       validAnnounce.Timestamp,
			},
			wantErr: ErrEmptyAnnounceVersion,
		},
		{
			name: "empty pubkey",
			announce: Announce{
				Protocol:        ProtocolVersion,
				AnnounceVersion: AnnounceFormatVersion,
				Pubkey:          "",
				Packages:        validAnnounce.Packages,
				Signature:       validAnnounce.Signature,
				Timestamp:       validAnnounce.Timestamp,
			},
			wantErr: ErrEmptyPubkey,
		},
		{
			name: "empty packages",
			announce: Announce{
				Protocol:        ProtocolVersion,
				AnnounceVersion: AnnounceFormatVersion,
				Pubkey:          validAnnounce.Pubkey,
				Packages:        []AnnouncePackage{},
				Signature:       validAnnounce.Signature,
				Timestamp:       validAnnounce.Timestamp,
			},
			wantErr: ErrEmptyPackages,
		},
		{
			name: "empty signature",
			announce: Announce{
				Protocol:        ProtocolVersion,
				AnnounceVersion: AnnounceFormatVersion,
				Pubkey:          validAnnounce.Pubkey,
				Packages:        validAnnounce.Packages,
				Signature:       "",
				Timestamp:       validAnnounce.Timestamp,
			},
			wantErr: ErrEmptySignature,
		},
		{
			name: "zero timestamp",
			announce: Announce{
				Protocol:        ProtocolVersion,
				AnnounceVersion: AnnounceFormatVersion,
				Pubkey:          validAnnounce.Pubkey,
				Packages:        validAnnounce.Packages,
				Signature:       validAnnounce.Signature,
				Timestamp:       0,
			},
			wantErr: ErrInvalidTimestamp,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.announce.Validate()
			if tt.wantErr == nil {
				if err != nil {
					t.Errorf("Validate() unexpected error = %v", err)
				}
			} else {
				if err == nil {
					t.Errorf("Validate() expected error %v, got nil", tt.wantErr)
				} else if !isErrorType(err, tt.wantErr) {
					t.Errorf("Validate() error = %v, want error type %v", err, tt.wantErr)
				}
			}
		})
	}
}

// TestAnnounce_FindPackage tests the FindPackage helper method.
func TestAnnounce_FindPackage(t *testing.T) {
	announce := Announce{
		Packages: []AnnouncePackage{
			{Name: "package1"},
			{Name: "package2"},
			{Name: "package3"},
		},
	}

	tests := []struct {
		name      string
		pkgName   string
		wantFound bool
	}{
		{"found first", "package1", true},
		{"found middle", "package2", true},
		{"found last", "package3", true},
		{"not found", "nonexistent", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pkg := announce.FindPackage(tt.pkgName)
			found := pkg != nil
			if found != tt.wantFound {
				t.Errorf("FindPackage() found = %v, want %v", found, tt.wantFound)
			}
			if found {
				if pkg.Name != tt.pkgName {
					t.Errorf("FindPackage() returned wrong package: got %q, want %q", pkg.Name, tt.pkgName)
				}
			} else if pkg != nil {
				t.Errorf("FindPackage() returned non-nil package for not found case")
			}
		})
	}
}

// TestSeederStatus_Validate tests validation for SeederStatus.
func TestSeederStatus_Validate(t *testing.T) {
	validStatus := SeederStatus{
		Protocol:  ProtocolVersion,
		SeederID:  "seeder123",
		Pubkey:    "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=",
		Signature: "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
		Timestamp: time.Now().UnixMilli(),
		BandwidthStats: BandwidthStats{
			TotalUploadBytes:    1000,
			TotalDownloadBytes:  500,
			CurrentUploadRate:   100,
			CurrentDownloadRate: 50,
		},
	}

	tests := []struct {
		name    string
		status  SeederStatus
		wantErr error
	}{
		{
			name:    "valid status",
			status:  validStatus,
			wantErr: nil,
		},
		{
			name: "empty protocol",
			status: SeederStatus{
				Protocol:       "",
				SeederID:       "seeder123",
				Pubkey:         validStatus.Pubkey,
				Signature:      validStatus.Signature,
				Timestamp:      validStatus.Timestamp,
				BandwidthStats: validStatus.BandwidthStats,
			},
			wantErr: ErrEmptyProtocol,
		},
		{
			name: "empty seederID",
			status: SeederStatus{
				Protocol:       ProtocolVersion,
				SeederID:       "",
				Pubkey:         validStatus.Pubkey,
				Signature:      validStatus.Signature,
				Timestamp:      validStatus.Timestamp,
				BandwidthStats: validStatus.BandwidthStats,
			},
			wantErr: ErrEmptySeederID,
		},
		{
			name: "empty pubkey",
			status: SeederStatus{
				Protocol:       ProtocolVersion,
				SeederID:       "seeder123",
				Pubkey:         "",
				Signature:      validStatus.Signature,
				Timestamp:      validStatus.Timestamp,
				BandwidthStats: validStatus.BandwidthStats,
			},
			wantErr: ErrEmptyPubkey,
		},
		{
			name: "empty signature",
			status: SeederStatus{
				Protocol:       ProtocolVersion,
				SeederID:       "seeder123",
				Pubkey:         validStatus.Pubkey,
				Signature:      "",
				Timestamp:      validStatus.Timestamp,
				BandwidthStats: validStatus.BandwidthStats,
			},
			wantErr: ErrEmptySignature,
		},
		{
			name: "zero timestamp",
			status: SeederStatus{
				Protocol:       ProtocolVersion,
				SeederID:       "seeder123",
				Pubkey:         validStatus.Pubkey,
				Signature:      validStatus.Signature,
				Timestamp:      0,
				BandwidthStats: validStatus.BandwidthStats,
			},
			wantErr: ErrInvalidTimestamp,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.status.Validate()
			if tt.wantErr == nil {
				if err != nil {
					t.Errorf("Validate() unexpected error = %v", err)
				}
			} else {
				if err == nil {
					t.Errorf("Validate() expected error %v, got nil", tt.wantErr)
				} else if !isErrorType(err, tt.wantErr) {
					t.Errorf("Validate() error = %v, want error type %v", err, tt.wantErr)
				}
			}
		})
	}
}

// TestSeederStatus_SigningData tests SigningData() for SeederStatus.
func TestSeederStatus_SigningData(t *testing.T) {
	status := SeederStatus{
		Protocol:  ProtocolVersion,
		SeederID:  "seeder123",
		Pubkey:    "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=",
		Signature: "SHOULD_BE_EXCLUDED",
		Timestamp: 1234567890000,
		BandwidthStats: BandwidthStats{
			TotalUploadBytes:    1000,
			TotalDownloadBytes:  500,
			CurrentUploadRate:   100,
			CurrentDownloadRate: 50,
		},
	}

	// Test determinism
	data1, err1 := status.SigningData()
	if err1 != nil {
		t.Fatalf("SigningData() first call error = %v", err1)
	}

	data2, err2 := status.SigningData()
	if err2 != nil {
		t.Fatalf("SigningData() second call error = %v", err2)
	}

	if string(data1) != string(data2) {
		t.Errorf("SigningData() not deterministic")
	}

	// Test signature exclusion
	var unmarshaled map[string]interface{}
	if err := json.Unmarshal(data1, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal signing data: %v", err)
	}

	if _, hasSignature := unmarshaled["signature"]; hasSignature {
		t.Error("SigningData() should not include signature field")
	}
}

// TestSeederStatus_IsExpired tests expiration logic for SeederStatus.
func TestSeederStatus_IsExpired(t *testing.T) {
	now := time.Now()
	ttl := 1 * time.Hour

	tests := []struct {
		name      string
		timestamp int64
		ttl       time.Duration
		want      bool
	}{
		{
			name:      "not expired",
			timestamp: now.UnixMilli(),
			ttl:       ttl,
			want:      false,
		},
		{
			name:      "expired",
			timestamp: now.Add(-2 * time.Hour).UnixMilli(),
			ttl:       ttl,
			want:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status := SeederStatus{Timestamp: tt.timestamp}
			got := status.IsExpired(tt.ttl)
			if got != tt.want {
				t.Errorf("IsExpired() = %v, want %v", got, tt.want)
			}
		})
	}
}

// isErrorType checks if the given error wraps the expected error type.
// This helper handles both direct matches and wrapped errors.
func isErrorType(got, want error) bool {
	if got == want {
		return true
	}
	// Check if the error message contains the expected error's message
	return got != nil && want != nil && contains(got.Error(), want.Error())
}

// contains checks if a string contains a substring.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || stringContains(s, substr))
}

// stringContains is a simple substring search implementation.
func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
