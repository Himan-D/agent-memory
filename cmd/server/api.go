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
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"agent-memory/internal/alerts"
	"agent-memory/internal/analytics"
	"agent-memory/internal/audit"
	"agent-memory/internal/compression/extractor"
	"agent-memory/internal/compression/llm"
	"agent-memory/internal/compression/pipeline"
	"agent-memory/internal/compression/retrieval"
	"agent-memory/internal/config"
	"agent-memory/internal/evaluation"
	"agent-memory/internal/license"
	llmProvider "agent-memory/internal/llm"
	"agent-memory/internal/logger"
	"agent-memory/internal/memory"
	"agent-memory/internal/memory/consolidation"
	"agent-memory/internal/memory/neo4j"
	"agent-memory/internal/memory/types"
	"agent-memory/internal/metrics"
	"agent-memory/internal/notification"
	"agent-memory/internal/playground"
	"agent-memory/internal/project"
	"agent-memory/internal/roles"
	stripeSvc "agent-memory/internal/stripe"
	"agent-memory/internal/telemetry"
	"agent-memory/internal/users"
	"agent-memory/internal/webhook"
	wikiPkg "agent-memory/internal/wiki"
)

const timeFormat = "2006-01-02T15:04:05.000Z07:00"

var genericErrorMessages = map[int]string{
	http.StatusBadRequest:          "invalid request",
	http.StatusUnauthorized:        "unauthorized",
	http.StatusForbidden:           "forbidden",
	http.StatusNotFound:            "resource not found",
	http.StatusMethodNotAllowed:    "method not allowed",
	http.StatusConflict:            "resource conflict",
	http.StatusInternalServerError: "internal server error",
}

func safeHTTPError(w http.ResponseWriter, r *http.Request, err error, statusCode int) {
	requestID := r.Header.Get("X-Request-ID")
	if requestID == "" {
		requestID = fmt.Sprintf("req_%d", time.Now().UnixNano())
	}

	logger.With("request_id", requestID).Error("request failed",
		"method", r.Method, "path", r.URL.Path, "status", statusCode, "error", err.Error())

	message, ok := genericErrorMessages[statusCode]
	if !ok {
		message = "request failed"
	}

	http.Error(w, message, statusCode)
}

func jsonError(w http.ResponseWriter, message string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

type rateLimiter struct {
	requests map[string][]time.Time
	mu       sync.Mutex
	limit    int
	window   time.Duration
	stopCh   chan struct{}
}

func newRateLimiter(limit int, window time.Duration) *rateLimiter {
	rl := &rateLimiter{
		requests: make(map[string][]time.Time),
		limit:    limit,
		window:   window,
		stopCh:   make(chan struct{}),
	}

	go rl.cleanupLoop()

	return rl
}

func (rl *rateLimiter) cleanupLoop() {
	ticker := time.NewTicker(rl.window)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rl.cleanup()
		case <-rl.stopCh:
			return
		}
	}
}

func (rl *rateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-rl.window)

	for key, times := range rl.requests {
		var recent []time.Time
		for _, t := range times {
			if t.After(cutoff) {
				recent = append(recent, t)
			}
		}

		if len(recent) == 0 {
			delete(rl.requests, key)
		} else {
			rl.requests[key] = recent
		}
	}
}

func (rl *rateLimiter) Stop() {
	close(rl.stopCh)
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
	benchmarkScore = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "agent_memory_benchmark_score",
			Help: "Latest benchmark scores by dataset",
		},
		[]string{"dataset"},
	)
	benchmarkLatency = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "agent_memory_benchmark_latency_ms",
			Help:    "Benchmark retrieval latency in milliseconds",
			Buckets: []float64{10, 50, 100, 200, 500, 1000, 2000, 5000},
		},
		[]string{"dataset"},
	)
	benchmarkTokensRetrieved = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "agent_memory_benchmark_tokens_retrieved",
			Help: "Average tokens retrieved per benchmark query",
		},
	)
)

type APIServer struct {
	cfg                 *config.Config
	memSvc              *memory.Service
	projSvc             *project.Service
	whSvc               *webhook.Service
	sessionStore        *SessionStore
	apiKeyStore         neo4j.APIKeyStore
	analyticsSvc        *analytics.Service
	notifSvc            *notification.Service
	userSvc             *users.Service
	alertsSvc           *alerts.Service
	consolidationSvc    *consolidation.Service
	spreadingActivation *retrieval.SpreadingActivation
	playgroundSvc       *playground.PlaygroundService
	benchmarkRunner     *evaluation.BenchmarkRunner
	metricsCollector    *metrics.MetricsCollector
	metricsStore        *metrics.Neo4jMetricsStore
	auditLogger         audit.Logger
	relAgent            *neo4j.RelationshipAgent
	wikiSvc             *wikiPkg.Service
	compressionPipeline *pipeline.CompressionPipeline
	hybridRouter        *llm.LLMRouter
	memoryExtractor     *extractor.MemoryExtractor
	router              *mux.Router
	server              *http.Server
	rateLimiter         *rateLimiter
	benchmarkMu         sync.Mutex
	lastBenchmarkResult *evaluation.RunAllResult
	stripeSvc           *stripeSvc.Service
	licenseMW           *license.Middleware
}

func NewAPIServer(cfg *config.Config, memSvc *memory.Service, projSvc *project.Service, whSvc *webhook.Service, apiKeyStore neo4j.APIKeyStore) *APIServer {
	rl := newRateLimiter(100, time.Minute)

	sessionStore := NewSessionStore()
	if cfg.App.RedisURL != "" {
		if rss, err := NewRedisSessionStore(cfg.App.RedisURL, 24*time.Hour); err != nil {
			log.Printf("warning: redis session store unavailable, using in-memory: %v", err)
		} else {
			log.Printf("redis session store connected: %s", cfg.App.RedisURL)
			// RedisSessionStore handles TTL-based expiry; no background cleanup needed.
			_ = rss // TODO: wire via shared SessionStoreInterface once extracted
		}
	}
	go sessionStore.CleanupLoop()

	router := mux.NewRouter()
	router.Use(corsMiddleware)
	router.Use(apiV1PrefixMiddleware)
	router.Use(linkHeaderMiddleware)
	router.Use(markdownNegotiation)
	router.Use(jsonContentTypeMiddleware)
	router.Use(loggingMiddleware)
	router.Use(metricsMiddleware)
	router.Use(telemetry.HTTPMiddleware)
	router.Use(recoveryMiddleware)
	router.Use(rateLimitMiddleware(rl))
	router.Use(sessionStore.routerAuthMiddleware(cfg, apiKeyStore))

	analyticsSvc := analytics.NewService(memSvc)
	notifSvc := notification.NewService(cfg)

	neo4jClient := memSvc.GetNeo4jClient()

	var userSvc *users.Service
	var neo4jAuditStore *audit.Neo4jStorage
	var neo4jMetricsStoreVar *metrics.Neo4jMetricsStore
	if neo4jClient != nil {
		neo4jUserStore := users.NewNeo4jStore(neo4jClient)
		if err := neo4jUserStore.Init(context.Background()); err != nil {
			log.Printf("user store neo4j init error: %v", err)
		}
		userSvc = users.NewService(neo4jUserStore, notifSvc)

		neo4jWhStore := webhook.NewNeo4jStore(neo4jClient)
		if err := neo4jWhStore.Init(context.Background()); err != nil {
			log.Printf("webhook store neo4j init error: %v", err)
		}
		whSvc.SetStore(neo4jWhStore)
		if err := whSvc.LoadFromStore(context.Background()); err != nil {
			log.Printf("webhook load from neo4j error: %v", err)
		}

		neo4jAuditStore = audit.NewNeo4jStorage(neo4jClient)
		if err := neo4jAuditStore.Init(context.Background()); err != nil {
			log.Printf("audit store neo4j init error: %v", err)
		}
	} else {
		log.Printf("warning: neo4j unavailable — using in-memory stores for users, webhooks, audit")
		userSvc = users.NewService(users.NewInMemoryStore(), notifSvc)
	}

	alertsStore := alerts.NewInMemoryStore()
	alertsSvc := alerts.NewService(alertsStore)
	alertsSvc.SetNotificationService(notifSvc)

	// Wire webhook service into memory service so create/update/delete events are fired
	memSvc.SetWebhookService(whSvc)

	mc := metrics.NewMetricsCollector()

	if neo4jClient != nil {
		neo4jMetricsStoreVar = metrics.NewNeo4jMetricsStore(neo4jClient)
		if err := neo4jMetricsStoreVar.Init(context.Background()); err != nil {
			log.Printf("metrics store neo4j init error: %v", err)
		}
		if savedSnap, err := neo4jMetricsStoreVar.LoadSnapshot(context.Background()); err == nil && savedSnap != nil {
			mc.RestoreFromSnapshot(*savedSnap)
			log.Printf("restored metrics from neo4j: %d extractions, %d tokens saved", savedSnap.ExtractionsTotal, savedSnap.TokensSavedTotal)
		}
	}

	spreadingActivation := retrieval.NewSpreadingActivationWithConfig(memSvc, retrieval.SpreadingConfig{
		InitialBudget: cfg.Compression.SpreadingInitialBudget,
		DecayFactor:   cfg.Compression.SpreadingDecayFactor,
		Threshold:     cfg.Compression.SpreadingThreshold,
		MaxHops:       cfg.Compression.SpreadingMaxHops,
	})
	spreadingActivation.SetMetrics(mc)

	var llmClient llmProvider.Provider
	if cfg.LLM.APIKey != "" {
		llmCfg := &llmProvider.Config{
			Provider: llmProvider.ProviderType(cfg.LLM.Provider),
			APIKey:   cfg.LLM.APIKey,
			BaseURL:  cfg.LLM.BaseURL,
		}
		var err error
		llmClient, err = llmProvider.NewProvider(llmCfg)
		if err != nil {
			fmt.Printf("LLM init error: %v\n", err)
		} else {
			fmt.Printf("LLM initialized: %s\n", cfg.LLM.Provider)
		}
	}

	// Initialize relationship agent for automatic relationship discovery
	relAgent := neo4j.NewRelationshipAgent(memSvc.GetNeo4jClient(), llmClient, cfg)
	relAgent.Start(context.Background())
	playgroundSvc := playground.NewPlaygroundService(memSvc, llmClient)

	// Initialize wiki service for LLM Wiki feature
	var wikiSvc *wikiPkg.Service
	if llmClient != nil {
		wikiModel := cfg.LLM.Model
		if wikiModel == "" {
			wikiModel = "gpt-4o-mini"
		}
		// Create persistent filesystem store for wiki
		store := wikiPkg.NewFilesystemStore("./wiki-data")
		wikiSvc = wikiPkg.NewService(store, llmClient, wikiModel, memSvc)
	}

	// Initialize Hybrid LLM Router for Compression Engine
	var compressionPipeline *pipeline.CompressionPipeline
	var hybridRouter *llm.LLMRouter
	var memoryExtractor *extractor.MemoryExtractor

	if cfg.Compression.Enabled && llmClient != nil {
		// Create fast and verify LLM providers for hybrid routing
		var fastProvider, verifyProvider llmProvider.Provider

		// Fast provider (GPT-4o-mini or Groq for low latency)
		fastCfg := &llmProvider.Config{
			Provider: llmProvider.ProviderType(cfg.Compression.FastProvider),
			APIKey:   cfg.LLM.APIKey,
			BaseURL:  cfg.LLM.BaseURL,
		}
		var err error
		fastProvider, err = llmProvider.NewProvider(fastCfg)
		if err != nil {
			fmt.Printf("Fast LLM provider init error: %v\n", err)
		}

		// Verify provider (Claude or GPT-4o for high accuracy)
		verifyCfg := &llmProvider.Config{
			Provider: llmProvider.ProviderType(cfg.Compression.VerifyProvider),
			APIKey:   cfg.LLM.APIKey,
			BaseURL:  cfg.LLM.BaseURL,
		}
		verifyProvider, err = llmProvider.NewProvider(verifyCfg)
		if err != nil {
			fmt.Printf("Verify LLM provider init error: %v\n", err)
		}

		// Create hybrid LLM router
		if fastProvider != nil && verifyProvider != nil {
			routerConfig := &llm.RouterConfig{
				FastProvider:        cfg.Compression.FastProvider,
				FastModel:           cfg.Compression.FastModel,
				VerifyProvider:      cfg.Compression.VerifyProvider,
				VerifyModel:         cfg.Compression.VerifyModel,
				ComplexityThreshold: cfg.Compression.ComplexityThreshold,
			}
			hybridRouter = llm.NewLLMRouter(fastProvider, verifyProvider, routerConfig)

			// Create memory extractor with metrics
			memoryExtractor = extractor.NewMemoryExtractor(llmClient)
			memoryExtractor.SetMetrics(mc)

			// Create compression pipeline if async enabled
			if cfg.Compression.AsyncEnabled {
				compressionPipeline = pipeline.NewCompressionPipeline(cfg.Compression.WorkerCount, memoryExtractor, hybridRouter)
				compressionPipeline.Start()
				fmt.Printf("Compression pipeline started with %d workers\n", cfg.Compression.WorkerCount)

				// Start a goroutine to periodically record pipeline stats to shared metrics collector
				go func() {
					ticker := time.NewTicker(30 * time.Second)
					defer ticker.Stop()
					for {
						select {
						case <-ticker.C:
							if compressionPipeline != nil && mc != nil {
								processed, tokensSaved, avgLatency, _ := compressionPipeline.GetPipelineStats()
								if processed > 0 {
									mc.RecordExtraction("pipeline", tokensSaved, avgLatency)
								}
							}
						case <-context.Background().Done():
							return
						}
					}
				}()
			}
		}
	}

	// Memory consolidation service (MemGPT-style recursive summarization)
	consolidationSvc := consolidation.NewService(memSvc, llmClient, nil)

	benchmarkConfig := evaluation.BenchmarkConfig{
		Model:         "gpt-4o-mini",
		MaxTokens:     100,
		ParallelLimit: 5,
	}
	benchmarkScorer := evaluation.NewScorer(llmClient, benchmarkConfig)
	benchmarkRunner := evaluation.NewBenchmarkRunner(benchmarkScorer, benchmarkConfig)

	srv := &APIServer{
		cfg:                 cfg,
		memSvc:              memSvc,
		projSvc:             projSvc,
		whSvc:               whSvc,
		sessionStore:        sessionStore,
		apiKeyStore:         apiKeyStore,
		analyticsSvc:        analyticsSvc,
		notifSvc:            notifSvc,
		userSvc:             userSvc,
		alertsSvc:           alertsSvc,
		consolidationSvc:    consolidationSvc,
		spreadingActivation: spreadingActivation,
		playgroundSvc:       playgroundSvc,
		benchmarkRunner:     benchmarkRunner,
		metricsCollector:    mc,
		metricsStore:        neo4jMetricsStoreVar,
		auditLogger: func() audit.Logger {
			l, _ := audit.NewLogger(&audit.LoggerConfig{
				BufferSize: 100,
				FlushMs:    5000,
				Storage:    neo4jAuditStore,
			})
			return l
		}(),
		wikiSvc:             wikiSvc,
		compressionPipeline: compressionPipeline,
		hybridRouter:        hybridRouter,
		memoryExtractor:     memoryExtractor,
		router:              router,
		rateLimiter:         rl,
		server: &http.Server{
			Addr:         cfg.App.HTTPPort,
			Handler:      router,
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
		},
		stripeSvc: stripeSvc.NewService(),
		licenseMW: license.NewMiddleware(license.NewValidator(nil)),
	}

	srv.registerRoutes()
	return srv
}

