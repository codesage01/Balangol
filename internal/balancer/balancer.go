package balancer

import (
	"errors"
	"sync"

	"github.com/codesage01/Balangol/internal/models"
)

var ErrNoHealthyBackend = errors.New("no healthy backend available")

// Balancer is the interface for load balancing algorithms
// Swap RoundRobin for LeastConnections, WeightedRoundRobin etc. easily
type Balancer interface {
	Next() (*models.Backend, error)
	Add(b *models.Backend)
	Remove(id string)
	All() []*models.Backend
}

// Registry holds all backends and the active balancer
type Registry struct {
	mu       sync.RWMutex
	backends []*models.Backend
	balancer Balancer
}

func NewRegistry(b Balancer) *Registry {
	return &Registry{balancer: b}
}

func (r *Registry) Add(b *models.Backend) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.backends = append(r.backends, b)
	r.balancer.Add(b)
}

func (r *Registry) Remove(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	filtered := make([]*models.Backend, 0, len(r.backends))
	for _, b := range r.backends {
		if b.ID != id {
			filtered = append(filtered, b)
		}
	}
	r.backends = filtered
	r.balancer.Remove(id)
}

func (r *Registry) Next() (*models.Backend, error) {
	return r.balancer.Next()
}

func (r *Registry) All() []*models.Backend {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return append([]*models.Backend{}, r.backends...)
}
