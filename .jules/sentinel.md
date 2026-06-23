## 2024-06-23 - [SQL Injection via Dynamic Configuration Identifiers]
**Vulnerability:** SQL injection vulnerability in `internal/memory/pgvector/client.go` due to unvalidated `cfg.Table` configuration variable used directly in `fmt.Sprintf` for SQL query construction.
**Learning:** Configuration values like table names, which cannot be parameterized using standard SQL driver arguments (`$1`, `$2`), are often blindly trusted. If an attacker can control the environment or configuration, they can execute arbitrary SQL.
**Prevention:** Always validate SQL identifiers loaded from configuration using strict allowlists or regex (e.g., `^[a-zA-Z0-9_.]+$`) before interpolating them into queries.