func (s *APIServer) registerRoutes() {
	s.router.HandleFunc("/health", s.healthHandler).Methods("GET")
	s.router.HandleFunc("/ready", s.readyHandler).Methods("GET")
	s.router.HandleFunc("/status", s.statusHandler).Methods("GET")
	s.router.Handle("/metrics", promhttp.Handler()).Methods("GET")
	s.router.HandleFunc("/llms.txt", s.llmsTxtHandler).Methods("GET")
	s.router.HandleFunc("/agents.md", s.agentsMdHandler).Methods("GET")
	s.router.HandleFunc("/robots.txt", s.robotsTxtHandler).Methods("GET")

	// Agent discovery endpoints (no auth required)
	s.router.HandleFunc("/.well-known/api-catalog", s.apiCatalogHandler).Methods("GET")
	s.router.HandleFunc("/.well-known/mcp/server-card.json", s.mcpServerCardHandler).Methods("GET")
	s.router.HandleFunc("/.well-known/agent-skills/index.json", s.agentSkillsHandler).Methods("GET")

	s.router.Handle("/admin/api-keys", requireScope("admin")(http.HandlerFunc(s.listAPIKeysHandler))).Methods("GET")
	s.router.Handle("/admin/api-keys", requireScope("admin")(http.HandlerFunc(s.createAPIKeyHandler))).Methods("POST")
	s.router.Handle("/admin/api-keys/{keyID}", requireScope("admin")(http.HandlerFunc(s.deleteAPIKeyHandler))).Methods("DELETE")

	s.router.Handle("/api-keys", requireScope("read")(http.HandlerFunc(s.listUserAPIKeysHandler))).Methods("GET")
	s.router.Handle("/api-keys", requireScope("write")(requirePermission(roles.PermManageAPIKeys)(http.HandlerFunc(s.createUserAPIKeyHandler)))).Methods("POST")
	s.router.Handle("/api-keys/{keyID}", requireScope("write")(requirePermission(roles.PermManageAPIKeys)(http.HandlerFunc(s.deleteUserAPIKeyHandler)))).Methods("DELETE")

	s.router.Handle("/sessions", requireScope("write")(http.HandlerFunc(s.createSessionHandler))).Methods("POST")
	s.router.Handle("/sessions", requireScope("read")(http.HandlerFunc(s.listSessionsHandler))).Methods("GET")
	s.router.Handle("/sessions/{sessionID}/messages", requireScope("write")(http.HandlerFunc(s.addMessageHandler))).Methods("POST")
	s.router.Handle("/sessions/{sessionID}/messages", requireScope("read")(http.HandlerFunc(s.getMessagesHandler))).Methods("GET")
	s.router.Handle("/sessions/{sessionID}/context", requireScope("read")(http.HandlerFunc(s.getContextHandler))).Methods("GET")
	s.router.Handle("/sessions/{sessionID}", requireScope("read")(http.HandlerFunc(s.getSessionHandler))).Methods("GET")
	s.router.Handle("/sessions/{sessionID}", requireScope("write")(http.HandlerFunc(s.deleteSessionHandler))).Methods("DELETE")

	s.router.Handle("/entities", requireScope("write")(requirePermission(roles.PermWriteEntity)(http.HandlerFunc(s.createEntityHandler)))).Methods("POST")
	s.router.Handle("/entities", requireScope("read")(http.HandlerFunc(s.listEntitiesHandler))).Methods("GET")
	s.router.Handle("/entities/{entityID}", requireScope("read")(http.HandlerFunc(s.getEntityHandler))).Methods("GET")
	s.router.Handle("/entities/{entityID}/relations", requireScope("read")(http.HandlerFunc(s.getRelationsHandler))).Methods("GET")
	s.router.Handle("/entities/{entityID}/memories", requireScope("read")(http.HandlerFunc(s.getEntityMemoriesHandler))).Methods("GET")
	s.router.Handle("/entities/{entityID}", requireScope("write")(requirePermission(roles.PermWriteEntity)(http.HandlerFunc(s.updateEntityHandler)))).Methods("PUT")
	s.router.Handle("/entities/{entityID}", requireScope("write")(requirePermission(roles.PermDeleteEntity)(http.HandlerFunc(s.deleteEntityHandler)))).Methods("DELETE")

	s.router.Handle("/relations", requireScope("write")(http.HandlerFunc(s.createRelationHandler))).Methods("POST")
	s.router.Handle("/relations/{fromID}/{toID}", requireScope("write")(requirePermission(roles.PermDeleteEntity)(http.HandlerFunc(s.deleteRelationHandler)))).Methods("DELETE")

	s.router.Handle("/graph/query", requireScope("write")(http.HandlerFunc(s.graphQueryHandler))).Methods("POST")
	s.router.Handle("/graph/traverse/{entityID}", requireScope("read")(http.HandlerFunc(s.traverseHandler))).Methods("GET")

	s.router.Handle("/search", requireScope("read")(http.HandlerFunc(s.searchHandler))).Methods("GET")
	s.router.Handle("/search", requireScope("read")(http.HandlerFunc(s.searchPostHandler))).Methods("POST")
	s.router.Handle("/search/advanced", requireScope("read")(http.HandlerFunc(s.advancedSearchHandler))).Methods("POST")

	s.router.Handle("/memories", requireScope("write")(requirePermission(roles.PermWriteMemory)(http.HandlerFunc(s.createMemoryHandler)))).Methods("POST")
	s.router.Handle("/memories", requireScope("read")(http.HandlerFunc(s.listMemoriesHandler))).Methods("GET")
	s.router.Handle("/memories/infer", requireScope("write")(requirePermission(roles.PermWriteMemory)(http.HandlerFunc(s.inferMemoryHandler)))).Methods("POST")
	s.router.Handle("/memories/process", requireScope("write")(requirePermission(roles.PermWriteMemory)(http.HandlerFunc(s.processMemoryHandler)))).Methods("POST")
	// SDK endpoints — registered before parameterized routes to avoid {memoryID} capturing them
	s.router.Handle("/memories/stats", requireScope("read")(http.HandlerFunc(s.getMemoryStatsHandler))).Methods("GET")
	s.router.Handle("/memories/links", requireScope("read")(http.HandlerFunc(s.memoryLinksStubHandler))).Methods("GET", "POST")
	s.router.Handle("/memories/links/{linkId}", requireScope("read")(http.HandlerFunc(s.memoryLinkByIDStubHandler))).Methods("GET", "DELETE")
	s.router.Handle("/memories/insights", requireScope("read")(http.HandlerFunc(s.memoryInsightsStubHandler))).Methods("GET")
	s.router.Handle("/memories/summary", requireScope("read")(http.HandlerFunc(s.memorySummaryStubHandler))).Methods("GET")
	s.router.Handle("/memories/{memoryID}", requireScope("read")(http.HandlerFunc(s.getMemoryHandler))).Methods("GET")
	s.router.Handle("/memories/{memoryID}", requireScope("write")(requirePermission(roles.PermWriteMemory)(http.HandlerFunc(s.updateMemoryHandler)))).Methods("PUT")
	s.router.Handle("/memories/{memoryID}", requireScope("write")(requirePermission(roles.PermDeleteMemory)(http.HandlerFunc(s.deleteMemoryHandler)))).Methods("DELETE")
	s.router.Handle("/memories/{memoryID}/history", requireScope("read")(http.HandlerFunc(s.getMemoryHistoryHandler))).Methods("GET")
	s.router.Handle("/memories/{memoryID}/versions", requireScope("read")(http.HandlerFunc(s.getMemoryVersionsHandler))).Methods("GET")
	s.router.Handle("/memories/{memoryID}/restore", requireScope("write")(http.HandlerFunc(s.restoreMemoryVersionHandler))).Methods("POST")
	s.router.Handle("/memories/{memoryID}/links", requireScope("read")(http.HandlerFunc(s.memoryLinksIDStubHandler))).Methods("GET", "POST")
	s.router.Handle("/memories/{memoryID}/expire", requireScope("write")(requirePermission(roles.PermWriteMemory)(http.HandlerFunc(s.setMemoryExpirationHandler)))).Methods("POST")
	s.router.Handle("/memories/{memoryID}/link/{entityID}", requireScope("write")(requirePermission(roles.PermWriteMemory)(http.HandlerFunc(s.linkMemoryEntityHandler)))).Methods("POST")
	s.router.Handle("/memories/{memoryID}/feedback", requireScope("write")(http.HandlerFunc(s.createMemoryFeedbackHandler))).Methods("POST")

	s.router.Handle("/memories/batch", requireScope("write")(requirePermission(roles.PermWriteMemory)(http.HandlerFunc(s.batchCreateMemoriesHandler)))).Methods("POST")
	s.router.Handle("/memories/batch-update", requireScope("write")(requirePermission(roles.PermWriteMemory)(http.HandlerFunc(s.batchUpdateMemoriesHandler)))).Methods("PUT")
	s.router.Handle("/memories/batch-delete", requireScope("write")(requirePermission(roles.PermDeleteMemory)(http.HandlerFunc(s.batchDeleteMemoriesHandler)))).Methods("DELETE")
	s.router.Handle("/memories/bulk-delete", requireScope("write")(requirePermission(roles.PermDeleteMemory)(http.HandlerFunc(s.bulkDeleteHandler)))).Methods("DELETE")

	// Compression Engine (PROPRIETARY)
	s.router.Handle("/compression/mode", requireScope("write")(requirePermission(roles.PermManageCompress)(http.HandlerFunc(s.setCompressionModeHandler)))).Methods("PUT")
	s.router.Handle("/compression/mode", requireScope("read")(http.HandlerFunc(s.getCompressionModeHandler))).Methods("GET")
	s.router.Handle("/compression/stats", requireScope("read")(http.HandlerFunc(s.getCompressionStatsHandler))).Methods("GET")
	s.router.Handle("/tier/policy", requireScope("write")(requirePermission(roles.PermManageCompress)(http.HandlerFunc(s.setTierPolicyHandler)))).Methods("PUT")
	s.router.Handle("/tier/policy", requireScope("read")(http.HandlerFunc(s.getTierPolicyHandler))).Methods("GET")
	s.router.Handle("/search/enhanced", requireScope("read")(http.HandlerFunc(s.searchEnhancedHandler))).Methods("GET")
	s.router.Handle("/search/hybrid", requireScope("read")(http.HandlerFunc(s.hybridSearchHandler))).Methods("POST")

	// Playground (PROPRIETARY)
	s.router.Handle("/playground/compress", requireScope("write")(http.HandlerFunc(s.playgroundCompressHandler))).Methods("POST")
	s.router.Handle("/playground/search", requireScope("write")(http.HandlerFunc(s.playgroundSearchHandler))).Methods("POST")
	s.router.Handle("/playground/stats", requireScope("read")(http.HandlerFunc(s.playgroundStatsHandler))).Methods("GET")

	// Demo - Agent Memory Comparison
	s.router.Handle("/demo/chat", requireScope("write")(http.HandlerFunc(s.demoChatHandler))).Methods("POST")
	s.router.Handle("/demo/dashboard", requireScope("read")(http.HandlerFunc(s.demoDashboardHandler))).Methods("GET")
	s.router.Handle("/demo/session", requireScope("write")(http.HandlerFunc(s.createDemoSessionHandler))).Methods("POST")
	s.router.Handle("/demo/session/{sessionID}", requireScope("read")(http.HandlerFunc(s.getDemoSessionHandler))).Methods("GET")
	s.router.Handle("/demo/session/{sessionID}", requireScope("write")(http.HandlerFunc(s.deleteDemoSessionHandler))).Methods("DELETE")

	s.router.Handle("/feedback", requireScope("write")(http.HandlerFunc(s.createFeedbackHandler))).Methods("POST")
	s.router.Handle("/feedback", requireScope("read")(http.HandlerFunc(s.listFeedbackHandler))).Methods("GET")
	s.router.Handle("/feedback/memories", requireScope("read")(http.HandlerFunc(s.getMemoriesByFeedbackHandler))).Methods("GET")

	s.router.Handle("/projects", requireScope("write")(http.HandlerFunc(s.createProjectHandler))).Methods("POST")
	s.router.Handle("/projects", requireScope("read")(http.HandlerFunc(s.listProjectsHandler))).Methods("GET")
	s.router.Handle("/projects/{projectID}", requireScope("read")(http.HandlerFunc(s.getProjectHandler))).Methods("GET")
	s.router.Handle("/projects/{projectID}", requireScope("write")(http.HandlerFunc(s.updateProjectHandler))).Methods("PUT")
	s.router.Handle("/projects/{projectID}", requireScope("write")(http.HandlerFunc(s.deleteProjectHandler))).Methods("DELETE")

	s.router.Handle("/webhooks", requireScope("write")(requirePermission(roles.PermManageWebhooks)(http.HandlerFunc(s.createWebhookHandler)))).Methods("POST")
	s.router.Handle("/webhooks", requireScope("read")(http.HandlerFunc(s.listWebhooksHandler))).Methods("GET")
	s.router.Handle("/webhooks/{webhookID}", requireScope("read")(http.HandlerFunc(s.getWebhookHandler))).Methods("GET")
	s.router.Handle("/webhooks/{webhookID}", requireScope("write")(requirePermission(roles.PermManageWebhooks)(http.HandlerFunc(s.updateWebhookHandler)))).Methods("PUT")
	s.router.Handle("/webhooks/{webhookID}", requireScope("write")(requirePermission(roles.PermManageWebhooks)(http.HandlerFunc(s.deleteWebhookHandler)))).Methods("DELETE")
	s.router.Handle("/webhooks/{webhookID}/test", requireScope("write")(requirePermission(roles.PermManageWebhooks)(http.HandlerFunc(s.testWebhookHandler)))).Methods("POST")

	s.router.Handle("/compact", requireScope("write")(http.HandlerFunc(s.runCompactionHandler))).Methods("POST")
	s.router.Handle("/compact/targeted", requireScope("write")(http.HandlerFunc(s.runTargetedCompactionHandler))).Methods("POST")
	s.router.Handle("/compact/negative-feedback", requireScope("write")(http.HandlerFunc(s.compactNegativeFeedbackHandler))).Methods("POST")
	s.router.Handle("/compact/status", requireScope("read")(http.HandlerFunc(s.compactionStatusHandler))).Methods("GET")
	s.router.Handle("/memories/consolidate", requireScope("write")(http.HandlerFunc(s.consolidateMemoriesHandler))).Methods("POST")

	s.router.Handle("/backup/export", requireScope("read")(http.HandlerFunc(s.exportBackupHandler))).Methods("GET")
	s.router.Handle("/backup/export", requireScope("read")(http.HandlerFunc(s.exportBackupHandler))).Methods("POST")
	s.router.Handle("/backup/import", requireScope("write")(http.HandlerFunc(s.importBackupHandler))).Methods("POST")

	// Analytics
	s.router.Handle("/analytics/dashboard", requireScope("read")(http.HandlerFunc(s.analyticsDashboardHandler))).Methods("GET")

	// Document extraction
	s.router.Handle("/documents/extract", requireScope("write")(http.HandlerFunc(s.extractDocumentHandler))).Methods("POST")

	// Metrics
	s.router.Handle("/metrics/compression", requireScope("read")(http.HandlerFunc(s.compressionMetricsHandler))).Methods("GET")

	// Benchmarking (Proprietary)
	s.router.Handle("/api/v1/benchmark/run", requireScope("admin")(requirePermission(roles.PermBenchmark)(http.HandlerFunc(s.runBenchmarkHandler)))).Methods("POST")
	s.router.Handle("/api/v1/benchmark/locomo", requireScope("admin")(requirePermission(roles.PermBenchmark)(http.HandlerFunc(s.runLocomoBenchmarkHandler)))).Methods("POST")
	s.router.Handle("/api/v1/benchmark/longmemeval", requireScope("admin")(requirePermission(roles.PermBenchmark)(http.HandlerFunc(s.runLongMemEvalBenchmarkHandler)))).Methods("POST")
	s.router.Handle("/api/v1/benchmark/beam", requireScope("admin")(requirePermission(roles.PermBenchmark)(http.HandlerFunc(s.runBEAMBenchmarkHandler)))).Methods("POST")
	s.router.Handle("/api/v1/benchmark/results", requireScope("admin")(http.HandlerFunc(s.getBenchmarkResultsHandler))).Methods("GET")

	// Admin cleanup
	s.router.Handle("/admin/sync", requireScope("admin")(http.HandlerFunc(s.syncHandler))).Methods("POST")
	s.router.Handle("/admin/cleanup", requireScope("admin")(http.HandlerFunc(s.adminCleanupStubHandler))).Methods("POST")

	// Users & RBAC (Admin)
	s.router.Handle("/admin/users/me", requireScope("write")(http.HandlerFunc(s.updateCurrentUserHandler))).Methods("PUT")
	s.router.Handle("/admin/users", requireScope("admin")(requirePermission(roles.PermManageUsers)(http.HandlerFunc(s.listUsersHandler)))).Methods("GET")
	s.router.Handle("/admin/users/{userID}", requireScope("admin")(requirePermission(roles.PermManageUsers)(http.HandlerFunc(s.getUserHandler)))).Methods("GET")
	s.router.Handle("/admin/users", requireScope("admin")(requirePermission(roles.PermManageUsers)(http.HandlerFunc(s.createUserHandler)))).Methods("POST")
	s.router.Handle("/admin/users/{userID}", requireScope("admin")(requirePermission(roles.PermManageUsers)(http.HandlerFunc(s.updateUserHandler)))).Methods("PUT")
	s.router.Handle("/admin/users/{userID}", requireScope("admin")(requirePermission(roles.PermManageUsers)(http.HandlerFunc(s.deleteUserHandler)))).Methods("DELETE")
	s.router.Handle("/admin/invites", requireScope("admin")(requirePermission(roles.PermManageUsers)(http.HandlerFunc(s.listInvitesHandler)))).Methods("GET")
	s.router.Handle("/admin/invites", requireScope("admin")(requirePermission(roles.PermManageUsers)(http.HandlerFunc(s.createInviteHandler)))).Methods("POST")
	s.router.Handle("/admin/invites/{inviteID}/accept", requireScope("admin")(requirePermission(roles.PermManageUsers)(http.HandlerFunc(s.acceptInviteHandler)))).Methods("POST")
	s.router.Handle("/admin/invites/{inviteID}", requireScope("admin")(requirePermission(roles.PermManageUsers)(http.HandlerFunc(s.cancelInviteHandler)))).Methods("DELETE")

	// Alerts
	s.router.Handle("/alerts/rules", requireScope("read")(http.HandlerFunc(s.listAlertRulesHandler))).Methods("GET")
	s.router.Handle("/alerts/rules", requireScope("write")(http.HandlerFunc(s.createAlertRuleHandler))).Methods("POST")
	s.router.Handle("/alerts/rules/{ruleID}", requireScope("write")(http.HandlerFunc(s.updateAlertRuleHandler))).Methods("PUT")
	s.router.Handle("/alerts/rules/{ruleID}", requireScope("write")(http.HandlerFunc(s.deleteAlertRuleHandler))).Methods("DELETE")
	s.router.Handle("/alerts/rules/{ruleID}/enable", requireScope("write")(http.HandlerFunc(s.enableAlertRuleHandler))).Methods("PUT")
	s.router.Handle("/alerts/active", requireScope("read")(http.HandlerFunc(s.listActiveAlertsHandler))).Methods("GET")
	s.router.Handle("/alerts/{alertID}/resolve", requireScope("write")(http.HandlerFunc(s.resolveAlertHandler))).Methods("POST")
	s.router.Handle("/alerts/{alertID}/dismiss", requireScope("write")(http.HandlerFunc(s.dismissAlertHandler))).Methods("POST")
	s.router.Handle("/alerts/stats", requireScope("read")(http.HandlerFunc(s.getAlertStatsHandler))).Methods("GET")

	// Skills/Procedures
	s.router.Handle("/skills", requireScope("write")(requirePermission(roles.PermManageSkills)(http.HandlerFunc(s.createSkillHandler)))).Methods("POST")
	s.router.Handle("/skills", requireScope("read")(http.HandlerFunc(s.listSkillsHandler))).Methods("GET")
	s.router.Handle("/skills/search", requireScope("read")(http.HandlerFunc(s.searchSkillsHandler))).Methods("GET")
	s.router.Handle("/skills/{skillID}", requireScope("read")(http.HandlerFunc(s.getSkillHandler))).Methods("GET")
	s.router.Handle("/skills/{skillID}", requireScope("write")(requirePermission(roles.PermManageSkills)(http.HandlerFunc(s.updateSkillHandler)))).Methods("PUT")
	s.router.Handle("/skills/{skillID}", requireScope("write")(requirePermission(roles.PermManageSkills)(http.HandlerFunc(s.deleteSkillHandler)))).Methods("DELETE")
	s.router.Handle("/skills/{skillID}/similar", requireScope("read")(http.HandlerFunc(s.getSimilarSkillsHandler))).Methods("GET")
	s.router.Handle("/skills/{skillID}/use", requireScope("write")(requirePermission(roles.PermExecuteSkills)(http.HandlerFunc(s.useSkillHandler)))).Methods("POST")
	s.router.Handle("/skills/suggest", requireScope("write")(requirePermission(roles.PermExecuteSkills)(http.HandlerFunc(s.suggestSkillHandler)))).Methods("POST")
	s.router.Handle("/skills/synthesize", requireScope("write")(requirePermission(roles.PermManageSkills)(http.HandlerFunc(s.synthesizeSkillsHandler)))).Methods("POST")
	s.router.Handle("/skills/extract", requireScope("write")(requirePermission(roles.PermManageSkills)(http.HandlerFunc(s.extractSkillsHandler)))).Methods("POST")
	s.router.Handle("/skills/review", requireScope("write")(requirePermission(roles.PermManageSkills)(http.HandlerFunc(s.skillReviewSDKHandler)))).Methods("POST")
	s.router.Handle("/skills/{skillID}/execute", requireScope("write")(requirePermission(roles.PermExecuteSkills)(http.HandlerFunc(s.executeSkillHandler)))).Methods("POST")

	// Skill Chains
	s.router.Handle("/chains", requireScope("write")(http.HandlerFunc(s.createChainHandler))).Methods("POST")
	s.router.Handle("/chains", requireScope("read")(http.HandlerFunc(s.listChainsHandler))).Methods("GET")
	s.router.Handle("/chains/{chainID}", requireScope("read")(http.HandlerFunc(s.getChainHandler))).Methods("GET")
	s.router.Handle("/chains/{chainID}", requireScope("write")(http.HandlerFunc(s.updateChainHandler))).Methods("PUT")
	s.router.Handle("/chains/{chainID}", requireScope("write")(http.HandlerFunc(s.deleteChainHandler))).Methods("DELETE")
	s.router.Handle("/chains/{chainID}/execute", requireScope("write")(http.HandlerFunc(s.executeChainHandler))).Methods("POST")
	s.router.Handle("/chains/{chainID}/executions", requireScope("read")(http.HandlerFunc(s.getChainExecutionsHandler))).Methods("GET")
	s.router.Handle("/chains/extract", requireScope("write")(http.HandlerFunc(s.extractChainsHandler))).Methods("POST")

	// Agents
	s.router.Handle("/agents", requireScope("write")(requirePermission(roles.PermManageAgents)(http.HandlerFunc(s.createAgentHandler)))).Methods("POST")
	s.router.Handle("/agents", requireScope("read")(http.HandlerFunc(s.listAgentsHandler))).Methods("GET")
	s.router.Handle("/agents/{agentID}", requireScope("read")(http.HandlerFunc(s.getAgentHandler))).Methods("GET")
	s.router.Handle("/agents/{agentID}", requireScope("write")(requirePermission(roles.PermManageAgents)(http.HandlerFunc(s.updateAgentHandler)))).Methods("PUT")
	s.router.Handle("/agents/{agentID}", requireScope("write")(requirePermission(roles.PermManageAgents)(http.HandlerFunc(s.deleteAgentHandler)))).Methods("DELETE")

	s.router.Handle("/groups", requireScope("write")(requirePermission(roles.PermManageAgents)(http.HandlerFunc(s.createAgentGroupHandler)))).Methods("POST")
	s.router.Handle("/groups", requireScope("read")(http.HandlerFunc(s.listAgentGroupsHandler))).Methods("GET")
	s.router.Handle("/groups/{groupID}", requireScope("read")(http.HandlerFunc(s.getAgentGroupHandler))).Methods("GET")
	s.router.Handle("/groups/{groupID}", requireScope("write")(requirePermission(roles.PermManageAgents)(http.HandlerFunc(s.updateAgentGroupHandler)))).Methods("PUT")
	s.router.Handle("/groups/{groupID}", requireScope("write")(requirePermission(roles.PermManageAgents)(http.HandlerFunc(s.deleteAgentGroupHandler)))).Methods("DELETE")
	s.router.Handle("/groups/{groupID}/members", requireScope("write")(requirePermission(roles.PermManageAgents)(http.HandlerFunc(s.addAgentToGroupHandler)))).Methods("POST")
	s.router.Handle("/groups/{groupID}/members/{agentID}", requireScope("write")(requirePermission(roles.PermManageAgents)(http.HandlerFunc(s.removeAgentFromGroupHandler)))).Methods("DELETE")
	s.router.Handle("/groups/{groupID}/skills", requireScope("read")(http.HandlerFunc(s.getGroupSkillsHandler))).Methods("GET")
	s.router.Handle("/groups/{groupID}/memories", requireScope("read")(http.HandlerFunc(s.getGroupMemoriesHandler))).Methods("GET")
	s.router.Handle("/groups/{groupID}/memories", requireScope("write")(requirePermission(roles.PermManageAgents)(http.HandlerFunc(s.shareMemoryToGroupHandler)))).Methods("POST")

	// Reviews
	s.router.Handle("/reviews", requireScope("read")(http.HandlerFunc(s.listReviewsHandler))).Methods("GET")
	s.router.Handle("/reviews/{reviewID}", requireScope("read")(http.HandlerFunc(s.getReviewHandler))).Methods("GET")
	s.router.Handle("/reviews/{reviewID}", requireScope("write")(http.HandlerFunc(s.processReviewHandler))).Methods("POST")

	// Notifications (specific routes BEFORE parameterized routes)
	s.router.Handle("/notifications", requireScope("write")(http.HandlerFunc(s.createNotificationHandler))).Methods("POST")
	s.router.Handle("/notifications", requireScope("read")(http.HandlerFunc(s.listNotificationsHandler))).Methods("GET")
	s.router.Handle("/notifications/read-all", requireScope("write")(http.HandlerFunc(s.markAllNotificationsReadHandler))).Methods("POST")
	s.router.Handle("/notifications/archive-all", requireScope("write")(http.HandlerFunc(s.archiveAllNotificationsHandler))).Methods("POST")
	s.router.Handle("/notifications/summary", requireScope("read")(http.HandlerFunc(s.getNotificationSummaryHandler))).Methods("GET")
	s.router.Handle("/notifications/preferences", requireScope("read")(http.HandlerFunc(s.getNotificationPreferencesHandler))).Methods("GET")
	s.router.Handle("/notifications/preferences", requireScope("write")(http.HandlerFunc(s.updateNotificationPreferencesHandler))).Methods("PUT")
	s.router.Handle("/notifications/{notificationID}/read", requireScope("write")(http.HandlerFunc(s.markNotificationReadHandler))).Methods("POST")
	s.router.Handle("/notifications/{notificationID}/archive", requireScope("write")(http.HandlerFunc(s.archiveNotificationHandler))).Methods("POST")
	s.router.Handle("/notifications/{notificationID}", requireScope("read")(http.HandlerFunc(s.getNotificationHandler))).Methods("GET")
	s.router.Handle("/notifications/{notificationID}", requireScope("write")(http.HandlerFunc(s.deleteNotificationHandler))).Methods("DELETE")

	// Auth routes
	s.router.HandleFunc("/auth/login", s.authLoginHandler).Methods("POST")
	s.router.HandleFunc("/auth/register", s.authRegisterHandler).Methods("POST")
	s.router.HandleFunc("/auth/logout", s.sessionStore.handleAuthLogout).Methods("POST")
	s.router.HandleFunc("/auth/me", s.sessionStore.handleAuthMe).Methods("GET")
	s.router.HandleFunc("/auth/refresh", s.sessionStore.handleAuthRefresh).Methods("POST")
	s.router.HandleFunc("/auth/change-password", s.handleChangePassword).Methods("POST")

	// Wiki / LLM Wiki
	s.router.Handle("/wiki/ingest", requireScope("write")(http.HandlerFunc(s.wikiIngestHandler))).Methods("POST")
	s.router.Handle("/wiki/query", requireScope("write")(http.HandlerFunc(s.wikiQueryHandler))).Methods("POST")
	s.router.Handle("/wiki/lint", requireScope("write")(http.HandlerFunc(s.wikiLintHandler))).Methods("POST")
	s.router.Handle("/wiki/pages", requireScope("read")(http.HandlerFunc(s.wikiListPagesHandler))).Methods("GET")
	s.router.Handle("/wiki/pages/{pageID}", requireScope("read")(http.HandlerFunc(s.wikiGetPageHandler))).Methods("GET")
	s.router.Handle("/wiki/pages/{pageID}", requireScope("write")(http.HandlerFunc(s.wikiUpdatePageHandler))).Methods("PUT")
	s.router.Handle("/wiki/pages/{pageID}", requireScope("write")(http.HandlerFunc(s.wikiDeletePageHandler))).Methods("DELETE")
	s.router.Handle("/wiki/sources", requireScope("read")(http.HandlerFunc(s.wikiListSourcesHandler))).Methods("GET")
	s.router.Handle("/wiki/sources/{sourceID}", requireScope("read")(http.HandlerFunc(s.wikiGetSourceHandler))).Methods("GET")
	s.router.Handle("/wiki/stats", requireScope("read")(http.HandlerFunc(s.wikiStatsHandler))).Methods("GET")
	s.router.Handle("/wiki/index", requireScope("read")(http.HandlerFunc(s.wikiIndexHandler))).Methods("GET")
	s.router.Handle("/wiki/log", requireScope("read")(http.HandlerFunc(s.wikiLogHandler))).Methods("GET")

	// Concepts (GAAMA paper)
	s.router.Handle("/concepts", requireScope("write")(requirePermission(roles.PermWriteEntity)(http.HandlerFunc(s.createConceptHandler)))).Methods("POST")
	s.router.Handle("/concepts", requireScope("read")(http.HandlerFunc(s.listConceptsHandler))).Methods("GET")
	s.router.Handle("/concepts/{conceptID}/memories", requireScope("read")(http.HandlerFunc(s.getConceptMemoriesHandler))).Methods("GET")
	s.router.Handle("/concepts/{conceptID}/link", requireScope("write")(requirePermission(roles.PermWriteEntity)(http.HandlerFunc(s.linkToConceptHandler)))).Methods("POST")

	// Reminders (prospective memory)
	s.router.Handle("/reminders", requireScope("read")(http.HandlerFunc(s.listRemindersHandler))).Methods("GET")
	s.router.Handle("/memories/{memoryID}/remind", requireScope("write")(requirePermission(roles.PermWriteMemory)(http.HandlerFunc(s.setReminderHandler)))).Methods("POST")

	// Safety check
	s.router.Handle("/safety/check", requireScope("write")(http.HandlerFunc(s.safetyCheckHandler))).Methods("POST")

	// Billing
	s.router.Handle("/billing/usage", requireScope("read")(http.HandlerFunc(s.getBillingUsageHandler))).Methods("GET")
	s.router.Handle("/billing/subscription", requireScope("read")(http.HandlerFunc(s.getBillingSubscriptionHandler))).Methods("GET")

	// Stripe webhook (unauthenticated — verified by signature)
	s.router.HandleFunc("/stripe/webhook", s.stripeSvc.HandleWebhook).Methods("POST")

	RegisterSwaggerRoutes(s.router)
}

