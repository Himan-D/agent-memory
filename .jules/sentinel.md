## 2024-05-27 - [Sentinel] Prevent SQL Injection in pgvector Dynamic Queries
**Vulnerability:** SQL injection vulnerability in `internal/memory/pgvector/client.go` because the table name (`cfg.Table`) was directly interpolated into SQL statements using `fmt.Sprintf` without validation. Table names cannot be parameterized.
**Learning:** When dynamic SQL queries require identifiers like table names, they must be strictly validated before interpolation.
**Prevention:** Use regular expression validation (`regexp.MustCompile("^[a-zA-Z_][a-zA-Z0-9_.]*$")`) for table names (allowing periods for schema qualification) prior to query execution.
