# Agent Memory API Specification

> Complete API reference for Agent Memory System v2.0

## Base URL

```
http://localhost:8080/api/v1
```

## Authentication

Agent Memory uses API key authentication. Include your API key in the `X-API-Key` header:

```bash
curl -H "X-API-Key: your-api-key" http://localhost:8080/api/v1/health
```

## Response Format

All responses return JSON with the following structure:

**Success Response:**
```json
{
  "success": true,
  "data": { ... },
  "meta": {
    "request_id": "req_abc123",
    "timestamp": "2024-01-15T10:30:00Z"
  }
}
```

**Error Response:**
```json
{
  "success": false,
  "error": {
    "code": "MEMORY_NOT_FOUND",
    "message": "Memory with ID 'mem_123' not found",
    "details": {}
  },
  "meta": {
    "request_id": "req_abc123",
    "timestamp": "2024-01-15T10:30:00Z"
  }
}
```

## Common Status Codes

| Code | Description |
|------|-------------|
| 200 | Success |
| 201 | Created |
| 400 | Bad Request |
| 401 | Unauthorized |
| 403 | Forbidden |
| 404 | Not Found |
| 429 | Rate Limited |
| 500 | Internal Error |

---

## Memories

### Create Memory

Create a new semantic memory with automatic LLM processing.

**Request:**
```http
POST /api/v1/memories
Content-Type: application/json
X-API-Key: your-api-key
```

```json
{
  "content": "User prefers dark mode for their IDE settings",
  "user_id": "user_abc123",
  "agent_id": "assistant_bot",
  "session_id": "sess_xyz789",
  "type": "conversation",
  "category": "preferences",
  "tags": ["settings", "ide", "productivity"],
  "importance": "high",
  "metadata": {
    "source": "user_preference",
    "confidence": 0.95
  },
  "process": true,
  "immutable": false,
  "expiration_date": "2025-12-31T23:59:59Z"
}
```

**Response (201 Created):**
```json
{
  "success": true,
  "data": {
    "id": "mem_def456",
    "content": "User prefers dark mode for their IDE settings",
    "user_id": "user_abc123",
    "agent_id": "assistant_bot",
    "session_id": "sess_xyz789",
    "type": "conversation",
    "category": "preferences",
    "tags": ["settings", "ide", "productivity"],
    "importance": "high",
    "status": "active",
    "metadata": {
      "facts": ["User prefers dark mode", "User uses IDE daily"],
      "entities": [{"name": "IDE", "type": "Tool"}],
      "source": "user_preference",
      "confidence": 0.95
    },
    "version": 1,
    "access_count": 0,
    "created_at": "2024-01-15T10:30:00Z",
    "updated_at": "2024-01-15T10:30:00Z"
  }
}
```

**Fields:**
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| content | string | Yes | The memory content to store |
| user_id | string | Yes | User identifier |
| agent_id | string | No | Agent identifier |
| session_id | string | No | Session identifier |
| type | string | No | Memory type: `conversation`, `session`, `user`, `org` (default: `conversation`) |
| category | string | No | Category for organization |
| tags | string[] | No | Array of tags for filtering |
| importance | string | No | `critical`, `high`, `medium`, `low` (default: `medium`) |
| metadata | object | No | Custom key-value metadata |
| process | boolean | No | Enable LLM processing (default: true) |
| immutable | boolean | No | Prevent modifications (default: false) |
| expiration_date | string | No | ISO 8601 expiration timestamp |
| parent_memory_id | string | No | Link to parent memory |
| related_memory_ids | string[] | No | Link to related memories |

---

### Get Memory

Retrieve a specific memory by ID.

**Request:**
```http
GET /api/v1/memories/{id}
X-API-Key: your-api-key
```

**Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "id": "mem_def456",
    "content": "User prefers dark mode for their IDE settings",
    "user_id": "user_abc123",
    "category": "preferences",
    "tags": ["settings", "ide", "productivity"],
    "importance": "high",
    "status": "active",
    "metadata": { ... },
    "version": 1,
    "access_count": 42,
    "created_at": "2024-01-15T10:30:00Z",
    "updated_at": "2024-01-15T10:30:00Z",
    "last_accessed": "2024-01-20T15:45:00Z"
  }
}
```

---

### Update Memory

Update an existing memory's content and metadata.

**Request:**
```http
PUT /api/v1/memories/{id}
Content-Type: application/json
X-API-Key: your-api-key
```

```json
{
  "content": "Updated content",
  "metadata": {
    "verified": true
  }
}
```

---

### Delete Memory

Permanently delete a memory.

**Request:**
```http
DELETE /api/v1/memories/{id}
X-API-Key: your-api-key
```

**Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "deleted": true,
    "id": "mem_def456"
  }
}
```

