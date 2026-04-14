/**
 * Hystersis Integrations - Node.js SDK
 * 
 * Integrations with popular AI frameworks:
 * - LangChain: Memory components and retrievers
 * - LlamaIndex: Reader, index, and query engine
 * - AutoGen: Multi-agent shared memory
 * - LangGraph: Memory nodes for workflows
 * - Mastra: Tool and storage for agents
 * - Agno: Storage for AI agents
 * - CrewAI: Shared memory for crews
 */

export { HystersisMemory, HystersisRetriever } from './langchain.js';
export { HystersisRetriever as LlamaIndexRetriever } from './langchain.js';

export {
  HystersisReader,
  HystersisIndex,
  HystersisRetriever,
  HystersisQueryEngine,
  HystersisStore,
} from './llamaindex.js';

export { AutoGenMemory, AutoGenAgentMemory } from './autogen.js';

export {
  HystersisChecker,
  HystersisUpdater,
  HystersisNode,
  type LangGraphMemoryState,
} from './langgraph.js';

export { MastraMemoryTool, MastraMemoryStorage } from './mastra.js';

export {
  HystersisStorageImpl as HystersisStorage,
  HystersisField,
  createHystersisStorage,
} from './agno.js';

export { CrewMemory, CrewAgentMemory } from './crewai.js';

export type {
  LangChainMemoryConfig,
  MastraMemoryConfig,
  AgnoMemoryConfig,
  AutoGenMemoryConfig,
  LlamaIndexReaderConfig,
  CrewMemoryConfig,
  CrewAgentConfig,
} from './types.js';
