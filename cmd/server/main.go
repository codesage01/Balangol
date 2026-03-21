package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/yourname/loadbalancer/internal/balancer"
	"github.com/yourname/loadbalancer/internal/health"
	"github.com/yourname/loadbalancer/internal/proxy"
	"github.com/yourname/loadbalancer/internal/server"
)

func main() {
	lbPort := getEnv("LB_PORT", ":8080")   // load balancer port
	apiPort := getEnv("API_PORT", ":9090")  // admin API + dashboard port

	// Setup round robin balancer and backend registry
	rr := balancer.NewRoundRobin()
	registry := balancer.NewRegistry(rr)

	// WebSocket hub for real-time dashboard updates
	hub := server.NewHub()
	go hub.Run()

	// Health checker — pings every backend every 10 seconds
	checker := health.NewChecker(registry, 10*time.Second, hub.Broadcast)
	go checker.Run()

	// --- Load Balancer Server (port 8080) ---
	// All traffic here gets forwarded to a healthy backend
	lbProxy := proxy.NewProxy(registry)
	go func() {
		log.Printf("Load Balancer listening on %s", lbPort)
		if err := http.ListenAndServe(lbPort, lbProxy); err != nil {
			log.Fatal(err)
		}
	}()

	// --- Admin API + Dashboard Server (port 9090) ---
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

	log.Printf("Admin Dashboard on %s", apiPort)
	if err := http.ListenAndServe(apiPort, mux); err != nil {
		log.Fatal(err)
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
