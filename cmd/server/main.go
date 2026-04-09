package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/codesage01/Balangol/internal/balancer"
	"github.com/codesage01/Balangol/internal/health"
	"github.com/codesage01/Balangol/internal/proxy"
	"github.com/codesage01/Balangol/internal/server"
)

func main() {
	// 1. Define the ports properly
	// Vercel usually provides a PORT env, but since you need two ports, 
	// we will define them clearly here.
	lbPort  := ":" + getEnv("PORT", "8080")    // Load Balancer Port
	apiPort := ":" + getEnv("API_PORT", "9090") // Admin/Dashboard Port

	// Setup round robin balancer and backend registry
	rr := balancer.NewRoundRobin()
	registry := balancer.NewRegistry(rr)

	// WebSocket hub for real-time dashboard updates
	hub := server.NewHub()
	go hub.Run()

	// Health checker — pings every backend every 10 seconds
	checker := health.NewChecker(registry, 10*time.Second, hub.Broadcast)
	go checker.Run()

	// --- Load Balancer Server ---
	lbProxy := proxy.NewProxy(registry)
	go func() {
		log.Printf("Load Balancer listening on %s", lbPort)
		if err := http.ListenAndServe(lbPort, lbProxy); err != nil {
			log.Fatalf("Load Balancer failed: %v", err)
		}
	}()

	// --- Admin API + Dashboard Server ---
	mux := http.NewServeMux()

	apiHandler := server.NewAPIHandler(registry, hub)
	wsHandler := server.NewWSHandler(hub)

	// REST API
	mux.HandleFunc("GET /api/backends", apiHandler.ListBackends)
	mux.HandleFunc("POST /api/backends", apiHandler.AddBackend)
	mux.HandleFunc("DELETE /api/backends/{id}", apiHandler.RemoveBackend)
	mux.HandleFunc("GET /api/stats", apiHandler.Stats)

	// WebSocket for dashboard
	mux.HandleFunc("GET /ws", wsHandler.Handle)

	// Dashboard UI
	mux.Handle("/", http.FileServer(http.Dir("./web")))

	log.Printf("Admin Dashboard listening on %s", apiPort)
	if err := http.ListenAndServe(apiPort, mux); err != nil {
		log.Fatalf("Admin Server failed: %v", err)
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
