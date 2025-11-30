package daemon

import (
	"sync"
	"time"
)

// DaemonState represents the current operational state of the daemon.
type DaemonState struct {
	mu sync.RWMutex

	// StartTime is when the daemon was started
	StartTime time.Time

	// Status is the current daemon status
	Status DaemonStatus

	// ActivePackages is the number of packages currently being seeded
	ActivePackages int

	// TotalPeers is the current number of connected peers
	TotalPeers int

	// DHTNodes is the number of DHT nodes currently known
	DHTNodes int

	// LastError is the most recent error encountered (if any)
	LastError error

	// LastErrorTime is when the last error occurred
	LastErrorTime time.Time
}

// DaemonStatus represents the possible daemon states.
type DaemonStatus string

const (
	// StatusStarting indicates the daemon is initializing
	StatusStarting DaemonStatus = "starting"

	// StatusRunning indicates the daemon is fully operational
	StatusRunning DaemonStatus = "running"

	// StatusStopping indicates the daemon is shutting down
	StatusStopping DaemonStatus = "stopping"

	// StatusStopped indicates the daemon has stopped
	StatusStopped DaemonStatus = "stopped"

	// StatusError indicates the daemon encountered a fatal error
	StatusError DaemonStatus = "error"
)

// NewDaemonState creates a new DaemonState with initial values.
func NewDaemonState() *DaemonState {
	return &DaemonState{
		StartTime: time.Now(),
		Status:    StatusStarting,
	}
}

// SetStatus updates the daemon status.
func (s *DaemonState) SetStatus(status DaemonStatus) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Status = status
}

// GetStatus returns the current daemon status.
func (s *DaemonState) GetStatus() DaemonStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Status
}

// SetActivePackages updates the active package count.
func (s *DaemonState) SetActivePackages(count int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ActivePackages = count
}

// GetActivePackages returns the current active package count.
func (s *DaemonState) GetActivePackages() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.ActivePackages
}

// SetTotalPeers updates the total peer count.
func (s *DaemonState) SetTotalPeers(count int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.TotalPeers = count
}

// GetTotalPeers returns the current total peer count.
func (s *DaemonState) GetTotalPeers() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.TotalPeers
}

// SetDHTNodes updates the DHT node count.
func (s *DaemonState) SetDHTNodes(count int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.DHTNodes = count
}

// GetDHTNodes returns the current DHT node count.
func (s *DaemonState) GetDHTNodes() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.DHTNodes
}

// SetError records an error and its timestamp.
func (s *DaemonState) SetError(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.LastError = err
	s.LastErrorTime = time.Now()
}

// GetError returns the last error and when it occurred.
func (s *DaemonState) GetError() (error, time.Time) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.LastError, s.LastErrorTime
}

// GetUptime returns how long the daemon has been running.
func (s *DaemonState) GetUptime() time.Duration {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return time.Since(s.StartTime)
}

// Snapshot returns a thread-safe copy of the current state.
func (s *DaemonState) Snapshot() DaemonStateSnapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return DaemonStateSnapshot{
		StartTime:      s.StartTime,
		Status:         s.Status,
		ActivePackages: s.ActivePackages,
		TotalPeers:     s.TotalPeers,
		DHTNodes:       s.DHTNodes,
		LastError:      s.LastError,
		LastErrorTime:  s.LastErrorTime,
		Uptime:         time.Since(s.StartTime),
	}
}

// DaemonStateSnapshot is an immutable snapshot of DaemonState.
type DaemonStateSnapshot struct {
	StartTime      time.Time
	Status         DaemonStatus
	ActivePackages int
	TotalPeers     int
	DHTNodes       int
	LastError      error
	LastErrorTime  time.Time
	Uptime         time.Duration
}
