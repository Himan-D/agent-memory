package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"agent-memory/internal/config"
	"agent-memory/internal/memory"
	"agent-memory/internal/memory/types"
)

var port = flag.String("port", "8081", "Port to listen on")

type MemoryAPIServer struct {
	httpServer *http.Server
	cfg       *config.Config
	memSvc    *memory.Service
}

func NewMemoryAPIServer(cfg *config.Config) (*MemoryAPIServer, error) {
	memSvc, err := memory.NewService(cfg)
	if err != nil {
		return nil, fmt.Errorf("init memory service: %w", err)
	}

	return &MemoryAPIServer{
		cfg:    cfg,
		memSvc: memSvc,
	}, nil
}

func (s *MemoryAPIServer) Start() error {
	mux := http.NewServeMux()

	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/ready", s.handleReady)

	mux.HandleFunc("/memories", s.handleMemories)
	mux.HandleFunc("/memories/", s.handleMemoryByID)
	mux.HandleFunc("/search", s.handleSearch)
	mux.HandleFunc("/api/v1/benchmark/", s.handleBenchmark)

	mux.HandleFunc("/metrics", s.handleMetrics)

	s.httpServer = &http.Server{
		Addr:         ":" + *port,
		Handler:      s.loggingMiddleware(mux),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	log.Printf("Memory API starting on %s", *port)
	return s.httpServer.ListenAndServe()
}

func (s *MemoryAPIServer) Stop() error {
	log.Println("Shutting down Memory API...")
	if s.httpServer != nil {
		return s.httpServer.Shutdown(nil)
	}
	return nil
}

func main() {
	flag.Parse()

	cfg := config.Load()

	server, err := NewMemoryAPIServer(cfg)
	if err != nil {
		log.Fatalf("Server init error: %v", err)
	}

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

func (s *MemoryAPIServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok", "service": "memory-api"})
}

func (s *MemoryAPIServer) handleReady(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ready"})
}

func (s *MemoryAPIServer) handleMemories(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.listMemories(w, r)
	case http.MethodPost:
		s.createMemory(w, r)
	default:
		safeHTTPError(w, r, fmt.Errorf("method not allowed"), http.StatusMethodNotAllowed)
	}
}

func (s *MemoryAPIServer) handleMemoryByID(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/memories/")

	switch r.Method {
	case http.MethodGet:
		s.getMemory(w, r, id)
	case http.MethodPut:
		s.updateMemory(w, r, id)
	case http.MethodDelete:
		s.deleteMemory(w, r, id)
	default:
		safeHTTPError(w, r, fmt.Errorf("method not allowed"), http.StatusMethodNotAllowed)
	}
}

func (s *MemoryAPIServer) handleSearch(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.searchMemories(w, r)
	case http.MethodPost:
		s.hybridSearch(w, r)
	default:
		safeHTTPError(w, r, fmt.Errorf("method not allowed"), http.StatusMethodNotAllowed)
	}
}

func (s *MemoryAPIServer) handleBenchmark(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "benchmark endpoint"})
}

func (s *MemoryAPIServer) handleMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(`# Memory API metrics
# Type: gauge, help
memory_total 0
memory_created_total 0
memory_search_total 0
`))
}

func (s *MemoryAPIServer) listMemories(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		safeHTTPError(w, r, fmt.Errorf("user_id required"), http.StatusBadRequest)
		return
	}

	memories, err := s.memSvc.GetMemoriesByUser(r.Context(), userID)
	if err != nil {
		safeHTTPError(w, r, fmt.Errorf("get memories: %w", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"items": memories,
		"total": len(memories),
	})
}

func (s *MemoryAPIServer) createMemory(w http.ResponseWriter, r *http.Request) {
	var mem types.Memory
	if err := json.NewDecoder(r.Body).Decode(&mem); err != nil {
		safeHTTPError(w, r, fmt.Errorf("invalid request: %w", err), http.StatusBadRequest)
		return
	}

	if mem.Content == "" {
		safeHTTPError(w, r, fmt.Errorf("content is required"), http.StatusBadRequest)
		return
	}

	mem.ID = ""
	mem.CreatedAt = time.Now()
	mem.UpdatedAt = time.Now()

	createdMem, err := s.memSvc.CreateMemory(r.Context(), &mem)
	if err != nil {
		safeHTTPError(w, r, fmt.Errorf("create memory: %w", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(createdMem)
}

func (s *MemoryAPIServer) getMemory(w http.ResponseWriter, r *http.Request, id string) {
	mem, err := s.memSvc.GetMemory(r.Context(), id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			safeHTTPError(w, r, fmt.Errorf("memory not found"), http.StatusNotFound)
			return
		}
		safeHTTPError(w, r, fmt.Errorf("get memory: %w", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(mem)
}

func (s *MemoryAPIServer) updateMemory(w http.ResponseWriter, r *http.Request, id string) {
	var update struct {
		Content  string                 `json:"content"`
		Metadata map[string]interface{} `json:"metadata"`
	}
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		safeHTTPError(w, r, fmt.Errorf("invalid request: %w", err), http.StatusBadRequest)
		return
	}

	mem, err := s.memSvc.GetMemory(r.Context(), id)
	if err != nil {
		safeHTTPError(w, r, fmt.Errorf("get memory: %w", err), http.StatusInternalServerError)
		return
	}

	if update.Content != "" {
		mem.Content = update.Content
	}
	if update.Metadata != nil {
		if mem.Metadata == nil {
			mem.Metadata = update.Metadata
		} else {
			for k, v := range update.Metadata {
				mem.Metadata[k] = v
			}
		}
	}

	var content string
	var metadata map[string]interface{}
	if update.Content != "" {
		content = update.Content
	}
	if update.Metadata != nil {
		metadata = update.Metadata
	}

	mem.UpdatedAt = time.Now()
	if err := s.memSvc.UpdateMemory(r.Context(), mem.ID, content, metadata); err != nil {
		safeHTTPError(w, r, fmt.Errorf("update memory: %w", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(mem)
}

func (s *MemoryAPIServer) deleteMemory(w http.ResponseWriter, r *http.Request, id string) {
	if err := s.memSvc.DeleteMemory(r.Context(), id); err != nil {
		safeHTTPError(w, r, fmt.Errorf("delete memory: %w", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *MemoryAPIServer) searchMemories(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	userID := r.URL.Query().Get("user_id")
	if query == "" {
		safeHTTPError(w, r, fmt.Errorf("query is required"), http.StatusBadRequest)
		return
	}

	results, err := s.memSvc.SearchMemories(r.Context(), &types.SearchRequest{
		Query:  query,
		UserID: userID,
		Limit:  10,
	})
	if err != nil {
		safeHTTPError(w, r, fmt.Errorf("search: %w", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"results": results,
		"total":   len(results),
	})
}

func (s *MemoryAPIServer) hybridSearch(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "hybrid search"})
}

func safeHTTPError(w http.ResponseWriter, r *http.Request, err error, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error": err.Error(),
	})
}

func (s *MemoryAPIServer) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %d %v", r.Method, r.URL.Path, http.StatusOK, time.Since(start))
	})
}