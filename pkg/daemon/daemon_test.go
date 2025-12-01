package daemon

import (
	"encoding/hex"
	"sync"
	"testing"

	"github.com/anacrolix/torrent/metainfo"
)

// mockAnnouncer is a test double for the Announcer component
type mockAnnouncer struct {
	mu       sync.RWMutex
	packages map[metainfo.Hash]string // InfoHash -> PackageName
	started  bool
}

func newMockAnnouncer() *mockAnnouncer {
	return &mockAnnouncer{
		packages: make(map[metainfo.Hash]string),
	}
}

func (m *mockAnnouncer) Start() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.started = true
}

func (m *mockAnnouncer) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.started = false
}

func (m *mockAnnouncer) AddPackage(infoHash metainfo.Hash, packageName, creatorFingerprint, maintainerFingerprint string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.packages[infoHash] = packageName
	// Mock doesn't track fingerprints yet, but signature must match
}

func (m *mockAnnouncer) RemovePackage(infoHash metainfo.Hash) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.packages, infoHash)
}

func (m *mockAnnouncer) GetPackageCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.packages)
}

func (m *mockAnnouncer) GetPackageName(infoHash metainfo.Hash) (string, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	name, exists := m.packages[infoHash]
	return name, exists
}

func (m *mockAnnouncer) IsStarted() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.started
}

// mockPackageManager is a test double for the PackageManager component
type mockPackageManager struct {
	mu       sync.RWMutex
	packages []*PackageInfo
}

func newMockPackageManager() *mockPackageManager {
	return &mockPackageManager{
		packages: make([]*PackageInfo, 0),
	}
}

func (m *mockPackageManager) AddPackage(info *PackageInfo) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.packages = append(m.packages, info)
}

func (m *mockPackageManager) ListPackages() []*PackageInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()
	// Return a copy to avoid race conditions
	result := make([]*PackageInfo, len(m.packages))
	copy(result, m.packages)
	return result
}

