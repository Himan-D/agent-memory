package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var port = flag.String("port", "8081", "Port to listen on")

type MemoryAPIServer struct {
	httpServer *http.Server
}

func NewMemoryAPIServer() *MemoryAPIServer {
	mux := http.NewServeMux()
	
	// Health
	mux.HandleFunc("/health", handleHealth)
	mux.HandleFunc("/ready", handleReady)
	
	// Memory endpoints (delegate to existing handlers)
	mux.HandleFunc("/memories", handleMemories)
	mux.HandleFunc("/memories/", handleMemoryByID)
	mux.HandleFunc("/search", handleSearch)
	mux.HandleFunc("/api/v1/benchmark/", handleBenchmark)
	
	// Metrics
	mux.HandleFunc("/metrics", handleMetrics)

	httpServer := &http.Server{
		Addr:         *port,
		Handler:      loggingMiddleware(mux),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	return &MemoryAPIServer{httpServer: httpServer}
}

func (s *MemoryAPIServer) Start() error {
	log.Printf("Memory API starting on %s", *port)
	return s.httpServer.ListenAndServe()
}

func (s *MemoryAPIServer) Stop() error {
	log.Println("Shutting down Memory API...")
	return s.httpServer.Shutdown(nil)
}

func main() {
	flag.Parse()
	
	server := NewMemoryAPIServer()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Shutdown signal")
		server.Stop()
	}()

	if err := server.Start(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Error: %v", err)
	}
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status": "ok", "service": "memory-api"}`))
}

func handleReady(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status": "ready"}`))
}

func handleMemories(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status": "memories endpoint - delegate to memory service"}`))
}

func handleMemoryByID(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status": "memory by id endpoint"}`))
}

func handleSearch(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status": "search endpoint - delegate to multi-signal"}`))
}

func handleBenchmark(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status": "benchmark endpoint"}`))
}

func handleMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(`# Memory API metrics`))
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %d %v", r.Method, r.URL.Path, 0, time.Since(start))
	})
}