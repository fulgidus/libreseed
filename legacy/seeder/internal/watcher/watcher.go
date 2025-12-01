package watcher

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"go.uber.org/zap"

	"github.com/fulgidus/libreseed/seeder/internal/config"
	"github.com/fulgidus/libreseed/seeder/internal/torrent"
)

const (
	// debounceDelay is the time to wait after the last file event before processing
	debounceDelay = 2 * time.Second

	// seededSubdir is where successfully processed packages are moved
	seededSubdir = "seeded"

	// invalidSubdir is where invalid packages are moved
	invalidSubdir = "invalid"
)

// Watcher monitors a directory for new package files and automatically seeds them.
type Watcher struct {
	cfg     *config.Config
	logger  *zap.Logger
	engine  *torrent.Engine
	fsWatch *fsnotify.Watcher

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// debounceTimers tracks debounce timers for each file
	debounceTimers map[string]*time.Timer
	timerMu        sync.Mutex
}

// NewWatcher creates a new file watcher for automatic package seeding.
func NewWatcher(cfg *config.Config, logger *zap.Logger, engine *torrent.Engine) (*Watcher, error) {
	if cfg.Manifest.WatchDir == "" {
		return nil, fmt.Errorf("watch directory not configured")
	}

	if engine == nil {
		return nil, fmt.Errorf("torrent engine cannot be nil")
	}

	fsWatch, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create fsnotify watcher: %w", err)
	}

	return &Watcher{
		cfg:            cfg,
		logger:         logger,
		engine:         engine,
		fsWatch:        fsWatch,
		debounceTimers: make(map[string]*time.Timer),
	}, nil
}

// Start begins watching the configured directory for new packages.
func (w *Watcher) Start(ctx context.Context) error {
	watchDir := w.cfg.Manifest.WatchDir

	// Ensure watch directory exists
	if err := os.MkdirAll(watchDir, 0755); err != nil {
		return fmt.Errorf("failed to create watch directory: %w", err)
	}

	// Create subdirectories for processed files
	seededDir := filepath.Join(watchDir, seededSubdir)
	if err := os.MkdirAll(seededDir, 0755); err != nil {
		return fmt.Errorf("failed to create seeded directory: %w", err)
	}

	invalidDir := filepath.Join(watchDir, invalidSubdir)
	if err := os.MkdirAll(invalidDir, 0755); err != nil {
		return fmt.Errorf("failed to create invalid directory: %w", err)
	}

	// Add watch on the directory
	if err := w.fsWatch.Add(watchDir); err != nil {
		return fmt.Errorf("failed to watch directory: %w", err)
	}

	w.ctx, w.cancel = context.WithCancel(ctx)

	// Start event loop
	w.wg.Add(1)
	go w.eventLoop()

	w.logger.Info("file watcher started",
		zap.String("watch_dir", watchDir),
		zap.String("seeded_dir", seededDir),
		zap.String("invalid_dir", invalidDir),
	)

	return nil
}

// Stop gracefully stops the watcher and waits for pending operations to complete.
func (w *Watcher) Stop() error {
	if w.cancel != nil {
		w.cancel()
	}

	// Cancel all pending debounce timers
	w.timerMu.Lock()
	for _, timer := range w.debounceTimers {
		timer.Stop()
	}
	w.timerMu.Unlock()

	// Close fsnotify watcher
	if w.fsWatch != nil {
		if err := w.fsWatch.Close(); err != nil {
			w.logger.Error("failed to close fsnotify watcher", zap.Error(err))
		}
	}

	// Wait for event loop to finish
	w.wg.Wait()

	w.logger.Info("file watcher stopped")
	return nil
}

// eventLoop is the main event processing loop.
func (w *Watcher) eventLoop() {
	defer w.wg.Done()

	for {
		select {
		case <-w.ctx.Done():
			return

		case event, ok := <-w.fsWatch.Events:
			if !ok {
				return
			}
			w.handleFileEvent(event)

		case err, ok := <-w.fsWatch.Errors:
			if !ok {
				return
			}
			w.logger.Error("fsnotify error", zap.Error(err))
		}
	}
}

// handleFileEvent processes fsnotify events and triggers package processing with debouncing.
func (w *Watcher) handleFileEvent(event fsnotify.Event) {
	// Only interested in Create, Write, and Rename events
	if event.Op&(fsnotify.Create|fsnotify.Write|fsnotify.Rename) == 0 {
		return
	}

	// Only process .tar.gz or .tgz files
	if !isTarGzFile(event.Name) {
		return
	}

	// Ignore files in subdirectories (seeded/, invalid/)
	if isInSubdirectory(event.Name, w.cfg.Manifest.WatchDir) {
		return
	}

	w.logger.Debug("file event detected",
		zap.String("file", filepath.Base(event.Name)),
		zap.String("operation", event.Op.String()),
	)

	// Debounce: wait for file writes to finish
	w.scheduleProcessing(event.Name)
}

