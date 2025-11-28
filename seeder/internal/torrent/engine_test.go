// Package torrent provides the BitTorrent engine for the LibreSeed seeder.
package torrent

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/fulgidus/libreseed/seeder/internal/config"
	"go.uber.org/zap"
)

// testConfig creates a minimal test configuration with temp directories.
func testConfig(t *testing.T) *config.Config {
	t.Helper()
	dataDir := t.TempDir()

	return &config.Config{
		Server: config.ServerConfig{
			MetricsPort: 9090,
			HealthPort:  8080,
		},
		Network: config.NetworkConfig{
			BindAddress: "127.0.0.1",
			Port:        0, // Let the OS assign a port
			EnableIPv6:  false,
		},
		DHT: config.DHTConfig{
			Enabled:        false, // Disable DHT for tests
			BootstrapPeers: []string{},
		},
		Storage: config.StorageConfig{
			DataDir:      dataDir,
			MaxDiskUsage: "1GB",
			CacheSizeMB:  64,
		},
		Limits: config.LimitsConfig{
			MaxUploadKBps:     0,
			MaxDownloadKBps:   0,
			MaxConnections:    10,
			MaxActiveTorrents: 5,
		},
		Manifest: config.ManifestConfig{
			Source:          "",
			RefreshInterval: time.Hour,
		},
		Log: config.LogConfig{
			Level:  "info",
			Format: "json",
		},
	}
}

// testLogger creates a no-op logger for tests.
func testLogger() *zap.Logger {
	return zap.NewNop()
}

// =============================================================================
// EngineState Tests
// =============================================================================

