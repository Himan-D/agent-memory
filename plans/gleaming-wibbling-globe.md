# Implement ChatGPT + Claude + MemGPT Memory Patterns

## Context

Hystersis already has a world-class foundation (20+ research papers, full pipeline, tiered storage, sleep consolidation, skills system, VFS). But it's missing the **specific patterns** that make ChatGPT and Claude's memory feel magical to end users. This plan adds those patterns to make Hystersis the most complete agent memory system available.

### How Modern LLMs Actually Maintain Memory

**ChatGPT** uses a dual-layer system:
1. **Saved Memories** — explicit fact list (user preferences, facts, decisions). LLM decides what's worth saving. NOT vector-based — pattern matching retrieval.
2. **Reference Chat History** — implicit patterns from past conversations, injected when relevant.

**Claude** uses a filesystem-based system:
1. **MEMORY.md index** — first 200 lines loaded at session start, rest on-demand
2. **Topic files** — `debugging.md`, `patterns.md` etc. read only when needed
3. **Auto-consolidation** — Claude decides what to write, periodically merges/deduplicates
4. **Path-scoped rules** — conditional memory loading based on file patterns

**MemGPT** pioneered virtual context management:
1. **Working memory** (context window) ↔ **Archival memory** (database) — OS paging metaphor
2. **Self-directed edits** — agent calls functions to save/retrieve/edit its own memory
3. **Interrupts** — control flow between memory operations and user interaction

### What Hystersis Already Has
- Memory pipeline (chunk→embed→extract→graphify) ✅
- Sleep consolidation (Auto-Dreamer) ✅
- Tiered storage (Working→Hot→Cold→Archive) ✅
- Skills/procedural memory ✅
- VFS filesystem layer (FUSE/WebDAV/NFS) ✅
- Self-improvement from feedback ✅
- Prospective memory (reminders) ✅
- Provenance DAG, temporal reasoning, MW scoring ✅

### What's Missing
1. **MemGPT virtual context manager** — agent doesn't manage its own context window
2. **ChatGPT-style dual-layer** — no explicit "saved memories" + "chat history reference"
3. **Claude-style filesystem memory** — VFS exists but no MEMORY.md pattern
4. **Self-directed memory tools** — agent can't edit its own memory via function calls
5. **System prompt injection** — memories not auto-injected into LLM prompts
6. **Conversation-boundary consolidation** — only consolidates on timer, not at session end

---

## Phase 1: Virtual Context Manager (MemGPT Pattern)

The core innovation: treat the LLM context window like OS virtual memory.

**New file:** `internal/memory/context/manager.go`

```
ContextManager
├── WorkingMemory    (active context — fits in LLM window)
│   ├── SystemPrompt
│   ├── CoreMemory    (user bio, preferences, key facts)
│   ├── RecentMessages (last N conversation turns)
│   └── RecalledMemories (just-in-time retrieved memories)
├── ExternalMemory   (overflow — stored in graph/vector DB)
│   ├── ConversationArchive
│   ├── FactStore
│   └── EntityKnowledge
└── ContextBudget    (token budget management)
    ├── MaxTokens       (model limit)
    ├── SystemReserve   (tokens reserved for system prompt)
    ├── MemoryBudget    (tokens allocated for recalled memories)
    └── ConversationBudget (remaining for conversation)
```

**Key methods:**
- `ComposeContext(userMessage) → []Message` — builds the full LLM context:
  1. System prompt + core memory
  2. Retrieve relevant memories for current query (semantic + recency)
  3. Recent conversation turns (as many as budget allows)
  4. Token counting → evict oldest turns if over budget
- `UpdateCoreMemory(key, value)` — agent edits its own persistent facts
- `ArchiveConversation(messages)` — moves old turns to external memory
- `RecallRelevant(query, budget) → []Memory` — retrieves within token budget

**Files:**
- `internal/memory/context/manager.go` — ContextManager struct + methods
- `internal/memory/context/budget.go` — token counting and budget allocation
- `internal/memory/context/core.go` — CoreMemory (persistent user/agent facts)

---

## Phase 2: ChatGPT-Style Dual-Layer Memory

Implement the two layers ChatGPT uses.

### Layer 1: Saved Memories (Explicit Facts)

**New file:** `internal/memory/saved/store.go`

