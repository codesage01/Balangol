package models

import (
	"sync"
	"sync/atomic"
	"time"
)

// Status represents the health state of a backend
type Status string

const (
	StatusHealthy   Status = "healthy"
	StatusUnhealthy Status = "unhealthy"
)

// Backend represents a single backend server
type Backend struct {
	ID           string    `json:"id"`
	URL          string    `json:"url"`
	Status       Status    `json:"status"`
	AddedAt      time.Time `json:"added_at"`

	// Atomic counters — safe to read/write from multiple goroutines
	TotalRequests  atomic.Int64 `json:"-"`
	ActiveRequests atomic.Int64 `json:"-"`
	TotalErrors    atomic.Int64 `json:"-"`

	mu           sync.RWMutex
}

func (b *Backend) SetStatus(s Status) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.Status = s
}

func (b *Backend) GetStatus() Status {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.Status
}

func (b *Backend) IsHealthy() bool {
	return b.GetStatus() == StatusHealthy
}

// BackendStats is a JSON-safe snapshot of backend metrics
type BackendStats struct {
	ID             string    `json:"id"`
	URL            string    `json:"url"`
	Status         Status    `json:"status"`
	AddedAt        time.Time `json:"added_at"`
	TotalRequests  int64     `json:"total_requests"`
	ActiveRequests int64     `json:"active_requests"`
	TotalErrors    int64     `json:"total_errors"`
}

func (b *Backend) Snapshot() BackendStats {
	return BackendStats{
		ID:             b.ID,
		URL:            b.URL,
		Status:         b.GetStatus(),
		AddedAt:        b.AddedAt,
		TotalRequests:  b.TotalRequests.Load(),
		ActiveRequests: b.ActiveRequests.Load(),
		TotalErrors:    b.TotalErrors.Load(),
	}
}
