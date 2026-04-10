/**
 * Agent Memory System - Node.js SDK
 * 
 * Persistent memory for AI agents with graph relationships and semantic search.
 * 
 * @example
 * ```typescript
 * import { AgentMemory } from 'agent-memory';
 * 
 * const client = new AgentMemory({
 *   baseUrl: 'http://localhost:8080',
 *   apiKey: 'your-api-key'
 * });
 * 
 * // Create a session
 * const session = await client.sessions.create({ agentId: 'my-agent' });
 * 
 * // Add messages
 * await client.sessions.messages.add(session.id, 'user', 'Hello!');
 * 
 * // Search memories
 * const results = await client.memories.search('previous conversations');
 * ```
 */

// Core SDK
export {
  AgentMemory,
  AgentMemoryError,
  AuthenticationError,
  NotFoundError,
  ValidationError,
  RateLimitError,
} from './src/index.js';

export type {
  AgentMemoryConfig,
  Memory,
  Message,
  Session,
  Entity,
  Relation,
  MemoryResult,
  SearchRequest,
  SearchFilters,
  Feedback,
  MemoryHistory,
  Project,
  Webhook,
  HealthStatus,
} from './src/index.js';

// Integrations
export {
  // LangChain
  AgentMemoryMemory,
  AgentMemoryRetriever,
  
  // LlamaIndex
  AgentMemoryReader,
  AgentMemoryIndex,
  AgentMemoryQueryEngine,
  AgentMemoryStore,
  
  // AutoGen
  AutoGenMemory,
  AutoGenAgentMemory,
  
  // LangGraph
  AgentMemoryChecker,
  AgentMemoryUpdater,
  AgentMemoryNode,
  
  // Mastra
  MastraMemoryTool,
  MastraMemoryStorage,
  
  // Agno
  AgentMemoryStorageImpl,
  AgentMemoryField,
  createAgentMemoryStorage,
} from './src/integrations/index.js';
