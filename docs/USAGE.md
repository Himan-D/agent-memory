# Agent Memory - Usage Examples

This document provides comprehensive examples for using Agent Memory effectively.

## Table of Contents

1. [Python SDK Examples](#python-sdk-examples)
2. [JavaScript SDK Examples](#javascript-sdk-examples)
3. [cURL Examples](#curl-examples)
4. [Advanced Patterns](#advanced-patterns)

---

## Python SDK Examples

### Installation

```bash
pip install agentmemory
```

### Basic Setup

```python
from agentmemory import AgentMemory

# Connect to local server
client = AgentMemory(
    base_url="http://localhost:8080",
    api_key="your-api-key"
)

# Or connect to cloud
client = AgentMemory(
    base_url="https://api.agentmemory.ai",
    api_key="your-cloud-api-key"
)
```

---

### Creating Memories

#### Simple Memory

```python
memory = client.create_memory(
    content="User prefers dark mode in their IDE",
    user_id="user_123"
)
print(f"Created memory: {memory['id']}")
```

#### Memory with Full Options

```python
memory = client.create_memory(
    content="User is learning Python and prefers video tutorials",
    user_id="user_123",
    agent_id="tutor_bot",
    session_id="session_abc",
    type="conversation",
    category="learning",
    tags=["python", "tutorials", "learning"],
    importance="high",
    metadata={
        "skill_level": "beginner",
        "preferred_format": "video"
    },
    process=True,  # Enable LLM extraction
    immutable=False,
    expiration_date="2025-12-31T23:59:59Z"
)
```

#### Skip LLM Processing

```python
# Store raw content without processing
memory = client.create_memory(
    content="EXACT_CONTENT_TO_STORE",
    user_id="user_123",
    process=False  # Skip LLM processing
)
```

---

### Searching Memories

#### Semantic Search

```python
results = client.search(
    query="What are the user's coding preferences?",
    user_id="user_123",
    limit=10
)

for result in results:
    print(f"[{result['score']:.2f}] {result['text']}")
```

#### Search with Filters

```python
results = client.search(
    query="Python tutorials",
    user_id="user_123",
    category="learning",
    tags=["python", "tutorials"],
    importance="high",
    limit=5,
    rerank=True  # Enable reranking for better results
)
```

#### Hybrid Search (Semantic + Keyword)

```python
results = client.hybrid_search(
    query="dark mode IDE settings",
    user_id="user_123",
    semantic_limit=10,
    keyword_limit=10,
    boost=1.5,
    threshold=0.6
)
```

#### Advanced Filter Search

```python
results = client.advanced_search(
    query="preferences",
    user_id="user_123",
    filters={
        "logic": "AND",
        "rules": [
            {"field": "category", "operator": "eq", "value": "preferences"},
            {"field": "importance", "operator": "in", "value": ["high", "critical"]}
        ],
        "nested": [
            {
                "logic": "OR",
                "rules": [
                    {"field": "tags", "operator": "contains", "value": "ide"},
                    {"field": "tags", "operator": "contains", "value": "editor"}
                ]
            }
        ]
    }
)
```

---

### Memory Management

#### Get Memory

```python
memory = client.get_memory("mem_abc123")
print(f"Content: {memory['content']}")
print(f"Importance: {memory['importance']}")
```

#### Update Memory

```python
client.update_memory(
    id="mem_abc123",
    content="Updated content here",
    metadata={"verified": True}
)
```

#### Delete Memory

```python
client.delete_memory("mem_abc123")
```

#### Batch Operations

```python
# Batch create
memories = client.create_memory_batch([
    {"content": "Memory 1", "user_id": "user_123"},
    {"content": "Memory 2", "user_id": "user_123"},
    {"content": "Memory 3", "user_id": "user_123"}
], process=True)

# Batch update
client.batch_update(
    ids=["mem_1", "mem_2", "mem_3"],
    action="archive"
)

# Batch delete
client.batch_delete(ids=["mem_4", "mem_5"])
```

---

### Memory Relationships

#### Link Memories

```python
# Create parent-child relationship
link = client.create_memory_link(
    from_id="mem_parent",
    to_id="mem_child",
    link_type="parent",
    weight=0.9,
    metadata={"context": "Sub-discussion"}
)

# Create related relationship
link = client.create_memory_link(
    from_id="mem_1",
    to_id="mem_2", 
    link_type="related",
    weight=0.75
)
```

#### Get Related Memories

```python
# Get all linked memories
links = client.get_memory_links("mem_abc123")

# Get memories by link type
related = client.get_related_memories(
    memory_id="mem_abc123",
    link_type="parent",
    limit=10
)
```

---

### Memory Versioning

#### Save Version

```python
# Automatically saves before update
version = client.save_memory_version(
    memory_id="mem_abc123",
    content="New content",
    created_by="user_123"
)
```

#### Get Version History

```python
versions = client.get_memory_versions("mem_abc123")
for v in versions:
    print(f"v{v['version']}: {v['content'][:50]}...")
```

#### Restore Version

```python
client.restore_memory_version(
    memory_id="mem_abc123",
    version_id="ver_xyz789"
)
```

---

### Feedback

```python
# Add positive feedback
client.add_feedback(
    memory_id="mem_abc123",
    feedback_type="positive",
    comment="This is accurate!"
)

# Add negative feedback
client.add_feedback(
    memory_id="mem_abc123", 
    feedback_type="negative",
    comment="This is outdated"
)

# Get memories with negative feedback
negative_memories = client.get_memories_by_feedback(
    feedback_type="negative",
    limit=10
)
```

---

### Sessions & Context

```python
# Create session
session = client.create_session(
    agent_id="assistant_bot",
    metadata={"channel": "web"}
)

# Add messages
client.add_message(session["id"], "user", "I need help with Python")
client.add_message(session["id"], "assistant", "What specific topic?")
client.add_message(session["id"], "user", "Web scraping with BeautifulSoup")

# Get conversation history
messages = client.get_messages(session["id"])

# Get formatted context for LLM
context = client.get_context(session["id"], limit=10)
```

---

### Analytics

#### Get Memory Stats

```python
stats = client.get_memory_stats(user_id="user_123")

print(f"Total memories: {stats['total_memories']}")
print(f"By category: {stats['by_category']}")
print(f"By importance: {stats['by_importance']}")
print(f"Top tags: {stats['top_tags']}")
```

#### Get AI Insights

```python
insights = client.get_memory_insights(user_id="user_123")
for insight in insights:
    print(f"[{insight['type']}] {insight['description']}")
```

---

### Memory Summary & Compression

```python
# Generate memory summary
summary = client.get_memory_summary(user_id="user_123")

print(f"Summary: {summary['summary']}")
print(f"Key points: {summary['key_points']}")
print(f"Token savings: {summary['token_savings']:.0%}")
```

---

### Export & Import

```python
# Export all user memories
export = client.export_memories(user_id="user_123")

# Save to file
import json
with open("user_memories_backup.json", "w") as f:
    json.dump(export, f, indent=2)

# Import memories
import json
with open("user_memories_backup.json") as f:
    data = json.load(f)

client.import_memories(
    memories=data["memories"],
    overwrite=False,
    merge_mode="append"
)
```

---

## JavaScript SDK Examples

### Installation

```bash
npm install @agent-memory/sdk
```

### Basic Usage

```javascript
import { AgentMemory } from '@agent-memory/sdk';

const client = new AgentMemory({
  baseUrl: 'http://localhost:8080',
  apiKey: 'your-api-key'
});

// Create memory
const memory = await client.memories.create({
  content: 'User prefers TypeScript',
  userId: 'user_123',
  tags: ['typescript', 'programming'],
  importance: 'high'
});

// Search
const results = await client.search({
  query: 'What languages does user prefer?',
  userId: 'user_123'
});
```

---

## cURL Examples

### Create Memory

```bash
curl -X POST http://localhost:8080/api/v1/memories \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your-api-key" \
  -d '{
    "content": "User prefers dark mode",
    "user_id": "user_123",
    "category": "preferences",
    "tags": ["settings", "ui"],
    "importance": "high"
  }'
```

### Search

```bash
curl -X POST http://localhost:8080/api/v1/search \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your-api-key" \
  -d '{
    "query": "What are user preferences?",
    "user_id": "user_123",
    "limit": 10,
    "rerank": true
  }'
```

### Get Memory

```bash
curl http://localhost:8080/api/v1/memories/mem_abc123 \
  -H "X-API-Key: your-api-key"
```

### List Memories with Filters

```bash
curl "http://localhost:8080/api/v1/memories?user_id=user_123&category=preferences&page=1&page_size=20" \
  -H "X-API-Key: your-api-key"
```

### Add Feedback

```bash
curl -X POST http://localhost:8080/api/v1/feedback \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your-api-key" \
  -d '{
    "memory_id": "mem_abc123",
    "type": "positive",
    "comment": "Helpful memory!"
  }'
```

### Create Session

```bash
curl -X POST http://localhost:8080/api/v1/sessions \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your-api-key" \
  -d '{
    "agent_id": "assistant_bot",
    "metadata": {"channel": "web"}
  }'
```

### Add Message to Session

```bash
curl -X POST http://localhost:8080/api/v1/sessions/sess_xyz/messages \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your-api-key" \
  -d '{
    "role": "user",
    "content": "I need help with coding"
  }'
```

---

## Advanced Patterns

### Multi-Agent Memory

```python
# Agent-specific memories
client.create_memory(
    content="Code review agent notes",
    user_id="user_123",
    agent_id="code_review_agent",
    category="reviews"
)

client.create_memory(
    content="Research agent findings",
    user_id="user_123", 
    agent_id="research_agent",
    category="research"
)

# Search across all agents
results = client.search(
    query="user preferences",
    user_id="user_123"
)

# Search specific agent
results = client.search(
    query="code patterns",
    user_id="user_123",
    agent_id="code_review_agent"
)
```

### Memory Organization

```python
# Hierarchical categories
client.create_memory(
    content="Project deadline",
    user_id="user_123",
    category="work/projects/alpha"
)

# Use tags for cross-cutting concerns
client.create_memory(
    content="Important client meeting",
    user_id="user_123",
    category="meetings",
    tags=["client", "important", "work"]
)
```

### Importance-Based Processing

```python
# Critical - preserved exactly
client.create_memory(
    content="User's birthday: March 15",
    user_id="user_123",
    importance="critical",
    immutable=True  # Never delete
)

# High importance
client.create_memory(
    content="User's preferred coffee order",
    user_id="user_123",
    importance="high"
)

# Low importance - aggressive compression
client.create_memory(
    content="Casual chat about weather",
    user_id="user_123",
    importance="low"
)
```

### Async Operations

```python
# For high-throughput scenarios
results = client.create_memory_batch_async(
    memories=[...],  # Large batch
    callback=lambda r: print(f"Created: {r.id}")
)
```

### Compaction Scheduler

```python
import schedule

def run_daily_compaction():
    client.run_compaction(
        user_id="user_123",
        action="full"
    )

# Schedule daily at 2am
schedule.every().day.at("02:00").do(run_daily_compaction)
```

---

## Error Handling

```python
from agentmemory.exceptions import (
    MemoryNotFoundError,
    RateLimitError,
    AuthenticationError
)

try:
    memory = client.get_memory("non_existent_id")
except MemoryNotFoundError:
    print("Memory not found!")
except RateLimitError:
    print("Rate limited, wait and retry")
except AuthenticationError:
    print("Invalid API key")
```

---

## Best Practices

1. **Use appropriate importance levels** - Critical memories are preserved, low importance memories may be compressed

2. **Use tags for cross-cutting concerns** - Tags work across categories

3. **Enable reranking for precision** - Reranking improves result quality at cost of latency

4. **Use sessions for conversations** - Sessions provide better context than individual memories

5. **Provide feedback** - Feedback improves future search relevance

6. **Run compaction periodically** - Compaction deduplicates and summarizes old memories

7. **Export regularly** - Back up important memories

8. **Use immutable for critical data** - Critical facts should be immutable
