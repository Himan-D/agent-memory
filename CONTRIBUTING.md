Contributing to Hystersis

Quickstart
----------
1. Clone the repo:
   git clone https://github.com/Himan-D/agent-memory.git
2. Start local dev services:
   docker compose -f docker-compose.dev.yml up -d
3. Run build & tests:
   go build ./...
   go test ./... 

Local development
-----------------
- Use Go 1.20+ (1.21 recommended).
- Run the API server:
  go run ./cmd/server
- Use the provided docker-compose.dev.yml for Neo4j, Qdrant, and Redis.

Testing & CI
------------
- Unit tests: go test ./...
- Integration tests: go test ./tests/integration
- CI runs build, tests, vet. Keep tests fast; mock LLM calls in CI when possible.

PR process
----------
- Fork → feature branch → open PR to main
- Include tests for behavior changes
- If touching proprietary directories (see AGENTS.md), add the `proprietary-review` label and request review from code owners.

Code style and quality
----------------------
- Run go vet and gofmt (or goimports) before committing.
- Keep functions small and well-tested.

Security
--------
- Never commit secrets or API keys. Use .env files locally and GitHub Secrets for CI.

Proprietary boundaries
----------------------
See AGENTS.md for a short description of which directories are proprietary and how to handle them. If in doubt, ask maintainers on the PR.

License
-------
By contributing, you agree that your contributions will be licensed under the project's MIT License.
