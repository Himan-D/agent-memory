.PHONY: all build test lint run clean docker migrate

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
	@echo "  migrate       Run database migrations"
	@echo "  clean         Remove binaries and coverage files"
