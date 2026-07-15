## 2024-05-24 - SQL Injection in pgvector table formatting
**Vulnerability:** The pgvector implementation was directly inserting `c.cfg.Table` into SQL statements using `fmt.Sprintf` (e.g., `fmt.Sprintf("SELECT ... FROM %s", c.cfg.Table)`). Table names cannot be parameterized with `$1`, making this a critical SQL injection vector if `PGVECTOR_TABLE` config is externally controlled.
**Learning:** Dynamic table/column names in Go SQL must be validated strictly before use, as they bypass standard parameterization protections.
**Prevention:** Added strict regex validation (`^[a-zA-Z0-9_.]+$`) for the table name in the `NewClient` constructor to ensure only valid SQL identifiers are used.
