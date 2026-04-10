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
	"regexp"
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
	"agent-memory/internal/project"
	"agent-memory/internal/webhook"
)

const timeFormat = "2006-01-02T15:04:05.000Z07:00"

type rateLimiter struct {
	requests map[string][]time.Time
	mu       sync.Mutex
	limit    int
	window   time.Duration
}

func newRateLimiter(limit int, window time.Duration) *rateLimiter {
	return &rateLimiter{
		requests: make(map[string][]time.Time),
		limit:    limit,
		window:   window,
	}
}

func (rl *rateLimiter) allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-rl.window)

	var recent []time.Time
	for _, t := range rl.requests[key] {
		if t.After(cutoff) {
			recent = append(recent, t)
		}
	}

	if len(recent) >= rl.limit {
		rl.requests[key] = recent
		return false
	}

	rl.requests[key] = append(recent, now)
	return true
}

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
	cfg         *config.Config
	memSvc      *memory.Service
	projSvc     *project.Service
	whSvc       *webhook.Service
	router      *mux.Router
	server      *http.Server
	rateLimiter *rateLimiter
}

func NewAPIServer(cfg *config.Config, memSvc *memory.Service, projSvc *project.Service, whSvc *webhook.Service) *APIServer {
	rl := newRateLimiter(100, time.Minute)

	router := mux.NewRouter()
	router.Use(loggingMiddleware)
	router.Use(metricsMiddleware)
	router.Use(recoveryMiddleware)
	router.Use(rateLimitMiddleware(rl))

	if cfg.Auth.Enabled {
		router.Use(authMiddleware(cfg))
	}

	srv := &APIServer{
		cfg:         cfg,
		memSvc:      memSvc,
		projSvc:     projSvc,
		whSvc:       whSvc,
		router:      router,
		rateLimiter: rl,
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

	s.router.HandleFunc("/admin/api-keys", s.listAPIKeysHandler).Methods("GET")
	s.router.HandleFunc("/admin/api-keys", s.createAPIKeyHandler).Methods("POST")
	s.router.HandleFunc("/admin/api-keys/{keyID}", s.deleteAPIKeyHandler).Methods("DELETE")

	s.router.HandleFunc("/sessions", s.createSessionHandler).Methods("POST")
	s.router.HandleFunc("/sessions/{sessionID}/messages", s.addMessageHandler).Methods("POST")
	s.router.HandleFunc("/sessions/{sessionID}/messages", s.getMessagesHandler).Methods("GET")
	s.router.HandleFunc("/sessions/{sessionID}/context", s.getContextHandler).Methods("GET")
	s.router.HandleFunc("/sessions/{sessionID}", s.getSessionHandler).Methods("GET")
	s.router.HandleFunc("/sessions/{sessionID}", s.deleteSessionHandler).Methods("DELETE")

	s.router.HandleFunc("/entities", s.createEntityHandler).Methods("POST")
	s.router.HandleFunc("/entities", s.listEntitiesHandler).Methods("GET")
	s.router.HandleFunc("/entities/{entityID}", s.getEntityHandler).Methods("GET")
	s.router.HandleFunc("/entities/{entityID}/relations", s.getRelationsHandler).Methods("GET")
	s.router.HandleFunc("/entities/{entityID}/memories", s.getEntityMemoriesHandler).Methods("GET")
	s.router.HandleFunc("/entities/{entityID}", s.updateEntityHandler).Methods("PUT")
	s.router.HandleFunc("/entities/{entityID}", s.deleteEntityHandler).Methods("DELETE")

	s.router.HandleFunc("/relations", s.createRelationHandler).Methods("POST")
	s.router.HandleFunc("/relations/{relationID}", s.deleteRelationHandler).Methods("DELETE")

	s.router.HandleFunc("/graph/query", s.graphQueryHandler).Methods("POST")
	s.router.HandleFunc("/graph/traverse/{entityID}", s.traverseHandler).Methods("GET")

	s.router.HandleFunc("/search", s.searchHandler).Methods("GET")
	s.router.HandleFunc("/search", s.searchPostHandler).Methods("POST")
	s.router.HandleFunc("/search/advanced", s.advancedSearchHandler).Methods("POST")

	s.router.HandleFunc("/memories", s.createMemoryHandler).Methods("POST")
	s.router.HandleFunc("/memories", s.listMemoriesHandler).Methods("GET")
	s.router.HandleFunc("/memories/infer", s.inferMemoryHandler).Methods("POST")
	s.router.HandleFunc("/memories/process", s.processMemoryHandler).Methods("POST")
	s.router.HandleFunc("/memories/{memoryID}", s.getMemoryHandler).Methods("GET")
	s.router.HandleFunc("/memories/{memoryID}", s.updateMemoryHandler).Methods("PUT")
	s.router.HandleFunc("/memories/{memoryID}", s.deleteMemoryHandler).Methods("DELETE")
	s.router.HandleFunc("/memories/{memoryID}/history", s.getMemoryHistoryHandler).Methods("GET")
	s.router.HandleFunc("/memories/{memoryID}/expire", s.setMemoryExpirationHandler).Methods("POST")
	s.router.HandleFunc("/memories/{memoryID}/link/{entityID}", s.linkMemoryEntityHandler).Methods("POST")

	s.router.HandleFunc("/memories/batch", s.batchCreateMemoriesHandler).Methods("POST")
	s.router.HandleFunc("/memories/batch-update", s.batchUpdateMemoriesHandler).Methods("PUT")
	s.router.HandleFunc("/memories/batch-delete", s.batchDeleteMemoriesHandler).Methods("DELETE")
	s.router.HandleFunc("/memories/bulk-delete", s.bulkDeleteHandler).Methods("DELETE")

	s.router.HandleFunc("/feedback", s.createFeedbackHandler).Methods("POST")
	s.router.HandleFunc("/feedback", s.listFeedbackHandler).Methods("GET")
	s.router.HandleFunc("/feedback/memories", s.getMemoriesByFeedbackHandler).Methods("GET")

	s.router.HandleFunc("/projects", s.createProjectHandler).Methods("POST")
	s.router.HandleFunc("/projects", s.listProjectsHandler).Methods("GET")
	s.router.HandleFunc("/projects/{projectID}", s.getProjectHandler).Methods("GET")
	s.router.HandleFunc("/projects/{projectID}", s.updateProjectHandler).Methods("PUT")
	s.router.HandleFunc("/projects/{projectID}", s.deleteProjectHandler).Methods("DELETE")

	s.router.HandleFunc("/webhooks", s.createWebhookHandler).Methods("POST")
	s.router.HandleFunc("/webhooks", s.listWebhooksHandler).Methods("GET")
	s.router.HandleFunc("/webhooks/{webhookID}", s.getWebhookHandler).Methods("GET")
	s.router.HandleFunc("/webhooks/{webhookID}", s.updateWebhookHandler).Methods("PUT")
	s.router.HandleFunc("/webhooks/{webhookID}", s.deleteWebhookHandler).Methods("DELETE")
	s.router.HandleFunc("/webhooks/{webhookID}/test", s.testWebhookHandler).Methods("POST")

	s.router.HandleFunc("/compact", s.runCompactionHandler).Methods("POST")
	s.router.HandleFunc("/compact/targeted", s.runTargetedCompactionHandler).Methods("POST")
	s.router.HandleFunc("/compact/negative-feedback", s.compactNegativeFeedbackHandler).Methods("POST")
	s.router.HandleFunc("/compact/status", s.compactionStatusHandler).Methods("GET")

	s.router.HandleFunc("/admin/cleanup", s.cleanupExpiredHandler).Methods("POST")
	s.router.HandleFunc("/admin/sync", s.syncHandler).Methods("POST")
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
		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(rw, r)
		duration := time.Since(start)

		log.Printf(`{"timestamp":"%s","method":"%s","path":"%s","status":%d,"duration":"%s"}`,
			time.Now().Format(timeFormat), r.Method, r.URL.Path, rw.statusCode, duration)
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

func rateLimitMiddleware(rl *rateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/health" || r.URL.Path == "/ready" || r.URL.Path == "/metrics" {
				next.ServeHTTP(w, r)
				return
			}

			apiKey := r.Header.Get("X-API-Key")
			if apiKey == "" {
				apiKey = r.RemoteAddr
			}

			if !rl.allow(apiKey) {
				http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
				return
			}

			r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
			next.ServeHTTP(w, r)
		})
	}
}

