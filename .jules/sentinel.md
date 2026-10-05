## 2024-07-04 - Fix SQL injection via dynamic table name
**Vulnerability:** SQL injection vulnerability in pgvector client due to unsanitized dynamic table names (`cfg.Table`) passed into `fmt.Sprintf` when constructing SQL queries.
**Learning:** Table names cannot be parameterized in SQL queries, making them a common vector for SQL injection if sourced from configuration without strict validation.
**Prevention:** Validate SQL identifiers such as table names with strict regex matching (`^[a-zA-Z0-9_.]+$`) before using them in dynamic query construction.