// scheduleProcessing schedules package processing after a debounce delay.
func (w *Watcher) scheduleProcessing(filePath string) {
	w.timerMu.Lock()
	defer w.timerMu.Unlock()

	// Cancel existing timer for this file
	if timer, exists := w.debounceTimers[filePath]; exists {
		timer.Stop()
	}

	// Create new timer
	w.debounceTimers[filePath] = time.AfterFunc(debounceDelay, func() {
		w.processPackage(filePath)

		// Clean up timer
		w.timerMu.Lock()
		delete(w.debounceTimers, filePath)
		w.timerMu.Unlock()
	})
}

// processPackage validates and adds a package to the torrent engine.
func (w *Watcher) processPackage(packagePath string) {
	fileName := filepath.Base(packagePath)
	logger := w.logger.With(zap.String("file", fileName))

	logger.Info("processing package")

	// Validate file still exists and is readable
	fileInfo, err := os.Stat(packagePath)
	if err != nil {
		if os.IsNotExist(err) {
			logger.Debug("file no longer exists, skipping")
			return
		}
		logger.Error("failed to stat file", zap.Error(err))
		w.moveToInvalid(packagePath, logger)
		return
	}

	if fileInfo.IsDir() {
		logger.Warn("path is a directory, expected file")
		return
	}

	// Validate it's actually a tar.gz file (basic check)
	if !isTarGzFile(packagePath) {
		logger.Warn("file does not have .tar.gz or .tgz extension")
		w.moveToInvalid(packagePath, logger)
		return
	}

	// Add package to torrent engine
	handle, err := w.engine.AddPackage(w.ctx, packagePath)
	if err != nil {
		// Check if it's because the torrent already exists
		if err == torrent.ErrTorrentExists {
			logger.Info("package already seeded, moving to seeded directory")
			w.moveToSeeded(packagePath, logger)
			return
		}

		logger.Error("failed to add package to engine", zap.Error(err))
		w.moveToInvalid(packagePath, logger)
		return
	}

	logger.Info("package successfully added to seeder",
		zap.String("info_hash", handle.InfoHash),
		zap.String("name", handle.Name),
	)

	// Move successfully processed package to seeded directory
	w.moveToSeeded(packagePath, logger)
}

// moveToSeeded moves a successfully processed package to the seeded subdirectory.
func (w *Watcher) moveToSeeded(packagePath string, logger *zap.Logger) {
	destPath := filepath.Join(w.cfg.Manifest.WatchDir, seededSubdir, filepath.Base(packagePath))

	if err := moveFile(packagePath, destPath); err != nil {
		logger.Error("failed to move package to seeded directory",
			zap.String("dest", destPath),
			zap.Error(err),
		)
		return
	}

	logger.Debug("package moved to seeded directory", zap.String("dest", destPath))
}

// moveToInvalid moves an invalid package to the invalid subdirectory.
func (w *Watcher) moveToInvalid(packagePath string, logger *zap.Logger) {
	destPath := filepath.Join(w.cfg.Manifest.WatchDir, invalidSubdir, filepath.Base(packagePath))

	if err := moveFile(packagePath, destPath); err != nil {
		logger.Error("failed to move package to invalid directory",
			zap.String("dest", destPath),
			zap.Error(err),
		)
		return
	}

	logger.Debug("package moved to invalid directory", zap.String("dest", destPath))
}

// moveFile moves a file from src to dest, handling duplicates by appending a timestamp.
func moveFile(src, dest string) error {
	// Check if destination already exists
	if _, err := os.Stat(dest); err == nil {
		// Append timestamp to avoid overwriting
		ext := filepath.Ext(dest)
		base := strings.TrimSuffix(filepath.Base(dest), ext)
		timestamp := time.Now().Format("20060102-150405")
		dest = filepath.Join(filepath.Dir(dest), fmt.Sprintf("%s-%s%s", base, timestamp, ext))
	}

	// Rename/move the file
	if err := os.Rename(src, dest); err != nil {
		return fmt.Errorf("failed to move file: %w", err)
	}

	return nil
}

// isTarGzFile checks if a file has a .tar.gz or .tgz extension.
func isTarGzFile(filePath string) bool {
	lower := strings.ToLower(filePath)
	return strings.HasSuffix(lower, ".tar.gz") || strings.HasSuffix(lower, ".tgz")
}

// isInSubdirectory checks if a file path is within a subdirectory of the watch directory.
func isInSubdirectory(filePath, watchDir string) bool {
	rel, err := filepath.Rel(watchDir, filePath)
	if err != nil {
		return false
	}

	// If the relative path contains a separator, it's in a subdirectory
	return strings.Contains(rel, string(os.PathSeparator))
}
