package main

import (
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/codesage01/Balangol/internal/balancer"
	"github.com/codesage01/Balangol/internal/health"
	"github.com/codesage01/Balangol/internal/proxy"
	"github.com/codesage01/Balangol/internal/server"
)

func main() {
	// 1. Railway provides a single PORT. We use it for EVERYTHING.
	port := getEnv("PORT", "8080")

	// Setup round robin balancer and backend registry
	rr := balancer.NewRoundRobin()
	registry := balancer.NewRegistry(rr)

	// WebSocket hub for real-time dashboard updates
	hub := server.NewHub()
	go hub.Run()

	// Health checker — pings every backend every 10 seconds
	checker := health.NewChecker(registry, 10*time.Second, hub.Broadcast)
	go checker.Run()

	// Initialize handlers
	lbProxy := proxy.NewProxy(registry)
	apiHandler := server.NewAPIHandler(registry, hub)
	wsHandler := server.NewWSHandler(hub)

	// Create a single Mux (Router)
	mux := http.NewServeMux()

	// --- Admin API Routes ---
	mux.HandleFunc("GET /api/backends", apiHandler.ListBackends)
	mux.HandleFunc("POST /api/backends", apiHandler.AddBackend)
	mux.HandleFunc("DELETE /api/backends/{id}", apiHandler.RemoveBackend)
	mux.HandleFunc("GET /api/stats", apiHandler.Stats)

	// WebSocket for dashboard
	mux.HandleFunc("GET /ws", wsHandler.Handle)

	// --- Dashboard UI ---
	// We serve the UI on the /dashboard path so it doesn't conflict with the Proxy
	fileServer := http.FileServer(http.Dir("./web"))
	mux.Handle("/dashboard/", http.StripPrefix("/dashboard/", fileServer))

	// --- Main Handler Logic ---
	// We use a custom handler to decide: Is this an API call, a Dashboard call, or Proxy traffic?
	finalHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 1. If the path starts with /api, /ws, or /dashboard, use the Mux (Admin/UI)
		if strings.HasPrefix(r.URL.Path, "/api") || 
		   strings.HasPrefix(r.URL.Path, "/ws") || 
		   strings.HasPrefix(r.URL.Path, "/dashboard") {
			mux.ServeHTTP(w, r)
			return
		}

		// 2. Everything else is treated as traffic that needs to be Load Balanced
		lbProxy.ServeHTTP(w, r)
	})

	log.Printf("Balangol Unified Server started on port %s", port)
	log.Printf("Access Dashboard at: /dashboard/")
	
	// Start the single server
	if err := http.ListenAndServe(":"+port, finalHandler); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
