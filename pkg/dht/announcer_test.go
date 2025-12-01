package dht

import (
	"fmt"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/anacrolix/torrent/metainfo"
)

// mockDHTClient implements the DHTClient interface for testing
type mockDHTClient struct {
	mu              sync.RWMutex
	started         bool
	announceFunc    func(infoHash [20]byte, port int) error
	getPeersFunc    func(infoHash [20]byte) ([]net.Addr, error)
	stats           ClientStats
	nodeID          [20]byte
	announceCount   int
	announcedHashes map[[20]byte]int
}

func newMockDHTClient() *mockDHTClient {
	return &mockDHTClient{
		started:         false,
		nodeID:          [20]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20},
		announcedHashes: make(map[[20]byte]int),
	}
}

func (m *mockDHTClient) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.started {
		return fmt.Errorf("already started")
	}
	m.started = true
	return nil
}

func (m *mockDHTClient) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.started = false
	return nil
}

func (m *mockDHTClient) Announce(infoHash [20]byte, port int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.started {
		return fmt.Errorf("DHT client not started")
	}

	m.announceCount++
	m.announcedHashes[infoHash]++
	m.stats.TotalAnnounces++

	if m.announceFunc != nil {
		return m.announceFunc(infoHash, port)
	}
	return nil
}

func (m *mockDHTClient) GetPeers(infoHash [20]byte) ([]net.Addr, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.started {
		return nil, fmt.Errorf("DHT client not started")
	}

	m.stats.TotalLookups++

	if m.getPeersFunc != nil {
		return m.getPeersFunc(infoHash)
	}

	// Return empty peer list by default
	return []net.Addr{}, nil
}

func (m *mockDHTClient) GetStats() ClientStats {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.stats
}

func (m *mockDHTClient) NodeID() [20]byte {
	return m.nodeID
}

func (m *mockDHTClient) IsStarted() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.started
}

func (m *mockDHTClient) getAnnounceCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.announceCount
}

func (m *mockDHTClient) getHashAnnounceCount(infoHash [20]byte) int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.announcedHashes[infoHash]
}

// Helper function to create a test InfoHash
func testInfoHash(suffix byte) metainfo.Hash {
	var hash [20]byte
	for i := range hash {
		hash[i] = suffix
	}
	return metainfo.Hash(hash)
}

// TestNewAnnouncer verifies announcer initialization
func TestNewAnnouncer(t *testing.T) {
	client := newMockDHTClient()
	interval := 100 * time.Millisecond

	announcer := NewAnnouncer(client, interval)

	if announcer == nil {
		t.Fatal("NewAnnouncer returned nil")
	}
	if announcer.client == nil {
		t.Error("Announcer client is nil")
	}
	if announcer.interval != interval {
		t.Errorf("Interval mismatch: got %v, want %v", announcer.interval, interval)
	}
	if len(announcer.packages) != 0 {
		t.Errorf("Packages map should be empty, got %d entries", len(announcer.packages))
	}
}

// TestAddPackage verifies adding packages
func TestAddPackage(t *testing.T) {
	client := newMockDHTClient()
	announcer := NewAnnouncer(client, time.Hour)

	infoHash := testInfoHash(1)
	packageName := "test-package"
	creatorFP := "creator-fingerprint"
	maintainerFP := "maintainer-fingerprint"

	announcer.AddPackage(infoHash, packageName, creatorFP, maintainerFP)

	packages := announcer.GetPackages()
	if len(packages) != 1 {
		t.Fatalf("Expected 1 package, got %d", len(packages))
	}

	pkg := packages[0]
	if pkg.PackageName != packageName {
		t.Errorf("PackageName mismatch: got %s, want %s", pkg.PackageName, packageName)
	}
	if pkg.CreatorFingerprint != creatorFP {
		t.Errorf("CreatorFingerprint mismatch: got %s, want %s", pkg.CreatorFingerprint, creatorFP)
	}
	if pkg.MaintainerFingerprint != maintainerFP {
		t.Errorf("MaintainerFingerprint mismatch: got %s, want %s", pkg.MaintainerFingerprint, maintainerFP)
	}
}

// TestAddPackageIdempotent verifies adding same package twice doesn't duplicate
func TestAddPackageIdempotent(t *testing.T) {
	client := newMockDHTClient()
	announcer := NewAnnouncer(client, time.Hour)

	infoHash := testInfoHash(1)

	announcer.AddPackage(infoHash, "pkg1", "creator1", "maintainer1")
	announcer.AddPackage(infoHash, "pkg2", "creator2", "maintainer2") // Same infoHash

	packages := announcer.GetPackages()
	if len(packages) != 1 {
		t.Errorf("Expected 1 package after duplicate add, got %d", len(packages))
	}

	// First add should be preserved
	if packages[0].PackageName != "pkg1" {
		t.Errorf("Expected first package name to be preserved, got %s", packages[0].PackageName)
	}
}

