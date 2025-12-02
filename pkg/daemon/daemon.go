package daemon

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/anacrolix/torrent/metainfo"
	"github.com/libreseed/libreseed/pkg/crypto"
	"github.com/libreseed/libreseed/pkg/dht"
	"github.com/libreseed/libreseed/pkg/storage"
)

// Daemon represents the libreseed daemon server.
type Daemon struct {
	config *DaemonConfig
	state  *DaemonState
	stats  *DaemonStatistics

	httpServer *http.Server
	listener   net.Listener

	// DHT components
	dhtClient   *dht.Client
	announcer   *dht.Announcer
	discovery   *dht.Discovery
	peerManager *dht.PeerManager

	// Package management components
	keyManager         *crypto.KeyManager
	packageManager     *PackageManager
	maintainerRegistry *MaintainerRegistry

	// Channels for lifecycle management
	stopCh    chan struct{}
	stoppedCh chan struct{}

	mu sync.Mutex
}

// New creates a new Daemon instance with the given configuration.
func New(config *DaemonConfig) (*Daemon, error) {
	if config == nil {
		config = DefaultConfig()
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	if err := config.EnsureStorageDir(); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}

	d := &Daemon{
		config:    config,
		state:     NewDaemonState(),
		stats:     NewDaemonStatistics(),
		stopCh:    make(chan struct{}),
		stoppedCh: make(chan struct{}),
	}

	// Initialize package management components
	baseDir := filepath.Dir(config.StorageDir)
	keysDir := filepath.Join(baseDir, "keys")
	packagesDir := filepath.Join(baseDir, "packages")
	metaFile := filepath.Join(baseDir, "packages.yaml")

	// Ensure required directories exist
	if err := os.MkdirAll(packagesDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create packages directory: %w", err)
	}

	// Initialize KeyManager
	keyManager, err := crypto.NewKeyManager(keysDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create key manager: %w", err)
	}
	if err := keyManager.EnsureKeysExist(); err != nil {
		return nil, fmt.Errorf("failed to ensure keys exist: %w", err)
	}
	d.keyManager = keyManager

	// Initialize PackageManager
	packageManager := NewPackageManager(packagesDir, metaFile)
	if err := packageManager.LoadState(); err != nil {
		return nil, fmt.Errorf("failed to load package state: %w", err)
	}
	d.packageManager = packageManager

	// Initialize MaintainerRegistry
	maintainersFile := filepath.Join(baseDir, "maintainers.yaml")
	maintainerRegistry, err := NewMaintainerRegistry(maintainersFile)
	if err != nil {
		return nil, fmt.Errorf("failed to create maintainer registry: %w", err)
	}
	d.maintainerRegistry = maintainerRegistry

	// Initialize DHT components
	dhtConfig := &dht.ClientConfig{
		Port:           config.DHTPort,
		BootstrapNodes: config.DHTBootstrapNodes,
	}
	dhtClient, err := dht.NewClient(dhtConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create DHT client: %w", err)
	}
	d.dhtClient = dhtClient
	d.announcer = dht.NewAnnouncer(dhtClient, 30*time.Minute)
	d.discovery = dht.NewDiscovery(dhtClient, 15*time.Minute)
	d.peerManager = dht.NewPeerManager()

	// Setup HTTP server
	mux := http.NewServeMux()
	d.registerRoutes(mux)

	d.httpServer = &http.Server{
		Addr:         config.ListenAddr,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return d, nil
}

// Start starts the daemon and begins serving requests.
func (d *Daemon) Start() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Check if already running
	if d.state.GetStatus() == StatusRunning {
		return fmt.Errorf("daemon is already running")
	}

	d.state.SetStatus(StatusStarting)

	// Create listener
	listener, err := net.Listen("tcp", d.config.ListenAddr)
	if err != nil {
		d.state.SetStatus(StatusError)
		d.state.SetError(err)
		return fmt.Errorf("failed to listen on %s: %w", d.config.ListenAddr, err)
	}
	d.listener = listener

	// Start DHT client if enabled
	if d.config.EnableDHT {
		if err := d.dhtClient.Start(); err != nil {
			d.state.SetStatus(StatusError)
			d.state.SetError(err)
			return fmt.Errorf("failed to start DHT client: %w", err)
		}

		// Start announcer
		d.announcer.Start()

		// Populate announcer with existing packages from database
		log.Println("=== Populating announcer with existing packages ===")
		existingPackages := d.packageManager.ListPackages()
		log.Printf("Found %d packages in database to sync to announcer", len(existingPackages))
		for _, pkg := range existingPackages {
			log.Printf("Adding package to announcer: %s (%s)", pkg.Name, pkg.PackageID)

			// Convert package ID (hex string) to InfoHash
			infoHashBytes, err := hex.DecodeString(pkg.PackageID)
			if err != nil {
				log.Printf("Warning: Failed to decode package ID %s: %v", pkg.PackageID, err)
				continue
			}
			if len(infoHashBytes) < 20 {
				log.Printf("Warning: Package ID %s too short (need 20 bytes, got %d)", pkg.PackageID, len(infoHashBytes))
				continue
			}

			var infoHash metainfo.Hash
			copy(infoHash[:], infoHashBytes[:20])
			// Use package fingerprints for DHT announcement
			d.announcer.AddPackage(infoHash, pkg.Name, pkg.CreatorFingerprint, pkg.MaintainerFingerprint)
		}
		log.Println("=== Announcer population complete ===")
	}

	// Start HTTP server in background
	go func() {
		if err := d.httpServer.Serve(d.listener); err != nil && err != http.ErrServerClosed {
			d.state.SetError(err)
		}
	}()

	// Start background tasks
	go d.backgroundWorker()

	d.state.SetStatus(StatusRunning)
	return nil
}

