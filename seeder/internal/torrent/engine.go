// Package torrent provides the BitTorrent engine for the LibreSeed seeder.
// It wraps the anacrolix/torrent library and provides a high-level API
// for managing torrents, including adding, removing, and monitoring them.
package torrent

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/anacrolix/torrent"
	"github.com/anacrolix/torrent/metainfo"
	"github.com/anacrolix/torrent/storage"
	"github.com/fulgidus/libreseed/seeder/internal/config"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

// Common errors returned by the Engine.
var (
	ErrEngineNotStarted   = errors.New("engine not started")
	ErrEngineStopped      = errors.New("engine stopped")
	ErrTorrentNotFound    = errors.New("torrent not found")
	ErrTorrentExists      = errors.New("torrent already exists")
	ErrInvalidInfoHash    = errors.New("invalid info hash")
	ErrInvalidMagnetLink  = errors.New("invalid magnet link")
	ErrInvalidTorrentFile = errors.New("invalid torrent file")
	ErrMaxTorrentsReached = errors.New("maximum active torrents reached")
)

// EngineState represents the current state of the engine.
type EngineState int

const (
	// EngineStateStopped indicates the engine is not running.
	EngineStateStopped EngineState = iota
	// EngineStateStarting indicates the engine is starting up.
	EngineStateStarting
	// EngineStateRunning indicates the engine is running and ready.
	EngineStateRunning
	// EngineStateStopping indicates the engine is shutting down.
	EngineStateStopping
)

// String returns a string representation of the engine state.
func (s EngineState) String() string {
	switch s {
	case EngineStateStopped:
		return "stopped"
	case EngineStateStarting:
		return "starting"
	case EngineStateRunning:
		return "running"
	case EngineStateStopping:
		return "stopping"
	default:
		return "unknown"
	}
}

// TorrentState represents the current state of a torrent.
type TorrentState int

const (
	// TorrentStateIdle indicates the torrent is idle (not downloading or seeding).
	TorrentStateIdle TorrentState = iota
	// TorrentStateDownloading indicates the torrent is downloading.
	TorrentStateDownloading
	// TorrentStateSeeding indicates the torrent is seeding.
	TorrentStateSeeding
	// TorrentStatePaused indicates the torrent is paused.
	TorrentStatePaused
	// TorrentStateChecking indicates the torrent is checking/verifying.
	TorrentStateChecking
	// TorrentStateError indicates the torrent encountered an error.
	TorrentStateError
)

// String returns a string representation of the torrent state.
func (s TorrentState) String() string {
	switch s {
	case TorrentStateIdle:
		return "idle"
	case TorrentStateDownloading:
		return "downloading"
	case TorrentStateSeeding:
		return "seeding"
	case TorrentStatePaused:
		return "paused"
	case TorrentStateChecking:
		return "checking"
	case TorrentStateError:
		return "error"
	default:
		return "unknown"
	}
}

// TorrentHandle wraps a torrent.Torrent with additional metadata and controls.
type TorrentHandle struct {
	// InfoHash is the unique identifier for this torrent.
	InfoHash string
	// Name is the display name of the torrent.
	Name string
	// AddedAt is the time the torrent was added to the engine.
	AddedAt time.Time
	// torrent is the underlying anacrolix/torrent.Torrent.
	torrent *torrent.Torrent
	// paused indicates if the torrent is paused.
	paused bool
	// mu protects the TorrentHandle fields.
	mu sync.RWMutex
}

// State returns the current state of the torrent.
func (h *TorrentHandle) State() TorrentState {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if h.paused {
		return TorrentStatePaused
	}

	if h.torrent == nil {
		return TorrentStateError
	}

	stats := h.torrent.Stats()

	// Check if we have all pieces
	info := h.torrent.Info()
	if info == nil {
		// Metadata not yet received
		return TorrentStateDownloading
	}

	bytesCompleted := h.torrent.BytesCompleted()
	totalLength := info.TotalLength()

	if bytesCompleted >= totalLength {
		// Fully downloaded, seeding
		if stats.ActivePeers > 0 {
			return TorrentStateSeeding
		}
		return TorrentStateIdle
	}

	// Still downloading
	if stats.ActivePeers > 0 {
		return TorrentStateDownloading
	}

	return TorrentStateIdle
}

