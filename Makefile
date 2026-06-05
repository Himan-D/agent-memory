.PHONY: all build test lint run clean docker migrate test-integration test-compression test-tenant benchmark benchmark-longmemeval benchmark-locomo build-mcp

BINARY_SERVER := hystersis-server
BINARY_CLI    := hystersis
BINARY_AGENT  := hystersis-agent
GO            := go
DOCKER        := docker
COMPOSE       := docker compose
GOLANGCI_LINT := golangci-lint

all: build

build:
	$(GO) build -o $(BINARY_SERVER) ./cmd/server
	$(GO) build -o $(BINARY_CLI)    ./cmd/cli
	$(GO) build -o $(BINARY_AGENT)  ./cmd/agent

build-server:
	$(GO) build -o $(BINARY_SERVER) ./cmd/server

build-cli:
	$(GO) build -o $(BINARY_CLI) ./cmd/cli

build-agent:
	$(GO) build -o $(BINARY_AGENT) ./cmd/agent

test:
	$(GO) test ./...

test-verbose:
	$(GO) test -v ./...

test-cover:
	$(GO) test -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

test-short:
	$(GO) test -short ./...

test-integration:
	$(GO) test -v ./tests/...

# Requires: docker compose up -d neo4j qdrant redis, server on :8080, env from tests/.env.e2e.example
e2e:
	bash tests/e2e_test.sh

lint:
	$(GOLANGCI_LINT) run ./...

lint-fix:
	$(GOLANGCI_LINT) run --fix ./...

vet:
	$(GO) vet ./...

fmt:
	gofmt -w .

tidy:
	$(GO) mod tidy

run-server: build-server
	./$(BINARY_SERVER)

run-agent: build-agent
	./$(BINARY_AGENT)

deps:
	$(GO) mod download

generate:
	$(GO) generate ./...

docker-build:
	$(DOCKER) build -t hystersis:latest .

docker-up:
	$(COMPOSE) up -d

docker-down:
	$(COMPOSE) down

docker-logs:
	$(COMPOSE) logs -f

docker-ps:
	$(COMPOSE) ps

test-integration: ## Run integration tests with testcontainers
	$(GO) test -v -tags=integration ./tests/...

test-compression: ## Run compression engine tests
	$(GO) test -v ./internal/compression/...

test-tenant: ## Run tenant isolation tests
	$(GO) test -v ./internal/memory/...

benchmark: ## Run full benchmark suite
	$(GO) run ./cmd/benchmark --dataset=all --mode=hybrid --concurrency=10

benchmark-all-modes: ## Run benchmarks across all search modes
	$(GO) run ./cmd/benchmark --dataset=all --mode=vector --concurrency=10
	$(GO) run ./cmd/benchmark --dataset=loocmo --mode=vector --concurrency=10
	$(GO) run ./cmd/benchmark --dataset=all --mode=spreading --concurrency=10
	$(GO) run ./cmd/benchmark --dataset=all --mode=hybrid --concurrency=10

benchmark-longmemeval: ## Run LongMemEval benchmark
	$(GO) run ./cmd/benchmark --dataset=longmemeval --mode=hybrid --concurrency=10

benchmark-locomo: ## Run LoCoMo benchmark
	$(GO) run ./cmd/benchmark --dataset=locomo --mode=hybrid --concurrency=10

benchmark-es: ## Run ES-MemEval benchmark
	$(GO) run ./cmd/benchmark --dataset=es_memeval --mode=hybrid --concurrency=10

benchmark-quick: ## Run single-mode smoke test (1 question per dataset, for CI)
	$(GO) run ./cmd/benchmark --dataset=all --mode=vector --concurrency=1 --max-questions=1

benchmark-deterministic: ## Run full benchmark with determinism for reproducibility
	$(GO) run ./cmd/benchmark --dataset=all --mode=hybrid --concurrency=10 --deterministic

build-mcp: ## Build unified MCP server binary
	$(GO) build -o hystersis-mcp ./cmd/mcp-cloud

migrate:
	$(GO) run ./cmd/server -migrate

clean:
	rm -f $(BINARY_SERVER) $(BINARY_CLI) $(BINARY_AGENT)
	rm -f coverage.out coverage.html
	$(GO) clean ./...

help:
	@echo "Targets:"
	@echo "  build         Build all binaries (server, cli, agent)"
	@echo "  test          Run all tests"
	@echo "  test-cover    Run tests with HTML coverage report"
	@echo "  lint          Run golangci-lint"
	@echo "  vet           Run go vet"
	@echo "  fmt           Run gofmt"
	@echo "  tidy          Run go mod tidy"
	@echo "  docker-build  Build Docker image"
	@echo "  docker-up     Start docker compose"
	@echo "  docker-down   Stop docker compose"
	@echo "  test-integration  Run integration tests with testcontainers"
	@echo "  test-compression  Run compression engine tests"
	@echo "  test-tenant       Run tenant isolation tests"
	@echo "  benchmark         Run full benchmark suite"
	@echo "  benchmark-longmemeval  Run LongMemEval benchmark"
	@echo "  benchmark-locomo       Run LoCoMo benchmark"
	@echo "  build-mcp         Build unified MCP server binary"
	@echo "  migrate       Run database migrations"
	@echo "  clean         Remove binaries and coverage files"
