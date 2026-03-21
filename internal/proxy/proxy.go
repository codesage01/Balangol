package proxy

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/yourname/loadbalancer/internal/balancer"
	"github.com/yourname/loadbalancer/internal/models"
)

// Proxy forwards incoming requests to the next healthy backend
type Proxy struct {
	registry *balancer.Registry
}

func NewProxy(registry *balancer.Registry) *Proxy {
	return &Proxy{registry: registry}
}

func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	backend, err := p.registry.Next()
	if err != nil {
		http.Error(w, "No healthy backend available", http.StatusServiceUnavailable)
		return
	}

	backend.TotalRequests.Add(1)
	backend.ActiveRequests.Add(1)
	defer backend.ActiveRequests.Add(-1)

	target, err := url.Parse(backend.URL)
	if err != nil {
		backend.TotalErrors.Add(1)
		http.Error(w, "Invalid backend URL", http.StatusInternalServerError)
		return
	}

	rp := httputil.NewSingleHostReverseProxy(target)
	rp.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		backend.TotalErrors.Add(1)
		backend.SetStatus(models.StatusUnhealthy)
		log.Printf("[proxy] error forwarding to %s: %v", backend.URL, err)
		http.Error(w, "Backend error", http.StatusBadGateway)
	}

	// Add header so backend knows it came through LB
	r.Header.Set("X-Forwarded-By", "GoLoadBalancer")
	r.Header.Set("X-Backend-ID", backend.ID)

	log.Printf("[proxy] %s %s → %s", r.Method, r.URL.Path, backend.URL)
	rp.ServeHTTP(w, r)
}
