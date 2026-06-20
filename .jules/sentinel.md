## 2025-01-20 - [SQL Injection via Dynamic Table Name in Pgvector]
**Vulnerability:** The pgvector client used `fmt.Sprintf` to directly inject the table name from configuration (`cfg.Table`) into SQL queries without validation. Since table names cannot be parameterized in PostgreSQL, a malicious or malformed table name in the configuration could lead to SQL injection.
**Learning:** Table names in configuration cannot be parameterized and require strict validation before being used in dynamic SQL queries via `fmt.Sprintf` to prevent SQL injection.
**Prevention:** Always validate SQL identifiers like table names against a strict regex (e.g., `^[a-zA-Z0-9_\.]+$`) to ensure they only contain safe characters before injecting them into queries.
