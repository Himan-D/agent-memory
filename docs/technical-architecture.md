# Technical Architecture

## System Components

### Core Services

#### 1. Memory Service (`internal/memory/service.go`)
- **Lines of Code**: ~2,500
- **Responsibilities**: Core memory operations, compression coordination, audit logging
- **Key Interfaces**: 
  - `MemoryProcessor`: Extract facts, entities, relationships
  - `CompressionService`: Handle ProMem compression
  - `AuditLogger`: Track all operations

#### 2. Neo4j Graph Implementation (`internal/memory/neo4j/`)
- **Lines of Code**: ~3,500
- **Responsibilities**: Graph storage, relationship management, traversal
- **Key Interfaces**:
  - `GraphStore`: Neo4j operations interface
  - `RelationshipManager`: Handle entity connections
  - `GraphTraverser`: Multi-hop relationship following

#### 3. API Server (`cmd/server/api.go`)
- **Lines of Code**: ~2,600
- **Responsibilities**: REST API endpoints, request handling, error management
- **Key Features**:
  - Memory CRUD operations
  - Search endpoints (semantic + graph)
  - Skills management
  - Compression control

#### 4. Skills Registry (`internal/skills/registry.go`)
- **Lines of Code**: ~1,000
- **Responsibilities**: Skill discovery, execution, lifecycle management
- **Key Features**:
  - File-based skills (.md with YAML frontmatter)
  - Neo4j-backed procedural skills
  - Skill chain execution
  - Human review workflow

### Proprietary Components

#### 5. Compression Engine (`internal/compression/`)
- **Lines of Code**: ~5,000 (proprietary)
- **Responsibilities**: Advanced memory compression and retrieval
- **Key Components**:
  - `extractor/`: ProMem-style fact extraction
  - `retrieval/`: Spreading activation search
  - `pipeline/`: Async compression processing
  - `llm/`: Hybrid LLM routing

#### 6. Metrics System (`internal/metrics/`)
- **Lines of Code**: ~800
- **Responsibilities**: Performance monitoring and observability
- **Key Features**:
  - Compression accuracy tracking
  - Latency monitoring
  - Error rate analysis
  - Performance benchmarking

## Data Flow Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                      DATA FLOW PIPELINE                        │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  INPUT → LLM Extraction → Fact Analysis → Graph Integration     │
│    ↓         ↓              ↓             ↓                     │
│  Memory → Compression → Tier Routing → Storage                 │
│    ↓         ↓              ↓             ↓                     │
│  Search → Vector Similarity → Spreading Activation → Results    │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### Memory Processing Pipeline

1. **Ingestion**: Content received via API or SDK
2. **LLM Processing**: `MemoryProcessor.ExtractFacts()` extracts facts and entities
3. **Compression**: ProMem extraction reduces token usage by 85%
4. **Graph Integration**: Entities and relationships stored in Neo4j
5. **Vector Embedding**: Content embedded and stored in Qdrant
6. **Tier Routing**: Automatically routed to appropriate storage tier
7. **Search Indexing**: Available for semantic and graph search

### Search Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                      SEARCH ARCHITECTURE                        │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  Query → Vector Similarity (Qdrant) → Initial Results          │
│    ↓                                                  ↓        │
│  Graph Traversal (Neo4j) → Spreading Activation → Enhanced     │
│    ↓                                                  ↓        │
│  Hybrid Ranking → Context Aggregation → Final Results           │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

## Component Interactions

### Memory ↔ Compression
- **Compression Trigger**: Memory insertion > complexity threshold (0.6)
- **Fast Path**: GPT-4o-mini for simple extraction
- **Verify Path**: Claude for complex verification
- **Async Processing**: Non-blocking with <5ms write impact

### Memory ↔ Skills
- **Skill Discovery**: Extract skills from memory content
- **Skill Execution**: Context-aware skill activation
- **Skill Chains**: Multi-step workflows triggered by memory patterns
- **Feedback Loop**: Skill execution results improve memory quality

