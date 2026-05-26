# Fix Plan: PR #19 Issues + Codebase Cleanup

## Context

PR #19 (`feat/full-memory-pipeline-and-hardening`) is open with critical findings from GitGuardian (9 leaked secrets) and Gemini code review (nil pointer crash, N+1 query, missing auth, unsafe file ops). Integration tests crash with a nil pointer dereference. This plan addresses all P0/P1 issues.

---

## Phase 1: Security — Remove Leaked Secrets (P0)

**Files:**
- `.gitignore` — add key patterns
- `deploy_key`, `PRODUCTION_SSH_KEY.txt`, `PRODUCTION_SSH_KEY_OLD.txt` — untrack

**Steps:**
1. Add to `.gitignore`: `deploy_key`, `PRODUCTION_SSH_KEY*.txt`, `*.pem`, `*.key`
2. `git rm --cached deploy_key PRODUCTION_SSH_KEY.txt PRODUCTION_SSH_KEY_OLD.txt`
3. Commit: `fix: remove leaked SSH keys from tracking`

> Note: Keys in git history are still exposed. User must rotate them separately.

---

## Phase 2: Nil Pointer Crash Fix (P0)

**File:** `internal/memory/neo4j/client.go:241-249`

**Problem:** `Ping()` calls `c.driver.NewSession()` without nil check. When Neo4j is unavailable, `c.driver` is nil → SIGSEGV.

**Fix:** Add nil guard at top of `Ping()`:
```go
func (c *Client) Ping(ctx context.Context) error {
    if c.driver == nil {
        return fmt.Errorf("neo4j driver not initialized")
    }
    // ... rest unchanged
}
```

---

## Phase 3: Unsafe Audit Log Truncation (P1)

**File:** `internal/audit/logger.go:471-503`

**Problem:** `DeleteOld()` truncates the file with `O_TRUNC` before writing new data. Crash mid-write = total data loss. Also ignores `Write()` errors on line 499.

**Fix:** Atomic write-then-rename:
1. Write kept events to `s.filePath + ".tmp"`
2. Check all `Write()` errors
3. `Sync()` the temp file
4. Close original, `os.Rename(tmp, original)`
5. Reopen for append

---

## Phase 4: Reminder Worker Error Handling (P1)

**File:** `internal/memory/reminder/worker.go:83-89`

**Problem:** Callbacks execute without panic recovery. A panicking callback crashes the entire worker goroutine. Reminder is cleared even if callback fails.

**Fix:**
1. Wrap each callback in `func()` with `defer recover()`
2. Log panics
3. Only clear `RemindAt` if all callbacks succeed

---

## Phase 5: Migration System (P2 — Stub Only)

**File:** `internal/migration/migration.go`

**Problem:** `GetCurrentVersion()` returns 0 always; `RunPending()` body is empty. Full implementation requires a Neo4j session dependency.

**Fix:** The Migrator needs a Neo4j session to function. Since it currently has no DB dependency injected, this is a design gap. For now:
1. Add a `driver` field to `Migrator` struct
2. Implement `GetCurrentVersion()` to query `MATCH (s:SchemaVersion) RETURN s.version`
3. Implement `RunPending()` to execute Cypher DDL per migration version and upsert SchemaVersion node
4. Define Cypher DDL for each migration version (indexes from `ensureIndexes()` as v1)

---

## Phase 6: Session Store Wiring (P2)

**File:** `cmd/server/api.go:243-256`

**Problem:** `rss` (RedisSessionStore) is created then discarded with `_ = rss`.

**Fix:**
1. Replace `sessionStore` with `rss` when Redis is available:
```go
if rss, err := NewRedisSessionStore(cfg.App.RedisURL, 24*time.Hour); err != nil {
    log.Printf("warning: redis session store unavailable, using in-memory: %v", err)
} else {
    sessionStore = rss  // ← use Redis store instead of discarding
}
```
2. Both types already implement the same methods, so this should work if they share an interface. Verify method compatibility.

---

## Phase 7: Extractor Hardcoded Models (P2)

**File:** `internal/compression/extractor/proprietary.go`

**Lines:** 171, 265, 301, 349, 389, 426, 507, 563

**Problem:** Model names `"gpt-4o-mini"` and `"claude-3-5-sonnet"` hardcoded instead of using router config.

**Fix:**
1. Add `fastModel` and `verifyModel` fields to extractor struct
2. Set from config during initialization
3. Replace all hardcoded strings with field references

---

## Verification

```bash
go build ./...                    # Must compile
go test ./...                     # Integration test should skip cleanly (not panic)
git log --oneline -5              # Verify commits
git diff --cached --name-only     # Verify no secrets staged
```

---

## Out of Scope

- Rotating compromised SSH keys (user must do manually)
- N+1 query fix (requires new `GetMemoriesByIDs` method on GraphStore interface — larger change)
- MCP auth forwarding (requires OAuth token validation infrastructure — separate PR)
- GroupPolicy lookup from request context (needs design for group resolution)