// TestRemovePackage verifies removing packages
func TestRemovePackage(t *testing.T) {
	client := newMockDHTClient()
	announcer := NewAnnouncer(client, time.Hour)

	infoHash1 := testInfoHash(1)
	infoHash2 := testInfoHash(2)

	announcer.AddPackage(infoHash1, "pkg1", "creator1", "maintainer1")
	announcer.AddPackage(infoHash2, "pkg2", "creator2", "maintainer2")

	if len(announcer.GetPackages()) != 2 {
		t.Fatal("Expected 2 packages before removal")
	}

	announcer.RemovePackage(infoHash1)

	packages := announcer.GetPackages()
	if len(packages) != 1 {
		t.Fatalf("Expected 1 package after removal, got %d", len(packages))
	}

	if packages[0].PackageName != "pkg2" {
		t.Errorf("Wrong package remained: got %s, want pkg2", packages[0].PackageName)
	}
}

// TestGetPackage verifies retrieving specific packages
func TestGetPackage(t *testing.T) {
	client := newMockDHTClient()
	announcer := NewAnnouncer(client, time.Hour)

	infoHash := testInfoHash(1)
	announcer.AddPackage(infoHash, "test-pkg", "creator", "maintainer")

	pkg, exists := announcer.GetPackage(infoHash)
	if !exists {
		t.Fatal("Package should exist")
	}
	if pkg.PackageName != "test-pkg" {
		t.Errorf("PackageName mismatch: got %s, want test-pkg", pkg.PackageName)
	}

	// Non-existent package
	nonExistent := testInfoHash(99)
	_, exists = announcer.GetPackage(nonExistent)
	if exists {
		t.Error("Non-existent package should not exist")
	}
}

// TestAnnouncerStartStop verifies lifecycle management
func TestAnnouncerStartStop(t *testing.T) {
	client := newMockDHTClient()
	client.Start() // Start the mock client
	announcer := NewAnnouncer(client, 50*time.Millisecond)

	infoHash := testInfoHash(1)
	announcer.AddPackage(infoHash, "test-pkg", "creator", "maintainer")

	// Start announcer
	announcer.Start()

	// Wait for at least one announcement cycle
	time.Sleep(150 * time.Millisecond)

	// Stop announcer
	announcer.Stop()

	// Verify announcements occurred
	if client.getAnnounceCount() == 0 {
		t.Error("Expected at least one announcement")
	}
}

// TestAnnouncerMultiplePackages verifies multiple packages are announced
func TestAnnouncerMultiplePackages(t *testing.T) {
	client := newMockDHTClient()
	client.Start()
	announcer := NewAnnouncer(client, 50*time.Millisecond)

	hash1 := testInfoHash(1)
	hash2 := testInfoHash(2)
	hash3 := testInfoHash(3)

	announcer.AddPackage(hash1, "pkg1", "creator1", "maintainer1")
	announcer.AddPackage(hash2, "pkg2", "creator2", "maintainer2")
	announcer.AddPackage(hash3, "pkg3", "creator3", "maintainer3")

	announcer.Start()
	time.Sleep(150 * time.Millisecond)
	announcer.Stop()

	// Verify all packages were announced
	if client.getHashAnnounceCount(hash1) == 0 {
		t.Error("Package 1 was not announced")
	}
	if client.getHashAnnounceCount(hash2) == 0 {
		t.Error("Package 2 was not announced")
	}
	if client.getHashAnnounceCount(hash3) == 0 {
		t.Error("Package 3 was not announced")
	}
}

// TestAnnouncerErrorHandling verifies error tracking
func TestAnnouncerErrorHandling(t *testing.T) {
	client := newMockDHTClient()
	client.Start()

	// Make announces fail
	client.announceFunc = func(infoHash [20]byte, port int) error {
		return fmt.Errorf("simulated announce failure")
	}

	announcer := NewAnnouncer(client, 50*time.Millisecond)
	infoHash := testInfoHash(1)
	announcer.AddPackage(infoHash, "test-pkg", "creator", "maintainer")

	announcer.Start()
	time.Sleep(150 * time.Millisecond)
	announcer.Stop()

	pkg, exists := announcer.GetPackage(infoHash)
	if !exists {
		t.Fatal("Package should exist")
	}
	if !pkg.Failed {
		t.Error("Package should be marked as failed")
	}
	if pkg.LastError == nil {
		t.Error("LastError should be set")
	}
}

