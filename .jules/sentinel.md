## 2025-02-13 - [Prevent SQL Injection via Configuration]
**Vulnerability:** The pgvector client initialized queries utilizing `fmt.Sprintf` allowing injection through dynamically inserting unvalidated configuration `cfg.Table` strings. Table names cannot be easily parameterized.
**Learning:** Found table configurations read from environment variables and directly injected into SQL formatting without regex or structural validation against SQL Injection.
**Prevention:** Always parse and restrict dynamically constructed database identifier inputs against strict regex constraints like `^[a-zA-Z0-9_]+$` prior to injecting it to `fmt.Sprintf` calls used for raw SQL operations.
