## 2024-05-17 - SQL Injection via Unvalidated Table Name
**Vulnerability:** SQL injection vulnerability in pgvector implementation because `cfg.Table` is used directly in `fmt.Sprintf` to build SQL queries without validation. Since table names cannot be parameterized, this allows malicious table names (via config injection) to execute arbitrary SQL.
**Learning:** Configurations derived from environment variables or external sources must be treated as untrusted input. When interpolating them as SQL identifiers (table names, column names) where parameterization is not supported, strict validation is required.
**Prevention:** Always use regex matching (e.g., `^[a-zA-Z_][a-zA-Z0-9_.]*$`) to validate SQL identifiers before using them in dynamic SQL construction.
