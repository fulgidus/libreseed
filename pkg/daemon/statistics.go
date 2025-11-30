package daemon

import (
	"sync"
	"time"
)

// DaemonStatistics tracks performance metrics and operational statistics.
type DaemonStatistics struct {
	mu sync.RWMutex

	// TotalBytesUploaded is the cumulative bytes uploaded since daemon start
	TotalBytesUploaded int64

	// TotalBytesDownloaded is the cumulative bytes downloaded since daemon start
	TotalBytesDownloaded int64

	// TotalPackagesSeeded is the total number of packages seeded
	TotalPackagesSeeded int

	// TotalPeersConnected is the cumulative number of peers ever connected
	TotalPeersConnected int64

	// CurrentUploadRate is the current upload speed in bytes/sec
	CurrentUploadRate int64

	// CurrentDownloadRate is the current download speed in bytes/sec
	CurrentDownloadRate int64

	// PeakUploadRate is the highest upload speed seen in bytes/sec
	PeakUploadRate int64

	// PeakDownloadRate is the highest download speed seen in bytes/sec
	PeakDownloadRate int64

	// LastUpdateTime is when statistics were last updated
	LastUpdateTime time.Time
}

// NewDaemonStatistics creates a new DaemonStatistics with zero values.
func NewDaemonStatistics() *DaemonStatistics {
	return &DaemonStatistics{
		LastUpdateTime: time.Now(),
	}
}

// AddBytesUploaded increments the total bytes uploaded.
func (s *DaemonStatistics) AddBytesUploaded(bytes int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.TotalBytesUploaded += bytes
	s.LastUpdateTime = time.Now()
}

// AddBytesDownloaded increments the total bytes downloaded.
func (s *DaemonStatistics) AddBytesDownloaded(bytes int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.TotalBytesDownloaded += bytes
	s.LastUpdateTime = time.Now()
}

// IncrementPackagesSeeded increments the packages seeded counter.
func (s *DaemonStatistics) IncrementPackagesSeeded() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.TotalPackagesSeeded++
	s.LastUpdateTime = time.Now()
}

// IncrementPeersConnected increments the total peers connected counter.
func (s *DaemonStatistics) IncrementPeersConnected() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.TotalPeersConnected++
	s.LastUpdateTime = time.Now()
}

// UpdateUploadRate updates the current upload rate and tracks peak.
func (s *DaemonStatistics) UpdateUploadRate(bytesPerSec int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.CurrentUploadRate = bytesPerSec
	if bytesPerSec > s.PeakUploadRate {
		s.PeakUploadRate = bytesPerSec
	}
	s.LastUpdateTime = time.Now()
}

// UpdateDownloadRate updates the current download rate and tracks peak.
func (s *DaemonStatistics) UpdateDownloadRate(bytesPerSec int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.CurrentDownloadRate = bytesPerSec
	if bytesPerSec > s.PeakDownloadRate {
		s.PeakDownloadRate = bytesPerSec
	}
	s.LastUpdateTime = time.Now()
}

// GetTotalBytesUploaded returns the total bytes uploaded.
func (s *DaemonStatistics) GetTotalBytesUploaded() int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.TotalBytesUploaded
}

// GetTotalBytesDownloaded returns the total bytes downloaded.
func (s *DaemonStatistics) GetTotalBytesDownloaded() int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.TotalBytesDownloaded
}

// GetTotalPackagesSeeded returns the total packages seeded.
func (s *DaemonStatistics) GetTotalPackagesSeeded() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.TotalPackagesSeeded
}

// GetTotalPeersConnected returns the total peers ever connected.
func (s *DaemonStatistics) GetTotalPeersConnected() int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.TotalPeersConnected
}

// GetCurrentRates returns the current upload and download rates.
func (s *DaemonStatistics) GetCurrentRates() (upload, download int64) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.CurrentUploadRate, s.CurrentDownloadRate
}

// GetPeakRates returns the peak upload and download rates.
func (s *DaemonStatistics) GetPeakRates() (upload, download int64) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.PeakUploadRate, s.PeakDownloadRate
}

// Snapshot returns a thread-safe copy of the current statistics.
func (s *DaemonStatistics) Snapshot() DaemonStatisticsSnapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return DaemonStatisticsSnapshot{
		TotalBytesUploaded:   s.TotalBytesUploaded,
		TotalBytesDownloaded: s.TotalBytesDownloaded,
		TotalPackagesSeeded:  s.TotalPackagesSeeded,
		TotalPeersConnected:  s.TotalPeersConnected,
		CurrentUploadRate:    s.CurrentUploadRate,
		CurrentDownloadRate:  s.CurrentDownloadRate,
		PeakUploadRate:       s.PeakUploadRate,
		PeakDownloadRate:     s.PeakDownloadRate,
		LastUpdateTime:       s.LastUpdateTime,
	}
}

// DaemonStatisticsSnapshot is an immutable snapshot of DaemonStatistics.
type DaemonStatisticsSnapshot struct {
	TotalBytesUploaded   int64
	TotalBytesDownloaded int64
	TotalPackagesSeeded  int
	TotalPeersConnected  int64
	CurrentUploadRate    int64
	CurrentDownloadRate  int64
	PeakUploadRate       int64
	PeakDownloadRate     int64
	LastUpdateTime       time.Time
}

// Reset clears all statistics (useful for testing or manual resets).
func (s *DaemonStatistics) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.TotalBytesUploaded = 0
	s.TotalBytesDownloaded = 0
	s.TotalPackagesSeeded = 0
	s.TotalPeersConnected = 0
	s.CurrentUploadRate = 0
	s.CurrentDownloadRate = 0
	s.PeakUploadRate = 0
	s.PeakDownloadRate = 0
	s.LastUpdateTime = time.Now()
}