---

### List Memories

List memories with pagination and filtering.

**Request:**
```http
GET /api/v1/memories?user_id=user_abc123&category=preferences&page=1&page_size=20
X-API-Key: your-api-key
```

**Query Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| user_id | string | Filter by user |
| org_id | string | Filter by organization |
| agent_id | string | Filter by agent |
| category | string | Filter by category |
| type | string | Filter by memory type |
| tags | string | Comma-separated tags |
| importance | string | Filter by importance |
| status | string | Filter by status: `active`, `archived`, `deleted` |
| page | int | Page number (default: 1) |
| page_size | int | Items per page (default: 20, max: 100) |

**Response (200 OK):**
```json
{
  "success": true,
  "data": [
    { "id": "mem_1", "content": "...", ... },
    { "id": "mem_2", "content": "...", ... }
  ],
  "meta": {
    "page": 1,
    "page_size": 20,
    "total_items": 150,
    "total_pages": 8,
    "has_more": true
  }
}
```

---

### Batch Create Memories

Create up to 1000 memories in a single request.

**Request:**
```http
POST /api/v1/memories/batch
Content-Type: application/json
X-API-Key: your-api-key
```

```json
{
  "memories": [
    {
      "content": "Memory 1",
      "user_id": "user_abc123"
    },
    {
      "content": "Memory 2",
      "user_id": "user_abc123"
    }
  ]
}
```

---

### Memory History

Get the modification history of a memory.

**Request:**
```http
GET /api/v1/memories/{id}/history
X-API-Key: your-api-key
```

**Response (200 OK):**
```json
{
  "success": true,
  "data": [
    {
      "id": "hist_1",
      "memory_id": "mem_def456",
      "action": "create",
      "old_value": null,
      "new_value": "Original content",
      "changed_by": "system",
      "reason": null,
      "created_at": "2024-01-15T10:30:00Z"
    },
    {
      "id": "hist_2",
      "memory_id": "mem_def456",
      "action": "update",
      "old_value": "Original content",
      "new_value": "Updated content",
      "changed_by": "user_abc123",
      "reason": "Correction",
      "created_at": "2024-01-16T14:20:00Z"
    }
  ]
}
```

---

## Search

### Semantic Search

Perform vector-based semantic search.

**Request:**
```http
POST /api/v1/search
Content-Type: application/json
X-API-Key: your-api-key
```

```json
{
  "query": "What are the user's IDE preferences?",
  "user_id": "user_abc123",
  "limit": 10,
  "threshold": 0.7,
  "category": "preferences",
  "rerank": true,
  "rerank_top_k": 20
}
```

**Response (200 OK):**
```json
{
  "success": true,
  "data": [
    {
      "id": "mem_def456",
      "score": 0.92,
      "text": "User prefers dark mode for their IDE settings",
      "source": "qdrant",
      "memory_id": "mem_def456",
      "metadata": {
        "category": "preferences",
        "tags": ["settings", "ide"],
        "importance": "high"
      }
    }
  ]
}
```

---

### Hybrid Search

Combine semantic and keyword search for better results.

**Request:**
```http
POST /api/v1/search/hybrid
Content-Type: application/json
X-API-Key: your-api-key
```

```json
{
  "query": "dark mode IDE settings",
  "semantic_limit": 10,
  "keyword_limit": 10,
  "boost": 1.5,
  "threshold": 0.6,
  "filters": {
    "user_id": "user_abc123"
  },
  "tags": ["settings", "productivity"],
  "importance": "high",
  "date_from": "2024-01-01T00:00:00Z",
  "date_to": "2024-12-31T23:59:59Z"
}
```

---

### Advanced Search

Search with complex filter logic.

**Request:**
```http
POST /api/v1/search/advanced
Content-Type: application/json
X-API-Key: your-api-key
```