func authMiddleware(cfg *config.Config) func(http.Handler) http.Handler {
	apiKeys := make(map[string]string)
	for _, key := range cfg.Auth.APIKeys {
		parts := splitKey(key)
		if len(parts) == 2 {
			apiKeys[parts[0]] = parts[1]
		} else {
			apiKeys[key] = "default"
		}
	}

	adminKeys := make(map[string]bool)
	for _, key := range cfg.Auth.AdminAPIKeys {
		adminKeys[key] = true
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/health" || r.URL.Path == "/ready" || r.URL.Path == "/metrics" {
				next.ServeHTTP(w, r)
				return
			}

			apiKey := r.Header.Get("X-API-Key")
			if apiKey == "" {
				apiKey = r.URL.Query().Get("api_key")
			}

			tenantID := ""
			isAdmin := false
			valid := false

			if tenantID = apiKeys[apiKey]; tenantID != "" {
				valid = true
			} else if adminKeys[apiKey] {
				tenantID = "admin"
				isAdmin = true
				valid = true
			} else {
				keyMu.Lock()
				for _, k := range apiKeyStore {
					if k.Key == apiKey && !k.IsExpired() {
						tenantID = k.TenantID
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

			ctx := r.Context()
			ctx = context.WithValue(ctx, "tenant_id", tenantID)
			ctx = context.WithValue(ctx, "is_admin", isAdmin)
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

var (
	validAgentID      = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,64}$`)
	validEntityID     = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,64}$`)
	validMessageRole  = regexp.MustCompile(`^(user|assistant|system|tool)$`)
	validMemoryType   = regexp.MustCompile(`^(conversation|session|user|org)$`)
	validFeedbackType = regexp.MustCompile(`^(positive|negative|very_negative)$`)
)

func validateAgentID(id string) error {
	if id == "" {
		return fmt.Errorf("agent_id is required")
	}
	if !validAgentID.MatchString(id) {
		return fmt.Errorf("agent_id must be 1-64 alphanumeric characters, dashes, or underscores")
	}
	return nil
}

func validateEntityID(id string) error {
	if id == "" {
		return fmt.Errorf("entity_id is required")
	}
	if !validEntityID.MatchString(id) {
		return fmt.Errorf("entity_id must be 1-64 alphanumeric characters, dashes, or underscores")
	}
	return nil
}

func validateMessageRole(role string) error {
	if role == "" {
		return fmt.Errorf("role is required")
	}
	if !validMessageRole.MatchString(role) {
		return fmt.Errorf("role must be one of: user, assistant, system, tool")
	}
	return nil
}

func validateMemoryType(memType string) error {
	if memType == "" {
		return nil
	}
	if !validMemoryType.MatchString(memType) {
		return fmt.Errorf("memory_type must be one of: conversation, session, user, org")
	}
	return nil
}

func validateFeedbackType(fbType string) error {
	if fbType == "" {
		return nil
	}
	if !validFeedbackType.MatchString(fbType) {
		return fmt.Errorf("feedback_type must be one of: positive, negative, very_negative")
	}
	return nil
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

func isAdmin(r *http.Request) bool {
	if ctx := r.Context(); ctx != nil {
		if admin, ok := ctx.Value("is_admin").(bool); ok {
			return admin
		}
	}
	return false
}

func (s *APIServer) readyHandler(w http.ResponseWriter, r *http.Request) {
	status := s.memSvc.HealthCheck(r.Context())

	allHealthy := status.Neo4j == "healthy" && status.Qdrant == "healthy"

	w.Header().Set("Content-Type", "application/json")
	if allHealthy {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ready"})
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(status)
	}
}

// ==================== Session Handlers ====================

func (s *APIServer) createSessionHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		AgentID  string                 `json:"agent_id"`
		Metadata map[string]interface{} `json:"metadata"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := validateAgentID(req.AgentID); err != nil {
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
		http.Error(w, "Failed to create session", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(sess)
}

func (s *APIServer) getSessionHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["sessionID"]

	messages, err := s.memSvc.GetContext(sessionID, 1)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	if len(messages) == 0 {
		http.Error(w, "Session not found", http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"session_id": sessionID,
		"messages":   messages,
	})
}

func (s *APIServer) deleteSessionHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["sessionID"]

	if err := s.memSvc.ClearContext(sessionID); err != nil {
		http.Error(w, "Failed to delete session", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
}

func (s *APIServer) addMessageHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["sessionID"]

	var msg types.Message
	if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := validateMessageRole(msg.Role); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if msg.Content == "" || len(msg.Content) > 100000 {
		http.Error(w, "content is required and must be under 100KB", http.StatusBadRequest)
		return
	}

	if err := s.memSvc.AddToContext(sessionID, msg); err != nil {
		http.Error(w, "Failed to add message", http.StatusInternalServerError)
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

// ==================== Entity Handlers ====================

func (s *APIServer) createEntityHandler(w http.ResponseWriter, r *http.Request) {
	var entity types.Entity
	if err := json.NewDecoder(r.Body).Decode(&entity); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if entity.Name == "" {
		http.Error(w, "entity name is required", http.StatusBadRequest)
		return
	}
	if entity.Type == "" {
		http.Error(w, "entity type is required", http.StatusBadRequest)
		return
	}

	tenantID := getTenantID(r)
	if tenantID != "" {
		entity.TenantID = tenantID
	}

	created, err := s.memSvc.AddEntity(entity)
	if err != nil {
		http.Error(w, "Failed to create entity", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(created)
}

func (s *APIServer) listEntitiesHandler(w http.ResponseWriter, r *http.Request) {
	limit := 100
	if l := r.URL.Query().Get("limit"); l != "" {
		fmt.Sscanf(l, "%d", &limit)
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"entities": []types.Entity{},
		"limit":    limit,
	})
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

func (s *APIServer) updateEntityHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	entityID := vars["entityID"]

	var req struct {
		Name       string                 `json:"name"`
		Type       string                 `json:"type"`
		Properties map[string]interface{} `json:"properties"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	entity, err := s.memSvc.GetEntity(entityID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	if req.Name != "" {
		entity.Name = req.Name
	}
	if req.Type != "" {
		entity.Type = req.Type
	}
	if req.Properties != nil {
		entity.Properties = req.Properties
	}

	updated, err := s.memSvc.AddEntity(*entity)
	if err != nil {
		http.Error(w, "Failed to update entity", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(updated)
}

func (s *APIServer) deleteEntityHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	entityID := vars["entityID"]

	err := s.memSvc.DeleteMemoryByID(context.Background(), entityID)
	if err != nil {
		http.Error(w, "Failed to delete entity", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
}

func (s *APIServer) getEntityMemoriesHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	entityID := vars["entityID"]

	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		fmt.Sscanf(l, "%d", &limit)
	}

	results, err := s.memSvc.GetEntityMemories(context.Background(), entityID, limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(results)
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
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := validateEntityID(req.FromID); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := validateEntityID(req.ToID); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if req.Type == "" {
		http.Error(w, "relation type is required", http.StatusBadRequest)
		return
	}

	if err := s.memSvc.AddRelation(req.FromID, req.ToID, req.Type, req.Metadata); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (s *APIServer) deleteRelationHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	relationID := vars["relationID"]

	json.NewEncoder(w).Encode(map[string]string{"status": "deleted", "id": relationID})
}

// ==================== Graph Handlers ====================

func (s *APIServer) graphQueryHandler(w http.ResponseWriter, r *http.Request) {
	if !isAdmin(r) {
		http.Error(w, "Forbidden: Admin access required", http.StatusForbidden)
		return
	}

	var req struct {
		Cypher string                 `json:"cypher"`
		Params map[string]interface{} `json:"params"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	results, err := s.memSvc.QueryGraph(req.Cypher, req.Params)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(results)
}

func (s *APIServer) traverseHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	entityID := vars["entityID"]

	depth := 3
	if d := r.URL.Query().Get("depth"); d != "" {
		fmt.Sscanf(d, "%d", &depth)
	}

	paths, err := s.memSvc.Traverse(entityID, depth)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(paths)
}

// ==================== Search Handlers ====================

func (s *APIServer) searchHandler(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		http.Error(w, "missing query param 'q'", http.StatusBadRequest)
		return
	}
	if len(query) > 1000 {
		http.Error(w, "query too long (max 1000 chars)", http.StatusBadRequest)
		return
	}

	limit := 10
	if l := r.URL.Query().Get("limit"); l != "" {
		fmt.Sscanf(l, "%d", &limit)
	}
	if limit < 1 {
		limit = 1
	}
	if limit > 100 {
		limit = 100
	}

	threshold := float32(0.5)
	if t := r.URL.Query().Get("threshold"); t != "" {
		var f float64
		fmt.Sscanf(t, "%f", &f)
		threshold = float32(f)
	}

	memType := r.URL.Query().Get("memory_type")
	if err := validateMemoryType(memType); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	req := &types.SearchRequest{
		Query:      query,
		Limit:      limit,
		Threshold:  threshold,
		MemoryType: types.MemoryType(memType),
		UserID:     r.URL.Query().Get("user_id"),
		OrgID:      r.URL.Query().Get("org_id"),
		AgentID:    r.URL.Query().Get("agent_id"),
		Category:   r.URL.Query().Get("category"),
	}

	results, err := s.memSvc.SearchMemories(context.Background(), req)
	if err != nil {
		http.Error(w, "Search failed", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(results)
}

func (s *APIServer) searchPostHandler(w http.ResponseWriter, r *http.Request) {
	var req types.SearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := validateMemoryType(string(req.MemoryType)); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	results, err := s.memSvc.SearchMemories(context.Background(), &req)
	if err != nil {
		http.Error(w, "Search failed", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(results)
}

func (s *APIServer) advancedSearchHandler(w http.ResponseWriter, r *http.Request) {
	var req types.SearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	results, err := s.memSvc.AdvancedSearch(context.Background(), &req)
	if err != nil {
		http.Error(w, "Search failed", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(results)
}

// ==================== Memory Handlers ====================

func (s *APIServer) createMemoryHandler(w http.ResponseWriter, r *http.Request) {
	var mem types.Memory
	if err := json.NewDecoder(r.Body).Decode(&mem); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if mem.Content == "" {
		http.Error(w, "content is required", http.StatusBadRequest)
		return
	}
	if err := validateMemoryType(string(mem.Type)); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	tenantID := getTenantID(r)
	if tenantID != "" {
		mem.TenantID = tenantID
	}

	created, err := s.memSvc.CreateMemory(context.Background(), &mem)
	if err != nil {
		http.Error(w, "Failed to create memory", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(created)
}

func (s *APIServer) inferMemoryHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Content string `json:"content"`
		UserID  string `json:"user_id"`
		Type    string `json:"type"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Content == "" {
		http.Error(w, "content is required", http.StatusBadRequest)
		return
	}

	if req.Type == "" {
		req.Type = "user"
	}

	result, err := s.memSvc.InferMemoryContent(context.Background(), req.Content, req.UserID, types.MemoryType(req.Type))
	if err != nil {
		http.Error(w, "Failed to infer memory content", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(result)
}

func (s *APIServer) processMemoryHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Content        string `json:"content"`
		UserID         string `json:"user_id"`
		Type           string `json:"type"`
		SkipProcessing bool   `json:"skip_processing"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Content == "" {
		http.Error(w, "content is required", http.StatusBadRequest)
		return
	}

	if req.Type == "" {
		req.Type = "user"
	}

	mem := &types.Memory{
		Content: req.Content,
		UserID:  req.UserID,
		Type:    types.MemoryType(req.Type),
	}

	created, err := s.memSvc.CreateMemoryWithOptions(context.Background(), mem, req.SkipProcessing)
	if err != nil {
		http.Error(w, "Failed to process memory", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(created)
}

func (s *APIServer) listMemoriesHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	orgID := r.URL.Query().Get("org_id")
	agentID := r.URL.Query().Get("agent_id")
	category := r.URL.Query().Get("category")

	var memories []*types.Memory
	var err error

	if userID != "" {
		memories, err = s.memSvc.GetMemoriesByUser(context.Background(), userID)
	} else if orgID != "" {
		memories, err = s.memSvc.GetMemoriesByOrg(context.Background(), orgID)
	} else {
		memories = []*types.Memory{}
		err = nil
	}

	if err != nil {
		http.Error(w, "Failed to list memories", http.StatusInternalServerError)
		return
	}

	if agentID != "" {
		var filtered []*types.Memory
		for _, m := range memories {
			if m.AgentID == agentID {
				filtered = append(filtered, m)
			}
		}
		memories = filtered
	}

	if category != "" {
		var filtered []*types.Memory
		for _, m := range memories {
			if m.Category == category {
				filtered = append(filtered, m)
			}
		}
		memories = filtered
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"memories": memories,
		"count":    len(memories),
	})
}

func (s *APIServer) getMemoryHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	memoryID := vars["memoryID"]

	mem, err := s.memSvc.GetMemory(context.Background(), memoryID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(mem)
}

func (s *APIServer) updateMemoryHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	memoryID := vars["memoryID"]

	var req struct {
		Content  string                 `json:"content"`
		Metadata map[string]interface{} `json:"metadata"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Content == "" {
		http.Error(w, "content is required", http.StatusBadRequest)
		return
	}

	if err := s.memSvc.UpdateMemory(context.Background(), memoryID, req.Content, req.Metadata); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	mem, _ := s.memSvc.GetMemory(context.Background(), memoryID)
	json.NewEncoder(w).Encode(mem)
}

func (s *APIServer) deleteMemoryHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	memoryID := vars["memoryID"]

	if err := s.memSvc.DeleteMemory(context.Background(), memoryID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
}

func (s *APIServer) getMemoryHistoryHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	memoryID := vars["memoryID"]

	history, err := s.memSvc.GetMemoryHistory(context.Background(), memoryID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(history)
}

func (s *APIServer) setMemoryExpirationHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	memoryID := vars["memoryID"]

	var req struct {
		ExpirationDate string `json:"expiration_date"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	expDate, err := time.Parse(time.RFC3339, req.ExpirationDate)
	if err != nil {
		http.Error(w, "Invalid date format, use RFC3339", http.StatusBadRequest)
		return
	}

	if err := s.memSvc.SetMemoryExpiration(context.Background(), memoryID, expDate); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
}

func (s *APIServer) linkMemoryEntityHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	memoryID := vars["memoryID"]
	entityID := vars["entityID"]

	if err := s.memSvc.LinkMemoryToEntity(context.Background(), memoryID, entityID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"status": "linked"})
}

// ==================== Batch Handlers ====================

func (s *APIServer) batchCreateMemoriesHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Memories []*types.Memory `json:"memories"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if len(req.Memories) > 1000 {
		http.Error(w, "Maximum 1000 memories per batch", http.StatusBadRequest)
		return
	}

	tenantID := getTenantID(r)
	for _, mem := range req.Memories {
		if tenantID != "" {
			mem.TenantID = tenantID
		}
	}

	created, err := s.memSvc.BatchCreateMemories(context.Background(), req.Memories)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"created": created,
		"count":   len(created),
	})
}

func (s *APIServer) batchUpdateMemoriesHandler(w http.ResponseWriter, r *http.Request) {
	var req types.BatchUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if len(req.IDs) == 0 {
		http.Error(w, "ids are required", http.StatusBadRequest)
		return
	}
	if len(req.IDs) > 1000 {
		http.Error(w, "Maximum 1000 IDs per batch", http.StatusBadRequest)
		return
	}
	if req.Action == "" {
		http.Error(w, "action is required (update, archive, delete)", http.StatusBadRequest)
		return
	}

	if err := s.memSvc.BatchUpdateMemories(context.Background(), &req); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"status": "ok", "updated": fmt.Sprintf("%d", len(req.IDs))})
}

func (s *APIServer) batchDeleteMemoriesHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		IDs []string `json:"ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if len(req.IDs) == 0 {
		http.Error(w, "ids are required", http.StatusBadRequest)
		return
	}
	if len(req.IDs) > 1000 {
		http.Error(w, "Maximum 1000 IDs per batch", http.StatusBadRequest)
		return
	}

	if err := s.memSvc.DeleteMemories(context.Background(), req.IDs); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"status": "deleted", "count": fmt.Sprintf("%d", len(req.IDs))})
}

func (s *APIServer) bulkDeleteHandler(w http.ResponseWriter, r *http.Request) {
	var req types.BatchDeleteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.UserID == "" && req.OrgID == "" && req.Category == "" {
		http.Error(w, "At least one filter (user_id, org_id, or category) is required", http.StatusBadRequest)
		return
	}

	count, err := s.memSvc.BulkDeleteByFilter(context.Background(), &req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{"status": "deleted", "count": count})
}

// ==================== Feedback Handlers ====================

func (s *APIServer) createFeedbackHandler(w http.ResponseWriter, r *http.Request) {
	var feedback types.Feedback
	if err := json.NewDecoder(r.Body).Decode(&feedback); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if feedback.MemoryID == "" {
		http.Error(w, "memory_id is required", http.StatusBadRequest)
		return
	}
	if err := validateFeedbackType(string(feedback.Type)); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	created, err := s.memSvc.AddFeedback(context.Background(), &feedback)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(created)
}

func (s *APIServer) listFeedbackHandler(w http.ResponseWriter, r *http.Request) {
	memID := r.URL.Query().Get("memory_id")
	if memID != "" {
		history, _ := s.memSvc.GetMemoryHistory(context.Background(), memID)
		var feedback []types.MemoryHistory
		for _, h := range history {
			if h.Action == types.HistoryActionFeedback {
				feedback = append(feedback, h)
			}
		}
		json.NewEncoder(w).Encode(feedback)
		return
	}

	json.NewEncoder(w).Encode([]types.Feedback{})
}

func (s *APIServer) getMemoriesByFeedbackHandler(w http.ResponseWriter, r *http.Request) {
	fbType := r.URL.Query().Get("type")
	if err := validateFeedbackType(fbType); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		fmt.Sscanf(l, "%d", &limit)
	}

	memories, err := s.memSvc.GetMemoriesByFeedback(context.Background(), types.FeedbackType(fbType), limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(memories)
}

// ==================== Admin Handlers ====================

func (s *APIServer) cleanupExpiredHandler(w http.ResponseWriter, r *http.Request) {
	if !isAdmin(r) {
		http.Error(w, "Forbidden: Admin access required", http.StatusForbidden)
		return
	}

	count, err := s.memSvc.CleanupExpiredMemories(context.Background())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{"cleaned_up": count})
}

func (s *APIServer) syncHandler(w http.ResponseWriter, r *http.Request) {
	if !isAdmin(r) {
		http.Error(w, "Forbidden: Admin access required", http.StatusForbidden)
		return
	}

	var req struct {
		EntityIDs []string `json:"entity_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if len(req.EntityIDs) > 0 {
		if err := s.memSvc.BatchSyncEntities(req.EntityIDs); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	json.NewEncoder(w).Encode(map[string]string{"status": "synced"})
}

// ==================== API Key Management ====================

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
	keyMu       sync.RWMutex
)

func (s *APIServer) listAPIKeysHandler(w http.ResponseWriter, r *http.Request) {
	keyMu.RLock()
	defer keyMu.RUnlock()

	var keys []APIKey
	for _, k := range apiKeyStore {
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

	json.NewEncoder(w).Encode(map[string]interface{}{
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
	if _, err := rand.Read(b); err != nil {
		log.Fatalf("Failed to generate random string: %v", err)
	}
	for i := range b {
		b[i] = charset[int(b[i])%len(charset)]
	}
	return string(b)
}

// ==================== Helper Methods for Service ====================

func (s *APIServer) GetMemoriesByUser(ctx context.Context, userID string) ([]*types.Memory, error) {
	return s.memSvc.GetMemoriesByUser(ctx, userID)
}

func (s *APIServer) GetMemoriesByOrg(ctx context.Context, orgID string) ([]*types.Memory, error) {
	return s.memSvc.GetMemoriesByOrg(ctx, orgID)
}

func (s *APIServer) DeleteMemories(ctx context.Context, ids []string) error {
	return s.memSvc.DeleteMemories(ctx, ids)
}

func (s *APIServer) BulkDeleteByFilter(ctx context.Context, req *types.BatchDeleteRequest) (int, error) {
	return s.memSvc.BulkDeleteByFilter(ctx, req)
}

// ==================== Project Handlers ====================

func (s *APIServer) createProjectHandler(w http.ResponseWriter, r *http.Request) {
	var proj types.Project
	if err := json.NewDecoder(r.Body).Decode(&proj); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if proj.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}

	tenantID := getTenantID(r)
	if tenantID != "" {
		proj.UserID = tenantID
	}

	created, err := s.projSvc.CreateProject(r.Context(), &proj)
	if err != nil {
		http.Error(w, "Failed to create project", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(created)
}

func (s *APIServer) listProjectsHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	orgID := r.URL.Query().Get("org_id")

	projects := s.projSvc.ListProjects(userID, orgID)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"projects": projects,
		"count":    len(projects),
	})
}

func (s *APIServer) getProjectHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	projectID := vars["projectID"]

	proj, err := s.projSvc.GetProject(projectID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(proj)
}

func (s *APIServer) updateProjectHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	projectID := vars["projectID"]

	var updates types.Project
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	updated, err := s.projSvc.UpdateProject(r.Context(), projectID, &updates)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(updated)
}

func (s *APIServer) deleteProjectHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	projectID := vars["projectID"]

	if err := s.projSvc.DeleteProject(projectID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
}

// ==================== Webhook Handlers ====================

func (s *APIServer) createWebhookHandler(w http.ResponseWriter, r *http.Request) {
	var wh types.Webhook
	if err := json.NewDecoder(r.Body).Decode(&wh); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if wh.URL == "" {
		http.Error(w, "url is required", http.StatusBadRequest)
		return
	}
	if len(wh.Events) == 0 {
		http.Error(w, "events are required", http.StatusBadRequest)
		return
	}

	created, err := s.whSvc.CreateWebhook(r.Context(), &wh)
	if err != nil {
		http.Error(w, "Failed to create webhook", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(created)
}

func (s *APIServer) listWebhooksHandler(w http.ResponseWriter, r *http.Request) {
	projectID := r.URL.Query().Get("project_id")

	webhooks := s.whSvc.ListWebhooks(projectID)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"webhooks": webhooks,
		"count":    len(webhooks),
	})
}

func (s *APIServer) getWebhookHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	webhookID := vars["webhookID"]

	wh, err := s.whSvc.GetWebhook(webhookID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(wh)
}

func (s *APIServer) updateWebhookHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	webhookID := vars["webhookID"]

	var updates types.Webhook
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	updated, err := s.whSvc.UpdateWebhook(r.Context(), webhookID, &updates)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(updated)
}

