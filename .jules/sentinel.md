## 2025-05-26 - SQL Injection via Table Name in pgvector
**Vulnerability:** The table name for pgvector queries (c.cfg.Table) is concatenated directly into SQL queries using fmt.Sprintf without any validation or sanitization, allowing potential SQL injection if the configuration value is manipulated.
**Learning:** Table names cannot be parameterized in SQL queries, so they must be validated before being used in dynamic queries.
**Prevention:** Always validate SQL identifiers like table names using a strict regex (e.g., ^[a-zA-Z0-9_.]+$) before constructing dynamic SQL queries.