func TestEngineState_String(t *testing.T) {
	tests := []struct {
		name  string
		state EngineState
		want  string
	}{
		{
			name:  "stopped state",
			state: EngineStateStopped,
			want:  "stopped",
		},
		{
			name:  "starting state",
			state: EngineStateStarting,
			want:  "starting",
		},
		{
			name:  "running state",
			state: EngineStateRunning,
			want:  "running",
		},
		{
			name:  "stopping state",
			state: EngineStateStopping,
			want:  "stopping",
		},
		{
			name:  "unknown state",
			state: EngineState(999),
			want:  "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.state.String()
			if got != tt.want {
				t.Errorf("EngineState.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

// =============================================================================
// TorrentState Tests
// =============================================================================

func TestTorrentState_String(t *testing.T) {
	tests := []struct {
		name  string
		state TorrentState
		want  string
	}{
		{
			name:  "idle state",
			state: TorrentStateIdle,
			want:  "idle",
		},
		{
			name:  "downloading state",
			state: TorrentStateDownloading,
			want:  "downloading",
		},
		{
			name:  "seeding state",
			state: TorrentStateSeeding,
			want:  "seeding",
		},
		{
			name:  "paused state",
			state: TorrentStatePaused,
			want:  "paused",
		},
		{
			name:  "checking state",
			state: TorrentStateChecking,
			want:  "checking",
		},
		{
			name:  "error state",
			state: TorrentStateError,
			want:  "error",
		},
		{
			name:  "unknown state",
			state: TorrentState(999),
			want:  "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.state.String()
			if got != tt.want {
				t.Errorf("TorrentState.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

// =============================================================================
// Engine Creation Tests
// =============================================================================

func TestNewEngine(t *testing.T) {
	t.Run("with valid config and logger", func(t *testing.T) {
		cfg := testConfig(t)
		logger := testLogger()

		engine := NewEngine(cfg, logger)

		if engine == nil {
			t.Fatal("NewEngine() returned nil")
		}
		if engine.State() != EngineStateStopped {
			t.Errorf("NewEngine() state = %v, want %v", engine.State(), EngineStateStopped)
		}
		if engine.config != cfg {
			t.Error("NewEngine() config not set correctly")
		}
	})

	t.Run("with nil logger creates nop logger", func(t *testing.T) {
		cfg := testConfig(t)

		engine := NewEngine(cfg, nil)

		if engine == nil {
			t.Fatal("NewEngine() returned nil")
		}
		if engine.logger == nil {
			t.Error("NewEngine() should create nop logger when nil is passed")
		}
	})
}

// =============================================================================
// Engine Lifecycle Tests
// =============================================================================

func TestEngine_StartStop(t *testing.T) {
	t.Run("start and stop successfully", func(t *testing.T) {
		cfg := testConfig(t)
		engine := NewEngine(cfg, testLogger())
		ctx := context.Background()

		// Start engine
		err := engine.Start(ctx)
		if err != nil {
			t.Fatalf("Start() error = %v", err)
		}

		if engine.State() != EngineStateRunning {
			t.Errorf("After Start(), state = %v, want %v", engine.State(), EngineStateRunning)
		}

		// Stop engine
		err = engine.Stop(ctx)
		if err != nil {
			t.Fatalf("Stop() error = %v", err)
		}

		if engine.State() != EngineStateStopped {
			t.Errorf("After Stop(), state = %v, want %v", engine.State(), EngineStateStopped)
		}
	})

	t.Run("start already running engine is idempotent", func(t *testing.T) {
		cfg := testConfig(t)
		engine := NewEngine(cfg, testLogger())
		ctx := context.Background()

		// Start first time
		err := engine.Start(ctx)
		if err != nil {
			t.Fatalf("First Start() error = %v", err)
		}
		defer engine.Stop(ctx)

		// Start second time should be idempotent
		err = engine.Start(ctx)
		if err != nil {
			t.Errorf("Second Start() should be idempotent, got error = %v", err)
		}

		if engine.State() != EngineStateRunning {
			t.Errorf("State should still be running, got %v", engine.State())
		}
	})

	t.Run("stop already stopped engine is idempotent", func(t *testing.T) {
		cfg := testConfig(t)
		engine := NewEngine(cfg, testLogger())
		ctx := context.Background()

		// Stop without starting
		err := engine.Stop(ctx)
		if err != nil {
			t.Errorf("Stop() on stopped engine should be idempotent, got error = %v", err)
		}

		if engine.State() != EngineStateStopped {
			t.Errorf("State should still be stopped, got %v", engine.State())
		}
	})
}

func TestEngine_State(t *testing.T) {
	cfg := testConfig(t)
	engine := NewEngine(cfg, testLogger())

	if engine.State() != EngineStateStopped {
		t.Errorf("Initial state = %v, want %v", engine.State(), EngineStateStopped)
	}
}

// =============================================================================
// Engine Stats Tests
// =============================================================================

func TestEngine_Stats(t *testing.T) {
	t.Run("stats before start", func(t *testing.T) {
		cfg := testConfig(t)
		engine := NewEngine(cfg, testLogger())

		stats := engine.Stats()

		if stats.State != EngineStateStopped {
			t.Errorf("Stats().State = %v, want %v", stats.State, EngineStateStopped)
		}
		if stats.TotalTorrents != 0 {
			t.Errorf("Stats().TotalTorrents = %d, want 0", stats.TotalTorrents)
		}
	})

	t.Run("stats after start", func(t *testing.T) {
		cfg := testConfig(t)
		engine := NewEngine(cfg, testLogger())
		ctx := context.Background()

		err := engine.Start(ctx)
		if err != nil {
			t.Fatalf("Start() error = %v", err)
		}
		defer engine.Stop(ctx)

		stats := engine.Stats()

		if stats.State != EngineStateRunning {
			t.Errorf("Stats().State = %v, want %v", stats.State, EngineStateRunning)
		}
		if stats.StartedAt.IsZero() {
			t.Error("Stats().StartedAt should not be zero after starting")
		}
	})
}

// =============================================================================
// ListTorrents Tests
// =============================================================================

func TestEngine_ListTorrents(t *testing.T) {
	t.Run("empty list when no torrents", func(t *testing.T) {
		cfg := testConfig(t)
		engine := NewEngine(cfg, testLogger())
		ctx := context.Background()

		err := engine.Start(ctx)
		if err != nil {
			t.Fatalf("Start() error = %v", err)
		}
		defer engine.Stop(ctx)

		torrents := engine.ListTorrents()
		if len(torrents) != 0 {
			t.Errorf("ListTorrents() returned %d torrents, want 0", len(torrents))
		}
	})
}

// =============================================================================
// Error Cases Tests
// =============================================================================

func TestEngine_OperationsWhenNotStarted(t *testing.T) {
	cfg := testConfig(t)
	engine := NewEngine(cfg, testLogger())
	ctx := context.Background()

	t.Run("GetTorrent fails when not started", func(t *testing.T) {
		_, err := engine.GetTorrent("dummy-hash")
		if err != ErrEngineNotStarted {
			t.Errorf("GetTorrent() error = %v, want %v", err, ErrEngineNotStarted)
		}
	})

	t.Run("AddTorrentFromFile fails when not started", func(t *testing.T) {
		_, err := engine.AddTorrentFromFile(ctx, "/dummy/path.torrent")
		if err != ErrEngineNotStarted {
			t.Errorf("AddTorrentFromFile() error = %v, want %v", err, ErrEngineNotStarted)
		}
	})

	t.Run("AddTorrentFromMagnet fails when not started", func(t *testing.T) {
		_, err := engine.AddTorrentFromMagnet(ctx, "magnet:?xt=urn:btih:dummy")
		if err != ErrEngineNotStarted {
			t.Errorf("AddTorrentFromMagnet() error = %v, want %v", err, ErrEngineNotStarted)
		}
	})

	t.Run("AddTorrentFromInfoHash fails when not started", func(t *testing.T) {
		_, err := engine.AddTorrentFromInfoHash(ctx, "0000000000000000000000000000000000000000")
		if err != ErrEngineNotStarted {
			t.Errorf("AddTorrentFromInfoHash() error = %v, want %v", err, ErrEngineNotStarted)
		}
	})

	t.Run("RemoveTorrent fails when not started", func(t *testing.T) {
		err := engine.RemoveTorrent(ctx, "dummy-hash", false)
		if err != ErrEngineNotStarted {
			t.Errorf("RemoveTorrent() error = %v, want %v", err, ErrEngineNotStarted)
		}
	})
}

func TestEngine_AddTorrentFromInfoHash_Validation(t *testing.T) {
	cfg := testConfig(t)
	engine := NewEngine(cfg, testLogger())
	ctx := context.Background()

	err := engine.Start(ctx)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer engine.Stop(ctx)

	tests := []struct {
		name        string
		infoHash    string
		wantErrType error
	}{
		{
			name:        "too short info hash",
			infoHash:    "0000000000",
			wantErrType: ErrInvalidInfoHash,
		},
		{
			name:        "too long info hash",
			infoHash:    "00000000000000000000000000000000000000000000000000",
			wantErrType: ErrInvalidInfoHash,
		},
		{
			name:        "invalid hex characters",
			infoHash:    "zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz",
			wantErrType: ErrInvalidInfoHash,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := engine.AddTorrentFromInfoHash(ctx, tt.infoHash)
			if err == nil {
				t.Error("AddTorrentFromInfoHash() expected error, got nil")
				return
			}
			// Check that the error wraps or is the expected error type
			if err.Error()[:len(tt.wantErrType.Error())] != tt.wantErrType.Error() {
				t.Errorf("AddTorrentFromInfoHash() error = %v, want error containing %v", err, tt.wantErrType)
			}
		})
	}
}

func TestEngine_AddTorrentFromFile_InvalidFile(t *testing.T) {
	cfg := testConfig(t)
	engine := NewEngine(cfg, testLogger())
	ctx := context.Background()

	err := engine.Start(ctx)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer engine.Stop(ctx)

	t.Run("non-existent file", func(t *testing.T) {
		_, err := engine.AddTorrentFromFile(ctx, "/nonexistent/file.torrent")
		if err == nil {
			t.Error("AddTorrentFromFile() expected error for non-existent file, got nil")
		}
	})

	t.Run("invalid torrent file", func(t *testing.T) {
		// Create a file with invalid content
		tmpFile := filepath.Join(t.TempDir(), "invalid.torrent")
		if err := os.WriteFile(tmpFile, []byte("not a valid torrent file"), 0644); err != nil {
			t.Fatalf("Failed to create invalid torrent file: %v", err)
		}

		_, err := engine.AddTorrentFromFile(ctx, tmpFile)
		if err == nil {
			t.Error("AddTorrentFromFile() expected error for invalid file, got nil")
		}
	})
}

func TestEngine_AddTorrentFromMagnet_InvalidMagnet(t *testing.T) {
	cfg := testConfig(t)
	engine := NewEngine(cfg, testLogger())
	ctx := context.Background()

	err := engine.Start(ctx)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer engine.Stop(ctx)

	t.Run("invalid magnet link", func(t *testing.T) {
		_, err := engine.AddTorrentFromMagnet(ctx, "not-a-magnet-link")
		if err == nil {
			t.Error("AddTorrentFromMagnet() expected error for invalid magnet, got nil")
		}
	})

	t.Run("empty magnet link", func(t *testing.T) {
		_, err := engine.AddTorrentFromMagnet(ctx, "")
		if err == nil {
			t.Error("AddTorrentFromMagnet() expected error for empty magnet, got nil")
		}
	})
}

func TestEngine_GetTorrent_NotFound(t *testing.T) {
	cfg := testConfig(t)
	engine := NewEngine(cfg, testLogger())
	ctx := context.Background()

	err := engine.Start(ctx)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer engine.Stop(ctx)

	_, err = engine.GetTorrent("0000000000000000000000000000000000000000")
	if err != ErrTorrentNotFound {
		t.Errorf("GetTorrent() error = %v, want %v", err, ErrTorrentNotFound)
	}
}

func TestEngine_RemoveTorrent_NotFound(t *testing.T) {
	cfg := testConfig(t)
	engine := NewEngine(cfg, testLogger())
	ctx := context.Background()

	err := engine.Start(ctx)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer engine.Stop(ctx)

	err = engine.RemoveTorrent(ctx, "0000000000000000000000000000000000000000", false)
	if err != ErrTorrentNotFound {
		t.Errorf("RemoveTorrent() error = %v, want %v", err, ErrTorrentNotFound)
	}
}

// =============================================================================
// Max Torrents Limit Tests
// =============================================================================

func TestEngine_MaxTorrentsLimit(t *testing.T) {
	cfg := testConfig(t)
	cfg.Limits.MaxActiveTorrents = 2 // Set a low limit for testing
	engine := NewEngine(cfg, testLogger())
	ctx := context.Background()

	err := engine.Start(ctx)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer engine.Stop(ctx)

	// Add torrents up to the limit using valid 40-char hex info hashes
	infoHashes := []string{
		"0000000000000000000000000000000000000001",
		"0000000000000000000000000000000000000002",
	}

	for _, ih := range infoHashes {
		_, err := engine.AddTorrentFromInfoHash(ctx, ih)
		if err != nil {
			t.Fatalf("AddTorrentFromInfoHash(%s) error = %v", ih, err)
		}
	}

	// Third torrent should fail with max limit reached
	_, err = engine.AddTorrentFromInfoHash(ctx, "0000000000000000000000000000000000000003")
	if err != ErrMaxTorrentsReached {
		t.Errorf("AddTorrentFromInfoHash() error = %v, want %v", err, ErrMaxTorrentsReached)
	}
}

// =============================================================================
// Duplicate Torrent Tests
// =============================================================================

func TestEngine_AddDuplicateTorrent(t *testing.T) {
	cfg := testConfig(t)
	engine := NewEngine(cfg, testLogger())
	ctx := context.Background()

	err := engine.Start(ctx)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer engine.Stop(ctx)

	infoHash := "0000000000000000000000000000000000000001"

	// Add first time
	_, err = engine.AddTorrentFromInfoHash(ctx, infoHash)
	if err != nil {
		t.Fatalf("First AddTorrentFromInfoHash() error = %v", err)
	}

	// Add same torrent again
	_, err = engine.AddTorrentFromInfoHash(ctx, infoHash)
	if err != ErrTorrentExists {
		t.Errorf("Second AddTorrentFromInfoHash() error = %v, want %v", err, ErrTorrentExists)
	}
}

// =============================================================================
// TorrentHandle Tests
// =============================================================================

func TestTorrentHandle_PauseResume(t *testing.T) {
	cfg := testConfig(t)
	engine := NewEngine(cfg, testLogger())
	ctx := context.Background()

	err := engine.Start(ctx)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer engine.Stop(ctx)

	// Add a torrent
	handle, err := engine.AddTorrentFromInfoHash(ctx, "0000000000000000000000000000000000000001")
	if err != nil {
		t.Fatalf("AddTorrentFromInfoHash() error = %v", err)
	}

	// Initially not paused
	if handle.IsPaused() {
		t.Error("IsPaused() = true, want false (initial state)")
	}

	// Pause
	handle.Pause()
	if !handle.IsPaused() {
		t.Error("IsPaused() = false after Pause(), want true")
	}

	// Resume
	handle.Resume()
	if handle.IsPaused() {
		t.Error("IsPaused() = true after Resume(), want false")
	}
}

func TestTorrentHandle_Stats(t *testing.T) {
	cfg := testConfig(t)
	engine := NewEngine(cfg, testLogger())
	ctx := context.Background()

	err := engine.Start(ctx)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer engine.Stop(ctx)

	infoHash := "0000000000000000000000000000000000000001"
	handle, err := engine.AddTorrentFromInfoHash(ctx, infoHash)
	if err != nil {
		t.Fatalf("AddTorrentFromInfoHash() error = %v", err)
	}

	stats := handle.Stats()

	if stats.InfoHash != infoHash {
		t.Errorf("Stats().InfoHash = %q, want %q", stats.InfoHash, infoHash)
	}
	if stats.AddedAt.IsZero() {
		t.Error("Stats().AddedAt should not be zero")
	}
}

func TestTorrentHandle_State(t *testing.T) {
	cfg := testConfig(t)
	engine := NewEngine(cfg, testLogger())
	ctx := context.Background()

	err := engine.Start(ctx)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer engine.Stop(ctx)

	handle, err := engine.AddTorrentFromInfoHash(ctx, "0000000000000000000000000000000000000001")
	if err != nil {
		t.Fatalf("AddTorrentFromInfoHash() error = %v", err)
	}

	// When paused, state should be TorrentStatePaused
	handle.Pause()
	if handle.State() != TorrentStatePaused {
		t.Errorf("State() = %v after Pause(), want %v", handle.State(), TorrentStatePaused)
	}
}

// =============================================================================
// WaitForMetadata / StartDownload / VerifyTorrent Tests (Error Cases)
// =============================================================================

func TestEngine_WaitForMetadata_TorrentNotFound(t *testing.T) {
	cfg := testConfig(t)
	engine := NewEngine(cfg, testLogger())
	ctx := context.Background()

	err := engine.Start(ctx)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer engine.Stop(ctx)

	err = engine.WaitForMetadata(ctx, "nonexistent-hash")
	if err != ErrTorrentNotFound {
		t.Errorf("WaitForMetadata() error = %v, want %v", err, ErrTorrentNotFound)
	}
}

func TestEngine_StartDownload_TorrentNotFound(t *testing.T) {
	cfg := testConfig(t)
	engine := NewEngine(cfg, testLogger())
	ctx := context.Background()

	err := engine.Start(ctx)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer engine.Stop(ctx)

	err = engine.StartDownload(ctx, "nonexistent-hash")
	if err != ErrTorrentNotFound {
		t.Errorf("StartDownload() error = %v, want %v", err, ErrTorrentNotFound)
	}
}

func TestEngine_VerifyTorrent_TorrentNotFound(t *testing.T) {
	cfg := testConfig(t)
	engine := NewEngine(cfg, testLogger())
	ctx := context.Background()

	err := engine.Start(ctx)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer engine.Stop(ctx)

	err = engine.VerifyTorrent(ctx, "nonexistent-hash")
	if err != ErrTorrentNotFound {
		t.Errorf("VerifyTorrent() error = %v, want %v", err, ErrTorrentNotFound)
	}
}

// =============================================================================
// Integration Test: Full Workflow
// =============================================================================

func TestEngine_FullWorkflow(t *testing.T) {
	cfg := testConfig(t)
	engine := NewEngine(cfg, testLogger())
	ctx := context.Background()

	// Start
	err := engine.Start(ctx)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	// Add torrent
	infoHash := "0000000000000000000000000000000000000001"
	handle, err := engine.AddTorrentFromInfoHash(ctx, infoHash)
	if err != nil {
		t.Fatalf("AddTorrentFromInfoHash() error = %v", err)
	}

	// Verify it's in the list
	torrents := engine.ListTorrents()
	if len(torrents) != 1 {
		t.Errorf("ListTorrents() returned %d torrents, want 1", len(torrents))
	}

	// Get torrent
	retrieved, err := engine.GetTorrent(infoHash)
	if err != nil {
		t.Fatalf("GetTorrent() error = %v", err)
	}
	if retrieved.InfoHash != handle.InfoHash {
		t.Errorf("Retrieved torrent InfoHash = %q, want %q", retrieved.InfoHash, handle.InfoHash)
	}

	// Check stats
	stats := engine.Stats()
	if stats.TotalTorrents != 1 {
		t.Errorf("Stats().TotalTorrents = %d, want 1", stats.TotalTorrents)
	}

	// Pause and resume
	handle.Pause()
	if !handle.IsPaused() {
		t.Error("Torrent should be paused")
	}
	handle.Resume()
	if handle.IsPaused() {
		t.Error("Torrent should be resumed")
	}

	// Remove torrent
	err = engine.RemoveTorrent(ctx, infoHash, false)
	if err != nil {
		t.Fatalf("RemoveTorrent() error = %v", err)
	}

	// Verify it's removed
	torrents = engine.ListTorrents()
	if len(torrents) != 0 {
		t.Errorf("ListTorrents() after removal returned %d torrents, want 0", len(torrents))
	}

	// Stop
	err = engine.Stop(ctx)
	if err != nil {
		t.Fatalf("Stop() error = %v", err)
	}

	if engine.State() != EngineStateStopped {
		t.Errorf("Final state = %v, want %v", engine.State(), EngineStateStopped)
	}
}

// =============================================================================
// Concurrent Access Tests
// =============================================================================

func TestEngine_ConcurrentAccess(t *testing.T) {
	cfg := testConfig(t)
	cfg.Limits.MaxActiveTorrents = 100 // Allow more torrents for concurrent test
	engine := NewEngine(cfg, testLogger())
	ctx := context.Background()

	err := engine.Start(ctx)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer engine.Stop(ctx)

	// Launch multiple goroutines to add and query torrents concurrently
	done := make(chan bool)
	numGoroutines := 10

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer func() { done <- true }()

			// Each goroutine adds a unique torrent (id+1 to avoid zero hash)
			infoHash := infoHashFromInt(id + 1)
			_, err := engine.AddTorrentFromInfoHash(ctx, infoHash)
			if err != nil && err != ErrTorrentExists {
				t.Errorf("Goroutine %d: AddTorrentFromInfoHash() error = %v", id, err)
				return
			}

			// Query the torrent
			_, err = engine.GetTorrent(infoHash)
			if err != nil && err != ErrTorrentNotFound {
				t.Errorf("Goroutine %d: GetTorrent() error = %v", id, err)
			}

			// Get stats
			engine.Stats()

			// List torrents
			engine.ListTorrents()
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}
}

// infoHashFromInt generates a 40-character hex info hash from an integer.
func infoHashFromInt(n int) string {
	// Pad the number to create a valid 40-char hex string
	return padInfoHash(n)
}

// padInfoHash creates a valid 40-char hex info hash from an integer.
func padInfoHash(n int) string {
	base := "0000000000000000000000000000000000000000" // 40 zeros
	suffix := formatHex(n)
	return base[:40-len(suffix)] + suffix
}

// formatHex formats an int as hex string.
func formatHex(n int) string {
	return string(hexDigit(n/16)) + string(hexDigit(n%16))
}

// hexDigit returns the hex digit for values 0-15.
func hexDigit(n int) byte {
	if n < 10 {
		return '0' + byte(n)
	}
	return 'a' + byte(n-10)
}
