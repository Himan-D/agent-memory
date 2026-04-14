/**
 * Hystersis - Node.js SDK
 * 
 * Persistent memory for AI agents with graph relationships and semantic search.
 * 
 * @example
 * ```typescript
 * import { Hystersis } from 'hystersis';
 * 
 * const client = new Hystersis({
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
  Hystersis,
  HystersisError,
  AuthenticationError,
  NotFoundError,
  ValidationError,
  RateLimitError,
} from './src/index.js';

export type {
  HystersisConfig,
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
  HystersisMemory,
  HystersisRetriever,
  
  // LlamaIndex
  HystersisReader,
  HystersisIndex,
  HystersisQueryEngine,
  HystersisStore,
  
  // AutoGen
  AutoGenMemory,
  AutoGenAgentMemory,
  
  // LangGraph
  HystersisChecker,
  HystersisUpdater,
  HystersisNode,
  
  // Mastra
  MastraMemoryTool,
  MastraMemoryStorage,
  
  // Agno
  HystersisStorage,
  HystersisField,
  createHystersisStorage,
} from './src/integrations/index.js';