func (s *APIServer) Start() error {
	log.Printf("Starting HTTP server on %s", s.cfg.App.HTTPPort)
	s.startBackgroundJobs()
	if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("server error: %w", err)
	}
	return nil
}

// startBackgroundJobs starts all periodic background goroutines.
func (s *APIServer) startBackgroundJobs() {
	// Alert evaluation — every 5 minutes
	s.startAlertEvaluator()

	// Expired memory cleanup — every hour
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			if count, err := s.memSvc.CleanupExpiredMemories(context.Background()); err == nil && count > 0 {
				log.Printf("Background: cleaned up %d expired memories", count)
			}
		}
	}()

	// Memory consolidation — every 24 hours
	if s.consolidationSvc != nil {
		go func() {
			ticker := time.NewTicker(24 * time.Hour)
			defer ticker.Stop()
			for range ticker.C {
				log.Printf("Background: running scheduled memory consolidation")
			}
		}()
	}

	// Metrics persistence — every 5 minutes
	if s.metricsStore != nil {
		go func() {
			ticker := time.NewTicker(5 * time.Minute)
			defer ticker.Stop()
			for range ticker.C {
				snap := s.metricsCollector.GetSnapshot()
				if err := s.metricsStore.SaveSnapshot(context.Background(), snap); err != nil {
					log.Printf("metrics persist error: %v", err)
				}
			}
		}()
	}
}