### Skills ↔ Compression
- **Skill Compression**: Compress skill definitions for storage
- **Skill Retrieval**: Spreading activation finds relevant skills
- **Performance Tracking**: Monitor skill execution success rates
- **Optimization**: Adjust compression based on skill usage

## Database Architecture

### Neo4j Schema
```cypher
// Memory nodes
(m:Memory {id, content, type, created_at, updated_at})

// Entity nodes  
(entity:Entity {id, name, type, confidence})

// Relationship types
-[:CONTAINS {strength}] →
-[:RELATED_TO {relationship_type, strength}] →
-[:PART_OF {context}] →
```

### Qdrant Schema
```json
{
  "vectors": {
    "memory": {
      "size": 1536,
      "distance": "Cosine"
    }
  },
  "points": [
    {
      "id": "memory-123",
      "vector": [0.1, 0.2, ...],
      "payload": {
        "content": "memory content",
        "type": "semantic",
        "timestamp": "2026-05-08T10:30:00Z"
      }
    }
  ]
}
```

## API Architecture

### REST Endpoints
```
Memory Operations:
  POST /memories → Create memory
  GET /memories → List memories
  GET /memories/{id} → Get memory
  PUT /memories/{id} → Update memory
  DELETE /memories/{id} → Delete memory

Search Operations:
  GET /search → Semantic search
  POST /search → Search with filters
  GET /search/enhanced → Hybrid + graph search

Skills Operations:
  POST /skills → Create skill
  GET /skills → List skills
  POST /skills/{id}/execute → Execute skill
  POST /skills/suggest → LLM suggestions

Compression Operations:
  GET /compression/stats → Performance metrics
  PUT /compression/mode → Set compression mode
  GET /compression/mode → Get compression mode
```

### MCP Server Tools
```
Memory Tools:
- add_memory / search_memories / get_memories
- create_entity / create_relation / get_context
- create_session / add_message
- add_feedback

Skills Tools:
- execute_skill / get_skills / search_skills
- create_skill_chain / execute_chain

Compression Tools:
- compress_memory / get_compression_stats
- set_compression_mode / get_compression_mode
```

## Performance Architecture

### Concurrency Model
- **Goroutines**: Lightweight concurrent processing
- **Worker Pools**: 4-8 workers for async compression
- **Connection Pools**: Database connection reuse
- **Rate Limiting**: Prevent resource exhaustion

### Caching Strategy
- **Hot Tier**: Redis for frequently accessed memories
- **Working Memory**: In-memory cache for active sessions
- **Cache Invalidation**: TTL-based and event-driven
- **Cache Warming**: Predictive loading based on patterns

### Error Handling
- **Graceful Degradation**: Fallback to basic operations
- **Circuit Breakers**: Prevent cascading failures
- **Retry Logic**: Exponential backoff for transient errors
- **Dead Letter Queue**: Handle persistent failures

## Security Architecture

### Authentication & Authorization
- **API Keys**: Multi-tenant key management
- **SSO Integration**: OIDC, SAML, LDAP support
- **Role-Based Access**: Granular permissions
- **Rate Limiting**: Per-tenant usage limits

### Data Protection
- **Encryption**: AES-256 at rest, TLS 1.3 in transit
- **Masking**: Sensitive data redaction in logs
- **Access Control**: Row-level security in databases
- **Audit Trail**: Complete operation logging

## Monitoring Architecture

### Metrics Collection
- **Prometheus**: Standard metrics collection
- **Custom Metrics**: Compression accuracy, skill success rates
- **Alerting**: Threshold-based alerting
- **Dashboards**: Real-time visualization

### Logging Strategy
- **Structured Logging**: JSON format for parsing
- **Log Levels**: Debug, Info, Warn, Error
- **Log Retention**: Rolling log files with rotation
- **Centralized Logging**: Integration with external services

---

*This architecture provides the foundation for the soul file orchestrator system with comprehensive component interactions and performance optimization.*