// Stop gracefully stops the daemon.
func (d *Daemon) Stop() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	status := d.state.GetStatus()
	if status == StatusStopped || status == StatusStopping {
		return nil
	}

	d.state.SetStatus(StatusStopping)

	// Signal background workers to stop
	close(d.stopCh)

	// Stop DHT components if enabled
	if d.config.EnableDHT {
		d.announcer.Stop()
		d.dhtClient.Stop()
	}

	// Shutdown HTTP server with timeout (only if it was started)
	if d.httpServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := d.httpServer.Shutdown(ctx); err != nil {
			d.state.SetError(err)
			return fmt.Errorf("failed to shutdown HTTP server: %w", err)
		}
	}

	// Wait for background workers (only if they were started)
	// Use select with timeout to avoid hanging if Start() was never called
	select {
	case <-d.stoppedCh:
		// Background worker finished normally
	case <-time.After(100 * time.Millisecond):
		// Timeout - background worker was never started, that's ok
	}

	d.state.SetStatus(StatusStopped)
	return nil
}

// GetState returns a snapshot of the current daemon state.
func (d *Daemon) GetState() DaemonStateSnapshot {
	return d.state.Snapshot()
}

// GetStatistics returns a snapshot of the current daemon statistics.
func (d *Daemon) GetStatistics() DaemonStatisticsSnapshot {
	return d.stats.Snapshot()
}

// GetConfig returns the daemon configuration.
func (d *Daemon) GetConfig() *DaemonConfig {
	return d.config
}

// backgroundWorker runs periodic maintenance tasks.
func (d *Daemon) backgroundWorker() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	defer close(d.stoppedCh)

	for {
		select {
		case <-d.stopCh:
			return
		case <-ticker.C:
			// Periodic tasks would go here:
			// - Update peer counts
			// - Update DHT stats
			// - Calculate current transfer rates
			// - Clean up stale data
			d.performPeriodicTasks()
		}
	}
}

// performPeriodicTasks executes periodic maintenance and updates.
func (d *Daemon) performPeriodicTasks() {
	if !d.config.EnableDHT {
		return
	}

	// Update DHT node count from DHT client
	dhtStats := d.dhtClient.GetStats()
	d.state.mu.Lock()
	d.state.DHTNodes = dhtStats.NodesInRoutingTable
	d.state.mu.Unlock()

	// Update total peer count from peer manager
	peerStats := d.peerManager.GetStats()
	d.state.mu.Lock()
	d.state.TotalPeers = peerStats.TotalPeers
	d.state.mu.Unlock()

	// Remove stale peers (not connected for more than 5 minutes)
	d.peerManager.RemoveStalePeers(5 * time.Minute)

	// Clear expired entries from discovery cache
	d.discovery.ClearExpired()
}

// registerRoutes sets up HTTP API routes.
func (d *Daemon) registerRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/health", d.handleHealth)
	mux.HandleFunc("/status", d.handleStatus)
	mux.HandleFunc("/stats", d.handleStats)
	mux.HandleFunc("/shutdown", d.handleShutdown)

	// Package management endpoints
	mux.HandleFunc("POST /packages/add", d.handlePackageAdd)
	mux.HandleFunc("GET /packages/list", d.handlePackageList)
	mux.HandleFunc("DELETE /packages/remove", d.handlePackageRemove)

	// Maintainer management endpoints
	mux.HandleFunc("GET /maintainers", d.handleMaintainerList)
	mux.HandleFunc("GET /maintainers/", d.handleMaintainerGet)
	mux.HandleFunc("POST /maintainers", d.handleMaintainerRegister)
	mux.HandleFunc("POST /maintainers/activate/", d.handleMaintainerActivate)
	mux.HandleFunc("POST /maintainers/deactivate/", d.handleMaintainerDeactivate)

	// Signature management endpoints
	mux.HandleFunc("GET /signatures/pending", d.handlePendingSignatures)
	mux.HandleFunc("POST /packages/sign/", d.handlePackageSign)

	// DHT-specific endpoints (only if DHT is enabled)
	if d.config.EnableDHT {
		mux.HandleFunc("/dht/stats", d.handleDHTStats)
		mux.HandleFunc("/dht/announcements", d.handleDHTAnnouncements)
		mux.HandleFunc("/dht/peers", d.handleDHTPeers)
		mux.HandleFunc("/dht/discovery", d.handleDHTDiscovery)
	}
}

