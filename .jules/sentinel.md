## 2026-07-09 - Fix SQL Injection in pgvector config
**Vulnerability:** The pgvector client was vulnerable to SQL injection because it dynamically constructed SQL queries using the table name (`c.cfg.Table`) directly in `fmt.Sprintf` without prior validation. Table names cannot be parameterized in Postgres.
**Learning:** Configurations that are used to construct SQL statements as identifiers (like table names) must be strictly validated against an allowlist pattern before use.
**Prevention:** Always validate dynamically injected SQL identifiers against a strict regular expression like `^[a-zA-Z0-9_.]+$`.