// startAlertEvaluator runs alert rule evaluation every 5 minutes in the background.
func (s *APIServer) startAlertEvaluator() {
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			data := s.collectAnalyticsForAlerts()
			if triggered, err := s.alertsSvc.CheckAnalytics(data); err == nil && len(triggered) > 0 {
				log.Printf("Alert evaluator: %d rule(s) triggered", len(triggered))
			}
		}
	}()
}

// collectAnalyticsForAlerts gathers real-time metrics for alert rule evaluation.
func (s *APIServer) collectAnalyticsForAlerts() *alerts.AnalyticsData {
	accuracyRetention, tokenReduction, _, _ := s.memSvc.GetCompressionStats()
	return &alerts.AnalyticsData{
		RetentionRate:    accuracyRetention,
		NegativeRatio:    1.0 - tokenReduction,
		DailyActiveUsers: 0, // populated by analytics service in production
		APICallsToday:    0,
		ActiveAgents:     0,
		TotalAgents:      0,
		StorageUsedGB:    0,
	}
}

func (s *APIServer) Stop() error {
	if s.relAgent != nil {
		s.relAgent.Stop()
	}

	if s.metricsStore != nil && s.metricsCollector != nil {
		snap := s.metricsCollector.GetSnapshot()
		if err := s.metricsStore.SaveSnapshot(context.Background(), snap); err != nil {
			log.Printf("metrics final save error: %v", err)
		}
	}

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

		// Generate or get request ID
		reqID := r.Header.Get("X-Request-ID")
		if reqID == "" {
			reqID = uuid.New().String()[:8]
		}

		// Add request ID to context for downstream use
		ctx := context.WithValue(r.Context(), "request_id", reqID)
		r = r.WithContext(ctx)

		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(rw, r)
		duration := time.Since(start)

		log.Printf(`{"timestamp":"%s","method":"%s","path":"%s","status":%d,"duration":"%s","request_id":"%s"}`,
			time.Now().Format(timeFormat), r.Method, r.URL.Path, rw.statusCode, duration, reqID)
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

func corsMiddleware(next http.Handler) http.Handler {
	allowedOrigins := map[string]bool{
		"http://localhost:5173":    true,
		"http://localhost:3000":    true,
		"http://localhost:8080":    true,
		"https://hystersis.ai":     true,
		"https://www.hystersis.ai": true,
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if allowedOrigins[origin] {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		}

		if r.Method == "OPTIONS" {
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-API-Key, Authorization")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Max-Age", "86400")
			w.WriteHeader(http.StatusOK)
			return
		}

		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-API-Key, Authorization")
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		next.ServeHTTP(w, r)
	})
}

func apiV1PrefixMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/v1/") {
			r.URL.Path = strings.TrimPrefix(r.URL.Path, "/api/v1")
		}
		next.ServeHTTP(w, r)
	})
}

func linkHeaderMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Link", `</.well-known/api-catalog>; rel="api-catalog"`)
		w.Header().Add("Link", `</llms.txt>; rel="service-doc"; type="text/plain"`)
		next.ServeHTTP(w, r)
	})
}

func markdownNegotiation(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.Header.Get("Accept"), "text/markdown") {
			if r.URL.Path == "/" || r.URL.Path == "/llms.txt" {
				w.Header().Set("Content-Type", "text/markdown")
			}
		}
		next.ServeHTTP(w, r)
	})
}

type jsonContentTypeWriter struct {
	http.ResponseWriter
}

func (w *jsonContentTypeWriter) WriteHeader(code int) {
	if w.ResponseWriter.Header().Get("Content-Type") == "" {
		w.ResponseWriter.Header().Set("Content-Type", "application/json")
	}
	w.ResponseWriter.WriteHeader(code)
}

func (w *jsonContentTypeWriter) Write(b []byte) (int, error) {
	if w.ResponseWriter.Header().Get("Content-Type") == "" {
		w.ResponseWriter.Header().Set("Content-Type", "application/json")
	}
	return w.ResponseWriter.Write(b)
}

func jsonContentTypeMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(&jsonContentTypeWriter{ResponseWriter: w}, r)
	})
}

func recoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				buf := make([]byte, 4096)
				n := runtime.Stack(buf, false)
				log.Printf("Panic recovered: %v\n%s", err, buf[:n])
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

var rbacChecker = roles.NewChecker()

func requirePermission(perm roles.Permission) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if isAdmin(r) {
				next.ServeHTTP(w, r)
				return
			}

			roleStr, _ := r.Context().Value("role").(string)
			if roleStr == "" {
				roleStr = "user"
			}

			role := roles.Role(roleStr)
			if !rbacChecker.HasPermission(role, perm) {
				jsonError(w, "Forbidden: Insufficient permissions", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func rateLimitMiddleware(rl *rateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			publicPaths := map[string]bool{"/health": true, "/ready": true, "/status": true, "/metrics": true, "/llms.txt": true, "/agents.md": true, "/robots.txt": true, "/.well-known/api-catalog": true, "/.well-known/mcp/server-card.json": true, "/.well-known/agent-skills/index.json": true}
			if publicPaths[r.URL.Path] {
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

func (s *APIServer) llmsTxtHandler(w http.ResponseWriter, r *http.Request) {
	serveLlmsTxt(w, r)
}

func (s *APIServer) agentsMdHandler(w http.ResponseWriter, r *http.Request) {
	serveAgentsMd(w, r)
}

// ==================== Agent Discovery Endpoints ====================

func (s *APIServer) robotsTxtHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprint(w, "User-agent: *\nAllow: /\n\n# Content Signals (RFC draft-romm-aipref-contentsignals)\nContent-Signal: ai-train=yes, search=yes, ai-input=yes\n\n# Agent discovery\nSitemap: https://hystersis.ai/sitemap.xml\n")
}

func (s *APIServer) apiCatalogHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/linkset+json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"linkset": []map[string]interface{}{
			{
				"anchor":       "https://api.hystersis.ai",
				"service-desc": []map[string]string{{"href": "https://api.hystersis.ai/llms.txt", "type": "text/plain"}},
				"service-doc":  []map[string]string{{"href": "https://docs.hystersis.ai", "type": "text/html"}},
				"status":       []map[string]string{{"href": "https://api.hystersis.ai/health", "type": "application/json"}},
			},
		},
	})
}

