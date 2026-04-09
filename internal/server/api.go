package server

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/codesage01/Balangol/internal/balancer"
	"github.com/codesage01/Balangol/internal/models"
)

type APIHandler struct {
	registry *balancer.Registry
	hub      *Hub
}

func NewAPIHandler(registry *balancer.Registry, hub *Hub) *APIHandler {
	return &APIHandler{registry: registry, hub: hub}
}

// GET /api/backends — list all backends with stats
func (h *APIHandler) ListBackends(w http.ResponseWriter, r *http.Request) {
	backends := h.registry.All()
	stats := make([]models.BackendStats, 0, len(backends))
	for _, b := range backends {
		stats = append(stats, b.Snapshot())
	}
	jsonResponse(w, http.StatusOK, stats)
}

// POST /api/backends — add a new backend
func (h *APIHandler) AddBackend(w http.ResponseWriter, r *http.Request) {
	var req struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.URL == "" {
		http.Error(w, "url is required", http.StatusBadRequest)
		return
	}

	// Normalize URL
	req.URL = strings.TrimRight(req.URL, "/")

	b := &models.Backend{
		ID:      uuid.NewString(),
		URL:     req.URL,
		Status:  models.StatusHealthy,
		AddedAt: time.Now().UTC(),
	}

	h.registry.Add(b)

	// Notify dashboard via WebSocket
	h.hub.Broadcast(map[string]any{
		"type":    "backend_added",
		"backend": b.Snapshot(),
	})

	jsonResponse(w, http.StatusCreated, b.Snapshot())
}

// DELETE /api/backends/{id} — remove a backend
func (h *APIHandler) RemoveBackend(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "id is required", http.StatusBadRequest)
		return
	}

	h.registry.Remove(id)

	h.hub.Broadcast(map[string]any{
		"type": "backend_removed",
		"id":   id,
	})

	jsonResponse(w, http.StatusOK, map[string]string{"deleted": id})
}

// GET /api/stats — overall load balancer stats
func (h *APIHandler) Stats(w http.ResponseWriter, r *http.Request) {
	backends := h.registry.All()

	var totalReq, totalErr, healthy int64
	for _, b := range backends {
		totalReq += b.TotalRequests.Load()
		totalErr += b.TotalErrors.Load()
		if b.IsHealthy() {
			healthy++
		}
	}

	jsonResponse(w, http.StatusOK, map[string]any{
		"total_backends":   len(backends),
		"healthy_backends": healthy,
		"total_requests":   totalReq,
		"total_errors":     totalErr,
	})
}

func jsonResponse(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
