/**
 * LlamaIndex Integration for Agent Memory - Node.js SDK
 * 
 * Provides LlamaIndex components for Agent Memory.
 * 
 * @example
 * ```typescript
 * import { AgentMemoryIndex } from 'agent-memory/integrations/llamaindex';
 * 
 * const index = new AgentMemoryIndex({
 *   userId: 'user-123',
 *   baseUrl: 'http://localhost:8080'
 * });
 * 
 * // Query the index
 * const retriever = index.asRetriever();
 * const nodes = await retriever.retrieve('What did I learn?');
 * ```
 */

import { AgentMemory, type Memory, type MemoryResult } from '../index.js';

export interface LlamaIndexReaderConfig {
  baseUrl: string;
  apiKey?: string;
  userId?: string;
  orgId?: string;
  agentId?: string;
}

export interface LlamaIndexDocument {
  id: string;
  content: string;
  metadata: Record<string, unknown>;
}

/**
 * LlamaIndex Reader for loading memories from Agent Memory
 */
export class AgentMemoryReader {
  private client: AgentMemory;
  private userId?: string;
  private orgId?: string;
  private agentId?: string;

  constructor(config: LlamaIndexReaderConfig) {
    this.client = new AgentMemory({ baseUrl: config.baseUrl, apiKey: config.apiKey });
    this.userId = config.userId;
    this.orgId = config.orgId;
    this.agentId = config.agentId;
  }

  /**
   * Load memories as documents
   */
  async loadData(query?: string, limit = 10, memoryType?: string): Promise<LlamaIndexDocument[]> {
    if (query) {
      const results = await this.client.memories.search(query, {
        limit,
        userId: this.userId,
        orgId: this.orgId,
        agentId: this.agentId,
        memoryType: memoryType as 'user' | 'session' | 'conversation' | 'org' | undefined,
      });

      return results.map((r) => ({
        id: r.memoryId ?? r.entity.id,
        content: r.text,
        metadata: {
          memory_type: r.entity.properties?.memory_type,
          category: r.entity.properties?.category,
          user_id: r.entity.properties?.user_id,
          org_id: r.entity.properties?.org_id,
          agent_id: r.entity.properties?.agent_id,
          created_at: r.entity.properties?.created_at,
          score: r.score,
        },
      }));
    }

    const result = await this.client.memories.list({
      userId: this.userId,
      orgId: this.orgId,
      agentId: this.agentId,
    });

    return result.memories.slice(0, limit).map((m) => ({
      id: m.id,
      content: m.content,
      metadata: {
        memory_type: m.type,
        category: m.category,
        user_id: m.userId,
        org_id: m.orgId,
        agent_id: m.agentId,
        created_at: m.createdAt,
      },
    }));
  }

  /**
   * Load memories by feedback score
   */
  async loadMemoriesByFeedback(
    feedbackType: 'positive' | 'negative' | 'very_negative',
    limit = 50
  ): Promise<LlamaIndexDocument[]> {
    const memories = await this.client.feedback.getByType(feedbackType, limit);

    return memories.map((m) => ({
      id: m.id,
      content: m.content,
      metadata: {
        memory_type: m.type,
        category: m.category,
        feedback_score: m.feedbackScore,
        created_at: m.createdAt,
      },
    }));
  }
}

export interface LlamaIndexIndexConfig {
  baseUrl: string;
  apiKey?: string;
  userId?: string;
  orgId?: string;
  agentId?: string;
  memoryType?: 'user' | 'session' | 'conversation' | 'org';
}

/**
 * LlamaIndex-compatible index using Agent Memory
 */
export class AgentMemoryIndex {
  private client: AgentMemory;
  private userId?: string;
  private orgId?: string;
  private agentId?: string;
  private memoryType?: 'user' | 'session' | 'conversation' | 'org';

  constructor(config: LlamaIndexIndexConfig) {
    this.client = new AgentMemory({ baseUrl: config.baseUrl, apiKey: config.apiKey });
    this.userId = config.userId;
    this.orgId = config.orgId;
    this.agentId = config.agentId;
    this.memoryType = config.memoryType;
  }

  /**
   * Query the index
   */
  async query(
    query: string,
    limit = 5,
    threshold = 0.5
  ): Promise<MemoryResult[]> {
    return this.client.memories.search(query, {
      limit,
      threshold,
      userId: this.userId,
      orgId: this.orgId,
      agentId: this.agentId,
      memoryType: this.memoryType,
    });
  }

  /**
   * Retrieve relevant memories
   */
  async retrieve(query: string, limit = 5, threshold = 0.5): Promise<MemoryResult[]> {
    return this.query(query, limit, threshold);
  }

  /**
   * Get a retriever for this index
   */
  asRetriever(config?: { similarityTopK?: number; scoreThreshold?: number }): AgentMemoryRetriever {
    return new AgentMemoryRetriever({
      baseUrl: this.client['baseUrl'],
      apiKey: this.client['apiKey'],
      userId: this.userId,
      orgId: this.orgId,
      agentId: this.agentId,
      memoryType: this.memoryType,
      topK: config?.similarityTopK,
      scoreThreshold: config?.scoreThreshold,
    });
  }

