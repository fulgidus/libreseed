// Package dht provides DHT (Distributed Hash Table) functionality for the LibreSeed seeder.
// This file implements the DHT manager for storing and retrieving LibreSeed protocol records.
package dht

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/anacrolix/dht/v2"
	"github.com/anacrolix/dht/v2/bep44"
	"github.com/anacrolix/dht/v2/exts/getput"
	"github.com/anacrolix/torrent/bencode"
	"github.com/anacrolix/torrent/metainfo"
)

// Manager errors.
var (
	ErrManagerNotStarted = errors.New("DHT manager not started")
	ErrManagerClosed     = errors.New("DHT manager is closed")
	ErrInvalidKey        = errors.New("invalid DHT key")
	ErrRecordNotFound    = errors.New("DHT record not found")
	ErrEncodeFailed      = errors.New("failed to encode DHT record")
	ErrDecodeFailed      = errors.New("failed to decode DHT record")
	ErrStoreFailed       = errors.New("failed to store DHT record")
)

// ReannounceInterval is the time between re-announce cycles (22 hours per spec §3.2).
const ReannounceInterval = 22 * time.Hour

// GetTimeout is the maximum time to wait for a DHT Get operation.
const GetTimeout = 30 * time.Second

// PutTimeout is the maximum time to wait for a DHT Put operation.
const PutTimeout = 30 * time.Second

// ManagerConfig contains configuration options for the DHT Manager.
type ManagerConfig struct {
	// DisableReannounce disables automatic re-announcement (useful for testing).
	DisableReannounce bool

	// Logger is an optional logger for debugging (nil = no logging).
	Logger Logger
}

// Logger interface for DHT operations.
type Logger interface {
	Debugf(format string, args ...interface{})
	Infof(format string, args ...interface{})
	Warnf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
}

// Manager manages DHT operations for LibreSeed protocol records.
type Manager struct {
	config     ManagerConfig
	server     *dht.Server
	mu         sync.RWMutex
	started    bool
	closed     bool
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	announces  map[Key]*announceState
	announceMu sync.Mutex
}

// announceState tracks the state of an announced record for re-announcement.
type announceState struct {
	key       Key
	value     interface{} // Original value (MinimalManifest, NameIndex, PublisherAnnounce, or SeederStatus)
	lastSeen  time.Time
	nextRetry time.Time
}

// NewManager creates a new DHT Manager that uses an existing DHT server from the torrent client.
// This ensures there's only one DHT instance and avoids port conflicts.
func NewManager(server *dht.Server, config ManagerConfig) (*Manager, error) {
	if server == nil {
		return nil, errors.New("DHT server cannot be nil")
	}

	ctx, cancel := context.WithCancel(context.Background())

	m := &Manager{
		config:    config,
		server:    server,
		ctx:       ctx,
		cancel:    cancel,
		announces: make(map[Key]*announceState),
	}

	return m, nil
}

// Start begins DHT management operations using the existing server.
// The DHT server must already be initialized by the torrent client.
func (m *Manager) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return ErrManagerClosed
	}

	if m.started {
		return nil
	}

	if m.server == nil {
		return errors.New("DHT server not initialized")
	}

	m.started = true

	if m.config.Logger != nil {
		m.config.Logger.Infof("DHT Manager started (using torrent client's DHT server)")
	}

	// Start re-announce loop unless disabled
	if !m.config.DisableReannounce {
		m.wg.Add(1)
		go m.reannounceLoop()
	}

	return nil
}

// Stop gracefully shuts down the DHT manager.
// Note: This does NOT close the DHT server itself, as it's owned by the torrent client.
func (m *Manager) Stop() error {
	m.mu.Lock()

	if m.closed {
		m.mu.Unlock()
		return nil
	}

	m.closed = true
	m.cancel()

	// Note: We do NOT close m.server here because it's owned by the torrent client
	// and will be closed when the client shuts down.

	m.mu.Unlock()

	// Wait for background goroutines to finish
	m.wg.Wait()

	if m.config.Logger != nil {
		m.config.Logger.Infof("DHT Manager stopped")
	}

	return nil
}

// AnnouncePackage stores a MinimalManifest in the DHT at the manifest key.
//
// The manifest key is computed as:
//
//	sha256("libreseed:manifest:" + name + "@" + version)
//
// This operation will be automatically re-announced every 22 hours.
func (m *Manager) AnnouncePackage(manifest *MinimalManifest) error {
	if err := m.checkStarted(); err != nil {
		return err
	}

	if err := manifest.Validate(); err != nil {
		return fmt.Errorf("invalid manifest: %w", err)
	}

	// Verify signature before storing
	if err := manifest.VerifySignature(); err != nil {
		return fmt.Errorf("signature verification failed: %w", err)
	}

	key := ManifestKey(manifest.Name, manifest.Version)

	if err := m.putRecord(key, manifest); err != nil {
		return fmt.Errorf("failed to announce package: %w", err)
	}

	// Track for re-announcement
	m.trackAnnounce(key, manifest)

	if m.config.Logger != nil {
		m.config.Logger.Infof("Announced package %s@%s (key: %s)", manifest.Name, manifest.Version, key.String())
	}

	return nil
}

