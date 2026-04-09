package health

import (
	"log"
	"net/http"
	"time"

	"github.com/codesage01/Balangol/internal/models"
)

// Checker pings all backends on a schedule and updates their status
type Checker struct {
	registry  BackendLister
	interval  time.Duration
	client    *http.Client
	broadcast func(any) // sends status update to WebSocket dashboard
}

type BackendLister interface {
	All() []*models.Backend
}

func NewChecker(registry BackendLister, interval time.Duration, broadcast func(any)) *Checker {
	return &Checker{
		registry:  registry,
		interval:  interval,
		broadcast: broadcast,
		client: &http.Client{
			Timeout: 3 * time.Second,
		},
	}
}

// Run starts the health check loop — call in a goroutine
func (c *Checker) Run() {
	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()

	for range ticker.C {
		c.checkAll()
	}
}

func (c *Checker) checkAll() {
	backends := c.registry.All()
	for _, b := range backends {
		go c.check(b) // check each backend concurrently
	}
}

func (c *Checker) check(b *models.Backend) {
	prev := b.GetStatus()

	resp, err := c.client.Get(b.URL + "/health")
	if err != nil || resp.StatusCode >= 500 {
		b.SetStatus(models.StatusUnhealthy)
	} else {
		b.SetStatus(models.StatusHealthy)
	}

	// Only broadcast if status changed — avoids spamming dashboard
	if prev != b.GetStatus() {
		log.Printf("[health] %s → %s (%s)", b.URL, b.GetStatus(), b.ID)
		if c.broadcast != nil {
			c.broadcast(map[string]any{
				"type":   "health_update",
				"id":     b.ID,
				"status": b.GetStatus(),
			})
		}
	}
}
