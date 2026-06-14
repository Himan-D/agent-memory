## 2024-XX-XX - Initial
## 2024-XX-XX - Hardcoded Passwords
**Vulnerability:** Hardcoded Neo4j passwords in `cmd/list_memories/main.go` ("password123") and `cmd/agent/main.go` ("password") as a fallback for missing environment variables.
**Learning:** Development defaults often leak into production-ready configuration loading logic when environment variables are omitted.
**Prevention:** Remove fallback values for secrets like passwords. Instead, rely on the environment or config files securely managed externally. Let the program fail gracefully if required secrets are absent, or omit them, rather than falling back to guessable defaults.
