// Package dht provides DHT integration for libreseed
package dht

import (
	"context"
	"net"
	"sync"
	"time"

	"github.com/anacrolix/torrent/metainfo"
)

// DiscoveryResult represents the result of a package discovery query
type DiscoveryResult struct {
	InfoHash     metainfo.Hash
	PackageName  string
	Peers        []net.Addr
	DiscoveredAt time.Time
	QueryCount   int
}

// Discovery manages package discovery through the DHT
type Discovery struct {
	client    *Client
	mu        sync.RWMutex
	cache     map[metainfo.Hash]*DiscoveryResult
	cacheTTL  time.Duration
	statsLock sync.RWMutex
	stats     DiscoveryStats
}

// DiscoveryStats contains statistics about discovery operations
type DiscoveryStats struct {
	TotalQueries    int
	CacheHits       int
	CacheMisses     int
	PeersDiscovered int
	FailedQueries   int
	AveragePeers    float64
	LastQuery       time.Time
}

// NewDiscovery creates a new discovery manager
func NewDiscovery(client *Client, cacheTTL time.Duration) *Discovery {
	return &Discovery{
		client:   client,
		cache:    make(map[metainfo.Hash]*DiscoveryResult),
		cacheTTL: cacheTTL,
	}
}

// FindPeers finds peers for a package by its info hash
func (d *Discovery) FindPeers(ctx context.Context, infoHash metainfo.Hash, packageName string) ([]net.Addr, error) {
	// Check cache first
	if result := d.checkCache(infoHash); result != nil {
		d.updateStats(true, len(result.Peers), nil)
		return result.Peers, nil
	}

	// Query DHT
	peers, err := d.client.GetPeers(infoHash)
	if err != nil {
		d.updateStats(false, 0, err)
		return nil, err
	}

	// Update cache
	d.updateCache(infoHash, packageName, peers)
	d.updateStats(false, len(peers), nil)

	return peers, nil
}

// checkCache checks if a result is in cache and still valid
func (d *Discovery) checkCache(infoHash metainfo.Hash) *DiscoveryResult {
	d.mu.RLock()
	defer d.mu.RUnlock()

	result, exists := d.cache[infoHash]
	if !exists {
		return nil
	}

	// Check if cache entry is still valid
	if time.Since(result.DiscoveredAt) > d.cacheTTL {
		return nil
	}

	return result
}

// updateCache updates the cache with new discovery results
func (d *Discovery) updateCache(infoHash metainfo.Hash, packageName string, peers []net.Addr) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if result, exists := d.cache[infoHash]; exists {
		// Update existing entry
		result.Peers = peers
		result.DiscoveredAt = time.Now()
		result.QueryCount++
	} else {
		// Create new entry
		d.cache[infoHash] = &DiscoveryResult{
			InfoHash:     infoHash,
			PackageName:  packageName,
			Peers:        peers,
			DiscoveredAt: time.Now(),
			QueryCount:   1,
		}
	}
}

// GetCachedResult returns a cached result if available
func (d *Discovery) GetCachedResult(infoHash metainfo.Hash) (*DiscoveryResult, bool) {
	result := d.checkCache(infoHash)
	if result == nil {
		return nil, false
	}

	// Return a copy to avoid race conditions
	resultCopy := *result
	resultCopy.Peers = make([]net.Addr, len(result.Peers))
	copy(resultCopy.Peers, result.Peers)

	return &resultCopy, true
}

// GetAllResults returns all cached results
func (d *Discovery) GetAllResults() []*DiscoveryResult {
	d.mu.RLock()
	defer d.mu.RUnlock()

	results := make([]*DiscoveryResult, 0, len(d.cache))
	now := time.Now()

	for _, result := range d.cache {
		// Only return valid cache entries
		if now.Sub(result.DiscoveredAt) <= d.cacheTTL {
			resultCopy := *result
			resultCopy.Peers = make([]net.Addr, len(result.Peers))
			copy(resultCopy.Peers, result.Peers)
			results = append(results, &resultCopy)
		}
	}

	return results
}

// ClearCache clears all cached results
func (d *Discovery) ClearCache() {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.cache = make(map[metainfo.Hash]*DiscoveryResult)
}

// ClearExpired removes expired entries from the cache
func (d *Discovery) ClearExpired() int {
	d.mu.Lock()
	defer d.mu.Unlock()

	now := time.Now()
	removed := 0

	for hash, result := range d.cache {
		if now.Sub(result.DiscoveredAt) > d.cacheTTL {
			delete(d.cache, hash)
			removed++
		}
	}

	return removed
}

// updateStats updates discovery statistics
func (d *Discovery) updateStats(cacheHit bool, peersFound int, err error) {
	d.statsLock.Lock()
	defer d.statsLock.Unlock()

	d.stats.TotalQueries++
	d.stats.LastQuery = time.Now()

	if cacheHit {
		d.stats.CacheHits++
	} else {
		d.stats.CacheMisses++
	}

	if err != nil {
		d.stats.FailedQueries++
	} else {
		d.stats.PeersDiscovered += peersFound

		// Calculate average peers per successful query
		successfulQueries := d.stats.TotalQueries - d.stats.FailedQueries
		if successfulQueries > 0 {
			d.stats.AveragePeers = float64(d.stats.PeersDiscovered) / float64(successfulQueries)
		}
	}
}

// GetStats returns discovery statistics
func (d *Discovery) GetStats() DiscoveryStats {
	d.statsLock.RLock()
	defer d.statsLock.RUnlock()

	return d.stats
}

// GetCacheSize returns the current number of cached entries
func (d *Discovery) GetCacheSize() int {
	d.mu.RLock()
	defer d.mu.RUnlock()

	return len(d.cache)
}

// RefreshCache re-queries all cached packages to update peer lists
func (d *Discovery) RefreshCache(ctx context.Context) error {
	d.mu.RLock()
	packages := make([]struct {
		hash metainfo.Hash
		name string
	}, 0, len(d.cache))

	for hash, result := range d.cache {
		packages = append(packages, struct {
			hash metainfo.Hash
			name string
		}{hash: hash, name: result.PackageName})
	}
	d.mu.RUnlock()

	// Refresh each package
	for _, pkg := range packages {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			// Query DHT for updated peer list (ignore errors during refresh)
			peers, err := d.client.GetPeers(pkg.hash)
			if err == nil {
				d.updateCache(pkg.hash, pkg.name, peers)
			}
		}
	}

	return nil
}