// Stats returns statistics for this torrent.
func (h *TorrentHandle) Stats() TorrentStats {
	h.mu.RLock()
	defer h.mu.RUnlock()

	stats := TorrentStats{
		InfoHash: h.InfoHash,
		Name:     h.Name,
		AddedAt:  h.AddedAt,
		State:    h.State(),
	}

	if h.torrent == nil {
		return stats
	}

	tStats := h.torrent.Stats()
	stats.ConnectedPeers = tStats.ActivePeers
	stats.TotalPeers = tStats.TotalPeers

	// Get download/upload stats
	stats.BytesDownloaded = tStats.BytesReadUsefulData.Int64()
	stats.BytesUploaded = tStats.BytesWrittenData.Int64()

	// Get piece info
	info := h.torrent.Info()
	if info != nil {
		stats.TotalBytes = info.TotalLength()
		stats.BytesCompleted = h.torrent.BytesCompleted()
		stats.PieceLength = info.PieceLength
		stats.NumPieces = info.NumPieces()
		stats.PiecesCompleted = h.torrent.Stats().PiecesComplete
		if stats.TotalBytes > 0 {
			stats.Progress = float64(stats.BytesCompleted) / float64(stats.TotalBytes) * 100
		}
	}

	return stats
}

// Pause pauses the torrent (stops downloading/uploading).
func (h *TorrentHandle) Pause() {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.torrent != nil && !h.paused {
		h.torrent.DisallowDataDownload()
		h.torrent.DisallowDataUpload()
		h.paused = true
	}
}

// Resume resumes the torrent (allows downloading/uploading).
func (h *TorrentHandle) Resume() {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.torrent != nil && h.paused {
		h.torrent.AllowDataDownload()
		h.torrent.AllowDataUpload()
		h.paused = false
	}
}

// IsPaused returns true if the torrent is paused.
func (h *TorrentHandle) IsPaused() bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.paused
}

// TorrentStats contains statistics for a single torrent.
type TorrentStats struct {
	InfoHash        string       `json:"info_hash"`
	Name            string       `json:"name"`
	State           TorrentState `json:"state"`
	AddedAt         time.Time    `json:"added_at"`
	Progress        float64      `json:"progress"`
	BytesCompleted  int64        `json:"bytes_completed"`
	TotalBytes      int64        `json:"total_bytes"`
	BytesDownloaded int64        `json:"bytes_downloaded"`
	BytesUploaded   int64        `json:"bytes_uploaded"`
	ConnectedPeers  int          `json:"connected_peers"`
	TotalPeers      int          `json:"total_peers"`
	PieceLength     int64        `json:"piece_length"`
	NumPieces       int          `json:"num_pieces"`
	PiecesCompleted int          `json:"pieces_completed"`
}

// EngineStats contains aggregate statistics for the engine.
type EngineStats struct {
	State                EngineState `json:"state"`
	StartedAt            time.Time   `json:"started_at"`
	TotalTorrents        int         `json:"total_torrents"`
	ActiveTorrents       int         `json:"active_torrents"`
	SeedingTorrents      int         `json:"seeding_torrents"`
	DownloadingTorrents  int         `json:"downloading_torrents"`
	TotalBytesUploaded   int64       `json:"total_bytes_uploaded"`
	TotalBytesDownloaded int64       `json:"total_bytes_downloaded"`
	TotalConnectedPeers  int         `json:"total_connected_peers"`
}

// Engine is the main BitTorrent engine that manages all torrents.
type Engine struct {
	// config holds the engine configuration.
	config *config.Config
	// logger is the structured logger.
	logger *zap.Logger
	// client is the underlying anacrolix/torrent client.
	client *torrent.Client
	// state is the current engine state.
	state EngineState
	// startedAt is the time the engine was started.
	startedAt time.Time
	// torrents maps info hash to TorrentHandle.
	torrents map[string]*TorrentHandle
	// mu protects the Engine fields.
	mu sync.RWMutex
}

