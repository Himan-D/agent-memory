package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/joho/godotenv"

	"agent-memory/internal/config"
	"agent-memory/internal/logger"
	"agent-memory/internal/memory"
	"agent-memory/internal/memory/types"
	"agent-memory/internal/migration"
	"agent-memory/internal/project"
	"agent-memory/internal/storage"
	"agent-memory/internal/sync"
	"agent-memory/internal/telemetry"
	"agent-memory/internal/webhook"

	"github.com/grafana/pyroscope-go"
	_ "github.com/grafana/pyroscope-go/godeltaprof"
)

func loadSampleData(memSvc *memory.Service, projSvc *project.Service, whSvc *webhook.Service) {
	ctx := context.Background()

	sampleAgents := []*types.Agent{
		{ID: uuid.New().String(), Name: "Sales Agent", Description: "Sales and marketing automation", Status: types.AgentStatusActive, TenantID: "default", Config: types.AgentConfig{AutoExtract: true}, CreatedAt: time.Now()},
		{ID: uuid.New().String(), Name: "Support Agent", Description: "Customer support and success", Status: types.AgentStatusActive, TenantID: "default", Config: types.AgentConfig{AutoExtract: true}, CreatedAt: time.Now()},
		{ID: uuid.New().String(), Name: "Analysis Agent", Description: "Data analysis and reporting", Status: types.AgentStatusInactive, TenantID: "default", Config: types.AgentConfig{AutoExtract: false}, CreatedAt: time.Now()},
	}
	for _, agent := range sampleAgents {
		if err := memSvc.CreateAgent(ctx, agent); err != nil {
			logger.Errorf("Failed to create sample agent", "error", err, "agent", agent.Name)
		}
	}
	logger.Infof("Loaded %d sample agents", len(sampleAgents))

	sampleGroups := []*types.AgentGroup{
		{ID: uuid.New().String(), Name: "Sales Team", Description: "Sales and marketing automation", TenantID: "default", Policy: types.GroupPolicy{AllowCrossAgentMemory: true, SkillSharingEnabled: true}, CreatedAt: time.Now()},
		{ID: uuid.New().String(), Name: "Support Team", Description: "Customer support and success", TenantID: "default", Policy: types.GroupPolicy{AllowCrossAgentMemory: true, SkillSharingEnabled: true}, CreatedAt: time.Now()},
		{ID: uuid.New().String(), Name: "DevOps Team", Description: "Infrastructure and automation", TenantID: "default", Policy: types.GroupPolicy{AllowCrossAgentMemory: false, RequireHumanReview: true}, CreatedAt: time.Now()},
	}
	for _, group := range sampleGroups {
		if err := memSvc.CreateAgentGroup(ctx, group); err != nil {
			logger.Errorf("Failed to create sample group", "error", err, "group", group.Name)
		}
	}
	logger.Infof("Loaded %d sample groups", len(sampleGroups))

	sampleProjects := []*types.Project{
		{ID: uuid.New().String(), Name: "Website Redesign", Description: "Complete overhaul of company website", UserID: "default", OrgID: "default", CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{ID: uuid.New().String(), Name: "Mobile App", Description: "iOS and Android app development", UserID: "default", OrgID: "default", CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{ID: uuid.New().String(), Name: "Data Analytics", Description: "Business intelligence dashboard", UserID: "default", OrgID: "default", CreatedAt: time.Now(), UpdatedAt: time.Now()},
	}
	for _, proj := range sampleProjects {
		proj.Settings = types.ProjectSettings{MemoryTypes: []types.MemoryType{types.MemoryTypeUser, types.MemoryTypeSession}}
		if _, err := projSvc.CreateProject(ctx, proj); err != nil {
			logger.Errorf("Failed to create sample project", "error", err, "project", proj.Name)
		}
	}
	logger.Infof("Loaded %d sample projects", len(sampleProjects))

	sampleWebhooks := []*types.Webhook{
		{ID: uuid.New().String(), ProjectID: "default", URL: "https://hooks.slack.com/services/xxx", Events: []types.WebhookEvent{types.WebhookEventMemoryCreated}, Active: true, CreatedAt: time.Now()},
		{ID: uuid.New().String(), ProjectID: "default", URL: "https://api.example.com/email", Events: []types.WebhookEvent{types.WebhookEventMemoryUpdated}, Active: true, CreatedAt: time.Now()},
		{ID: uuid.New().String(), ProjectID: "default", URL: "https://backup.example.com/webhook", Events: []types.WebhookEvent{types.WebhookEventMemoryDeleted}, Active: false, CreatedAt: time.Now()},
	}
	for _, wh := range sampleWebhooks {
		if _, err := whSvc.CreateWebhook(ctx, wh); err != nil {
			logger.Errorf("Failed to create sample webhook", "error", err, "webhook_id", wh.ID)
		}
	}
	logger.Infof("Loaded %d sample webhooks", len(sampleWebhooks))

	demoMemories := []string{
		"User prefers Python over JavaScript",
		"User works on machine learning projects",
		"User is interested in neural networks and deep learning",
		"User's name is Demo User",
		"User works at a tech startup",
		"User likes dark mode interface",
		"User is building an AI agent",
		"User prefers async communication over meetings",
		"User's favorite framework is React",
		"User has experience with PostgreSQL and MongoDB",
	}
	for _, content := range demoMemories {
		mem := &types.Memory{
			ID:         uuid.New().String(),
			Content:    content,
			UserID:     "demo-user",
			OrgID:      "default",
			TenantID:   "default",
			Type:       types.MemoryTypeUser,
			Importance: types.ImportanceHigh,
			CreatedAt:  time.Now(),
		}
		if _, err := memSvc.CreateMemory(ctx, mem); err != nil {
			logger.Errorf("Failed to create demo memory", "error", err, "content", content[:30])
		}
	}
	logger.Infof("Loaded %d demo memories", len(demoMemories))
}

func main() {
	if err := godotenv.Load(".env"); err != nil {
		_ = godotenv.Load("/home/ubuntu/agent-memory/.env")
	}

	cfg := config.Load()

	env := cfg.App.Environment
	if env == "" {
		env = "development"
	}
	logger.Init(env, os.Getenv("LOG_LEVEL"))

	initSentry(&cfg.App)

	telCfg := telemetry.Config{
		Enabled:      cfg.Telemetry.Enabled,
		OTLPEndpoint: cfg.Telemetry.OTLPEndpoint,
		ServiceName:  cfg.Telemetry.ServiceName,
		Environment:  env,
		SampleRate:   cfg.Telemetry.SampleRate,
	}
	telCtx := context.Background()
	if _, err := telemetry.Init(telCtx, telCfg); err != nil {
		logger.Warn("telemetry init failed (non-fatal)", "error", err)
	} else if cfg.Telemetry.Enabled {
		logger.Info("OTLP tracing enabled", "endpoint", cfg.Telemetry.OTLPEndpoint)
	}

	if os.Getenv("PYROSCOPE_SERVER_ADDRESS") != "" {
		_, err := pyroscope.Start(pyroscope.Config{
			ApplicationName: "hystersis-server",
			ServerAddress:   os.Getenv("PYROSCOPE_SERVER_ADDRESS"),
			Tags: map[string]string{
				"environment": cfg.App.Environment,
			},
		})
		if err != nil {
			logger.Warn("Failed to initialize Pyroscope", "error", err)
		} else {
			logger.Info("Pyroscope profiling initialized")
		}
	}

	blobStore, err := storage.NewBlobStore(cfg.Storage.Provider, cfg.Storage.DataDir, cfg.GCP.BucketName, cfg.AWS.S3Bucket, cfg.AWS.Region)
	if err != nil {
		logger.Warn("blob storage unavailable", "error", err)
	} else {
		logger.Info("blob storage initialized", "provider", cfg.Storage.Provider)
	}
	_ = blobStore

	logger.Info("=== Hystersis System ===")
	logger.Info("startup", "environment", cfg.App.Environment, "neo4j", cfg.Neo4j.URI, "qdrant", cfg.Qdrant.URL)

	memSvc, err := memory.NewService(cfg)
	if err != nil {
		logger.Fatal("Failed to initialize memory service", "error", err)
	}
	defer memSvc.Close()

	neo4jClient := memSvc.GetNeo4jClient()
	if neo4jClient != nil {
		migExec := &migration.Neo4jExecutor{
			RunFn: func(ctx context.Context, cypher string, params map[string]any) error {
				session, cleanup, err := neo4jClient.AcquireSession(ctx)
				if err != nil {
					return err
				}
				defer cleanup()
				_, err = session.Run(ctx, cypher, params)
				return err
			},
			ReadFn: func(ctx context.Context, cypher string, params map[string]any) ([]map[string]any, error) {
				session, cleanup, err := neo4jClient.AcquireSession(ctx)
				if err != nil {
					return nil, err
				}
				defer cleanup()
				result, err := session.Run(ctx, cypher, params)
				if err != nil {
					return nil, err
				}
				var rows []map[string]any
				for result.Next(ctx) {
					rec := result.Record()
					row := make(map[string]any, len(rec.Keys))
					for i, key := range rec.Keys {
						row[key] = rec.Values[i]
					}
					rows = append(rows, row)
				}
				return rows, nil
			},
		}
		migrator := migration.NewMigrator(migExec)
		ctx := context.Background()
		if err := migrator.RunPending(ctx); err != nil {
			logger.Warn("Schema migration failed (non-fatal)", "error", err)
		} else {
			status, _ := migrator.Status(ctx)
			for _, s := range status {
				if s.Applied {
					logger.Infof("Migration applied", "version", s.Version, "description", s.Description)
				}
			}
		}
	}

	projSvc := project.NewService(cfg)
	whSvc := webhook.NewService(cfg)

	if os.Getenv("LOAD_SAMPLE_DATA") == "true" {
		loadSampleData(memSvc, projSvc, whSvc)
	} else {
		logger.Info("Sample data loading skipped (set LOAD_SAMPLE_DATA=true to enable)")
	}

	mode := os.Getenv("SERVER_MODE")
	if mode == "mcp-stdio" {
		logger.Info("Starting MCP server (stdio mode)...")
		RunMCPServer(memSvc, projSvc, whSvc)
		return
	}

	logger.Info("HTTP server", "port", cfg.App.HTTPPort)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	syncer := sync.New(memSvc, cfg.App.SyncInterval, cfg.App.BatchSize)
	go syncer.Start(ctx)

	apiServer := NewAPIServer(cfg, memSvc, projSvc, whSvc, memSvc.APIKeyStore())

	go func() {
		if err := apiServer.RunUntilShutdown(); err != nil {
			logger.Error("API server error", "error", err)
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	logger.Info("Shutting down...")
	syncer.Stop()
	cancel()
	apiServer.Stop()
	if err := telemetry.Shutdown(context.Background()); err != nil {
		logger.Warn("telemetry shutdown error", "error", err)
	}
	flushSentry()
	logger.Info("Goodbye!")
}