A curated fact store. Each entry is a discrete, user-visible fact:
```go
type SavedMemory struct {
    ID        string
    UserID    string
    Fact      string    // "User prefers dark mode"
    Category  string    // preference/fact/decision/goal
    Source    string    // which conversation it came from
    CreatedAt time.Time
    UpdatedAt time.Time
    Active    bool      // can be toggled off
}
```

**Decision logic** — LLM decides what to save:
- After each user message, run a lightweight classifier: "Is there a fact worth saving long-term?"
- If yes, extract the fact and add to SavedMemory store
- Deduplicates against existing facts (semantic similarity check)
- Resolves conflicts (newer fact supersedes older if contradictory)

**Key methods:**
- `ShouldSave(message) → (bool, SavedMemory)` — LLM classifier
- `Save(fact)` — store with dedup check
- `GetAll(userID) → []SavedMemory` — list all saved facts
- `Search(userID, query) → []SavedMemory` — find relevant facts
- `Delete(id)` / `Toggle(id, active)` — user control
- `InjectIntoPrompt(userID) → string` — format all active facts for system prompt

### Layer 2: Chat History Reference (Implicit Patterns)

**New file:** `internal/memory/history/reference.go`

Background indexing of past conversations:
- After each conversation ends, extract key patterns/topics
- Store lightweight summaries (not full transcripts)
- On new conversation, check if any past patterns are relevant
- Inject as "You previously discussed X with this user"

**Key methods:**
- `IndexConversation(sessionID, messages)` — extract patterns post-conversation
- `FindRelevantHistory(userID, currentQuery) → []Reference` — retrieve related past topics
- `FormatForPrompt(references) → string` — format as system prompt section

---

## Phase 3: Claude-Style Filesystem Memory

Implement the MEMORY.md pattern on top of existing VFS.

**New file:** `internal/memory/filesystem/markdown.go`

```
/memories/{user_id}/
├── MEMORY.md           # Index (first 200 lines loaded per session)
├── preferences.md      # Topic: user preferences
├── decisions.md        # Topic: decisions made
├── patterns.md         # Topic: interaction patterns
├── corrections.md      # Topic: things user corrected
└── daily/
    ├── 2026-05-26.md   # Daily log
    └── 2026-05-25.md
```

**Key methods:**
- `LoadIndex(userID) → string` — load first 200 lines of MEMORY.md
- `LoadTopic(userID, topic) → string` — on-demand topic file read
- `WriteFact(userID, topic, fact)` — append to appropriate topic file
- `WriteDaily(userID, entry)` — append to today's daily log
- `Consolidate(userID)` — merge duplicates, drop stale, update MEMORY.md index
- `ExportMarkdown(userID) → map[string]string` — export all memory as files

**Integration with VFS:** Build on existing `internal/fs/vfs/filesystem.go` which already mounts memories as files.

---

## Phase 4: Self-Directed Memory Tools (Agent Memory API)

Give the agent function-calling tools to manage its own memory — the MemGPT innovation.

**New file:** `internal/memory/tools/agent_tools.go`

Define MCP/function-calling tools the agent can invoke:

```go
var AgentMemoryTools = []Tool{
    {
        Name: "memory_save",
        Description: "Save an important fact about the user for future reference",
        Parameters: {content, category, importance},
    },
    {
        Name: "memory_search",
        Description: "Search your memories for relevant information",
        Parameters: {query, limit},
    },
    {
        Name: "memory_update",
        Description: "Update an existing memory with new information",
        Parameters: {memory_id, new_content},
    },
    {
        Name: "memory_delete",
        Description: "Remove a memory that is no longer accurate",
        Parameters: {memory_id},
    },
    {
        Name: "core_memory_edit",
        Description: "Edit your core understanding of the user (bio, preferences)",
        Parameters: {section, content},
    },
    {
        Name: "context_archive",
        Description: "Archive old conversation turns to free context space",
        Parameters: {before_turn_id},
    },
    {
        Name: "context_recall",
        Description: "Recall archived conversation context",
        Parameters: {query, max_tokens},
    },
}
```

**Execution handler:** Each tool maps to the corresponding service method.

---

## Phase 5: System Prompt Memory Injection

Auto-inject relevant memories into every LLM call.