// NewEngine creates a new Engine with the given configuration.
func NewEngine(cfg *config.Config, logger *zap.Logger) *Engine {
	if logger == nil {
		logger = zap.NewNop()
	}

	return &Engine{
		config:   cfg,
		logger:   logger.Named("torrent-engine"),
		state:    EngineStateStopped,
		torrents: make(map[string]*TorrentHandle),
	}
}

// Start initializes and starts the torrent engine.
func (e *Engine) Start(ctx context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.state == EngineStateRunning {
		return nil
	}

	if e.state != EngineStateStopped {
		return fmt.Errorf("cannot start engine in state %s", e.state)
	}

	e.state = EngineStateStarting
	e.logger.Info("starting torrent engine")

	// Build client configuration
	clientCfg, err := e.buildClientConfig()
	if err != nil {
		e.state = EngineStateStopped
		return fmt.Errorf("failed to build client config: %w", err)
	}

	// Create the torrent client
	client, err := torrent.NewClient(clientCfg)
	if err != nil {
		e.state = EngineStateStopped
		return fmt.Errorf("failed to create torrent client: %w", err)
	}

	e.client = client
	e.state = EngineStateRunning
	e.startedAt = time.Now()

	e.logger.Info("torrent engine started",
		zap.String("bind_address", e.config.Network.BindAddress),
		zap.Int("port", e.config.Network.Port),
		zap.Bool("dht_enabled", e.config.DHT.Enabled),
	)

	return nil
}

// buildClientConfig creates the anacrolix/torrent ClientConfig from our config.
func (e *Engine) buildClientConfig() (*torrent.ClientConfig, error) {
	cfg := torrent.NewDefaultClientConfig()

	// Network settings
	cfg.ListenHost = func(network string) string {
		return e.config.Network.BindAddress
	}
	cfg.ListenPort = e.config.Network.Port

	// Disable IPv6 if not enabled
	if !e.config.Network.EnableIPv6 {
		cfg.DisableIPv6 = true
	}

	// DHT settings
	if !e.config.DHT.Enabled {
		cfg.NoDHT = true
	}

	// Storage settings
	dataDir := e.config.Storage.DataDir
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory %s: %w", dataDir, err)
	}

	// Use file storage with the configured data directory
	cfg.DefaultStorage = storage.NewFileOpts(storage.NewFileClientOpts{
		ClientBaseDir: dataDir,
	})
	cfg.DataDir = dataDir

	// Rate limiting
	if e.config.Limits.MaxUploadKBps > 0 {
		cfg.UploadRateLimiter = rate.NewLimiter(
			rate.Limit(e.config.Limits.MaxUploadKBps*1024),
			e.config.Limits.MaxUploadKBps*1024,
		)
	}
	if e.config.Limits.MaxDownloadKBps > 0 {
		cfg.DownloadRateLimiter = rate.NewLimiter(
			rate.Limit(e.config.Limits.MaxDownloadKBps*1024),
			e.config.Limits.MaxDownloadKBps*1024,
		)
	}

	// Connection limits
	if e.config.Limits.MaxConnections > 0 {
		cfg.EstablishedConnsPerTorrent = e.config.Limits.MaxConnections
		cfg.HalfOpenConnsPerTorrent = e.config.Limits.MaxConnections / 2
		cfg.TotalHalfOpenConns = e.config.Limits.MaxConnections
	}

	// Seeding behavior - we want to seed
	cfg.Seed = true
	cfg.NoUpload = false

	// Debug/logging
	cfg.Debug = false

	return cfg, nil
}

// Stop gracefully stops the torrent engine.
func (e *Engine) Stop(ctx context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.state == EngineStateStopped {
		return nil
	}

	if e.state != EngineStateRunning {
		return fmt.Errorf("cannot stop engine in state %s", e.state)
	}

	e.state = EngineStateStopping
	e.logger.Info("stopping torrent engine")

	// Close all torrents gracefully
	for _, handle := range e.torrents {
		if handle.torrent != nil {
			handle.torrent.Drop()
		}
	}

	// Clear torrents map
	e.torrents = make(map[string]*TorrentHandle)

	// Close the client
	if e.client != nil {
		errs := e.client.Close()
		if len(errs) > 0 {
			e.logger.Warn("errors while closing torrent client",
				zap.Int("error_count", len(errs)),
			)
		}
		e.client = nil
	}

	e.state = EngineStateStopped
	e.logger.Info("torrent engine stopped")

	return nil
}