// AnnounceNameIndex stores a NameIndex in the DHT at the name index key.
//
// The name index key is computed as:
//
//	sha256("libreseed:name-index:" + name)
//
// This operation will be automatically re-announced every 22 hours.
func (m *Manager) AnnounceNameIndex(index *NameIndex) error {
	if err := m.checkStarted(); err != nil {
		return err
	}

	if err := index.Validate(); err != nil {
		return fmt.Errorf("invalid name index: %w", err)
	}

	key := NameIndexKey(index.Name)

	if err := m.putRecord(key, index); err != nil {
		return fmt.Errorf("failed to announce name index: %w", err)
	}

	// Track for re-announcement
	m.trackAnnounce(key, index)

	if m.config.Logger != nil {
		m.config.Logger.Infof("Announced name index for %s (key: %s)", index.Name, key.String())
	}

	return nil
}

// AnnouncePublisher stores a PublisherAnnounce in the DHT at the announce key.
//
// The announce key is computed as:
//
//	sha256("libreseed:announce:" + base64(pubkey))
//
// This operation will be automatically re-announced every 22 hours.
func (m *Manager) AnnouncePublisher(announce *Announce) error {
	if err := m.checkStarted(); err != nil {
		return err
	}

	if err := announce.Validate(); err != nil {
		return fmt.Errorf("invalid publisher announce: %w", err)
	}

	// Verify signature before storing
	if err := announce.VerifySignature(); err != nil {
		return fmt.Errorf("signature verification failed: %w", err)
	}

	// Decode pubkey from base64 to compute the key
	pubkeyBytes, err := decodeBase64(announce.Pubkey)
	if err != nil {
		return fmt.Errorf("invalid pubkey: %w", err)
	}

	key := AnnounceKey(pubkeyBytes)

	if err := m.putRecord(key, announce); err != nil {
		return fmt.Errorf("failed to announce publisher: %w", err)
	}

	// Track for re-announcement
	m.trackAnnounce(key, announce)

	if m.config.Logger != nil {
		m.config.Logger.Infof("Announced publisher (key: %s)", key.String())
	}

	return nil
}

// AnnounceSeeder stores a SeederStatus in the DHT at the seeder key.
//
// The seeder key is computed as:
//
//	sha256("libreseed:seeder:" + seederID)
//
// This operation will be automatically re-announced every 22 hours.
func (m *Manager) AnnounceSeeder(status *SeederStatus) error {
	if err := m.checkStarted(); err != nil {
		return err
	}

	if err := status.Validate(); err != nil {
		return fmt.Errorf("invalid seeder status: %w", err)
	}

	// Verify signature before storing
	if err := status.VerifySignature(); err != nil {
		return fmt.Errorf("signature verification failed: %w", err)
	}

	key := SeederKey(status.SeederID)

	if err := m.putRecord(key, status); err != nil {
		return fmt.Errorf("failed to announce seeder: %w", err)
	}

	// Track for re-announcement
	m.trackAnnounce(key, status)

	if m.config.Logger != nil {
		m.config.Logger.Infof("Announced seeder %s (key: %s)", status.SeederID, key.String())
	}

	return nil
}

// GetManifest retrieves a MinimalManifest from the DHT by package name and version.
func (m *Manager) GetManifest(name, version string) (*MinimalManifest, error) {
	if err := m.checkStarted(); err != nil {
		return nil, err
	}

	key := ManifestKey(name, version)

	var manifest MinimalManifest
	if err := m.getRecord(key, &manifest); err != nil {
		return nil, fmt.Errorf("failed to get manifest: %w", err)
	}

	if err := manifest.Validate(); err != nil {
		return nil, fmt.Errorf("invalid manifest retrieved: %w", err)
	}

	// Verify signature after retrieval
	if err := manifest.VerifySignature(); err != nil {
		return nil, fmt.Errorf("signature verification failed: %w", err)
	}

	return &manifest, nil
}

// GetNameIndex retrieves a NameIndex from the DHT by package name.
func (m *Manager) GetNameIndex(name string) (*NameIndex, error) {
	if err := m.checkStarted(); err != nil {
		return nil, err
	}

	key := NameIndexKey(name)

	var index NameIndex
	if err := m.getRecord(key, &index); err != nil {
		return nil, fmt.Errorf("failed to get name index: %w", err)
	}

	if err := index.Validate(); err != nil {
		return nil, fmt.Errorf("invalid name index retrieved: %w", err)
	}

	return &index, nil
}

// GetPublisherAnnounce retrieves a PublisherAnnounce from the DHT by publisher public key.
func (m *Manager) GetPublisherAnnounce(pubkey []byte) (*Announce, error) {
	if err := m.checkStarted(); err != nil {
		return nil, err
	}

	key := AnnounceKey(pubkey)

	var announce Announce
	if err := m.getRecord(key, &announce); err != nil {
		return nil, fmt.Errorf("failed to get publisher announce: %w", err)
	}

	if err := announce.Validate(); err != nil {
		return nil, fmt.Errorf("invalid publisher announce retrieved: %w", err)
	}

	// Verify signature after retrieval
	if err := announce.VerifySignature(); err != nil {
		return nil, fmt.Errorf("signature verification failed: %w", err)
	}

	return &announce, nil
}