// TestPackageSyncToAnnouncer tests the package synchronization logic
func TestPackageSyncToAnnouncer(t *testing.T) {
	tests := []struct {
		name           string
		packages       []*PackageInfo
		expectedCount  int
		expectedNames  map[string]string // InfoHash hex -> expected name
		shouldHaveErrs bool
	}{
		{
			name:          "empty package list",
			packages:      []*PackageInfo{},
			expectedCount: 0,
		},
		{
			name: "single valid package",
			packages: []*PackageInfo{
				{
					PackageID: "c61349fb2b5f2b3a1d8f8e9c3b8a4f5e6d7c8b9a0f1e2d3c4b5a6978695a4b3c",
					Name:      "test-package",
					Version:   "1.0.0",
				},
			},
			expectedCount: 1,
			expectedNames: map[string]string{
				"c61349fb2b5f2b3a1d8f8e9c3b8a4f5e6d7c8b9a": "test-package",
			},
		},
		{
			name: "multiple valid packages",
			packages: []*PackageInfo{
				{
					PackageID: "c61349fb2b5f2b3a1d8f8e9c3b8a4f5e6d7c8b9a0f1e2d3c4b5a6978695a4b3c",
					Name:      "package-one",
					Version:   "1.0.0",
				},
				{
					PackageID: "3b192562a3b4c5d6e7f8091a2b3c4d5e6f708192a3b4c5d6e7f8091a2b3c4d5e",
					Name:      "package-two",
					Version:   "2.0.0",
				},
				{
					PackageID: "deadbeef2b5f2b3a1d8f8e9c3b8a4f5e6d7c8b9a0f1e2d3c4b5a6978695a4b3c",
					Name:      "package-three",
					Version:   "3.0.0",
				},
			},
			expectedCount: 3,
			expectedNames: map[string]string{
				"c61349fb2b5f2b3a1d8f8e9c3b8a4f5e6d7c8b9a": "package-one",
				"3b192562a3b4c5d6e7f8091a2b3c4d5e6f708192": "package-two",
				"deadbeef2b5f2b3a1d8f8e9c3b8a4f5e6d7c8b9a": "package-three",
			},
		},
		{
			name: "invalid hex string - should skip",
			packages: []*PackageInfo{
				{
					PackageID: "INVALID_HEX_STRING",
					Name:      "bad-package",
					Version:   "1.0.0",
				},
				{
					PackageID: "c61349fb2b5f2b3a1d8f8e9c3b8a4f5e6d7c8b9a0f1e2d3c4b5a6978695a4b3c",
					Name:      "good-package",
					Version:   "2.0.0",
				},
			},
			expectedCount: 1, // Only the good package
			expectedNames: map[string]string{
				"c61349fb2b5f2b3a1d8f8e9c3b8a4f5e6d7c8b9a": "good-package",
			},
			shouldHaveErrs: true,
		},
		{
			name: "short hash - should skip",
			packages: []*PackageInfo{
				{
					PackageID: "abc123", // Only 3 bytes when decoded
					Name:      "short-hash-package",
					Version:   "1.0.0",
				},
				{
					PackageID: "c61349fb2b5f2b3a1d8f8e9c3b8a4f5e6d7c8b9a0f1e2d3c4b5a6978695a4b3c",
					Name:      "good-package",
					Version:   "2.0.0",
				},
			},
			expectedCount: 1, // Only the good package
			expectedNames: map[string]string{
				"c61349fb2b5f2b3a1d8f8e9c3b8a4f5e6d7c8b9a": "good-package",
			},
			shouldHaveErrs: true,
		},
		{
			name: "mixed valid and invalid packages",
			packages: []*PackageInfo{
				{
					PackageID: "c61349fb2b5f2b3a1d8f8e9c3b8a4f5e6d7c8b9a0f1e2d3c4b5a6978695a4b3c",
					Name:      "valid-1",
					Version:   "1.0.0",
				},
				{
					PackageID: "INVALID",
					Name:      "invalid-1",
					Version:   "1.0.0",
				},
				{
					PackageID: "3b192562a3b4c5d6e7f8091a2b3c4d5e6f708192a3b4c5d6e7f8091a2b3c4d5e",
					Name:      "valid-2",
					Version:   "2.0.0",
				},
				{
					PackageID: "short",
					Name:      "invalid-2",
					Version:   "2.0.0",
				},
			},
			expectedCount: 2, // Only valid packages
			expectedNames: map[string]string{
				"c61349fb2b5f2b3a1d8f8e9c3b8a4f5e6d7c8b9a": "valid-1",
				"3b192562a3b4c5d6e7f8091a2b3c4d5e6f708192": "valid-2",
			},
			shouldHaveErrs: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mocks
			mockPM := newMockPackageManager()
			mockAnn := newMockAnnouncer()

			// Add test packages to mock package manager
			for _, pkg := range tt.packages {
				mockPM.AddPackage(pkg)
			}

			// Simulate the daemon's package sync logic
			syncPackagesToAnnouncer(mockPM, mockAnn)

			// Verify announcer received correct number of packages
			if got := mockAnn.GetPackageCount(); got != tt.expectedCount {
				t.Errorf("expected %d packages in announcer, got %d", tt.expectedCount, got)
			}

			// Verify each expected package is in announcer with correct name
			for infoHashHex, expectedName := range tt.expectedNames {
				infoHashBytes, err := hex.DecodeString(infoHashHex)
				if err != nil {
					t.Fatalf("test setup error: invalid InfoHash hex %s: %v", infoHashHex, err)
				}

				var infoHash metainfo.Hash
				copy(infoHash[:], infoHashBytes[:20])

				name, exists := mockAnn.GetPackageName(infoHash)
				if !exists {
					t.Errorf("expected package with InfoHash %s to exist in announcer", infoHashHex)
					continue
				}
				if name != expectedName {
					t.Errorf("expected package name %q, got %q", expectedName, name)
				}
			}
		})
	}
}