func (s *APIServer) deleteWebhookHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	webhookID := vars["webhookID"]

	if err := s.whSvc.DeleteWebhook(webhookID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
}

func (s *APIServer) testWebhookHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	webhookID := vars["webhookID"]

	if err := s.whSvc.TestWebhook(r.Context(), webhookID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"status": "test_delivered"})
}

// ==================== Compaction Handlers ====================

func (s *APIServer) runCompactionHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID string `json:"user_id"`
		OrgID  string `json:"org_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.UserID == "" && req.OrgID == "" {
		http.Error(w, "user_id or org_id is required", http.StatusBadRequest)
		return
	}

	result, err := s.memSvc.RunCompaction(r.Context(), req.UserID, req.OrgID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(result)
}

func (s *APIServer) runTargetedCompactionHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		MemoryIDs []string `json:"memory_ids"`
		Action    string   `json:"action"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if len(req.MemoryIDs) == 0 {
		http.Error(w, "memory_ids is required", http.StatusBadRequest)
		return
	}
	if req.Action == "" {
		http.Error(w, "action is required (merge, summarize, archive, delete)", http.StatusBadRequest)
		return
	}

	result, err := s.memSvc.RunTargetedCompaction(r.Context(), req.MemoryIDs, req.Action)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(result)
}

func (s *APIServer) compactNegativeFeedbackHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Limit int `json:"limit"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Limit <= 0 {
		req.Limit = 50
	}

	result, err := s.memSvc.CompactNegativeFeedback(r.Context(), req.Limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(result)
}

func (s *APIServer) compactionStatusHandler(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]bool{"compaction_available": true})
}