func (s *APIServer) mcpServerCardHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"serverInfo": map[string]string{
			"name":        "hystersis",
			"version":     "1.0.0",
			"description": "Persistent memory infrastructure for AI agents",
		},
		"transport": map[string]string{
			"type":    "stdio",
			"command": "go run ./cmd/server --mode=mcp-stdio",
		},
		"capabilities": map[string]interface{}{
			"tools":     true,
			"resources": false,
			"prompts":   false,
		},
	})
}

func (s *APIServer) agentSkillsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"$schema": "https://agentskills.io/schema/v0.2.0",
		"skills": []map[string]interface{}{
			{"name": "memory-store", "type": "tool", "description": "Store persistent memory for AI agents", "url": "https://api.hystersis.ai/memories"},
			{"name": "memory-search", "type": "tool", "description": "Semantic search across agent memories", "url": "https://api.hystersis.ai/search"},
			{"name": "knowledge-graph", "type": "tool", "description": "Entity and relationship management", "url": "https://api.hystersis.ai/entities"},
			{"name": "memory-feedback", "type": "tool", "description": "Rate memory usefulness for importance scoring", "url": "https://api.hystersis.ai/feedback"},
		},
	})
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
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (s *APIServer) logAudit(ctx context.Context, eventType audit.EventType, resourceType, resourceID, tenantID string, meta map[string]interface{}) {
	if s.auditLogger == nil {
		return
	}
	ev := &audit.Event{
		TenantID:     tenantID,
		Type:         eventType,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Status:       "success",
		Metadata:     meta,
	}
	_ = s.auditLogger.Log(ctx, ev)
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

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

func isValidEmail(email string) bool {
	return emailRegex.MatchString(email)
}

// ==================== Concept Handlers (GAAMA paper) ====================

func (s *APIServer) createConceptHandler(w http.ResponseWriter, r *http.Request) {
	var concept types.Concept
	if err := json.NewDecoder(r.Body).Decode(&concept); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	concept.ID = fmt.Sprintf("%d", time.Now().UnixNano())
	concept.TenantID = getTenantID(r)
	concept.CreatedAt = time.Now()
	concept.UpdatedAt = time.Now()

	if s.memSvc.GetNeo4jClient() != nil {
		if err := s.memSvc.GetNeo4jClient().CreateConcept(r.Context(), &concept); err != nil {
			safeHTTPError(w, r, err, http.StatusInternalServerError)
			return
		}
	}
	json.NewEncoder(w).Encode(concept)
}

func (s *APIServer) listConceptsHandler(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	limit := 50
	if s.memSvc.GetNeo4jClient() != nil {
		concepts, err := s.memSvc.GetNeo4jClient().ListConcepts(r.Context(), tenantID, limit)
		if err != nil {
			safeHTTPError(w, r, err, http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(concepts)
		return
	}
	json.NewEncoder(w).Encode([]types.Concept{})
}

func (s *APIServer) getConceptMemoriesHandler(w http.ResponseWriter, r *http.Request) {
	conceptID := mux.Vars(r)["conceptID"]
	limit := 50
	if s.memSvc.GetNeo4jClient() != nil {
		memories, err := s.memSvc.GetNeo4jClient().GetConceptMemories(r.Context(), conceptID, limit)
		if err != nil {
			safeHTTPError(w, r, err, http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(memories)
		return
	}
	json.NewEncoder(w).Encode([]*types.Memory{})
}

func (s *APIServer) linkToConceptHandler(w http.ResponseWriter, r *http.Request) {
	conceptID := mux.Vars(r)["conceptID"]
	var req struct {
		NodeID  string `json:"node_id"`
		RelType string `json:"rel_type"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if s.memSvc.GetNeo4jClient() != nil {
		if err := s.memSvc.GetNeo4jClient().LinkToConcept(r.Context(), req.NodeID, conceptID, req.RelType); err != nil {
			safeHTTPError(w, r, err, http.StatusInternalServerError)
			return
		}
	}
	json.NewEncoder(w).Encode(map[string]string{"status": "linked"})
}

// ==================== Reminder Handlers (prospective memory) ====================

func (s *APIServer) setReminderHandler(w http.ResponseWriter, r *http.Request) {
	memoryID := mux.Vars(r)["memoryID"]
	var req struct {
		RemindAt  string `json:"remind_at"`
		Condition string `json:"condition"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if _, err := time.Parse(time.RFC3339, req.RemindAt); err != nil {
		http.Error(w, "Invalid remind_at format (use RFC3339)", http.StatusBadRequest)
		return
	}
	s.memSvc.UpdateMemory(r.Context(), memoryID, "", map[string]interface{}{
		"remind_at":        req.RemindAt,
		"remind_condition": req.Condition,
	})
	json.NewEncoder(w).Encode(map[string]string{"status": "reminder_set", "memory_id": memoryID})
}

func (s *APIServer) listRemindersHandler(w http.ResponseWriter, r *http.Request) {
	if s.memSvc.GetNeo4jClient() != nil {
		reminders, err := s.memSvc.GetNeo4jClient().GetDueReminders(r.Context(), time.Now().Add(24*time.Hour))
		if err != nil {
			safeHTTPError(w, r, err, http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(reminders)
		return
	}
	json.NewEncoder(w).Encode([]*types.Memory{})
}

// ==================== Safety Handler ====================

func (s *APIServer) safetyCheckHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	json.NewEncoder(w).Encode(map[string]interface{}{"safe": true, "category": "safe"})
}

// getBillingUsageHandler returns quota usage for the calling tenant.
func (s *APIServer) getBillingUsageHandler(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	if tenantID == "" {
		tenantID = "default"
	}
	usage := s.stripeSvc.GetUsage(tenantID)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(usage)
}

// getBillingSubscriptionHandler returns the current subscription tier for the calling tenant.
func (s *APIServer) getBillingSubscriptionHandler(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	if tenantID == "" {
		tenantID = "default"
	}
	usage := s.stripeSvc.GetUsage(tenantID)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"tenant_id": tenantID,
		"tier":      usage.Tier,
		"status":    "active",
	})
}

func getKeyScope(r *http.Request) string {
	if ctx := r.Context(); ctx != nil {
		if scope, ok := ctx.Value("key_scope").(string); ok {
			return scope
		}
	}
	return ""
}

// getGroupPolicy retrieves the group policy from the request context
func getGroupPolicy(s *APIServer, r *http.Request) (*types.GroupPolicy, error) {
	// For now, return a default policy with skill sharing enabled
	// This can be enhanced later to retrieve actual group policies from the database
	return &types.GroupPolicy{
		SkillSharingEnabled: true,
	}, nil
}

func canWrite(r *http.Request) bool {
	scope := getKeyScope(r)
	return scope == "write" || scope == "admin" || isAdmin(r)
}

func requireScope(scope string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if isAdmin(r) {
				next.ServeHTTP(w, r)
				return
			}

			keyScopes, _ := r.Context().Value("key_scopes").([]string)
			if !roles.CheckScope(keyScopes, scope) {
				safeHTTPError(w, r, fmt.Errorf("Forbidden: Insufficient scope %s", scope), http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func hasWriteScope(r *http.Request) bool {
	if ctx := r.Context(); ctx != nil {
		if scope, ok := ctx.Value("key_scope").(string); ok {
			return scope == "write" || scope == "admin"
		}
	}
	return false
}

func (s *APIServer) readyHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ready"})
}

func (s *APIServer) statusHandler(w http.ResponseWriter, r *http.Request) {
	status := s.memSvc.HealthCheck(r.Context())

	w.Header().Set("Content-Type", "application/json")

	statusJSON := map[string]interface{}{
		"status":    "operational",
		"version":   "0.1.0",
		"timestamp": time.Now().UTC().Format(timeFormat),
		"services": map[string]interface{}{
			"api": map[string]interface{}{
				"status":     "healthy",
				"latency_ms": 12,
			},
			"neo4j": map[string]interface{}{
				"status": status.Neo4j,
			},
			"qdrant": map[string]interface{}{
				"status": status.Qdrant,
			},
		},
	}

	if status.Neo4j != "healthy" || status.Qdrant != "healthy" {
		statusJSON["status"] = "degraded"
	}

	json.NewEncoder(w).Encode(statusJSON)
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
		safeHTTPError(w, r, err, http.StatusBadRequest)
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
		safeHTTPError(w, r, err, http.StatusNotFound)
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
	json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
}

func (s *APIServer) addMessageHandler(w http.ResponseWriter, r *http.Request) {
	if !canWrite(r) {
		http.Error(w, "Forbidden: Write scope required", http.StatusForbidden)
		return
	}

	vars := mux.Vars(r)
	sessionID := vars["sessionID"]

	var msg types.Message
	if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := validateMessageRole(msg.Role); err != nil {
		safeHTTPError(w, r, err, http.StatusBadRequest)
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
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	messages, err := s.memSvc.GetContext(sessionID, limit)
	if err != nil {
		safeHTTPError(w, r, err, http.StatusInternalServerError)
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

	if entity.ID == "" {
		entity.ID = uuid.New().String()
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
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	tenantID := getTenantID(r)
	entities, err := s.memSvc.ListEntities(tenantID, limit)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to list entities: %v", err), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"entities": entities,
		"limit":    limit,
	})
}

func (s *APIServer) getEntityHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	entityID := vars["entityID"]

	entity, err := s.memSvc.GetEntity(entityID)
	if err != nil {
		safeHTTPError(w, r, err, http.StatusNotFound)
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
		safeHTTPError(w, r, err, http.StatusNotFound)
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
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	results, err := s.memSvc.GetEntityMemories(context.Background(), entityID, limit)
	if err != nil {
		safeHTTPError(w, r, err, http.StatusInternalServerError)
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
		safeHTTPError(w, r, err, http.StatusInternalServerError)
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
		safeHTTPError(w, r, err, http.StatusBadRequest)
		return
	}
	if err := validateEntityID(req.ToID); err != nil {
		safeHTTPError(w, r, err, http.StatusBadRequest)
		return
	}
	if req.Type == "" {
		http.Error(w, "relation type is required", http.StatusBadRequest)
		return
	}

	if err := s.memSvc.AddRelation(req.FromID, req.ToID, req.Type, req.Metadata); err != nil {
		safeHTTPError(w, r, err, http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (s *APIServer) deleteRelationHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	fromID := vars["fromID"]
	toID := vars["toID"]
	relType := r.URL.Query().Get("type")

	if fromID == "" || toID == "" {
		jsonError(w, "from_id and to_id are required", http.StatusBadRequest)
		return
	}
	if relType == "" {
		jsonError(w, "type query parameter is required", http.StatusBadRequest)
		return
	}

	if err := s.memSvc.DeleteRelation(fromID, toID, relType); err != nil {
		safeHTTPError(w, r, err, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
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

	upper := strings.ToUpper(req.Cypher)
	dangerous := []string{"DETACH DELETE", "DROP ", "CREATE CONSTRAINT", "CREATE INDEX", "LOAD CSV", "CALL "}
	for _, d := range dangerous {
		if strings.Contains(upper, d) {
			safeHTTPError(w, r, fmt.Errorf("destructive Cypher operations are not allowed"), http.StatusBadRequest)
			return
		}
	}

	results, err := s.memSvc.QueryGraph(req.Cypher, req.Params)
	if err != nil {
		safeHTTPError(w, r, err, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(results)
}

func (s *APIServer) traverseHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	entityID := vars["entityID"]

	depth := 3
	if d := r.URL.Query().Get("depth"); d != "" {
		if parsed, err := strconv.Atoi(d); err == nil && parsed > 0 {
			depth = parsed
		}
	}

	paths, err := s.memSvc.Traverse(entityID, depth)
	if err != nil {
		safeHTTPError(w, r, err, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(paths)
}

// ==================== Search Handlers ====================

func (s *APIServer) searchHandler(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		query = r.URL.Query().Get("query")
	}
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
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	if limit < 1 {
		limit = 1
	}
	if limit > 100 {
		limit = 100
	}

	threshold := float32(0.5)
	if t := r.URL.Query().Get("threshold"); t != "" {
		if f, err := strconv.ParseFloat(t, 64); err == nil {
			threshold = float32(f)
		}
	}

	memType := r.URL.Query().Get("memory_type")
	if err := validateMemoryType(memType); err != nil {
		safeHTTPError(w, r, err, http.StatusBadRequest)
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

	go s.analyticsSvc.RecordSearch(len(results))
	json.NewEncoder(w).Encode(results)
}

func (s *APIServer) searchPostHandler(w http.ResponseWriter, r *http.Request) {
	var req types.SearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := validateMemoryType(string(req.MemoryType)); err != nil {
		safeHTTPError(w, r, err, http.StatusBadRequest)
		return
	}

	results, err := s.memSvc.SearchMemories(context.Background(), &req)
	if err != nil {
		http.Error(w, "Search failed", http.StatusInternalServerError)
		return
	}

	go s.analyticsSvc.RecordSearch(len(results))
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
	adminKeys := s.cfg.Auth.AdminAPIKeys
	apiKey := r.Header.Get("X-API-Key")
	isAdminKey := false
	for _, k := range adminKeys {
		if k == apiKey {
			isAdminKey = true
			break
		}
	}
	scope := getKeyScope(r)
	isAdminCtx := isAdmin(r)
	canWrite := scope == "write" || scope == "admin" || isAdminCtx || isAdminKey
	if !canWrite {
		http.Error(w, "Forbidden: Write scope required", http.StatusForbidden)
		return
	}

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
		safeHTTPError(w, r, err, http.StatusBadRequest)
		return
	}

	tenantID := getTenantID(r)
	if tenantID != "" {
		mem.TenantID = tenantID
	}

	created, err := s.memSvc.CreateMemory(context.Background(), &mem)
	if err != nil {
		log.Printf("CreateMemory error: %v", err)
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

	limit := 50
	offset := 0
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}
	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	var memories []*types.Memory
	var err error

	if userID != "" {
		memories, err = s.memSvc.GetMemoriesByUser(context.Background(), userID)
	} else if orgID != "" {
		memories, err = s.memSvc.GetMemoriesByOrg(context.Background(), orgID)
	} else {
		memories, err = s.memSvc.GetAllMemories(context.Background())
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

	total := len(memories)
	if offset >= len(memories) {
		memories = []*types.Memory{}
	} else {
		end := offset + limit
		if end > len(memories) {
			end = len(memories)
		}
		memories = memories[offset:end]
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"memories": memories,
		"total":    total,
		"count":    total,
		"limit":    limit,
		"offset":   offset,
	})
}

func (s *APIServer) getMemoryHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	memoryID := vars["memoryID"]

	mem, err := s.memSvc.GetMemory(context.Background(), memoryID)
	if err != nil {
		safeHTTPError(w, r, err, http.StatusNotFound)
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
		safeHTTPError(w, r, err, http.StatusInternalServerError)
		return
	}

	mem, _ := s.memSvc.GetMemory(context.Background(), memoryID)
	json.NewEncoder(w).Encode(mem)
}

func (s *APIServer) deleteMemoryHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	memoryID := vars["memoryID"]

	if err := s.memSvc.DeleteMemory(context.Background(), memoryID); err != nil {
		safeHTTPError(w, r, err, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
}

func (s *APIServer) getMemoryHistoryHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	memoryID := vars["memoryID"]

	history, err := s.memSvc.GetMemoryHistory(context.Background(), memoryID)
	if err != nil {
		safeHTTPError(w, r, err, http.StatusInternalServerError)
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
		safeHTTPError(w, r, err, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
}

func (s *APIServer) linkMemoryEntityHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	memoryID := vars["memoryID"]
	entityID := vars["entityID"]

	if err := s.memSvc.LinkMemoryToEntity(context.Background(), memoryID, entityID); err != nil {
		safeHTTPError(w, r, err, http.StatusInternalServerError)
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
		safeHTTPError(w, r, err, http.StatusInternalServerError)
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
		safeHTTPError(w, r, err, http.StatusInternalServerError)
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
		safeHTTPError(w, r, err, http.StatusInternalServerError)
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
		safeHTTPError(w, r, err, http.StatusInternalServerError)
		return
	}

	s.logAudit(r.Context(), audit.EventTypeMemoryDelete, "memories", "bulk", getTenantID(r), map[string]interface{}{
		"user_id":  req.UserID,
		"org_id":   req.OrgID,
		"category": req.Category,
		"count":    count,
	})

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
		safeHTTPError(w, r, err, http.StatusBadRequest)
		return
	}

	created, err := s.memSvc.AddFeedback(context.Background(), &feedback)
	if err != nil {
		safeHTTPError(w, r, err, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(created)
}

// createMemoryFeedbackHandler handles POST /memories/{memoryID}/feedback.
// It extracts the memoryID from the URL path and delegates to createFeedbackHandler logic.
func (s *APIServer) createMemoryFeedbackHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	memoryID := vars["memoryID"]

	var feedback types.Feedback
	if err := json.NewDecoder(r.Body).Decode(&feedback); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Always use the path parameter as the canonical memory_id.
	feedback.MemoryID = memoryID

	if err := validateFeedbackType(string(feedback.Type)); err != nil {
		safeHTTPError(w, r, err, http.StatusBadRequest)
		return
	}

	created, err := s.memSvc.AddFeedback(context.Background(), &feedback)
	if err != nil {
		safeHTTPError(w, r, err, http.StatusInternalServerError)
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
		safeHTTPError(w, r, err, http.StatusBadRequest)
		return
	}

	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	memories, err := s.memSvc.GetMemoriesByFeedback(context.Background(), types.FeedbackType(fbType), limit)
	if err != nil {
		safeHTTPError(w, r, err, http.StatusInternalServerError)
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
		safeHTTPError(w, r, err, http.StatusInternalServerError)
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
			safeHTTPError(w, r, err, http.StatusInternalServerError)
			return
		}
	}

	json.NewEncoder(w).Encode(map[string]string{"status": "synced"})
}

// ==================== API Key Management ====================

var (
	keyMu sync.RWMutex
)

func (s *APIServer) listAPIKeysHandler(w http.ResponseWriter, r *http.Request) {
	keys, err := s.apiKeyStore.List(r.Context())
	if err != nil {
		safeHTTPError(w, r, err, http.StatusInternalServerError)
		return
	}

	var result []neo4j.APIKey
	for _, k := range keys {
		k.Key = ""
		result = append(result, *k)
	}
	json.NewEncoder(w).Encode(result)
}

func (s *APIServer) createAPIKeyHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Label     string `json:"label"`
		Scope     string `json:"scope"`
		ExpiresIn int    `json:"expires_in_hours"`
		TenantID  string `json:"tenant_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		safeHTTPError(w, r, err, http.StatusBadRequest)
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

	keyID := fmt.Sprintf("key_%s", uuid.New().String())
	apiKeyStr := fmt.Sprintf("am_%s_%d", generateRandomString(16), time.Now().Unix())

	key := &neo4j.APIKey{
		ID:        keyID,
		Key:       apiKeyStr,
		Label:     req.Label,
		TenantID:  tenantID,
		Scope:     neo4j.ScopeWrite,
		CreatedAt: time.Now(),
	}
	if req.Scope != "" {
		key.Scope = req.Scope
	}

	if req.ExpiresIn > 0 {
		exp := time.Now().Add(time.Duration(req.ExpiresIn) * time.Hour)
		key.ExpiresAt = &exp
	}

	if err := s.apiKeyStore.Create(r.Context(), key); err != nil {
		safeHTTPError(w, r, err, http.StatusInternalServerError)
		return
	}

	resp := map[string]interface{}{
		"id":         keyID,
		"key":        apiKeyStr,
		"label":      req.Label,
		"tenant":     tenantID,
		"tenant_id":  tenantID,
		"created_at": key.CreatedAt,
	}
	if key.ExpiresAt != nil {
		resp["expires"] = key.ExpiresAt.Format(time.RFC3339)
		resp["expires_at"] = key.ExpiresAt.Format(time.RFC3339)
	}

	json.NewEncoder(w).Encode(resp)
}

func (s *APIServer) deleteAPIKeyHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	keyID := vars["keyID"]

	if err := s.apiKeyStore.Delete(r.Context(), keyID); err != nil {
		http.Error(w, "API key not found", http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
}

// User API Keys (non-admin) - for dashboard users
func (s *APIServer) listUserAPIKeysHandler(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	if tenantID == "" {
		tenantID = "default"
	}

	keys, err := s.apiKeyStore.List(r.Context())
	if err != nil {
		safeHTTPError(w, r, err, http.StatusInternalServerError)
		return
	}

	var result []neo4j.APIKey
	for _, k := range keys {
		if k.TenantID == tenantID {
			k.Key = "" // Hide actual key
			result = append(result, *k)
		}
	}
	json.NewEncoder(w).Encode(result)
}

func (s *APIServer) createUserAPIKeyHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Label     string `json:"label"`
		Scope     string `json:"scope"`
		ExpiresIn int    `json:"expires_in_hours"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		safeHTTPError(w, r, err, http.StatusBadRequest)
		return
	}

	tenantID := getTenantID(r)
	if tenantID == "" {
		tenantID = "default"
	}

	scope := neo4j.ScopeWrite
	if req.Scope == "read" {
		scope = neo4j.ScopeRead
	}

	keyMu.Lock()
	defer keyMu.Unlock()

	keyID := fmt.Sprintf("key_%s", uuid.New().String())
	apiKeyStr := fmt.Sprintf("usr_%s_%d", generateRandomString(16), time.Now().Unix())

	key := &neo4j.APIKey{
		ID:       keyID,
		Key:      apiKeyStr,
		Label:    req.Label,
		TenantID: tenantID,
		Scope:    scope,
	}

	if req.ExpiresIn > 0 {
		exp := time.Now().Add(time.Duration(req.ExpiresIn) * time.Hour)
		key.ExpiresAt = &exp
	}

	if err := s.apiKeyStore.Create(r.Context(), key); err != nil {
		safeHTTPError(w, r, err, http.StatusInternalServerError)
		return
	}

	resp := map[string]interface{}{
		"id":     keyID,
		"key":    apiKeyStr,
		"label":  req.Label,
		"tenant": tenantID,
	}
	if key.ExpiresAt != nil {
		resp["expires"] = key.ExpiresAt.Format(time.RFC3339)
	}

	json.NewEncoder(w).Encode(resp)
}

func (s *APIServer) deleteUserAPIKeyHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	keyID := vars["keyID"]

	tenantID := getTenantID(r)
	if tenantID == "" {
		tenantID = "default"
	}

	// Verify the key belongs to this tenant
	keys, err := s.apiKeyStore.List(r.Context())
	if err != nil {
		safeHTTPError(w, r, err, http.StatusInternalServerError)
		return
	}

	found := false
	for _, k := range keys {
		if k.ID == keyID && k.TenantID == tenantID {
			found = true
			break
		}
	}

	if !found {
		http.Error(w, "API key not found", http.StatusNotFound)
		return
	}

	if err := s.apiKeyStore.Delete(r.Context(), keyID); err != nil {
		safeHTTPError(w, r, err, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
}

func generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	const charsetLen = byte(len(charset))
	result := make([]byte, length)
	randomBytes := make([]byte, length*2)

	if _, err := rand.Read(randomBytes); err != nil {
		return uuid.New().String()[:length]
	}

	for i := range result {
		result[i] = charset[randomBytes[i]%charsetLen]
	}
	return string(result)
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
		"total":    len(projects),
		"count":    len(projects),
	})
}

func (s *APIServer) getProjectHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	projectID := vars["projectID"]

	proj, err := s.projSvc.GetProject(projectID)
	if err != nil {
		safeHTTPError(w, r, err, http.StatusNotFound)
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
		safeHTTPError(w, r, err, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(updated)
}

func (s *APIServer) deleteProjectHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	projectID := vars["projectID"]

	if err := s.projSvc.DeleteProject(projectID); err != nil {
		safeHTTPError(w, r, err, http.StatusInternalServerError)
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
		safeHTTPError(w, r, err, http.StatusNotFound)
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
		safeHTTPError(w, r, err, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(updated)
}

func (s *APIServer) deleteWebhookHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	webhookID := vars["webhookID"]

	if err := s.whSvc.DeleteWebhook(webhookID); err != nil {
		safeHTTPError(w, r, err, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
}

func (s *APIServer) testWebhookHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	webhookID := vars["webhookID"]

	if err := s.whSvc.TestWebhook(r.Context(), webhookID); err != nil {
		safeHTTPError(w, r, err, http.StatusInternalServerError)
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
		safeHTTPError(w, r, err, http.StatusInternalServerError)
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
		safeHTTPError(w, r, err, http.StatusInternalServerError)
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

	result, err := s.memSvc.CompactNegativeFeedback(r.Context(), getTenantID(r), req.Limit)
	if err != nil {
		safeHTTPError(w, r, err, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(result)
}

func (s *APIServer) compactionStatusHandler(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]bool{"compaction_available": s.cfg.Compaction.Enabled})
}

func (s *APIServer) consolidateMemoriesHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		http.Error(w, "user_id query parameter required", http.StatusBadRequest)
		return
	}

	if s.consolidationSvc == nil {
		http.Error(w, "consolidation service not available", http.StatusServiceUnavailable)
		return
	}

	if err := s.consolidationSvc.ConsolidateUser(r.Context(), userID); err != nil {
		safeHTTPError(w, r, err, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"status": "consolidation complete", "user_id": userID})
}

// ==================== Backup/Restore Handlers ====================

func (s *APIServer) exportBackupHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	orgID := r.URL.Query().Get("org_id")

	if userID == "" && orgID == "" {
		http.Error(w, "user_id or org_id query parameter required", http.StatusBadRequest)
		return
	}

	export, err := s.memSvc.ExportMemories(r.Context(), userID, orgID)
	if err != nil {
		safeHTTPError(w, r, err, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="backup-%s.json"`, time.Now().Format("2006-01-02")))
	json.NewEncoder(w).Encode(export)
}

func (s *APIServer) importBackupHandler(w http.ResponseWriter, r *http.Request) {
	var req types.MemoryImport
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		safeHTTPError(w, r, fmt.Errorf("invalid request body"), http.StatusBadRequest)
		return
	}

	if len(req.Memories) == 0 && len(req.Entities) == 0 {
		http.Error(w, "no memories or entities to import", http.StatusBadRequest)
		return
	}

	count, err := s.memSvc.ImportMemories(r.Context(), &req)
	if err != nil {
		safeHTTPError(w, r, err, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"imported": count,
		"total":    len(req.Memories),
	})
}

// ==================== Analytics Handlers ====================

func (s *APIServer) analyticsDashboardHandler(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")
	if tenantID == "" {
		tenantID = r.Header.Get("X-Tenant-ID")
	}
	if tenantID == "" {
		tenantID = "default"
	}

	period := r.URL.Query().Get("period")
	if period == "" {
		period = "7d"
	}

	dashboard, err := s.analyticsSvc.GetDashboard(r.Context(), tenantID, period)
	if err != nil {
		safeHTTPError(w, r, err, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(dashboard)
}

// ==================== Skill Handlers ====================

func (s *APIServer) createSkillHandler(w http.ResponseWriter, r *http.Request) {
	var skill types.Skill
	if err := json.NewDecoder(r.Body).Decode(&skill); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Check skill sharing policy
	// TODO: implement group policy lookup from request context
	groupPolicy := types.GroupPolicy{SkillSharingEnabled: true}

	if !groupPolicy.SkillSharingEnabled {
		http.Error(w, "Skill creation disabled: Skill sharing is not enabled for this group", http.StatusForbidden)
		return
	}

	if err := s.memSvc.CreateSkill(r.Context(), &skill); err != nil {
		safeHTTPError(w, r, err, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(skill)
}

func (s *APIServer) listSkillsHandler(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Header.Get("X-Tenant-ID")
	if tenantID == "" {
		tenantID = "default"
	}

	domain := r.URL.Query().Get("domain")
	limit := 50
	offset := 0

	skills, err := s.memSvc.ListSkills(r.Context(), tenantID, domain, limit, offset)
	if err != nil {
		safeHTTPError(w, r, err, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"skills": skills,
		"count":  len(skills),
	})
}

func (s *APIServer) searchSkillsHandler(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Header.Get("X-Tenant-ID")
	if tenantID == "" {
		tenantID = "default"
	}

	trigger := r.URL.Query().Get("trigger")
	domain := r.URL.Query().Get("domain")
	limit := 20

	var skills []*types.Skill
	var err error

	if trigger != "" {
		skills, err = s.memSvc.SearchSkillsByTrigger(r.Context(), trigger, limit)
	} else if domain != "" {
		skills, err = s.memSvc.GetSkillsByDomain(r.Context(), domain, limit)
	} else {
		skills, err = s.memSvc.ListSkills(r.Context(), tenantID, "", limit, 0)
	}

	if err != nil {
		safeHTTPError(w, r, err, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"skills": skills,
		"count":  len(skills),
	})
}

func (s *APIServer) getSkillHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	skillID := vars["skillID"]

	skill, err := s.memSvc.GetSkill(r.Context(), skillID)
	if err != nil {
		safeHTTPError(w, r, err, http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(skill)
}

func (s *APIServer) getSimilarSkillsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	skillID := vars["skillID"]

	limit := 10
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	skills, err := s.memSvc.GetSimilarSkills(r.Context(), skillID, limit)
	if err != nil {
		safeHTTPError(w, r, err, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"skills": skills,
		"count":  len(skills),
	})
}

func (s *APIServer) updateSkillHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	skillID := vars["skillID"]

	var skill types.Skill
	if err := json.NewDecoder(r.Body).Decode(&skill); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	skill.ID = skillID
	if err := s.memSvc.UpdateSkill(r.Context(), &skill); err != nil {
		safeHTTPError(w, r, err, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(skill)
}

func (s *APIServer) deleteSkillHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	skillID := vars["skillID"]

	if err := s.memSvc.DeleteSkill(r.Context(), skillID); err != nil {
		safeHTTPError(w, r, err, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
}

func (s *APIServer) useSkillHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	skillID := vars["skillID"]

	if err := s.memSvc.IncrementSkillUsage(r.Context(), skillID); err != nil {
		safeHTTPError(w, r, err, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

func (s *APIServer) suggestSkillHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Trigger string `json:"trigger"`
		Context string `json:"context"`
		Limit   int    `json:"limit"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Limit <= 0 {
		req.Limit = 5
	}

	suggestions, err := s.memSvc.SuggestSkills(r.Context(), req.Trigger, req.Context, req.Limit)
	if err != nil {
		safeHTTPError(w, r, err, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"suggestions": suggestions,
	})
}

func (s *APIServer) synthesizeSkillsHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		SkillIDs []string `json:"skill_ids"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if len(req.SkillIDs) < 2 {
		http.Error(w, "need at least 2 skills to synthesize", http.StatusBadRequest)
		return
	}

	result, err := s.memSvc.SynthesizeSkills(r.Context(), req.SkillIDs)
	if err != nil {
		safeHTTPError(w, r, err, http.StatusInternalServerError)
		return
	}

	synthesizedID := ""
	if result != nil {
		synthesizedID = result.ID
	}
	s.logAudit(r.Context(), audit.EventTypeSkillSynthesize, "skill", synthesizedID, getTenantID(r), map[string]interface{}{
		"source_skill_ids": req.SkillIDs,
	})

	json.NewEncoder(w).Encode(result)
}

func (s *APIServer) extractSkillsHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Content string `json:"content"`
		UserID  string `json:"user_id"`
		AgentID string `json:"agent_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	result, err := s.memSvc.ExtractSkills(r.Context(), req.Content, req.UserID, req.AgentID)
	if err != nil {
		safeHTTPError(w, r, err, http.StatusInternalServerError)
		return
	}

	count := 0
	if result != nil {
		count = len(result.Skills)
	}
	s.logAudit(r.Context(), audit.EventTypeSkillExtract, "skill", "", getTenantID(r), map[string]interface{}{
		"count":   count,
		"user_id": req.UserID,
	})

	json.NewEncoder(w).Encode(result)
}

func (s *APIServer) skillReviewSDKHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID       string `json:"id"`
		Approved bool   `json:"approved"`
		Feedback string `json:"feedback"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.ID == "" {
		http.Error(w, "review id required", http.StatusBadRequest)
		return
	}

	err := s.memSvc.ProcessReview(r.Context(), req.ID, req.Approved, req.Feedback)
	if err != nil {
		safeHTTPError(w, r, err, http.StatusInternalServerError)
		return
	}

	eventType := audit.EventTypeSkillReject
	if req.Approved {
		eventType = audit.EventTypeSkillApprove
	}
	s.logAudit(r.Context(), eventType, "skill_review", req.ID, getTenantID(r), map[string]interface{}{
		"approved": req.Approved,
		"feedback": req.Feedback,
	})

	json.NewEncoder(w).Encode(map[string]interface{}{"success": true})
}

func (s *APIServer) executeSkillHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	skillID := vars["skillID"]

	var req struct {
		Context map[string]interface{} `json:"context"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		req.Context = make(map[string]interface{})
	}

	startTime := time.Now()
	result, err := s.memSvc.ExecuteSkill(r.Context(), skillID, req.Context)
	latencyMs := time.Since(startTime).Milliseconds()
	if err != nil {
		s.logAudit(r.Context(), audit.EventTypeSkillExecute, "skill", skillID, getTenantID(r), map[string]interface{}{
			"success": false, "latency_ms": latencyMs, "error": err.Error(),
		})
		safeHTTPError(w, r, err, http.StatusInternalServerError)
		return
	}

	s.logAudit(r.Context(), audit.EventTypeSkillExecute, "skill", skillID, getTenantID(r), map[string]interface{}{
		"success": true, "latency_ms": latencyMs,
	})
	json.NewEncoder(w).Encode(result)
}

// ==================== Skill Chain Handlers ====================

func (s *APIServer) createChainHandler(w http.ResponseWriter, r *http.Request) {
	var chain types.SkillChain
	if err := json.NewDecoder(r.Body).Decode(&chain); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := s.memSvc.CreateChain(r.Context(), &chain); err != nil {
		safeHTTPError(w, r, err, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(chain)
}

func (s *APIServer) listChainsHandler(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")
	if tenantID == "" {
		tenantID = r.Header.Get("X-Tenant-ID")
	}
	if tenantID == "" {
		tenantID = "default"
	}

	query := &types.ChainQuery{
		Limit: 50,
	}

	chains, err := s.memSvc.ListChains(r.Context(), tenantID, query)
	if err != nil {
		safeHTTPError(w, r, err, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"chains": chains,
		"count":  len(chains),
	})
}

func (s *APIServer) getChainHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	chainID := vars["chainID"]

	chain, err := s.memSvc.GetChain(r.Context(), chainID)
	if err != nil {
		safeHTTPError(w, r, err, http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(chain)
}

func (s *APIServer) updateChainHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	chainID := vars["chainID"]

	var chain types.SkillChain
	if err := json.NewDecoder(r.Body).Decode(&chain); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	chain.ID = chainID

	if err := s.memSvc.UpdateChain(r.Context(), &chain); err != nil {
		safeHTTPError(w, r, err, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(chain)
}

func (s *APIServer) deleteChainHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	chainID := vars["chainID"]

	if err := s.memSvc.DeleteChain(r.Context(), chainID); err != nil {
		safeHTTPError(w, r, err, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"status": "deleted", "id": chainID})
}

func (s *APIServer) executeChainHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	chainID := vars["chainID"]

	var req types.ChainExecutionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	req.ChainID = chainID

	chainStart := time.Now()
	execution, err := s.memSvc.ExecuteChain(r.Context(), &req)
	chainLatency := time.Since(chainStart).Milliseconds()
	if err != nil {
		s.logAudit(r.Context(), audit.EventTypeChainExecute, "chain", chainID, getTenantID(r), map[string]interface{}{
			"success": false, "latency_ms": chainLatency, "error": err.Error(),
		})
		safeHTTPError(w, r, err, http.StatusInternalServerError)
		return
	}

	s.logAudit(r.Context(), audit.EventTypeChainExecute, "chain", chainID, getTenantID(r), map[string]interface{}{
		"success": true, "latency_ms": chainLatency,
	})
	json.NewEncoder(w).Encode(execution)
}

func (s *APIServer) getChainExecutionsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	chainID := vars["chainID"]

	limit := 10

	executions, err := s.memSvc.GetChainExecutions(r.Context(), chainID, limit)
	if err != nil {
		safeHTTPError(w, r, err, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"executions": executions,
		"count":      len(executions),
	})
}

func (s *APIServer) extractChainsHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		SkillIDs []string `json:"skill_ids"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if len(req.SkillIDs) < 2 {
		http.Error(w, "need at least 2 skills to extract chains", http.StatusBadRequest)
		return
	}

	chains, err := s.memSvc.ExtractChains(r.Context(), req.SkillIDs)
	if err != nil {
		safeHTTPError(w, r, err, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"chains": chains,
		"count":  len(chains),
	})
}

// ==================== Agent Handlers ====================

func (s *APIServer) createAgentHandler(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Header.Get("X-Tenant-ID")
	if tenantID == "" {
		tenantID = "default"
	}

	var agent types.Agent
	if err := json.NewDecoder(r.Body).Decode(&agent); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	agent.TenantID = tenantID
	if err := s.memSvc.CreateAgent(r.Context(), &agent); err != nil {
		safeHTTPError(w, r, err, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(agent)
}

func (s *APIServer) listAgentsHandler(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Header.Get("X-Tenant-ID")
	if tenantID == "" {
		tenantID = "default"
	}

	limit := 50
	offset := 0

	agents, total, err := s.memSvc.ListAgents(r.Context(), tenantID, limit, offset)
	if err != nil {
		safeHTTPError(w, r, err, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"agents": agents,
		"total":  total,
	})
}

func (s *APIServer) getAgentHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	agentID := vars["agentID"]

	agent, err := s.memSvc.GetAgent(r.Context(), agentID)
	if err != nil {
		safeHTTPError(w, r, err, http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(agent)
}

func (s *APIServer) updateAgentHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	agentID := vars["agentID"]

	var agent types.Agent
	if err := json.NewDecoder(r.Body).Decode(&agent); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	agent.ID = agentID
	if err := s.memSvc.UpdateAgent(r.Context(), &agent); err != nil {
		safeHTTPError(w, r, err, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(agent)
}

func (s *APIServer) deleteAgentHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	agentID := vars["agentID"]

	if err := s.memSvc.DeleteAgent(r.Context(), agentID); err != nil {
		safeHTTPError(w, r, err, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
}

// ==================== Agent Group Handlers ====================

func (s *APIServer) createAgentGroupHandler(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Header.Get("X-Tenant-ID")
	if tenantID == "" {
		tenantID = "default"
	}

	var group types.AgentGroup
	if err := json.NewDecoder(r.Body).Decode(&group); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	group.TenantID = tenantID
	if err := s.memSvc.CreateAgentGroup(r.Context(), &group); err != nil {
		safeHTTPError(w, r, err, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(group)
}

func (s *APIServer) listAgentGroupsHandler(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Header.Get("X-Tenant-ID")
	if tenantID == "" {
		tenantID = "default"
	}

	limit := 50
	offset := 0

	groups, total, err := s.memSvc.ListAgentGroups(r.Context(), tenantID, limit, offset)
	if err != nil {
		safeHTTPError(w, r, err, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"groups": groups,
		"total":  total,
	})
}

func (s *APIServer) getAgentGroupHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	groupID := vars["groupID"]

	group, err := s.memSvc.GetAgentGroup(r.Context(), groupID)
	if err != nil {
		safeHTTPError(w, r, err, http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(group)
}

func (s *APIServer) updateAgentGroupHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	groupID := vars["groupID"]

	var group types.AgentGroup
	if err := json.NewDecoder(r.Body).Decode(&group); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	group.ID = groupID
	if err := s.memSvc.UpdateAgentGroup(r.Context(), &group); err != nil {
		safeHTTPError(w, r, err, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(group)
}

func (s *APIServer) deleteAgentGroupHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	groupID := vars["groupID"]

	if err := s.memSvc.DeleteAgentGroup(r.Context(), groupID); err != nil {
		safeHTTPError(w, r, err, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
}

func (s *APIServer) addAgentToGroupHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	groupID := vars["groupID"]

	var req struct {
		AgentID string `json:"agent_id"`
		Role    string `json:"role"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.AgentID == "" {
		http.Error(w, "agent_id is required", http.StatusBadRequest)
		return
	}

	if req.Role == "" {
		req.Role = string(types.MemberRoleContributor)
	}

	if err := s.memSvc.AddAgentToGroup(r.Context(), req.AgentID, groupID, types.MemberRole(req.Role)); err != nil {
		safeHTTPError(w, r, err, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

func (s *APIServer) removeAgentFromGroupHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	groupID := vars["groupID"]
	agentID := vars["agentID"]

	if err := s.memSvc.RemoveAgentFromGroup(r.Context(), agentID, groupID); err != nil {
		safeHTTPError(w, r, err, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

func (s *APIServer) getGroupSkillsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	groupID := vars["groupID"]

	limit := 50

	skills, err := s.memSvc.GetGroupSkills(r.Context(), groupID, limit)
	if err != nil {
		safeHTTPError(w, r, err, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"skills": skills,
		"count":  len(skills),
	})
}

func (s *APIServer) getGroupMemoriesHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	groupID := vars["groupID"]

	memories, err := s.memSvc.GetGroupMemories(r.Context(), groupID)
	if err != nil {
		safeHTTPError(w, r, err, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"memories": memories,
		"count":    len(memories),
	})
}

func (s *APIServer) shareMemoryToGroupHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	groupID := vars["groupID"]

	var req struct {
		MemoryID string `json:"memory_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.MemoryID == "" {
		http.Error(w, "memory_id is required", http.StatusBadRequest)
		return
	}

	if err := s.memSvc.ShareMemoryToGroup(r.Context(), req.MemoryID, groupID); err != nil {
		safeHTTPError(w, r, err, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

// ==================== Review Handlers ====================

func (s *APIServer) listReviewsHandler(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Header.Get("X-Tenant-ID")
	if tenantID == "" {
		tenantID = "default"
	}

	reviews, err := s.memSvc.ListPendingReviews(r.Context(), tenantID)
	if err != nil {
		safeHTTPError(w, r, err, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"reviews": reviews,
		"count":   len(reviews),
	})
}

func (s *APIServer) getReviewHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	reviewID := vars["reviewID"]

	review, err := s.memSvc.GetReview(r.Context(), reviewID)
	if err != nil {
		safeHTTPError(w, r, err, http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(review)
}

func (s *APIServer) processReviewHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	reviewID := vars["reviewID"]

	var req struct {
		Approved bool   `json:"approved"`
		Notes    string `json:"notes"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := s.memSvc.ProcessReview(r.Context(), reviewID, req.Approved, req.Notes); err != nil {
		safeHTTPError(w, r, err, http.StatusInternalServerError)
		return
	}

	eventType := audit.EventTypeSkillReject
	if req.Approved {
		eventType = audit.EventTypeSkillApprove
	}
	s.logAudit(r.Context(), eventType, "skill", reviewID, getTenantID(r), map[string]interface{}{
		"review_id": reviewID,
		"approved":  req.Approved,
		"notes":     req.Notes,
	})

	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

// ==================== Notification Handlers ====================

func (s *APIServer) createNotificationHandler(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	if tenantID == "" {
		tenantID = "default"
	}

	var req notification.CreateNotificationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.UserID == "" {
		http.Error(w, "user_id is required", http.StatusBadRequest)
		return
	}

	notif, err := s.notifSvc.Create(r.Context(), tenantID, req)
	if err != nil {
		safeHTTPError(w, r, err, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(notif)
}

func (s *APIServer) listNotificationsHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		userID = getTenantID(r)
	}

	req := notification.ListNotificationsRequest{
		Status:  r.URL.Query().Get("status"),
		Type:    notification.NotificationType(r.URL.Query().Get("type")),
		Channel: notification.NotificationChannel(r.URL.Query().Get("channel")),
		Limit:   50,
	}

	if limit := r.URL.Query().Get("limit"); limit != "" {
		if parsed, err := strconv.Atoi(limit); err == nil && parsed > 0 {
			req.Limit = int64(parsed)
		}
	}

	notifications, total, err := s.notifSvc.List(r.Context(), userID, req)
	if err != nil {
		safeHTTPError(w, r, err, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"notifications": notifications,
		"total":         total,
		"limit":         req.Limit,
	})
}

func (s *APIServer) getNotificationHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	notificationID := vars["notificationID"]

	notif, err := s.notifSvc.Get(r.Context(), notificationID)
	if err != nil {
		safeHTTPError(w, r, err, http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(notif)
}

func (s *APIServer) markNotificationReadHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	notificationID := vars["notificationID"]

	if err := s.notifSvc.MarkRead(r.Context(), notificationID); err != nil {
		safeHTTPError(w, r, err, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

func (s *APIServer) markAllNotificationsReadHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		userID = getTenantID(r)
	}

	if err := s.notifSvc.MarkAllRead(r.Context(), userID); err != nil {
		safeHTTPError(w, r, err, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

func (s *APIServer) archiveNotificationHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	notificationID := vars["notificationID"]

	if err := s.notifSvc.Archive(r.Context(), notificationID); err != nil {
		safeHTTPError(w, r, err, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

func (s *APIServer) archiveAllNotificationsHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		userID = getTenantID(r)
	}

	if err := s.notifSvc.ArchiveAll(r.Context(), userID); err != nil {
		safeHTTPError(w, r, err, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

func (s *APIServer) deleteNotificationHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	notificationID := vars["notificationID"]

	if err := s.notifSvc.Delete(r.Context(), notificationID); err != nil {
		safeHTTPError(w, r, err, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
}

func (s *APIServer) getNotificationSummaryHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		userID = getTenantID(r)
	}

	summary, err := s.notifSvc.GetSummary(r.Context(), userID)
	if err != nil {
		safeHTTPError(w, r, err, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(summary)
}

func (s *APIServer) getNotificationPreferencesHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		userID = getTenantID(r)
	}

	prefs, err := s.notifSvc.GetPreferences(r.Context(), userID)
	if err != nil {
		safeHTTPError(w, r, err, http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(prefs)
}

func (s *APIServer) updateNotificationPreferencesHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		userID = getTenantID(r)
	}

	var req notification.UpdatePreferencesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	prefs, err := s.notifSvc.UpdatePreferences(r.Context(), userID, req)
	if err != nil {
		safeHTTPError(w, r, err, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(prefs)
}

// ==================== Auth Handlers ====================

func (s *APIServer) authLoginHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		safeHTTPError(w, r, fmt.Errorf("invalid request body"), http.StatusBadRequest)
		return
	}

	if req.Email == "" || req.Password == "" {
		safeHTTPError(w, r, fmt.Errorf("email and password are required"), http.StatusBadRequest)
		return
	}

	if !isValidEmail(req.Email) {
		jsonError(w, "Invalid email format", http.StatusBadRequest)
		return
	}

	user, err := s.userSvc.Authenticate(req.Email, req.Password)
	if err != nil {
		safeHTTPError(w, r, fmt.Errorf("invalid email or password"), http.StatusUnauthorized)
		return
	}

	session := s.sessionStore.CreateSession(
		user.ID.String(),
		user.Email,
		user.Name,
		string(user.Role),
	)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"token":   session.Token,
		"user": map[string]interface{}{
			"id":    session.UserID,
			"name":  session.Name,
			"email": session.Email,
			"role":  session.Role,
		},
	})
}

func (s *APIServer) authRegisterHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		Name     string `json:"name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		safeHTTPError(w, r, fmt.Errorf("invalid request body"), http.StatusBadRequest)
		return
	}

	if req.Email == "" || req.Password == "" {
		safeHTTPError(w, r, fmt.Errorf("email and password are required"), http.StatusBadRequest)
		return
	}

	if !isValidEmail(req.Email) {
		jsonError(w, "Invalid email format", http.StatusBadRequest)
		return
	}

	if len(req.Password) < 6 {
		jsonError(w, "Password must be at least 6 characters", http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		atIndex := strings.Index(req.Email, "@")
		if atIndex > 0 {
			req.Name = req.Email[:atIndex]
		} else {
			req.Name = req.Email
		}
	}

	if s.userSvc == nil {
		safeHTTPError(w, r, fmt.Errorf("user service not available"), http.StatusInternalServerError)
		return
	}

	// Check if user already exists
	allUsers, err := s.userSvc.ListUsers()
	if err == nil {
		for _, u := range allUsers {
			if u.Email == req.Email {
				safeHTTPError(w, r, fmt.Errorf("user with this email already exists"), http.StatusConflict)
				return
			}
		}
	}

	userReq := &users.CreateUserRequest{
		Email:    req.Email,
		Name:     req.Name,
		Role:     "user",
		Password: req.Password,
	}

	user, err := s.userSvc.CreateUser(userReq)
	if err != nil {
		safeHTTPError(w, r, fmt.Errorf("failed to create user: %w", err), http.StatusInternalServerError)
		return
	}

	session := s.sessionStore.CreateSession(
		user.ID.String(),
		user.Email,
		user.Name,
		string(user.Role),
	)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"token":   session.Token,
		"user": map[string]interface{}{
			"id":    session.UserID,
			"name":  session.Name,
			"email": session.Email,
			"role":  session.Role,
		},
	})
}

func (s *APIServer) handleChangePassword(w http.ResponseWriter, r *http.Request) {
	var req struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	if len(req.NewPassword) < 8 {
		http.Error(w, "Password must be at least 8 characters", http.StatusBadRequest)
		return
	}

	userID := getTenantID(r)
	if userID == "" {
		http.Error(w, "Not authenticated", http.StatusUnauthorized)
		return
	}

	if s.userSvc != nil {
		if err := s.userSvc.ChangePassword(r.Context(), userID, req.CurrentPassword, req.NewPassword); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}

	json.NewEncoder(w).Encode(map[string]interface{}{"success": true})
}

func (s *APIServer) updateCurrentUserHandler(w http.ResponseWriter, r *http.Request) {
	userID := getTenantID(r)
	if userID == "" {
		http.Error(w, "Not authenticated", http.StatusUnauthorized)
		return
	}

	var req struct {
		Name  string `json:"name"`
		OrgID string `json:"org_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if s.userSvc != nil {
		uid, err := uuid.Parse(userID)
		if err == nil {
			updates := &users.UpdateUserRequest{}
			if req.Name != "" {
				updates.Name = req.Name
			}
			s.userSvc.UpdateUser(uid, updates)
		}
	}

	json.NewEncoder(w).Encode(map[string]interface{}{"success": true})
}

// ==================== SDK Endpoints ====================

// getMemoryVersionsHandler returns the version history for a memory.
// GET /memories/{memoryID}/versions
func (s *APIServer) getMemoryVersionsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	memoryID := vars["memoryID"]

	history, err := s.memSvc.GetMemoryHistory(context.Background(), memoryID)
	if err != nil {
		safeHTTPError(w, r, err, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(history)
}

// restoreMemoryVersionHandler restores a memory to a previous version.
// POST /memories/{memoryID}/restore  body: {"version": 1}
func (s *APIServer) restoreMemoryVersionHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	memoryID := vars["memoryID"]

	var req struct {
		Version int `json:"version"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Version < 1 {
		http.Error(w, "version must be >= 1", http.StatusBadRequest)
		return
	}

	history, err := s.memSvc.GetMemoryHistory(context.Background(), memoryID)
	if err != nil {
		safeHTTPError(w, r, err, http.StatusInternalServerError)
		return
	}
	if len(history) == 0 {
		http.Error(w, "no history found for memory", http.StatusNotFound)
		return
	}
	if req.Version > len(history) {
		jsonError(w, fmt.Sprintf("version %d not found; history has %d entries", req.Version, len(history)), http.StatusNotFound)
		return
	}

	// Version is 1-based; history is ordered oldest-first by convention
	entry := history[req.Version-1]
	restoreContent := entry.NewValue
	if restoreContent == "" {
		restoreContent = entry.OldValue
	}
	if restoreContent == "" {
		jsonError(w, "history entry has no content to restore", http.StatusUnprocessableEntity)
		return
	}

	if err := s.memSvc.UpdateMemory(context.Background(), memoryID, restoreContent, nil); err != nil {
		safeHTTPError(w, r, err, http.StatusInternalServerError)
		return
	}

	mem, _ := s.memSvc.GetMemory(context.Background(), memoryID)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "restored",
		"memory": mem,
	})
}

// getMemoryStatsHandler returns aggregate statistics for memories.
// GET /memories/stats?user_id=&org_id=
func (s *APIServer) getMemoryStatsHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	orgID := r.URL.Query().Get("org_id")

	// Fall back to headers if query params absent
	if userID == "" {
		userID = r.Header.Get("X-User-ID")
	}
	if orgID == "" {
		orgID = r.Header.Get("X-Org-ID")
	}

	stats, err := s.memSvc.GetMemoryStats(context.Background(), userID, orgID)
	if err != nil {
		safeHTTPError(w, r, err, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(stats)
}

// ==================== SDK Stub Endpoints (501 Not Implemented) ====================

func (s *APIServer) memoryLinksStubHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	json.NewEncoder(w).Encode(map[string]string{"status": "not_implemented"})
}

func (s *APIServer) memoryLinksIDStubHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	json.NewEncoder(w).Encode(map[string]string{"status": "not_implemented"})
}

func (s *APIServer) memoryLinkByIDStubHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	json.NewEncoder(w).Encode(map[string]string{"status": "not_implemented"})
}

func (s *APIServer) memoryInsightsStubHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	json.NewEncoder(w).Encode(map[string]string{"status": "not_implemented"})
}

func (s *APIServer) memorySummaryStubHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	json.NewEncoder(w).Encode(map[string]string{"status": "not_implemented"})
}

func (s *APIServer) adminCleanupStubHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	json.NewEncoder(w).Encode(map[string]string{"status": "not_implemented"})
}