// GetSeederStatus retrieves a SeederStatus from the DHT by seeder ID.
func (m *Manager) GetSeederStatus(seederID string) (*SeederStatus, error) {
	if err := m.checkStarted(); err != nil {
		return nil, err
	}

	key := SeederKey(seederID)

	var status SeederStatus
	if err := m.getRecord(key, &status); err != nil {
		return nil, fmt.Errorf("failed to get seeder status: %w", err)
	}

	if err := status.Validate(); err != nil {
		return nil, fmt.Errorf("invalid seeder status retrieved: %w", err)
	}

	// Verify signature after retrieval
	if err := status.VerifySignature(); err != nil {
		return nil, fmt.Errorf("signature verification failed: %w", err)
	}

	return &status, nil
}

// putRecord encodes and stores a record in the DHT using BEP 44 distributed put.
func (m *Manager) putRecord(key Key, value interface{}) error {
	// Bencode the value
	encoded, err := bencode.Marshal(value)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrEncodeFailed, err)
	}

	// Convert Key to target (20-byte array)
	var target [20]byte
	copy(target[:], key[:])

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(m.ctx, PutTimeout)
	defer cancel()

	// Create BEP 44 Put request (immutable data)
	put := bep44.Put{
		V: encoded,
		// For immutable data, K/Sig/Seq are nil
	}

	// Store in DHT using distributed put (publishes to K closest nodes)
	_, err = getput.Put(ctx, target, m.server, nil, func(seq int64) bep44.Put {
		return put
	})
	if err != nil {
		return fmt.Errorf("%w: %v", ErrStoreFailed, err)
	}

	return nil
}

// getRecord retrieves and decodes a record from the DHT.
func (m *Manager) getRecord(key Key, dest interface{}) error {
	// Create a context with timeout
	ctx, cancel := context.WithTimeout(m.ctx, GetTimeout)
	defer cancel()

	// Convert Key to [20]byte target for getput.Get
	var target [20]byte
	copy(target[:], key[:])

	// Retrieve from DHT using getput extension
	result, stats, err := getput.Get(ctx, target, m.server, nil, nil)
	if err != nil {
		return fmt.Errorf("%w: failed after trying %d nodes with %d responses: %v",
			ErrRecordNotFound, stats.NumAddrsTried, stats.NumResponses, err)
	}

	// Bencode decode the value
	if err := bencode.Unmarshal(result.V, dest); err != nil {
		return fmt.Errorf("%w: %v", ErrDecodeFailed, err)
	}

	return nil
}

// trackAnnounce tracks an announced record for re-announcement.
func (m *Manager) trackAnnounce(key Key, value interface{}) {
	m.announceMu.Lock()
	defer m.announceMu.Unlock()

	m.announces[key] = &announceState{
		key:       key,
		value:     value,
		lastSeen:  time.Now(),
		nextRetry: time.Now().Add(ReannounceInterval),
	}
}

// reannounceLoop periodically re-announces tracked records.
func (m *Manager) reannounceLoop() {
	defer m.wg.Done()

	ticker := time.NewTicker(1 * time.Hour) // Check every hour
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			m.performReannounce()
		}
	}
}

// performReannounce re-announces records that are due for re-announcement.
func (m *Manager) performReannounce() {
	m.announceMu.Lock()
	toReannounce := make(map[Key]*announceState)
	now := time.Now()

	for key, state := range m.announces {
		if now.After(state.nextRetry) {
			toReannounce[key] = state
		}
	}
	m.announceMu.Unlock()

	for key, state := range toReannounce {
		if err := m.putRecord(key, state.value); err != nil {
			if m.config.Logger != nil {
				m.config.Logger.Warnf("Failed to re-announce key %s: %v", key.String(), err)
			}

			// Exponential backoff: retry in 1 hour
			m.announceMu.Lock()
			if s, ok := m.announces[key]; ok {
				s.nextRetry = time.Now().Add(1 * time.Hour)
			}
			m.announceMu.Unlock()
		} else {
			if m.config.Logger != nil {
				m.config.Logger.Debugf("Re-announced key %s", key.String())
			}

			// Schedule next re-announcement
			m.announceMu.Lock()
			if s, ok := m.announces[key]; ok {
				s.lastSeen = time.Now()
				s.nextRetry = time.Now().Add(ReannounceInterval)
			}
			m.announceMu.Unlock()
		}
	}
}

// checkStarted verifies the manager is started and not closed.
func (m *Manager) checkStarted() error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.closed {
		return ErrManagerClosed
	}

	if !m.started {
		return ErrManagerNotStarted
	}

	return nil
}

// keyToInfoHash converts a Key to a metainfo.Hash (InfoHash).
func keyToInfoHash(key Key) metainfo.Hash {
	var hash metainfo.Hash
	copy(hash[:], key[:])
	return hash
}

// decodeBase64 is a helper to decode base64-encoded strings.
func decodeBase64(s string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(s)
}