// State returns the current state of the engine.
func (e *Engine) State() EngineState {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.state
}

// AddTorrentFromFile adds a torrent from a .torrent file path.
func (e *Engine) AddTorrentFromFile(ctx context.Context, filePath string) (*TorrentHandle, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.state != EngineStateRunning {
		return nil, ErrEngineNotStarted
	}

	// Check max torrents limit
	if e.config.Limits.MaxActiveTorrents > 0 && len(e.torrents) >= e.config.Limits.MaxActiveTorrents {
		return nil, ErrMaxTorrentsReached
	}

	// Load metainfo from file
	mi, err := metainfo.LoadFromFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidTorrentFile, err)
	}

	// Get info hash
	infoHash := mi.HashInfoBytes().HexString()

	// Check if already exists
	if _, exists := e.torrents[infoHash]; exists {
		return nil, ErrTorrentExists
	}

	// Add torrent to client
	t, err := e.client.AddTorrent(mi)
	if err != nil {
		return nil, fmt.Errorf("failed to add torrent: %w", err)
	}

	// Create handle
	handle := &TorrentHandle{
		InfoHash: infoHash,
		Name:     t.Name(),
		AddedAt:  time.Now(),
		torrent:  t,
	}

	e.torrents[infoHash] = handle

	e.logger.Info("added torrent from file",
		zap.String("info_hash", infoHash),
		zap.String("name", handle.Name),
		zap.String("file", filePath),
	)

	return handle, nil
}

// AddTorrentFromMagnet adds a torrent from a magnet link.
func (e *Engine) AddTorrentFromMagnet(ctx context.Context, magnetLink string) (*TorrentHandle, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.state != EngineStateRunning {
		return nil, ErrEngineNotStarted
	}

	// Check max torrents limit
	if e.config.Limits.MaxActiveTorrents > 0 && len(e.torrents) >= e.config.Limits.MaxActiveTorrents {
		return nil, ErrMaxTorrentsReached
	}

	// Parse magnet link
	spec, err := torrent.TorrentSpecFromMagnetUri(magnetLink)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidMagnetLink, err)
	}

	infoHash := spec.InfoHash.HexString()

	// Check if already exists
	if _, exists := e.torrents[infoHash]; exists {
		return nil, ErrTorrentExists
	}

	// Add torrent to client
	t, _, err := e.client.AddTorrentSpec(spec)
	if err != nil {
		return nil, fmt.Errorf("failed to add torrent: %w", err)
	}

	// Create handle
	handle := &TorrentHandle{
		InfoHash: infoHash,
		Name:     t.Name(),
		AddedAt:  time.Now(),
		torrent:  t,
	}

	// If name is empty (metadata not received yet), use info hash
	if handle.Name == "" {
		handle.Name = infoHash
	}

	e.torrents[infoHash] = handle

	e.logger.Info("added torrent from magnet",
		zap.String("info_hash", infoHash),
		zap.String("name", handle.Name),
	)

	return handle, nil
}

// AddTorrentFromInfoHash adds a torrent by its info hash (hex string).
func (e *Engine) AddTorrentFromInfoHash(ctx context.Context, infoHashHex string) (*TorrentHandle, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.state != EngineStateRunning {
		return nil, ErrEngineNotStarted
	}

	// Check max torrents limit
	if e.config.Limits.MaxActiveTorrents > 0 && len(e.torrents) >= e.config.Limits.MaxActiveTorrents {
		return nil, ErrMaxTorrentsReached
	}

	// Validate and parse info hash
	if len(infoHashHex) != 40 {
		return nil, fmt.Errorf("%w: expected 40 hex characters, got %d", ErrInvalidInfoHash, len(infoHashHex))
	}

	hashBytes, err := hex.DecodeString(infoHashHex)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidInfoHash, err)
	}

	var ih metainfo.Hash
	copy(ih[:], hashBytes)

	// Check if already exists
	if _, exists := e.torrents[infoHashHex]; exists {
		return nil, ErrTorrentExists
	}

	// Add torrent to client
	t, _ := e.client.AddTorrentInfoHash(ih)

	// Create handle
	handle := &TorrentHandle{
		InfoHash: infoHashHex,
		Name:     infoHashHex, // Will be updated when metadata is received
		AddedAt:  time.Now(),
		torrent:  t,
	}

	e.torrents[infoHashHex] = handle

	e.logger.Info("added torrent from info hash",
		zap.String("info_hash", infoHashHex),
	)

	return handle, nil
}

