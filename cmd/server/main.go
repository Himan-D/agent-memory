package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/joho/godotenv"

	"agent-memory/internal/config"
	"agent-memory/internal/memory"
	"agent-memory/internal/memory/types"
	"agent-memory/internal/project"
	"agent-memory/internal/sync"
	"agent-memory/internal/webhook"
)

func loadSampleData(memSvc *memory.Service, projSvc *project.Service, whSvc *webhook.Service) {
	ctx := context.Background()
	
	// Load sample agents to Neo4j
	sampleAgents := []*types.Agent{
		{ID: uuid.New().String(), Name: "Sales Agent", Description: "Sales and marketing automation", Status: types.AgentStatusActive, TenantID: "default", Config: types.AgentConfig{AutoExtract: true}, CreatedAt: time.Now()},
		{ID: uuid.New().String(), Name: "Support Agent", Description: "Customer support and success", Status: types.AgentStatusActive, TenantID: "default", Config: types.AgentConfig{AutoExtract: true}, CreatedAt: time.Now()},
		{ID: uuid.New().String(), Name: "Analysis Agent", Description: "Data analysis and reporting", Status: types.AgentStatusInactive, TenantID: "default", Config: types.AgentConfig{AutoExtract: false}, CreatedAt: time.Now()},
	}
	for _, agent := range sampleAgents {
		if err := memSvc.CreateAgent(ctx, agent); err != nil {
			log.Printf("Failed to create sample agent: %v", err)
		}
	}
	log.Printf("Loaded %d sample agents", len(sampleAgents))

	// Load sample agent groups
	sampleGroups := []*types.AgentGroup{
		{ID: uuid.New().String(), Name: "Sales Team", Description: "Sales and marketing automation", TenantID: "default", Policy: types.GroupPolicy{AllowCrossAgentMemory: true, SkillSharingEnabled: true}, CreatedAt: time.Now()},
		{ID: uuid.New().String(), Name: "Support Team", Description: "Customer support and success", TenantID: "default", Policy: types.GroupPolicy{AllowCrossAgentMemory: true, SkillSharingEnabled: true}, CreatedAt: time.Now()},
		{ID: uuid.New().String(), Name: "DevOps Team", Description: "Infrastructure and automation", TenantID: "default", Policy: types.GroupPolicy{AllowCrossAgentMemory: false, RequireHumanReview: true}, CreatedAt: time.Now()},
	}
	for _, group := range sampleGroups {
		if err := memSvc.CreateAgentGroup(ctx, group); err != nil {
			log.Printf("Failed to create sample group: %v", err)
		}
	}
	log.Printf("Loaded %d sample groups", len(sampleGroups))

	// Load sample projects
	sampleProjects := []*types.Project{
		{ID: uuid.New().String(), Name: "Website Redesign", Description: "Complete overhaul of company website", UserID: "default", OrgID: "default", CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{ID: uuid.New().String(), Name: "Mobile App", Description: "iOS and Android app development", UserID: "default", OrgID: "default", CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{ID: uuid.New().String(), Name: "Data Analytics", Description: "Business intelligence dashboard", UserID: "default", OrgID: "default", CreatedAt: time.Now(), UpdatedAt: time.Now()},
	}
	for _, proj := range sampleProjects {
		proj.Settings = types.ProjectSettings{MemoryTypes: []types.MemoryType{types.MemoryTypeUser, types.MemoryTypeSession}}
		if _, err := projSvc.CreateProject(ctx, proj); err != nil {
			log.Printf("Failed to create sample project: %v", err)
		}
	}
	log.Printf("Loaded %d sample projects", len(sampleProjects))

	// Load sample webhooks
	sampleWebhooks := []*types.Webhook{
		{ID: uuid.New().String(), ProjectID: "default", URL: "https://hooks.slack.com/services/xxx", Events: []types.WebhookEvent{types.WebhookEventMemoryCreated}, Active: true, CreatedAt: time.Now()},
		{ID: uuid.New().String(), ProjectID: "default", URL: "https://api.example.com/email", Events: []types.WebhookEvent{types.WebhookEventMemoryUpdated}, Active: true, CreatedAt: time.Now()},
		{ID: uuid.New().String(), ProjectID: "default", URL: "https://backup.example.com/webhook", Events: []types.WebhookEvent{types.WebhookEventMemoryDeleted}, Active: false, CreatedAt: time.Now()},
	}
	for _, wh := range sampleWebhooks {
		if _, err := whSvc.CreateWebhook(ctx, wh); err != nil {
			log.Printf("Failed to create sample webhook: %v", err)
		}
	}
	log.Printf("Loaded %d sample webhooks", len(sampleWebhooks))
}

func main() {
	godotenv.Load("/home/ubuntu/agent-memory/.env")

	cfg := config.Load()

	initSentry(&cfg.App)

	log.Println("=== Hystersis System ===")
	log.Printf("Environment: %s", cfg.App.Environment)
	log.Printf("Neo4j:  %s", cfg.Neo4j.URI)
	log.Printf("Qdrant: %s", cfg.Qdrant.URL)

	memSvc, err := memory.NewService(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize memory service: %v", err)
	}
	defer memSvc.Close()

	projSvc := project.NewService(cfg)
	whSvc := webhook.NewService(cfg)

	// Load sample data
	loadSampleData(memSvc, projSvc, whSvc)

	mode := os.Getenv("SERVER_MODE")
	if mode == "mcp-stdio" {
		log.Println("Starting MCP server (stdio mode)...")
		RunMCPServer(memSvc, projSvc, whSvc)
		return
	}

	log.Printf("HTTP:   %s", cfg.App.HTTPPort)
	log.Println()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	syncer := sync.New(memSvc, cfg.App.SyncInterval, cfg.App.BatchSize)
	go syncer.Start(ctx)

	apiServer := NewAPIServer(cfg, memSvc, projSvc, whSvc, memSvc.APIKeyStore())

	go func() {
		if err := apiServer.RunUntilShutdown(); err != nil {
			log.Printf("API server error: %v", err)
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	log.Println("Shutting down...")
	syncer.Stop()
	cancel()
	apiServer.Stop()
	flushSentry()
	log.Println("Goodbye!")
}