  /**
   * Get a query engine for this index
   */
  asQueryEngine(config?: { similarityTopK?: number; scoreThreshold?: number }): AgentMemoryQueryEngine {
    return new AgentMemoryQueryEngine(this, config);
  }

  /**
   * Insert a new memory
   */
  async insertMemory(
    content: string,
    options?: {
      category?: string;
      metadata?: Record<string, unknown>;
      immutable?: boolean;
      expirationDate?: Date;
    }
  ): Promise<Memory> {
    return this.client.memories.create({
      content,
      memoryType: this.memoryType,
      userId: this.userId,
      orgId: this.orgId,
      agentId: this.agentId,
      category: options?.category,
      metadata: options?.metadata,
      immutable: options?.immutable,
      expirationDate: options?.expirationDate,
    });
  }

  /**
   * Delete a memory by ID
   */
  async deleteMemory(memoryId: string): Promise<boolean> {
    try {
      await this.client.memories.delete(memoryId);
      return true;
    } catch {
      return false;
    }
  }
}

/**
 * LlamaIndex Retriever for Agent Memory
 */
export class AgentMemoryRetriever {
  private client: AgentMemory;
  private userId?: string;
  private orgId?: string;
  private agentId?: string;
  private memoryType?: 'user' | 'session' | 'conversation' | 'org';
  private similarityTopK: number;
  private scoreThreshold: number;

  constructor(config: {
    baseUrl: string;
    apiKey?: string;
    userId?: string;
    orgId?: string;
    agentId?: string;
    memoryType?: 'user' | 'session' | 'conversation' | 'org';
    topK?: number;
    scoreThreshold?: number;
  }) {
    this.client = new AgentMemory({ baseUrl: config.baseUrl, apiKey: config.apiKey });
    this.userId = config.userId;
    this.orgId = config.orgId;
    this.agentId = config.agentId;
    this.memoryType = config.memoryType;
    this.similarityTopK = config.topK ?? 5;
    this.scoreThreshold = config.scoreThreshold ?? 0.5;
  }

  /**
   * Retrieve relevant memories
   */
  async retrieve(query: string): Promise<MemoryResult[]> {
    return this.client.memories.search(query, {
      limit: this.similarityTopK,
      threshold: this.scoreThreshold,
      userId: this.userId,
      orgId: this.orgId,
      agentId: this.agentId,
      memoryType: this.memoryType,
    });
  }
}

/**
 * LlamaIndex Query Engine for Agent Memory
 */
export class AgentMemoryQueryEngine {
  private index: AgentMemoryIndex;
  private similarityTopK: number;
  private scoreThreshold: number;

  constructor(
    index: AgentMemoryIndex,
    config?: { similarityTopK?: number; scoreThreshold?: number }
  ) {
    this.index = index;
    this.similarityTopK = config?.similarityTopK ?? 5;
    this.scoreThreshold = config?.scoreThreshold ?? 0.5;
  }

  /**
   * Query the memory index
   */
  async query(query: string): Promise<{
    query: string;
    results: MemoryResult[];
    response: string;
    sourceNodes: MemoryResult[];
  }> {
    const results = await this.index.query(query, this.similarityTopK, this.scoreThreshold);

    return {
      query,
      results,
      response: this.formatResponse(results),
      sourceNodes: results,
    };
  }

  /**
   * Format results into a response string
   */
  private formatResponse(results: MemoryResult[]): string {
    if (results.length === 0) {
      return 'No relevant memories found.';
    }

    const lines = ['Here are relevant memories:\n'];
    results.forEach((r, i) => {
      lines.push(`${i + 1}. ${r.text} (relevance: ${r.score.toFixed(2)})`);
    });

    return lines.join('\n');
  }
}

/**
 * LlamaIndex Node storage using Agent Memory
 */
export class AgentMemoryStore {
  private client: AgentMemory;
  private userId?: string;
  private orgId?: string;

  constructor(config: { baseUrl: string; apiKey?: string; userId?: string; orgId?: string }) {
    this.client = new AgentMemory({ baseUrl: config.baseUrl, apiKey: config.apiKey });
    this.userId = config.userId;
    this.orgId = config.orgId;
  }

  /**
   * Store a node
   */
  async put(key: string, node: { content: string; metadata?: Record<string, unknown> }): Promise<void> {
    await this.client.memories.create({
      content: node.content,
      type: 'user',
      userId: this.userId,
      orgId: this.orgId,
      metadata: { node_id: key, ...node.metadata },
    });
  }

  /**
   * Retrieve a node
   */
  async get(key: string): Promise<{ content: string; metadata?: Record<string, unknown> } | null> {
    const results = await this.client.memories.search(`node_id:${key}`, { limit: 1 });

    for (const r of results) {
      if (r.entity.properties?.node_id === key) {
        return {
          content: r.text,
          metadata: r.entity.properties,
        };
      }
    }

    return null;
  }

  /**
   * Delete a node
   */
  async delete(key: string): Promise<boolean> {
    const node = await this.get(key);
    if (node && key) {
      try {
        await this.client.memories.delete(key);
        return true;
      } catch {
        return false;
      }
    }
    return false;
  }

  /**
   * Check if a key exists
   */
  async has(key: string): Promise<boolean> {
    const node = await this.get(key);
    return node !== null;
  }
}