// handleHealth returns a simple health check response.
func (d *Daemon) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "ok",
	})
}

// handleStatus returns the current daemon state.
func (d *Daemon) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	state := d.state.Snapshot()

	response := map[string]interface{}{
		"status":          string(state.Status),
		"uptime_seconds":  state.Uptime.Seconds(),
		"start_time":      state.StartTime.Format(time.RFC3339),
		"active_packages": state.ActivePackages,
		"total_peers":     state.TotalPeers,
		"dht_nodes":       state.DHTNodes,
	}

	if state.LastError != nil {
		response["last_error"] = state.LastError.Error()
		response["last_error_time"] = state.LastErrorTime.Format(time.RFC3339)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleStats returns the current daemon statistics.
func (d *Daemon) handleStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	stats := d.stats.Snapshot()

	response := map[string]interface{}{
		"total_bytes_uploaded":   stats.TotalBytesUploaded,
		"total_bytes_downloaded": stats.TotalBytesDownloaded,
		"total_packages_seeded":  stats.TotalPackagesSeeded,
		"total_peers_connected":  stats.TotalPeersConnected,
		"current_upload_rate":    stats.CurrentUploadRate,
		"current_download_rate":  stats.CurrentDownloadRate,
		"peak_upload_rate":       stats.PeakUploadRate,
		"peak_download_rate":     stats.PeakDownloadRate,
		"last_update_time":       stats.LastUpdateTime.Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleShutdown initiates a graceful shutdown of the daemon.
func (d *Daemon) handleShutdown(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "shutdown initiated",
	})

	// Trigger shutdown in background to allow response to be sent
	go func() {
		time.Sleep(100 * time.Millisecond)
		d.Stop()
	}()
}

// handleDHTStats returns DHT client statistics.
func (d *Daemon) handleDHTStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if !d.config.EnableDHT {
		http.Error(w, "DHT is not enabled", http.StatusServiceUnavailable)
		return
	}

	stats := d.dhtClient.GetStats()

	response := map[string]interface{}{
		"nodes_in_routing_table": stats.NodesInRoutingTable,
		"total_queries":          stats.TotalQueries,
		"total_responses":        stats.TotalResponses,
		"total_announces":        stats.TotalAnnounces,
		"total_lookups":          stats.TotalLookups,
		"last_bootstrap":         stats.LastBootstrap.Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleDHTAnnouncements returns a list of packages announced to the DHT.
func (d *Daemon) handleDHTAnnouncements(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if !d.config.EnableDHT {
		http.Error(w, "DHT is not enabled", http.StatusServiceUnavailable)
		return
	}

	packages := d.announcer.GetPackages()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(packages)
}

// handleDHTPeers returns information about discovered peers.
func (d *Daemon) handleDHTPeers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if !d.config.EnableDHT {
		http.Error(w, "DHT is not enabled", http.StatusServiceUnavailable)
		return
	}

	peers := d.peerManager.GetAllPeers()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(peers)
}

// handleDHTDiscovery returns the current DHT discovery cache contents.
func (d *Daemon) handleDHTDiscovery(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if !d.config.EnableDHT {
		http.Error(w, "DHT is not enabled", http.StatusServiceUnavailable)
		return
	}

	// Get cache contents and statistics
	results := d.discovery.GetAllResults()
	stats := d.discovery.GetStats()

	response := map[string]interface{}{
		"cache_results": results,
		"cache_size":    d.discovery.GetCacheSize(),
		"stats":         stats,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// LoadConfig loads daemon configuration from a YAML file.
func LoadConfig(path string) (*DaemonConfig, error) {
	config := &DaemonConfig{}
	if err := storage.LoadYAMLFile(path, config); err != nil {
		return nil, fmt.Errorf("failed to load config from %s: %w", path, err)
	}
	return config, nil
}

// SaveConfig saves daemon configuration to a YAML file.
func SaveConfig(path string, config *DaemonConfig) error {
	if err := storage.SaveYAMLFile(path, config); err != nil {
		return fmt.Errorf("failed to save config to %s: %w", path, err)
	}
	return nil
}
