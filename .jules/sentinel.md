## Sentinel's Journal

## 2026-06-21 - [Preventing SQL Injection in Dynamic Schema Table Names]
**Vulnerability:** In `internal/memory/pgvector/client.go`, the PostgreSQL pgvector extension client concatenated table names directly into query strings using `fmt.Sprintf` because PostgreSQL does not support parameterization of table names or schema object identifiers.
**Learning:** Table names coming from application configuration (`PGVECTOR_TABLE`) are technically safe if under developer control, but they become a risk if modified maliciously or externally. Since they cannot be parameterized with ``, direct string concatenation causes an SQL injection risk.
**Prevention:** Always validate schema identifiers (like table names) using an explicit strict regex like `^[a-zA-Z0-9_.]+$` before inserting them into an SQL query via `fmt.Sprintf`.
