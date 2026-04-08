package main

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"agent-memory/internal/config"
	"agent-memory/internal/memory"
	"agent-memory/internal/memory/types"
)

var (
	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "agent_memory_http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "endpoint", "status"},
	)
	httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "agent_memory_http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1},
		},
		[]string{"method", "endpoint"},
	)
)

type APIServer struct {
	cfg    *config.Config
	memSvc *memory.Service
	router *mux.Router
	server *http.Server
}

func NewAPIServer(cfg *config.Config, memSvc *memory.Service) *APIServer {
	router := mux.NewRouter()
	router.Use(loggingMiddleware)
	router.Use(metricsMiddleware)
	router.Use(recoveryMiddleware)

	// Auth middleware (optional, enabled via AUTH_ENABLED)
	if cfg.Auth.Enabled {
		router.Use(authMiddleware(cfg))
	}

	srv := &APIServer{
		cfg:    cfg,
		memSvc: memSvc,
		router: router,
		server: &http.Server{
			Addr:         cfg.App.HTTPPort,
			Handler:      router,
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
		},
	}

	srv.registerRoutes()
	return srv
}

func (s *APIServer) registerRoutes() {
	s.router.HandleFunc("/health", s.healthHandler).Methods("GET")
	s.router.HandleFunc("/ready", s.readyHandler).Methods("GET")
	s.router.Handle("/metrics", promhttp.Handler()).Methods("GET")

	// API key management (requires auth)
	s.router.HandleFunc("/admin/api-keys", s.listAPIKeysHandler).Methods("GET")
	s.router.HandleFunc("/admin/api-keys", s.createAPIKeyHandler).Methods("POST")
	s.router.HandleFunc("/admin/api-keys/{keyID}", s.deleteAPIKeyHandler).Methods("DELETE")

	s.router.HandleFunc("/sessions", s.createSessionHandler).Methods("POST")
	s.router.HandleFunc("/sessions/{sessionID}/messages", s.addMessageHandler).Methods("POST")
	s.router.HandleFunc("/sessions/{sessionID}/messages", s.getMessagesHandler).Methods("GET")
	s.router.HandleFunc("/sessions/{sessionID}/context", s.getContextHandler).Methods("GET")

	s.router.HandleFunc("/entities", s.createEntityHandler).Methods("POST")
	s.router.HandleFunc("/entities/{entityID}", s.getEntityHandler).Methods("GET")
	s.router.HandleFunc("/entities/{entityID}/relations", s.getRelationsHandler).Methods("GET")

	s.router.HandleFunc("/relations", s.createRelationHandler).Methods("POST")

	s.router.HandleFunc("/search", s.searchHandler).Methods("GET")

	s.router.HandleFunc("/graph/query", s.graphQueryHandler).Methods("POST")
}

func (s *APIServer) Start() error {
	log.Printf("Starting HTTP server on %s", s.cfg.App.HTTPPort)
	if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("server error: %w", err)
	}
	return nil
}

func (s *APIServer) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return s.server.Shutdown(ctx)
}

func (s *APIServer) RunUntilShutdown() error {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := s.Start(); err != nil {
			log.Printf("Server error: %v", err)
		}
	}()

	<-sigCh
	log.Println("Shutting down server...")
	return s.Stop()
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
	})
}

func metricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(rw, r)
		duration := time.Since(start).Seconds()

		httpRequestsTotal.WithLabelValues(r.Method, r.URL.Path, fmt.Sprintf("%d", rw.statusCode)).Inc()
		httpRequestDuration.WithLabelValues(r.Method, r.URL.Path).Observe(duration)
	})
}

func recoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("Panic recovered: %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func authMiddleware(cfg *config.Config) func(http.Handler) http.Handler {
	apiKeys := make(map[string]string) // key -> tenant_id
	for _, key := range cfg.Auth.APIKeys {
		parts := splitKey(key) // format: "key:tenant" or just "key"
		if len(parts) == 2 {
			apiKeys[parts[0]] = parts[1]
		} else {
			apiKeys[key] = "default"
		}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip auth for health/ready/metrics
			if r.URL.Path == "/health" || r.URL.Path == "/ready" || r.URL.Path == "/metrics" {
				next.ServeHTTP(w, r)
				return
			}

			apiKey := r.Header.Get("X-API-Key")
			if apiKey == "" {
				apiKey = r.URL.Query().Get("api_key")
			}

			tenantID := ""
			valid := false

			// Check config keys first
			if tenantID = apiKeys[apiKey]; tenantID != "" {
				valid = true
			} else {
				// Check runtime keys
				keyMu.Lock()
				for _, k := range apiKeyStore {
					if k.Key == apiKey && !k.IsExpired() {
						tenantID = "default"
						valid = true
						break
					}
				}
				keyMu.Unlock()
			}

			if apiKey == "" || !valid {
				http.Error(w, "Unauthorized: Invalid or missing API key", http.StatusUnauthorized)
				return
			}

			// Add tenant ID to request context
			ctx := r.Context()
			ctx = context.WithValue(ctx, "tenant_id", tenantID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func splitKey(key string) []string {
	for i, c := range key {
		if c == ':' {
			return []string{key[:i], key[i+1:]}
		}
	}
	return []string{key}
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (s *APIServer) healthHandler(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func getTenantID(r *http.Request) string {
	if ctx := r.Context(); ctx != nil {
		if tenantID, ok := ctx.Value("tenant_id").(string); ok {
			return tenantID
		}
	}
	return ""
}

func (s *APIServer) readyHandler(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]string{"status": "ready"})
}

func (s *APIServer) createSessionHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		AgentID  string                 `json:"agent_id"`
		Metadata map[string]interface{} `json:"metadata"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	tenantID := getTenantID(r)
	metadata := req.Metadata
	if metadata == nil {
		metadata = map[string]interface{}{}
	}
	if tenantID != "" && tenantID != "default" {
		metadata["tenant_id"] = tenantID
	}

	sess, err := s.memSvc.CreateSession(req.AgentID, metadata)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(sess)
}

func (s *APIServer) addMessageHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["sessionID"]

	var msg types.Message
	if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := s.memSvc.AddToContext(sessionID, msg); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (s *APIServer) getMessagesHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["sessionID"]

	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		fmt.Sscanf(l, "%d", &limit)
	}

	messages, err := s.memSvc.GetContext(sessionID, limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(messages)
}

func (s *APIServer) getContextHandler(w http.ResponseWriter, r *http.Request) {
	s.getMessagesHandler(w, r)
}

func (s *APIServer) createEntityHandler(w http.ResponseWriter, r *http.Request) {
	var entity types.Entity
	if err := json.NewDecoder(r.Body).Decode(&entity); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	created, err := s.memSvc.AddEntity(entity)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"status": "ok", "id": created.ID})
}

func (s *APIServer) getEntityHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	entityID := vars["entityID"]

	entity, err := s.memSvc.GetEntity(entityID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(entity)
}

func (s *APIServer) getRelationsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	entityID := vars["entityID"]

	relType := r.URL.Query().Get("type")

	relations, err := s.memSvc.GetEntityRelations(entityID, relType)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(relations)
}

func (s *APIServer) createRelationHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		FromID   string                 `json:"from_id"`
		ToID     string                 `json:"to_id"`
		Type     string                 `json:"type"`
		Metadata map[string]interface{} `json:"metadata"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := s.memSvc.AddRelation(req.FromID, req.ToID, req.Type, req.Metadata); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (s *APIServer) searchHandler(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		http.Error(w, "missing query param 'q'", http.StatusBadRequest)
		return
	}

	limit := 10
	if l := r.URL.Query().Get("limit"); l != "" {
		fmt.Sscanf(l, "%d", &limit)
	}

	threshold := float32(0.5)
	if t := r.URL.Query().Get("threshold"); t != "" {
		var f float64
		fmt.Sscanf(t, "%f", &f)
		threshold = float32(f)
	}

	results, err := s.memSvc.SearchSemantic(query, limit, threshold, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(results)
}

func (s *APIServer) graphQueryHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Cypher string                 `json:"cypher"`
		Params map[string]interface{} `json:"params"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	results, err := s.memSvc.QueryGraph(req.Cypher, req.Params)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(results)
}

type APIKey struct {
	ID        string     `json:"id"`
	Key       string     `json:"key,omitempty"`
	Label     string     `json:"label"`
	CreatedAt time.Time  `json:"created_at"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	TenantID  string     `json:"tenant_id,omitempty"`
}

func (k APIKey) IsExpired() bool {
	if k.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*k.ExpiresAt)
}

var (
	apiKeyStore = make(map[string]APIKey)
	keyCounter  int
	keyMu       sync.Mutex
)

func (s *APIServer) listAPIKeysHandler(w http.ResponseWriter, r *http.Request) {
	keyMu.Lock()
	defer keyMu.Unlock()

	var keys []APIKey
	for _, k := range apiKeyStore {
		// Don't expose the actual key
		keyCopy := k
		keyCopy.Key = ""
		keys = append(keys, keyCopy)
	}
	json.NewEncoder(w).Encode(keys)
}

func (s *APIServer) createAPIKeyHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Label     string `json:"label"`
		ExpiresIn int    `json:"expires_in_hours"`
		TenantID  string `json:"tenant_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	tenantID := getTenantID(r)
	if tenantID == "" {
		tenantID = req.TenantID
	}
	if tenantID == "" {
		tenantID = "default"
	}

	keyMu.Lock()
	defer keyMu.Unlock()

	keyCounter++
	keyID := fmt.Sprintf("key_%d", keyCounter)
	apiKey := fmt.Sprintf("am_%s_%d", generateRandomString(16), time.Now().Unix())

	key := APIKey{
		ID:        keyID,
		Key:       apiKey,
		Label:     req.Label,
		CreatedAt: time.Now(),
		TenantID:  tenantID,
	}

	if req.ExpiresIn > 0 {
		exp := time.Now().Add(time.Duration(req.ExpiresIn) * time.Hour)
		key.ExpiresAt = &exp
	}

	apiKeyStore[keyID] = key

	json.NewEncoder(w).Encode(map[string]string{
		"id":      keyID,
		"key":     apiKey,
		"label":   req.Label,
		"tenant":  tenantID,
		"expires": key.ExpiresAt.Format(time.RFC3339),
	})
}

func (s *APIServer) deleteAPIKeyHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	keyID := vars["keyID"]

	keyMu.Lock()
	defer keyMu.Unlock()

	if _, ok := apiKeyStore[keyID]; !ok {
		http.Error(w, "API key not found", http.StatusNotFound)
		return
	}

	delete(apiKeyStore, keyID)
	json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
}

func generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	_, err := rand.Read(b)
	if err != nil {
		// Fallback to pseudo-random
		for i := range b {
			b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
		}
		return string(b)
	}
	for i := range b {
		b[i] = charset[int(b[i])%len(charset)]
	}
	return string(b)
}
