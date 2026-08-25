## 2024-05-24 - [CRITICAL] SQL Injection Risk in Dynamic Identifiers
**Vulnerability:** The `pgvector` store dynamically interpolates the `cfg.Table` configuration directly into raw SQL query strings using `fmt.Sprintf("INSERT INTO %s ...", c.cfg.Table)` without prior validation.
**Learning:** Table names and other SQL identifiers cannot be parameterized via prepared statements (`$1`, `$2`, etc.). If a configuration value is sourced from untrusted input, it allows direct SQL injection.
**Prevention:** Always validate dynamically injected SQL identifiers against a strict allowlist or regular expression (e.g., `^[a-zA-Z0-9_.]+$`) before executing queries to ensure they only contain safe characters.