// RemoveTorrent removes a torrent by its info hash.
func (e *Engine) RemoveTorrent(ctx context.Context, infoHash string, deleteData bool) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.state != EngineStateRunning {
		return ErrEngineNotStarted
	}

	handle, exists := e.torrents[infoHash]
	if !exists {
		return ErrTorrentNotFound
	}

	// Drop the torrent from the client
	if handle.torrent != nil {
		handle.torrent.Drop()
	}

	// Remove from map
	delete(e.torrents, infoHash)

	e.logger.Info("removed torrent",
		zap.String("info_hash", infoHash),
		zap.String("name", handle.Name),
		zap.Bool("delete_data", deleteData),
	)

	// TODO: Optionally delete data files if deleteData is true

	return nil
}

// GetTorrent returns the torrent handle for the given info hash.
func (e *Engine) GetTorrent(infoHash string) (*TorrentHandle, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if e.state != EngineStateRunning {
		return nil, ErrEngineNotStarted
	}

	handle, exists := e.torrents[infoHash]
	if !exists {
		return nil, ErrTorrentNotFound
	}

	return handle, nil
}

// ListTorrents returns all torrent handles.
func (e *Engine) ListTorrents() []*TorrentHandle {
	e.mu.RLock()
	defer e.mu.RUnlock()

	handles := make([]*TorrentHandle, 0, len(e.torrents))
	for _, h := range e.torrents {
		handles = append(handles, h)
	}

	return handles
}

// Stats returns aggregate statistics for the engine.
func (e *Engine) Stats() EngineStats {
	e.mu.RLock()
	defer e.mu.RUnlock()

	stats := EngineStats{
		State:         e.state,
		StartedAt:     e.startedAt,
		TotalTorrents: len(e.torrents),
	}

	for _, handle := range e.torrents {
		tStats := handle.Stats()
		stats.TotalBytesUploaded += tStats.BytesUploaded
		stats.TotalBytesDownloaded += tStats.BytesDownloaded
		stats.TotalConnectedPeers += tStats.ConnectedPeers

		switch tStats.State {
		case TorrentStateSeeding:
			stats.SeedingTorrents++
			stats.ActiveTorrents++
		case TorrentStateDownloading:
			stats.DownloadingTorrents++
			stats.ActiveTorrents++
		}
	}

	return stats
}

// WaitForMetadata waits for torrent metadata to be received.
// This is useful when adding torrents by info hash or magnet link.
func (e *Engine) WaitForMetadata(ctx context.Context, infoHash string) error {
	handle, err := e.GetTorrent(infoHash)
	if err != nil {
		return err
	}

	handle.mu.RLock()
	t := handle.torrent
	handle.mu.RUnlock()

	if t == nil {
		return ErrTorrentNotFound
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.GotInfo():
		// Update name now that we have metadata
		handle.mu.Lock()
		handle.Name = t.Name()
		handle.mu.Unlock()
		return nil
	}
}

// StartDownload starts downloading a torrent (downloads all files).
func (e *Engine) StartDownload(ctx context.Context, infoHash string) error {
	handle, err := e.GetTorrent(infoHash)
	if err != nil {
		return err
	}

	handle.mu.Lock()
	defer handle.mu.Unlock()

	if handle.torrent == nil {
		return ErrTorrentNotFound
	}

	// Request all data
	handle.torrent.DownloadAll()
	handle.paused = false

	return nil
}

// VerifyTorrent verifies the torrent data integrity.
func (e *Engine) VerifyTorrent(ctx context.Context, infoHash string) error {
	handle, err := e.GetTorrent(infoHash)
	if err != nil {
		return err
	}

	handle.mu.RLock()
	t := handle.torrent
	handle.mu.RUnlock()

	if t == nil {
		return ErrTorrentNotFound
	}

	t.VerifyData()
	return nil
}