// TestAnnouncerStats verifies statistics tracking
func TestAnnouncerStats(t *testing.T) {
	client := newMockDHTClient()
	client.Start()
	announcer := NewAnnouncer(client, 50*time.Millisecond)

	hash1 := testInfoHash(1)
	hash2 := testInfoHash(2)

	announcer.AddPackage(hash1, "pkg1", "creator1", "maintainer1")
	announcer.AddPackage(hash2, "pkg2", "creator2", "maintainer2")

	announcer.Start()
	time.Sleep(150 * time.Millisecond)
	announcer.Stop()

	stats := announcer.GetStats()

	if stats.TotalPackages != 2 {
		t.Errorf("TotalPackages: got %d, want 2", stats.TotalPackages)
	}
	if stats.TotalAnnounces == 0 {
		t.Error("TotalAnnounces should be > 0")
	}
	if stats.LastAnnounce.IsZero() {
		t.Error("LastAnnounce should be set")
	}
}

// TestAnnouncerConcurrency verifies thread safety
func TestAnnouncerConcurrency(t *testing.T) {
	client := newMockDHTClient()
	client.Start()
	announcer := NewAnnouncer(client, time.Hour) // Long interval to control timing

	var wg sync.WaitGroup
	concurrency := 10

	// Concurrent adds
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			hash := testInfoHash(byte(index))
			announcer.AddPackage(hash, fmt.Sprintf("pkg%d", index), "creator", "maintainer")
		}(i)
	}

	wg.Wait()

	packages := announcer.GetPackages()
	if len(packages) != concurrency {
		t.Errorf("Expected %d packages, got %d", concurrency, len(packages))
	}

	// Concurrent reads
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = announcer.GetPackages()
			_ = announcer.GetStats()
		}()
	}

	wg.Wait()
}

// TestAnnouncerImmediateAnnounce verifies immediate announcement on start
func TestAnnouncerImmediateAnnounce(t *testing.T) {
	client := newMockDHTClient()
	client.Start()
	announcer := NewAnnouncer(client, time.Hour) // Long interval

	infoHash := testInfoHash(1)
	announcer.AddPackage(infoHash, "test-pkg", "creator", "maintainer")

	announcer.Start()
	time.Sleep(100 * time.Millisecond) // Wait for immediate announcement
	announcer.Stop()

	// Should have at least one announcement (the immediate one)
	if client.getAnnounceCount() == 0 {
		t.Error("Expected immediate announcement on start")
	}
}

// TestAnnouncerPackageCount verifies package tracking
func TestAnnouncerPackageCount(t *testing.T) {
	client := newMockDHTClient()
	client.Start()
	announcer := NewAnnouncer(client, 50*time.Millisecond)

	// Start with 0
	if len(announcer.GetPackages()) != 0 {
		t.Error("Should start with 0 packages")
	}

	// Add packages
	for i := 0; i < 5; i++ {
		hash := testInfoHash(byte(i))
		announcer.AddPackage(hash, fmt.Sprintf("pkg%d", i), "creator", "maintainer")
	}

	if len(announcer.GetPackages()) != 5 {
		t.Error("Should have 5 packages")
	}

	// Remove some
	announcer.RemovePackage(testInfoHash(0))
	announcer.RemovePackage(testInfoHash(2))

	if len(announcer.GetPackages()) != 3 {
		t.Error("Should have 3 packages after removal")
	}
}

// BenchmarkAddPackage measures package addition performance
func BenchmarkAddPackage(b *testing.B) {
	client := newMockDHTClient()
	announcer := NewAnnouncer(client, time.Hour)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		hash := testInfoHash(byte(i % 256))
		announcer.AddPackage(hash, "test-pkg", "creator", "maintainer")
	}
}

// BenchmarkGetPackages measures package retrieval performance
func BenchmarkGetPackages(b *testing.B) {
	client := newMockDHTClient()
	announcer := NewAnnouncer(client, time.Hour)

	// Pre-populate
	for i := 0; i < 100; i++ {
		hash := testInfoHash(byte(i))
		announcer.AddPackage(hash, fmt.Sprintf("pkg%d", i), "creator", "maintainer")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = announcer.GetPackages()
	}
}

// BenchmarkAnnouncePackage measures announcement performance
func BenchmarkAnnouncePackage(b *testing.B) {
	client := newMockDHTClient()
	client.Start()
	announcer := NewAnnouncer(client, time.Hour)

	infoHash := testInfoHash(1)
	announcer.AddPackage(infoHash, "test-pkg", "creator", "maintainer")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		announcer.announcePackage(infoHash)
	}
}
