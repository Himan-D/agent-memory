# Skills System Orchestrator

> Actionable build spec for skill discovery, execution, chains, audit events, and group policy.
> Status as of 2026-05-09: Core CRUD functional. Audit events, SkillSharingEnabled, AgentConfig.SkillDomains unimplemented.

---

## What This System Does

The Skills System gives AI agents reusable procedural capabilities. A skill is a trigger-action pair: when a certain context is detected, execute a specific action. Skills can be:
- **File-based**: `.md` files with YAML frontmatter (13 built-in, loaded at startup)
- **Neo4j-backed**: created at runtime via API, stored in graph database
- **LLM-synthesized**: merged from multiple similar skills into a general one
- **Chained**: multi-step workflows where output of step N feeds step N+1

---

## Current State

| Component | File | Status |
|-----------|------|--------|
| Skill CRUD | `internal/memory/neo4j/client.go` | ✅ Full |
| File-based registry | `internal/skills/registry.go` | ✅ Full |
| Chain execution | `cmd/server/api.go` | ✅ Executes via LLM |
| Similar skills endpoint | `GET /skills/{id}/similar` | ✅ Added |
| NPM SDK endpoints | `/skills/review`, `/skills/{id}/execute` | ✅ Added |
| Audit events | `cmd/server/api.go` | ❌ Missing for skills |
| `SkillSharingEnabled` | `GroupPolicy.SkillSharingEnabled` | ❌ Defined, never checked |
| `AgentConfig.SkillDomains` | `AgentConfig.SkillDomains` | ❌ Defined, never enforced |
| Human review flow | `POST /reviews/{id}` | ✅ Full |

---

## Task 1: Skill Audit Events

**Why**: Security and compliance require a full audit trail. Currently `skill.approved`, `skill.rejected`, and `skill.synthesized` events are never emitted, breaking the audit log completeness.

**Files to modify**: 
- `internal/audit/audit.go` — add event type constants
- `cmd/server/api.go` — add `a.audit.Log()` calls

**Step 1** — Add constants to `internal/audit/audit.go`:

```go
const (
    // Existing memory events (already present):
    // EventMemoryCreated, EventMemoryUpdated, EventMemoryDeleted
    
    // New skill events:
    EventSkillCreated    = "skill.created"
    EventSkillUpdated    = "skill.updated"
    EventSkillDeleted    = "skill.deleted"
    EventSkillApproved   = "skill.approved"
    EventSkillRejected   = "skill.rejected"
    EventSkillSynthesized = "skill.synthesized"
    EventSkillExtracted  = "skill.extracted"
    EventSkillExecuted   = "skill.executed"
    EventChainExecuted   = "chain.executed"
)
```

**Step 2** — Add audit calls in `cmd/server/api.go`. Find these handlers and add logging:

**`handleProcessSkillReview`** (approve/reject):
```go
// After updating skill status:
eventType := audit.EventSkillRejected
if req.Approved {
    eventType = audit.EventSkillApproved
}
a.audit.Log(r.Context(), audit.Event{
    TenantID:   tenantID,
    EntityType: "skill",
    EntityID:   review.SkillID,
    Action:     eventType,
    ActorID:    apiKeyID,
    Metadata:   map[string]interface{}{
        "review_id": reviewID,
        "notes":     req.Notes,
    },
})
```

**`handleSynthesizeSkills`**:
```go
// After synthesis succeeds:
a.audit.Log(r.Context(), audit.Event{
    TenantID:   tenantID,
    EntityType: "skill",
    EntityID:   synthesized.ID,
    Action:     audit.EventSkillSynthesized,
    ActorID:    apiKeyID,
    Metadata:   map[string]interface{}{
        "source_skill_ids": req.SkillIDs,
        "name":             synthesized.Name,
    },
})
```

**`handleExtractSkills`**:
```go
// After extraction:
a.audit.Log(r.Context(), audit.Event{
    TenantID:   tenantID,
    EntityType: "skill",
    EntityID:   "",  // multiple skills extracted
    Action:     audit.EventSkillExtracted,
    ActorID:    apiKeyID,
    Metadata:   map[string]interface{}{
        "count":       len(skills),
        "source":      "content_extraction",
    },
})
```