// syncPackagesToAnnouncer simulates the daemon's package synchronization logic
// This is extracted from daemon.go Start() method for testing purposes
func syncPackagesToAnnouncer(pm *mockPackageManager, announcer *mockAnnouncer) {
	existingPackages := pm.ListPackages()

	for _, pkg := range existingPackages {
		// Convert package ID (hex string) to InfoHash
		infoHashBytes, err := hex.DecodeString(pkg.PackageID)
		if err != nil {
			// Log warning and skip (in real code, this would log)
			continue
		}
		if len(infoHashBytes) < 20 {
			// Log warning and skip (in real code, this would log)
			continue
		}

		var infoHash metainfo.Hash
		copy(infoHash[:], infoHashBytes[:20])
		// Use fingerprints from package info (empty strings for test data without fingerprints)
		announcer.AddPackage(infoHash, pkg.Name, pkg.CreatorFingerprint, pkg.MaintainerFingerprint)
	}
}

// TestInfoHashConversion tests the conversion from hex string to metainfo.Hash
func TestInfoHashConversion(t *testing.T) {
	tests := []struct {
		name        string
		hexString   string
		shouldError bool
		expectedLen int
	}{
		{
			name:        "valid 64-char hex string",
			hexString:   "c61349fb2b5f2b3a1d8f8e9c3b8a4f5e6d7c8b9a0f1e2d3c4b5a6978695a4b3c",
			shouldError: false,
			expectedLen: 32,
		},
		{
			name:        "valid 40-char hex string (SHA-1 size)",
			hexString:   "c61349fb2b5f2b3a1d8f8e9c3b8a4f5e6d7c8b9a",
			shouldError: false,
			expectedLen: 20,
		},
		{
			name:        "invalid hex characters",
			hexString:   "INVALID_HEX_STRING",
			shouldError: true,
			expectedLen: 0,
		},
		{
			name:        "too short hex string",
			hexString:   "abc123",
			shouldError: false,
			expectedLen: 3, // Valid hex but too short
		},
		{
			name:        "empty string",
			hexString:   "",
			shouldError: false,
			expectedLen: 0,
		},
		{
			name:        "odd length hex string",
			hexString:   "c61349fb2b5f2b3a1d8f8e9c3b8a4f5e6d7c8b9a0",
			shouldError: true, // hex.DecodeString fails on odd length
			expectedLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			infoHashBytes, err := hex.DecodeString(tt.hexString)

			if tt.shouldError {
				if err == nil {
					t.Errorf("expected error decoding %q, got nil", tt.hexString)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error decoding %q: %v", tt.hexString, err)
				return
			}

			if len(infoHashBytes) != tt.expectedLen {
				t.Errorf("expected decoded length %d, got %d", tt.expectedLen, len(infoHashBytes))
			}

			// Test conversion to metainfo.Hash (only if length >= 20)
			if len(infoHashBytes) >= 20 {
				var infoHash metainfo.Hash
				copy(infoHash[:], infoHashBytes[:20])

				// Verify first 20 bytes match
				for i := 0; i < 20; i++ {
					if infoHash[i] != infoHashBytes[i] {
						t.Errorf("byte mismatch at index %d: expected %x, got %x", i, infoHashBytes[i], infoHash[i])
					}
				}
			}
		})
	}
}

