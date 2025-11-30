// Package dht provides DHT integration for libreseed
package dht

import (
	"net"
	"sync"
	"time"
)

// PeerInfo contains information about a discovered peer
type PeerInfo struct {
	Addr          net.Addr
	InfoHash      string
	DiscoveredAt  time.Time
	LastSeen      time.Time
	ConnectionOK  bool
	BytesDownload int64
	BytesUpload   int64
	Failed        bool
	FailCount     int
	LastError     error
}

// PeerManager manages discovered peers and their connection status
type PeerManager struct {
	mu    sync.RWMutex
	peers map[string]*PeerInfo // key: addr.String()
	stats PeerStats
}

// PeerStats contains statistics about peer management
type PeerStats struct {
	TotalPeers       int
	ConnectedPeers   int
	FailedPeers      int
	TotalBytesDown   int64
	TotalBytesUp     int64
	AverageDownSpeed float64
	AverageUpSpeed   float64
	LastConnection   time.Time
}

// NewPeerManager creates a new peer manager
func NewPeerManager() *PeerManager {
	return &PeerManager{
		peers: make(map[string]*PeerInfo),
	}
}

// AddPeer adds a peer to the manager
func (pm *PeerManager) AddPeer(addr net.Addr, infoHash string) *PeerInfo {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	key := addr.String()

	if peer, exists := pm.peers[key]; exists {
		// Update existing peer
		peer.LastSeen = time.Now()
		return peer
	}

	// Create new peer
	peer := &PeerInfo{
		Addr:         addr,
		InfoHash:     infoHash,
		DiscoveredAt: time.Now(),
		LastSeen:     time.Now(),
	}

	pm.peers[key] = peer
	pm.updateStats()

	return peer
}

// GetPeer returns information about a specific peer
func (pm *PeerManager) GetPeer(addr net.Addr) (*PeerInfo, bool) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	peer, exists := pm.peers[addr.String()]
	if !exists {
		return nil, false
	}

	// Return a copy
	peerCopy := *peer
	return &peerCopy, true
}

// GetAllPeers returns all tracked peers
func (pm *PeerManager) GetAllPeers() []*PeerInfo {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	peers := make([]*PeerInfo, 0, len(pm.peers))
	for _, peer := range pm.peers {
		peerCopy := *peer
		peers = append(peers, &peerCopy)
	}

	return peers
}

// GetPeersByInfoHash returns peers for a specific package
func (pm *PeerManager) GetPeersByInfoHash(infoHash string) []*PeerInfo {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	peers := make([]*PeerInfo, 0)
	for _, peer := range pm.peers {
		if peer.InfoHash == infoHash {
			peerCopy := *peer
			peers = append(peers, &peerCopy)
		}
	}

	return peers
}

// UpdatePeerConnection updates the connection status of a peer
func (pm *PeerManager) UpdatePeerConnection(addr net.Addr, connected bool, err error) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	key := addr.String()
	peer, exists := pm.peers[key]
	if !exists {
		return
	}

	peer.LastSeen = time.Now()
	peer.ConnectionOK = connected

	if err != nil {
		peer.Failed = true
		peer.FailCount++
		peer.LastError = err
	} else if connected {
		peer.Failed = false
		peer.LastError = nil
		pm.stats.LastConnection = time.Now()
	}

	pm.updateStats()
}

// UpdatePeerStats updates transfer statistics for a peer
func (pm *PeerManager) UpdatePeerStats(addr net.Addr, bytesDown, bytesUp int64) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	key := addr.String()
	peer, exists := pm.peers[key]
	if !exists {
		return
	}

	peer.BytesDownload += bytesDown
	peer.BytesUpload += bytesUp
	peer.LastSeen = time.Now()

	pm.updateStats()
}

// RemovePeer removes a peer from tracking
func (pm *PeerManager) RemovePeer(addr net.Addr) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	delete(pm.peers, addr.String())
	pm.updateStats()
}

// RemoveStalePeers removes peers that haven't been seen for the specified duration
func (pm *PeerManager) RemoveStalePeers(maxAge time.Duration) int {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	now := time.Now()
	removed := 0

	for key, peer := range pm.peers {
		if now.Sub(peer.LastSeen) > maxAge {
			delete(pm.peers, key)
			removed++
		}
	}

	if removed > 0 {
		pm.updateStats()
	}

	return removed
}

// RemoveFailedPeers removes peers that have failed too many times
func (pm *PeerManager) RemoveFailedPeers(maxFailures int) int {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	removed := 0

	for key, peer := range pm.peers {
		if peer.FailCount >= maxFailures {
			delete(pm.peers, key)
			removed++
		}
	}

	if removed > 0 {
		pm.updateStats()
	}

	return removed
}

// GetStats returns peer statistics
func (pm *PeerManager) GetStats() PeerStats {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	return pm.stats
}

// updateStats recalculates statistics (must be called with lock held)
func (pm *PeerManager) updateStats() {
	pm.stats.TotalPeers = len(pm.peers)
	pm.stats.ConnectedPeers = 0
	pm.stats.FailedPeers = 0
	pm.stats.TotalBytesDown = 0
	pm.stats.TotalBytesUp = 0

	for _, peer := range pm.peers {
		if peer.ConnectionOK {
			pm.stats.ConnectedPeers++
		}
		if peer.Failed {
			pm.stats.FailedPeers++
		}

		pm.stats.TotalBytesDown += peer.BytesDownload
		pm.stats.TotalBytesUp += peer.BytesUpload
	}

	// Calculate average speeds (simple calculation based on active peers)
	if pm.stats.ConnectedPeers > 0 {
		pm.stats.AverageDownSpeed = float64(pm.stats.TotalBytesDown) / float64(pm.stats.ConnectedPeers)
		pm.stats.AverageUpSpeed = float64(pm.stats.TotalBytesUp) / float64(pm.stats.ConnectedPeers)
	}
}

// GetConnectedPeers returns only peers with successful connections
func (pm *PeerManager) GetConnectedPeers() []*PeerInfo {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	peers := make([]*PeerInfo, 0)
	for _, peer := range pm.peers {
		if peer.ConnectionOK {
			peerCopy := *peer
			peers = append(peers, &peerCopy)
		}
	}

	return peers
}

// GetFailedPeers returns only peers with failed connections
func (pm *PeerManager) GetFailedPeers() []*PeerInfo {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	peers := make([]*PeerInfo, 0)
	for _, peer := range pm.peers {
		if peer.Failed {
			peerCopy := *peer
			peers = append(peers, &peerCopy)
		}
	}

	return peers
}

// Clear removes all peers
func (pm *PeerManager) Clear() {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	pm.peers = make(map[string]*PeerInfo)
	pm.updateStats()
}

// GetPeerCount returns the total number of tracked peers
func (pm *PeerManager) GetPeerCount() int {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	return len(pm.peers)
}