```json
{
  "query": "preferences",
  "filters": {
    "logic": "AND",
    "rules": [
      { "field": "category", "operator": "eq", "value": "preferences" },
      { "field": "importance", "operator": "in", "value": ["high", "critical"] }
    ],
    "nested": [
      {
        "logic": "OR",
        "rules": [
          { "field": "tags", "operator": "contains", "value": "ide" },
          { "field": "tags", "operator": "contains", "value": "editor" }
        ]
      }
    ]
  }
}
```

---

## Memory Links

### Link Memories

Create relationships between memories.

**Request:**
```http
POST /api/v1/memories/links
Content-Type: application/json
X-API-Key: your-api-key
```

```json
{
  "from_id": "mem_parent123",
  "to_id": "mem_child456",
  "type": "parent",
  "weight": 0.9,
  "metadata": {
    "context": "Related discussion"
  }
}
```

**Link Types:** `parent`, `related`, `reply`, `cite`

---

### Get Memory Links

Get all memories linked to a memory.

**Request:**
```http
GET /api/v1/memories/{id}/links
X-API-Key: your-api-key
```

---

## Memory Versions

### Get Memory Versions

Get all versions of a memory.

**Request:**
```http
GET /api/v1/memories/{id}/versions
X-API-Key: your-api-key
```

---

### Restore Memory Version

Restore a memory to a previous version.

**Request:**
```http
POST /api/v1/memories/{id}/restore
Content-Type: application/json
X-API-Key: your-api-key
```

```json
{
  "version_id": "ver_abc123"
}
```

---

## Memory Insights

### Get Memory Stats

Get statistics about a user's memories.

**Request:**
```http
GET /api/v1/memories/stats?user_id=user_abc123
X-API-Key: your-api-key
```

**Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "total_memories": 1500,
    "by_category": {
      "preferences": 320,
      "conversations": 850,
      "tasks": 330
    },
    "by_type": {
      "conversation": 1200,
      "user": 300
    },
    "by_importance": {
      "critical": 50,
      "high": 300,
      "medium": 800,
      "low": 350
    },
    "by_status": {
      "active": 1400,
      "archived": 100
    },
    "avg_access_count": 5.5,
    "top_tags": [
      { "tag": "work", "count": 120 },
      { "tag": "personal", "count": 85 }
    ],
    "recent_memories": 45,
    "expired_memories": 5
  }
}
```

---

### Get Memory Insights

Get AI-generated insights from memory patterns.

**Request:**
```http
GET /api/v1/memories/insights?user_id=user_abc123
X-API-Key: your-api-key
```

---

## Compression

### Generate Summary

Generate a compressed summary of user's memories.

**Request:**
```http
GET /api/v1/memories/summary?user_id=user_abc123
X-API-Key: your-api-key
```

**Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "user_id": "user_abc123",
    "summary": "User is interested in machine learning and AI. Prefers dark mode IDE. Active in research.",
    "key_points": [
      "User prefers dark mode",
      "Interested in ML/AI",
      "Uses Python daily"
    ],
    "memory_count": 150,
    "token_savings": 0.85,
    "last_updated": "2024-01-15T10:30:00Z"
  }
}
```

---

## Export & Import

### Export Memories

Export all memories for a user.

**Request:**
```http
GET /api/v1/memories/export?user_id=user_abc123
X-API-Key: your-api-key
```

**Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "version": "1.0",
    "exported_at": "2024-01-15T10:30:00Z",
    "memories": [ ... ],
    "entities": [ ... ],
    "relations": [ ... ]
  }
}
```

---

### Import Memories

Import memories from an export.

**Request:**
```http
POST /api/v1/memories/import
Content-Type: application/json
X-API-Key: your-api-key
```

```json
{
  "memories": [ ... ],
  "entities": [ ... ],
  "relations": [ ... ],
  "overwrite": false,
  "merge_mode": "append"
}
```

---

## Feedback

### Add Feedback

Add feedback to improve memory quality.

**Request:**
```http
POST /api/v1/feedback
Content-Type: application/json
X-API-Key: your-api-key
```

```json
{
  "memory_id": "mem_def456",
  "type": "positive",
  "comment": "This is accurate"
}
```

**Feedback Types:** `positive`, `negative`, `very_negative`

---

### Get Memories by Feedback

Get memories with specific feedback.

**Request:**
```http
GET /api/v1/feedback/memories?type=negative&limit=10
X-API-Key: your-api-key
```

---

## Sessions

### Create Session

Create a new conversation session.

**Request:**
```http
POST /api/v1/sessions
Content-Type: application/json
X-API-Key: your-api-key
```

```json
{
  "agent_id": "assistant_bot",
  "metadata": {
    "channel": "web",
    "user_tier": "premium"
  }
}
```

---

### Add Message

Add a message to a session.

**Request:**
```http
POST /api/v1/sessions/{id}/messages
Content-Type: application/json
X-API-Key: your-api-key
```

```json
{
  "role": "user",
  "content": "I need help with my code"
}
```

**Roles:** `user`, `assistant`, `system`, `tool`

---

### Get Context

Get formatted context for an LLM.

**Request:**
```http
GET /api/v1/sessions/{id}/context?limit=10
X-API-Key: your-api-key
```

---

## Entities (Knowledge Graph)

### Create Entity

Create a knowledge graph entity.

**Request:**
```http
POST /api/v1/entities
Content-Type: application/json
X-API-Key: your-api-key
```

```json
{
  "type": "Person",
  "name": "John Doe",
  "properties": {
    "role": "Engineer",
    "company": "Acme Corp"
  }
}
```

---

### Create Relation

Create a relationship between entities.

**Request:**
```http
POST /api/v1/relations
Content-Type: application/json
X-API-Key: your-api-key
```

```json
{
  "from_id": "entity_123",
  "to_id": "entity_456",
  "type": "WORKS_AT",
  "weight": 0.95,
  "metadata": {
    "since": "2022"
  }
}
```

---

### Traverse Graph

Traverse relationships from an entity.

**Request:**
```http
GET /api/v1/graph/traverse/{entity_id}?depth=3
X-API-Key: your-api-key
```

---

## Compaction

### Run Compaction

Run memory compaction/deduplication.

**Request:**
```http
POST /api/v1/compact
Content-Type: application/json
X-API-Key: your-api-key
```

```json
{
  "user_id": "user_abc123",
  "action": "full"
}
```

---

### Compaction Status

Check compaction status.

**Request:**
```http
GET /api/v1/compact/status
X-API-Key: your-api-key
```

---

## Admin

### Cleanup Expired

Cleanup expired memories.

**Request:**
```http
POST /api/v1/admin/cleanup
X-API-Key: your-api-key
```

---

### Sync Entities

Sync entities to vector store.

**Request:**
```http
POST /api/v1/admin/sync
X-API-Key: your-api-key
```

---

## Health & Monitoring

### Health Check

```http
GET /api/v1/health
```

### Readiness Check

```http
GET /api/v1/ready
```

### Metrics

```http
GET /metrics
```

---

## Rate Limits

| Plan | Requests/Minute |
|------|-----------------|
| Free | 100 |
| Pro | 1000 |
| Enterprise | Unlimited |

---

## SDK Examples

### Python SDK

```python
from agentmemory import AgentMemory

client = AgentMemory(
    base_url="https://api.yourserver.com",
    api_key="your-api-key"
)

# Create memory
memory = client.create_memory(
    content="User prefers email notifications",
    user_id="user_123",
    category="preferences",
    tags=["notifications", "email"],
    importance="high"
)

# Search
results = client.search(
    query="What are user preferences?",
    user_id="user_123",
    rerank=True
)

# Batch create
client.create_memory_batch([
    {"content": "Memory 1", "user_id": "user_123"},
    {"content": "Memory 2", "user_id": "user_123"},
])

# Get stats
stats = client.get_memory_stats(user_id="user_123")
```

### JavaScript/TypeScript SDK

```typescript
import { AgentMemory } from '@agent-memory/sdk';

const client = new AgentMemory({
  baseUrl: 'https://api.yourserver.com',
  apiKey: 'your-api-key'
});

// Create memory
const memory = await client.memories.create({
  content: 'User prefers dark mode',
  userId: 'user_123',
  tags: ['settings'],
  importance: 'high'
});

// Search
const results = await client.search({
  query: 'What are user preferences?',
  userId: 'user_123'
});
```