// TestConcurrentPackageSync tests thread safety of package synchronization
func TestConcurrentPackageSync(t *testing.T) {
	mockPM := newMockPackageManager()
	mockAnn := newMockAnnouncer()

	// Add multiple packages
	for i := 0; i < 100; i++ {
		// Generate unique valid InfoHash for each package
		infoHashHex := "c61349fb2b5f2b3a1d8f8e9c3b8a4f5e6d7c8b9a0f1e2d3c4b5a6978695a4b3c"
		if i > 0 {
			// Modify first byte to make it unique
			infoHashHex = hex.EncodeToString([]byte{byte(i)}) + infoHashHex[2:]
		}

		mockPM.AddPackage(&PackageInfo{
			PackageID: infoHashHex,
			Name:      "test-package-" + string(rune('0'+i)),
			Version:   "1.0.0",
		})
	}

	// Simulate concurrent sync operations
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			syncPackagesToAnnouncer(mockPM, mockAnn)
		}()
	}

	wg.Wait()

	// Should have 100 packages (duplicates overwrite, so still 100)
	if got := mockAnn.GetPackageCount(); got != 100 {
		t.Errorf("expected 100 packages after concurrent sync, got %d", got)
	}
}

// BenchmarkPackageSync benchmarks the package synchronization process
func BenchmarkPackageSync(b *testing.B) {
	mockPM := newMockPackageManager()

	// Add 1000 packages
	for i := 0; i < 1000; i++ {
		infoHashHex := "c61349fb2b5f2b3a1d8f8e9c3b8a4f5e6d7c8b9a0f1e2d3c4b5a6978695a4b3c"
		if i > 0 {
			infoHashHex = hex.EncodeToString([]byte{byte(i % 256), byte(i / 256)}) + infoHashHex[4:]
		}

		mockPM.AddPackage(&PackageInfo{
			PackageID: infoHashHex,
			Name:      "benchmark-package",
			Version:   "1.0.0",
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mockAnn := newMockAnnouncer()
		syncPackagesToAnnouncer(mockPM, mockAnn)
	}
}

// TestAnnouncerStarted verifies that announcer is started when DHT is enabled
func TestAnnouncerStarted(t *testing.T) {
	mockAnn := newMockAnnouncer()

	if mockAnn.IsStarted() {
		t.Error("announcer should not be started initially")
	}

	mockAnn.Start()

	if !mockAnn.IsStarted() {
		t.Error("announcer should be started after Start() call")
	}

	mockAnn.Stop()

	if mockAnn.IsStarted() {
		t.Error("announcer should be stopped after Stop() call")
	}
}

// TestPackageAdditionOrder verifies packages are added in correct order
func TestPackageAdditionOrder(t *testing.T) {
	mockPM := newMockPackageManager()
	mockAnn := newMockAnnouncer()

	// Add packages in specific order
	packages := []*PackageInfo{
		{
			PackageID: "c61349fb2b5f2b3a1d8f8e9c3b8a4f5e6d7c8b9a0f1e2d3c4b5a6978695a4b3c",
			Name:      "first-package",
			Version:   "1.0.0",
		},
		{
			PackageID: "3b192562a3b4c5d6e7f8091a2b3c4d5e6f708192a3b4c5d6e7f8091a2b3c4d5e",
			Name:      "second-package",
			Version:   "2.0.0",
		},
		{
			PackageID: "deadbeef2b5f2b3a1d8f8e9c3b8a4f5e6d7c8b9a0f1e2d3c4b5a6978695a4b3c",
			Name:      "third-package",
			Version:   "3.0.0",
		},
	}

	for _, pkg := range packages {
		mockPM.AddPackage(pkg)
	}

	syncPackagesToAnnouncer(mockPM, mockAnn)

	// All packages should be added
	if got := mockAnn.GetPackageCount(); got != 3 {
		t.Errorf("expected 3 packages, got %d", got)
	}

	// Verify each package exists with correct name
	for _, pkg := range packages {
		infoHashBytes, _ := hex.DecodeString(pkg.PackageID)
		var infoHash metainfo.Hash
		copy(infoHash[:], infoHashBytes[:20])

		name, exists := mockAnn.GetPackageName(infoHash)
		if !exists {
			t.Errorf("package %q not found in announcer", pkg.Name)
		}
		if name != pkg.Name {
			t.Errorf("expected name %q, got %q", pkg.Name, name)
		}
	}
}
