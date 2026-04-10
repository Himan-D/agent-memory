/**
 * LangGraph Integration for Agent Memory - Node.js SDK
 * 
 * Provides memory integration for LangGraph workflows and agents.
 * 
 * @example
 * ```typescript
 * import { AgentMemoryChecker, AgentMemoryUpdater } from 'agent-memory/integrations/langgraph';
 * import { StateGraph } from '@langchain/langgraph';
 * 
 * // Create memory tools for LangGraph
 * const checker = new AgentMemoryChecker({
 *   userId: 'user-123',
 *   baseUrl: 'http://localhost:8080'
 * });
 * 
 * const updater = new AgentMemoryUpdater({
 *   userId: 'user-123', 
 *   baseUrl: 'http://localhost:8080'
 * });
 * ```
 */

import { AgentMemory, type Memory, type MemoryResult } from '../index.js';

export interface LangGraphMemoryConfig {
  userId?: string;
  orgId?: string;
  agentId?: string;
  baseUrl: string;
  apiKey?: string;
}

export interface CheckMemoryInput {
  query: string;
  limit?: number;
  threshold?: number;
  memoryType?: 'user' | 'session' | 'conversation' | 'org';
}

export interface CheckMemoryOutput {
  memories: Memory[];
  relevantMemories: MemoryResult[];
  hasRelevantInfo: boolean;
}

export interface UpdateMemoryInput {
  content: string;
  category?: string;
  metadata?: Record<string, unknown>;
  immutable?: boolean;
  expirationDate?: Date;
}

export interface UpdateMemoryOutput {
  success: boolean;
  memoryId?: string;
  error?: string;
}

/**
 * Memory checker for LangGraph - retrieves relevant memories
 */
export class AgentMemoryChecker {
  private client: AgentMemory;
  private userId?: string;
  private orgId?: string;
  private agentId?: string;

  constructor(config: LangGraphMemoryConfig) {
    this.client = new AgentMemory({
      baseUrl: config.baseUrl,
      apiKey: config.apiKey,
    });
    this.userId = config.userId;
    this.orgId = config.orgId;
    this.agentId = config.agentId;
  }

  /**
   * Check for relevant memories in the store
   */
  async check(input: CheckMemoryInput): Promise<CheckMemoryOutput> {
    const results = await this.client.memories.search(input.query, {
      limit: input.limit ?? 10,
      threshold: input.threshold ?? 0.5,
      userId: this.userId,
      orgId: this.orgId,
      agentId: this.agentId,
      memoryType: input.memoryType,
    });

    const memories = results
      .map((r) => r.metadata)
      .filter((m): m is Memory => m !== undefined);

    return {
      memories,
      relevantMemories: results,
      hasRelevantInfo: results.length > 0,
    };
  }

  /**
   * Check with feedback consideration - boosts positive memories
   */
  async checkWithFeedback(input: CheckMemoryInput): Promise<CheckMemoryOutput> {
    const results = await this.client.memories.search(input.query, {
      limit: input.limit ?? 10,
      threshold: input.threshold ?? 0.3,
      userId: this.userId,
      orgId: this.orgId,
      agentId: this.agentId,
      memoryType: input.memoryType,
      rerank: true,
    });

    const memories = results
      .map((r) => r.metadata)
      .filter((m): m is Memory => m !== undefined);

    return {
      memories,
      relevantMemories: results,
      hasRelevantInfo: results.length > 0,
    };
  }
}

/**
 * Memory updater for LangGraph - stores new memories
 */
export class AgentMemoryUpdater {
  private client: AgentMemory;
  private userId?: string;
  private orgId?: string;
  private agentId?: string;

  constructor(config: LangGraphMemoryConfig) {
    this.client = new AgentMemory({
      baseUrl: config.baseUrl,
      apiKey: config.apiKey,
    });
    this.userId = config.userId;
    this.orgId = config.orgId;
    this.agentId = config.agentId;
  }

  /**
   * Store a new memory
   */
  async update(input: UpdateMemoryInput): Promise<UpdateMemoryOutput> {
    try {
      const memory = await this.client.memories.create({
        content: input.content,
        userId: this.userId,
        orgId: this.orgId,
        agentId: this.agentId,
        category: input.category,
        metadata: input.metadata,
        immutable: input.immutable,
        expirationDate: input.expirationDate,
      });

      return {
        success: true,
        memoryId: memory.id,
      };
    } catch (error) {
      return {
        success: false,
        error: error instanceof Error ? error.message : 'Unknown error',
      };
    }
  }

  /**
   * Store multiple memories
   */
  async updateBatch(inputs: UpdateMemoryInput[]): Promise<{ success: boolean; count: number; memoryIds?: string[] }> {
    try {
      const memories = await this.client.memories.batch.create(
        inputs.map((input) => ({
          content: input.content,
          userId: this.userId,
          orgId: this.orgId,
          agentId: this.agentId,
          category: input.category,
          metadata: input.metadata,
          immutable: input.immutable,
          expirationDate: input.expirationDate,
        }))
      );

      return {
        success: true,
        count: memories.count,
        memoryIds: memories.created.map((m) => m.id),
      };
    } catch (error) {
      return {
        success: false,
        count: 0,
      };
    }
  }
}

/**
 * Memory node for LangGraph StateGraph
 */
export interface LangGraphMemoryState {
  messages: Array<{ role: string; content: string }>;
  memories?: Memory[];
  query?: string;
  response?: string;
}

export class AgentMemoryNode {
  private checker: AgentMemoryChecker;
  private updater: AgentMemoryUpdater;

  constructor(config: LangGraphMemoryConfig) {
    this.checker = new AgentMemoryChecker(config);
    this.updater = new AgentMemoryUpdater(config);
  }

  /**
   * Retrieve relevant memories based on last message
   */
  async retrieveMemories(state: LangGraphMemoryState): Promise<Partial<LangGraphMemoryState>> {
    const lastMessage = state.messages[state.messages.length - 1];
    if (!lastMessage) return {};

    const query = lastMessage.content;
    const results = await this.checker.checkWithFeedback({
      query,
      limit: 5,
      threshold: 0.4,
    });

    return {
      memories: results.memories,
    };
  }

  /**
   * Store important information from the conversation
   */
  async storeMemory(
    state: LangGraphMemoryState,
    options?: { category?: string; metadata?: Record<string, unknown> }
  ): Promise<Partial<LangGraphMemoryState>> {
    const lastMessage = state.messages[state.messages.length - 1];
    if (!lastMessage || lastMessage.role !== 'assistant') return {};

    const result = await this.updater.update({
      content: lastMessage.content,
      category: options?.category ?? 'conversation',
      metadata: options?.metadata,
    });

    return result.success ? {} : {};
  }

  /**
   * Full memory workflow: retrieve, respond, store
   */
  async memoryAwareResponse(
    state: LangGraphMemoryState,
    options?: {
      retrieveCategory?: string;
      storeCategory?: string;
    }
  ): Promise<Partial<LangGraphMemoryState>> {
    const lastMessage = state.messages[state.messages.length - 1];
    if (!lastMessage) return {};

    const query = lastMessage.content;

    const { memories } = await this.checker.checkWithFeedback({
      query,
      limit: 5,
      threshold: 0.3,
    });

    if (memories.length > 0) {
      await this.updater.update({
        content: query,
        category: options?.retrieveCategory ?? 'conversation',
      });
    }

    return {
      memories: memories.map((m) => m),
    };
  }
}

export default {
  AgentMemoryChecker,
  AgentMemoryUpdater,
  AgentMemoryNode,
};