**`handleExecuteSkill`**:
```go
// After execution (success or failure):
a.audit.Log(r.Context(), audit.Event{
    TenantID:   tenantID,
    EntityType: "skill",
    EntityID:   skillID,
    Action:     audit.EventSkillExecuted,
    ActorID:    apiKeyID,
    Metadata:   map[string]interface{}{
        "success":    err == nil,
        "latency_ms": time.Since(startTime).Milliseconds(),
    },
})
```

**`handleExecuteChain`**:
```go
a.audit.Log(r.Context(), audit.Event{
    TenantID:   tenantID,
    EntityType: "chain",
    EntityID:   chainID,
    Action:     audit.EventChainExecuted,
    ActorID:    apiKeyID,
    Metadata:   map[string]interface{}{
        "steps_completed": stepsCompleted,
        "success":         err == nil,
    },
})
```

---

## Task 2: SkillSharingEnabled Group Policy

**Why**: Groups can share memory pools, but skill sharing is gated by `GroupPolicy.SkillSharingEnabled`. Currently it's always effectively `true` (no check).

**Files to modify**: `cmd/server/api.go`

**Find**: `handleGetGroupSkills` — the endpoint `GET /groups/{id}/skills`

**Add check**:
```go
func (a *API) handleGetGroupSkills(w http.ResponseWriter, r *http.Request) {
    groupID := chi.URLParam(r, "id")
    
    group, err := a.graphStore.GetGroup(r.Context(), groupID)
    if err != nil {
        safeHTTPError(w, http.StatusNotFound, err)
        return
    }
    
    // Check skill sharing policy
    if !group.Policy.SkillSharingEnabled {
        jsonOK(w, []interface{}{})  // return empty list, don't error
        return
    }
    
    skills, err := a.graphStore.GetGroupSkills(r.Context(), groupID)
    if err != nil {
        safeHTTPError(w, http.StatusInternalServerError, err)
        return
    }
    
    jsonOK(w, skills)
}
```

**Also gate**: When an agent in a group executes a skill that belongs to another agent in the group, check the group's `SkillSharingEnabled` before allowing cross-agent skill access.

---

## Task 3: AgentConfig.SkillDomains Filtering

**Why**: An agent can be configured to only use skills in certain domains (e.g., an agent restricted to `["database", "sql"]` should not have access to `git-expert` skills).

**Files to modify**:
- `cmd/server/api.go` — in `handleListSkills` and `handleSearchSkills`
- `internal/skills/registry.go` — `ListSkills()` to accept domain filter

**In `handleListSkills`**:
```go
func (a *API) handleListSkills(w http.ResponseWriter, r *http.Request) {
    // Get agent config from context (set by auth middleware)
    agentConfig := getAgentConfigFromContext(r.Context())
    
    domain := r.URL.Query().Get("domain")
    
    // If agent has domain restrictions, apply them
    if agentConfig != nil && len(agentConfig.SkillDomains) > 0 {
        // domain query param must be within agent's allowed domains
        if domain == "" || !contains(agentConfig.SkillDomains, domain) {
            // Default to first allowed domain, or return all allowed
            skills := []types.Skill{}
            for _, allowedDomain := range agentConfig.SkillDomains {
                domainSkills, _ := a.graphStore.GetSkillsByDomain(r.Context(), allowedDomain, tenantID)
                skills = append(skills, domainSkills...)
            }
            jsonOK(w, skills)
            return
        }
    }
    
    // Normal path
    skills, err := a.graphStore.GetSkillsByDomain(r.Context(), domain, tenantID)
    // ...
}
```

**In `handleSearchSkills`**: Add domain filter before calling graph store:
```go
if agentConfig != nil && len(agentConfig.SkillDomains) > 0 {
    // Restrict search to agent's allowed domains
    req.AllowedDomains = agentConfig.SkillDomains
}
```

