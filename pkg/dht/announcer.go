// Package dht provides DHT integration for libreseed
package dht

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/anacrolix/torrent/metainfo"
)

// PackageAnnouncement represents a package that should be announced to the DHT
type PackageAnnouncement struct {
	InfoHash      metainfo.Hash
	PackageName   string
	LastAnnounced time.Time
	AnnounceCount int
	Failed        bool
	LastError     error
}

// Announcer manages periodic announcements of packages to the DHT
type Announcer struct {
	client   *Client
	mu       sync.RWMutex
	packages map[metainfo.Hash]*PackageAnnouncement
	interval time.Duration
	ctx      context.Context
	cancel   context.CancelFunc
	wg       sync.WaitGroup
}

// NewAnnouncer creates a new DHT announcer
func NewAnnouncer(client *Client, interval time.Duration) *Announcer {
	ctx, cancel := context.WithCancel(context.Background())
	return &Announcer{
		client:   client,
		packages: make(map[metainfo.Hash]*PackageAnnouncement),
		interval: interval,
		ctx:      ctx,
		cancel:   cancel,
	}
}

// Start begins the announcement worker
func (a *Announcer) Start() {
	log.Printf("=== ANNOUNCER START CALLED ===")
	a.wg.Add(1)
	go a.worker()
}

// Stop stops the announcement worker
func (a *Announcer) Stop() {
	a.cancel()
	a.wg.Wait()
}

// AddPackage adds a package to be announced
func (a *Announcer) AddPackage(infoHash metainfo.Hash, packageName string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if _, exists := a.packages[infoHash]; !exists {
		a.packages[infoHash] = &PackageAnnouncement{
			InfoHash:    infoHash,
			PackageName: packageName,
		}
	}
}

// RemovePackage removes a package from announcements
func (a *Announcer) RemovePackage(infoHash metainfo.Hash) {
	a.mu.Lock()
	defer a.mu.Unlock()

	delete(a.packages, infoHash)
}

// GetPackages returns all tracked packages
func (a *Announcer) GetPackages() []*PackageAnnouncement {
	a.mu.RLock()
	defer a.mu.RUnlock()

	packages := make([]*PackageAnnouncement, 0, len(a.packages))
	for _, pkg := range a.packages {
		// Create a copy to avoid race conditions
		pkgCopy := *pkg
		packages = append(packages, &pkgCopy)
	}
	return packages
}

// GetPackage returns a specific package announcement
func (a *Announcer) GetPackage(infoHash metainfo.Hash) (*PackageAnnouncement, bool) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	pkg, exists := a.packages[infoHash]
	if !exists {
		return nil, false
	}

	// Return a copy
	pkgCopy := *pkg
	return &pkgCopy, true
}

// worker runs the periodic announcement loop
func (a *Announcer) worker() {
	defer a.wg.Done()

	log.Printf("=== ANNOUNCER WORKER STARTED, interval=%v ===", a.interval)
	ticker := time.NewTicker(a.interval)
	defer ticker.Stop()

	// Announce immediately on startup
	log.Printf("=== ANNOUNCER: Initial announceAll() call ===")
	a.announceAll()

	for {
		select {
		case <-a.ctx.Done():
			log.Printf("=== ANNOUNCER WORKER STOPPED ===")
			return
		case <-ticker.C:
			log.Printf("=== ANNOUNCER: Periodic announceAll() triggered ===")
			a.announceAll()
		}
	}
}

// announceAll announces all packages to the DHT
func (a *Announcer) announceAll() {
	a.mu.RLock()
	packages := make([]*PackageAnnouncement, 0, len(a.packages))
	for _, pkg := range a.packages {
		packages = append(packages, pkg)
	}
	a.mu.RUnlock()

	log.Printf("=== announceAll: Found %d packages to announce ===", len(packages))

	// Announce each package (without holding the lock)
	for _, pkg := range packages {
		log.Printf("=== Announcing package: %s (InfoHash: %s) ===", pkg.PackageName, pkg.InfoHash.HexString())
		a.announcePackage(pkg.InfoHash)
	}
}

// announcePackage announces a single package to the DHT
func (a *Announcer) announcePackage(infoHash metainfo.Hash) {
	log.Printf("=== Calling client.Announce for InfoHash: %s ===", infoHash.HexString())
	err := a.client.Announce(infoHash, 6881) // Default BitTorrent port

	a.mu.Lock()
	defer a.mu.Unlock()

	pkg, exists := a.packages[infoHash]
	if !exists {
		log.Printf("=== ERROR: Package not found in map after announce! ===")
		return
	}

	pkg.LastAnnounced = time.Now()
	pkg.AnnounceCount++

	if err != nil {
		log.Printf("=== Announce FAILED: %v ===", err)
		pkg.Failed = true
		pkg.LastError = err
	} else {
		log.Printf("=== Announce SUCCESS: %s (count=%d, time=%s) ===", pkg.PackageName, pkg.AnnounceCount, pkg.LastAnnounced)
		pkg.Failed = false
		pkg.LastError = nil
	}
}

// GetStats returns statistics about announcements
func (a *Announcer) GetStats() AnnouncerStats {
	a.mu.RLock()
	defer a.mu.RUnlock()

	stats := AnnouncerStats{
		TotalPackages: len(a.packages),
	}

	for _, pkg := range a.packages {
		stats.TotalAnnounces += pkg.AnnounceCount
		if pkg.Failed {
			stats.FailedPackages++
		} else {
			stats.ActivePackages++
		}

		if !pkg.LastAnnounced.IsZero() {
			if stats.LastAnnounce.IsZero() || pkg.LastAnnounced.After(stats.LastAnnounce) {
				stats.LastAnnounce = pkg.LastAnnounced
			}
		}
	}

	return stats
}

// AnnouncerStats contains statistics about the announcer
type AnnouncerStats struct {
	TotalPackages  int
	ActivePackages int
	FailedPackages int
	TotalAnnounces int
	LastAnnounce   time.Time
}