**New file:** `internal/memory/injection/composer.go`

Before every LLM call, compose the system prompt:

```
[System Instructions]
[Core Memory: User bio, key preferences]
[Saved Memories: Relevant facts for this query]
[Chat History: Related past conversations]
[Session Context: Recent messages summary]
[User Message]
```

**Key methods:**
- `ComposeSystemPrompt(userID, query, tokenBudget) → string`
- Retrieves from saved memories, chat history, core memory
- Respects token budget — most relevant first, truncate at limit
- Includes "You remember:" section with injected memories

---

## Phase 6: Conversation-Boundary Consolidation

Consolidate at end of conversation, not just on a timer.

**Modify:** `internal/memory/sleep/scheduler_consolidation.go`

Add `OnSessionEnd(sessionID)` hook:
1. Extract all messages from the session
2. Run "what's worth remembering?" classifier on each turn
3. Save new facts to SavedMemory store
4. Index conversation patterns for ChatHistory reference
5. Write daily log entry
6. Update MEMORY.md index if anything important changed

**Trigger:** Called from session cleanup when a session expires or is explicitly ended.

---

## Phase 7: Memory-Aware Search Enhancement

Enhance search to use all memory layers.

**Modify:** `internal/memory/service.go` — `SearchMemories()`

Current search: vector similarity + graph + scoring pipeline.

Add multi-source retrieval:
1. Vector search (existing)
2. Saved memories scan (new — pattern match against facts)
3. Chat history reference (new — related past conversations)
4. Core memory check (new — direct lookup for known facts)
5. Merge + deduplicate + score

---

## Files Summary

| File | New/Modify | Purpose |
|------|-----------|---------|
| `internal/memory/context/manager.go` | NEW | MemGPT virtual context manager |
| `internal/memory/context/budget.go` | NEW | Token budget allocation |
| `internal/memory/context/core.go` | NEW | Core memory (persistent user facts) |
| `internal/memory/saved/store.go` | NEW | ChatGPT saved memories layer |
| `internal/memory/history/reference.go` | NEW | ChatGPT chat history reference layer |
| `internal/memory/filesystem/markdown.go` | NEW | Claude MEMORY.md pattern |
| `internal/memory/tools/agent_tools.go` | NEW | Self-directed memory tools |
| `internal/memory/injection/composer.go` | NEW | System prompt memory injection |
| `internal/memory/service.go` | MODIFY | Wire all new layers into pipeline |
| `internal/memory/sleep/scheduler_consolidation.go` | MODIFY | Add session-end consolidation |
| `cmd/server/api.go` | MODIFY | Add saved memory API endpoints |

---

## New API Endpoints

```
GET    /memories/saved           — list user's saved facts
POST   /memories/saved           — manually save a fact
DELETE /memories/saved/{id}      — delete a saved fact
PATCH  /memories/saved/{id}      — toggle/update a saved fact
GET    /memories/core            — get core memory (user bio, prefs)
PUT    /memories/core            — update core memory section
GET    /memories/filesystem      — export memory as markdown files
POST   /memories/consolidate     — trigger manual consolidation
GET    /context/compose          — preview composed context for a query
```

---

## Verification

```bash
go build ./...
go test -count=1 ./...
# Test context composition
curl -X POST localhost:8080/context/compose \
  -H "X-API-Key: test" \
  -d '{"query": "what does the user prefer?", "user_id": "alice"}'
# Test saved memories
curl -X POST localhost:8080/memories/saved \
  -H "X-API-Key: test" \
  -d '{"fact": "User prefers dark mode", "user_id": "alice"}'
```

---

## Why This Makes Hystersis Superior

After implementation, Hystersis will be the **only** memory system that combines:
- ChatGPT's dual-layer (saved facts + chat reference) 
- Claude's filesystem memory (MEMORY.md + topic files)
- MemGPT's virtual context management (OS paging metaphor)
- Self-directed memory tools (agent edits own memory)
- 20+ research papers (ProMem, spreading activation, temporal reasoning, etc.)
- Production infrastructure (tiered storage, sleep consolidation, RBAC, SSO)

No competitor has all of these. Mem0 has none of the ChatGPT/Claude patterns. ChatGPT has no graph-based retrieval. Claude has no vector search. Hystersis will have everything.