**In `internal/memory/neo4j/client.go`** — update `SearchSkills` and `GetSkillsByDomain` to accept optional domain slice filter:
```go
func (c *Client) SearchSkills(ctx context.Context, query, tenantID string, allowedDomains []string) ([]*types.Skill, error) {
    cypher := `
        MATCH (s:Skill {tenantID: $tenantID})
        WHERE s.trigger CONTAINS $query OR s.action CONTAINS $query
    `
    if len(allowedDomains) > 0 {
        cypher += ` AND s.domain IN $domains`
    }
    cypher += ` RETURN s LIMIT 20`
    // ...
}
```

---

## Task 4: Skill Suggestion Quality

**Current state**: `POST /skills/suggest` uses LLM to suggest skills but doesn't filter by the agent's actual skill library.

**Improvement**: Before returning LLM suggestions, cross-reference with actual stored skills to prefer existing skills over hallucinated ones.

**In `handleSuggestSkills`**:
```go
func (a *API) handleSuggestSkills(w http.ResponseWriter, r *http.Request) {
    // ... existing LLM suggestion call ...
    
    suggestions, err := a.llmProvider.SuggestSkills(ctx, req.Trigger, req.Context, req.Limit)
    
    // NEW: Enrich suggestions with actual skill IDs if they match stored skills
    for i, suggestion := range suggestions {
        // Search for existing skill with similar name/trigger
        existing, _ := a.graphStore.SearchSkills(ctx, suggestion.Name, tenantID, nil)
        if len(existing) > 0 && skillSimilarity(suggestion, existing[0]) > 0.8 {
            suggestions[i].SkillID = existing[0].ID  // link to real skill
            suggestions[i].Confidence = existing[0].Confidence
        }
    }
    
    jsonOK(w, suggestions)
}
```

---

## Task 5: Skill Chain — Conditional Execution

**Current state**: Chain steps execute sequentially regardless of previous step output.

**Add**: Evaluate `ChainStep.ContinueIf` condition against previous step output.

**In `handleExecuteChain`** (cmd/server/api.go), in the step execution loop:
```go
for i, step := range chain.Steps {
    result, err := a.executeStep(ctx, step, context)
    if err != nil {
        if step.OnError == "stop" {
            return nil, fmt.Errorf("chain step %d failed: %w", i, err)
        }
        // OnError == "continue" → proceed with empty result
        result = map[string]interface{}{}
    }
    
    // Evaluate ContinueIf condition
    if step.ContinueIf != "" {
        shouldContinue, evalErr := evaluateCondition(step.ContinueIf, result)
        if evalErr != nil || !shouldContinue {
            // Condition not met — stop chain here
            break
        }
    }
    
    // Merge result into context for next step
    mergeInto(context, result)
}
```

**`evaluateCondition`** — simple expression evaluator:
```go
// Supports: "result.success == true", "result.count > 0", "result.status != 'error'"
func evaluateCondition(expr string, result map[string]interface{}) (bool, error) {
    // Use goval or go-expr library for expression evaluation
    // Or implement simple field.subfield comparisons
}
```

---

## Testing Checklist

```bash
# Verify audit events fire
go test ./internal/audit/... -run TestSkillAuditEvents

# Verify skill domain filtering
curl -X POST http://localhost:8080/agents \
  -d '{"name":"restricted-agent","skill_domains":["database"]}'

curl http://localhost:8080/skills \
  -H "X-Agent-ID: restricted-agent-id"
# Should only return database/sql skills

# Verify SkillSharingEnabled
curl -X PUT http://localhost:8080/groups/{id} \
  -d '{"policy":{"skill_sharing_enabled":false}}'
curl http://localhost:8080/groups/{id}/skills
# Should return []

# Verify chain conditional execution
curl -X POST http://localhost:8080/chains/{id}/execute \
  -d '{"context":{"input":"test"}}'
# Verify chain stops at step where ContinueIf evaluates to false

# Full build
go build ./...
go test ./...
```
