# Hystersis

Persistent memory infrastructure for AI agents.

```bash
npm install hystersis
```

## Usage

```typescript
import { HystersisClient } from 'hystersis'

const client = new HystersisClient({
  baseUrl: 'http://localhost:8080',
  apiKey: '...'
})

// Store a memory
const memory = await client.createMemory({
  agentId: 'agent-1',
  content: 'The user prefers dark mode',
  type: 'observation'
})

// Search with spreading activation
const results = await client.searchEnhanced({
  query: 'user preferences',
  mode: 'spreading'
})
```

## Integrations

```typescript
import { HystersisMemory } from 'hystersis/integrations'
```

Supports: LangChain, LlamaIndex, LangGraph, CrewAI, AutoGen, OpenAI Agents, Vercel AI SDK, Mastra, Agno, Google ADK.

## Docs

https://hystersis.ai/docs
