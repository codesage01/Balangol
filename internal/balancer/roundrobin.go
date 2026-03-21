package balancer

import (
	"sync"
	"sync/atomic"

	"github.com/yourname/loadbalancer/internal/models"
)

// RoundRobin cycles through healthy backends in order
type RoundRobin struct {
	mu       sync.RWMutex
	backends []*models.Backend
	counter  atomic.Uint64 // atomic — no lock needed for increment
}

func NewRoundRobin() *RoundRobin {
	return &RoundRobin{}
}

func (rr *RoundRobin) Add(b *models.Backend) {
	rr.mu.Lock()
	defer rr.mu.Unlock()
	rr.backends = append(rr.backends, b)
}

func (rr *RoundRobin) Remove(id string) {
	rr.mu.Lock()
	defer rr.mu.Unlock()
	filtered := make([]*models.Backend, 0, len(rr.backends))
	for _, b := range rr.backends {
		if b.ID != id {
			filtered = append(filtered, b)
		}
	}
	rr.backends = filtered
}

// Next picks the next healthy backend using round robin
func (rr *RoundRobin) Next() (*models.Backend, error) {
	rr.mu.RLock()
	defer rr.mu.RUnlock()

	total := len(rr.backends)
	if total == 0 {
		return nil, ErrNoHealthyBackend
	}

	// Try every backend once to find a healthy one
	for i := 0; i < total; i++ {
		idx := rr.counter.Add(1) % uint64(total)
		b := rr.backends[idx]
		if b.IsHealthy() {
			return b, nil
		}
	}

	return nil, ErrNoHealthyBackend
}

func (rr *RoundRobin) All() []*models.Backend {
	rr.mu.RLock()
	defer rr.mu.RUnlock()
	return append([]*models.Backend{}, rr.backends...)
}
