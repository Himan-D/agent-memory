/**
 * Agno Integration for Hystersis - Node.js SDK
 * 
 * Provides memory integration for Agno AI agents with storage and retrieval.
 * 
 * @example
 * ```typescript
 * import { HystersisStorage } from 'hystersis/integrations/agno';
 * import { Agent } from 'agno';
 * 
 * const storage = new HystersisStorage({
 *   userId: 'user-123',
 *   baseUrl: 'http://localhost:8080'
 * });
 * 
 * const agent = Agent({
 *   name: 'Assistant',
 *   storage
 * });
 * ```
 */

import { Hystersis, type Memory, type MemoryResult } from '../index.js';

export interface AgnoMemoryConfig {
  userId?: string;
  orgId?: string;
  agentId?: string;
  sessionId?: string;
  baseUrl: string;
  apiKey?: string;
}

export interface MemoryEntry {
  id?: string;
  content: string;
  embedding?: number[];
  metadata?: Record<string, unknown>;
  createdAt?: string;
  updatedAt?: string;
}

export interface SearchResult {
  id: string;
  content: string;
  score: number;
  metadata?: Record<string, unknown>;
}

export interface HystersisStorage {
  /**
   * Store a memory
   */
  create(entry: MemoryEntry): Promise<string>;

  /**
   * Retrieve a memory by ID
   */
  get(id: string): Promise<MemoryEntry | null>;

  /**
   * Update a memory
   */
  update(id: string, entry: Partial<MemoryEntry>): Promise<void>;

  /**
   * Delete a memory
   */
  delete(id: string): Promise<void>;

  /**
   * Search memories by query
   */
  search(query: string, limit?: number, threshold?: number): Promise<SearchResult[]>;

  /**
   * List all memories
   */
  list(limit?: number): Promise<MemoryEntry[]>;

  /**
   * Count total memories
   */
  count(): Promise<number>;
}

/**
 * Agno-compatible memory storage using Hystersis
 */
export class HystersisStorageImpl implements HystersisStorage {
  private client: Hystersis;
  private userId?: string;
  private orgId?: string;
  private agentId?: string;
  private sessionId?: string;

  constructor(config: AgnoMemoryConfig) {
    this.client = new Hystersis({
      baseUrl: config.baseUrl,
      apiKey: config.apiKey,
    });
    this.userId = config.userId;
    this.orgId = config.orgId;
    this.agentId = config.agentId;
    this.sessionId = config.sessionId;
  }

  /**
   * Create a new memory entry
   */
  async create(entry: MemoryEntry): Promise<string> {
    const memory = await this.client.memories.create({
      content: entry.content,
      userId: this.userId,
      orgId: this.orgId,
      agentId: this.agentId,
      sessionId: this.sessionId,
      metadata: entry.metadata,
    });

    return memory.id;
  }

  /**
   * Get a memory by ID
   */
  async get(id: string): Promise<MemoryEntry | null> {
    try {
      const memory = await this.client.memories.get(id);
      return this.memoryToEntry(memory);
    } catch {
      return null;
    }
  }

  /**
   * Update a memory entry
   */
  async update(id: string, entry: Partial<MemoryEntry>): Promise<void> {
    if (entry.content) {
      await this.client.memories.update(id, {
        content: entry.content,
        metadata: entry.metadata,
      });
    } else if (entry.metadata) {
      await this.client.memories.update(id, {
        content: '',
        metadata: entry.metadata,
      });
    }
  }

  /**
   * Delete a memory entry
   */
  async delete(id: string): Promise<void> {
    await this.client.memories.delete(id);
  }

  /**
   * Search memories by query
   */
  async search(query: string, limit = 10, threshold = 0.5): Promise<SearchResult[]> {
    const results = await this.client.memories.search(query, {
      limit,
      threshold,
      userId: this.userId,
      orgId: this.orgId,
      agentId: this.agentId,
      rerank: true,
    });

    return results.map((r) => ({
      id: r.memoryId ?? r.entity.id,
      content: r.text,
      score: r.score,
      metadata: r.entity.properties,
    }));
  }

  /**
   * List all memories
   */
  async list(limit = 100): Promise<MemoryEntry[]> {
    const result = await this.client.memories.list({
      userId: this.userId,
      orgId: this.orgId,
    });

    return result.memories.slice(0, limit).map((m) => this.memoryToEntry(m));
  }

  /**
   * Count total memories
   */
  async count(): Promise<number> {
    const result = await this.client.memories.list({
      userId: this.userId,
      orgId: this.orgId,
    });

    return result.count;
  }

  private memoryToEntry(memory: Memory): MemoryEntry {
    return {
      id: memory.id,
      content: memory.content,
      metadata: memory.metadata,
      createdAt: memory.createdAt,
      updatedAt: memory.updatedAt,
    };
  }
}

/**
 * Hystersis Field for Agno agents
 */
export class HystersisField {
  private storage: HystersisStorageImpl;
  private fieldName: string;

  constructor(config: AgnoMemoryConfig & { fieldName: string }) {
    this.storage = new HystersisStorageImpl(config);
    this.fieldName = config.fieldName;
  }

  /**
   * Get the field value
   */
  async get(): Promise<string[]> {
    const entries = await this.storage.list();
    return entries.map((e) => e.content);
  }

  /**
   * Set the field value
   */
  async set(value: string | string[]): Promise<void> {
    const values = Array.isArray(value) ? value : [value];

    for (const v of values) {
      await this.storage.create({
        content: v,
        metadata: { field: this.fieldName },
      });
    }
  }

  /**
   * Clear the field
   */
  async clear(): Promise<void> {
    const entries = await this.storage.list();
    for (const entry of entries) {
      if (entry.metadata?.field === this.fieldName) {
        await this.storage.delete(entry.id!);
      }
    }
  }

  /**
   * Search within this field
   */
  async search(query: string, limit = 5): Promise<SearchResult[]> {
    const results = await this.storage.search(query, limit);
    return results.filter((r) => r.metadata?.field === this.fieldName);
  }
}

/**
 * Create Hystersis storage for Agno
 */
export function createHystersisStorage(config: AgnoMemoryConfig): HystersisStorage {
  return new HystersisStorageImpl(config);
}

export default {
  HystersisStorage: HystersisStorageImpl,
  HystersisField,
  createHystersisStorage,
};
