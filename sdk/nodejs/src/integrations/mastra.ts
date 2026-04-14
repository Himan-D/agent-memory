/**
 * Mastra Integration for Hystersis - Node.js SDK
 * 
 * Provides memory integration for Mastra AI agents and workflows.
 * 
 * @example
 * ```typescript
 * import { MastraMemoryTool } from 'hystersis/integrations/mastra';
 * import { Agent } from '@mastra/core';
 * 
 * const memoryTool = new MastraMemoryTool({
 *   userId: 'user-123',
 *   baseUrl: 'http://localhost:8080'
 * });
 * 
 * const agent = new Agent({
 *   name: 'Assistant',
 *   instructions: 'You have access to memory',
 *   tools: [memoryTool]
 * });
 * ```
 */

import { Hystersis, type Memory, type MemoryResult } from '../index.js';

export interface MastraMemoryConfig {
  userId?: string;
  orgId?: string;
  agentId?: string;
  baseUrl: string;
  apiKey?: string;
}

export interface MastraToolInput {
  query?: string;
  memoryType?: 'user' | 'session' | 'conversation' | 'org';
  action?: 'search' | 'store' | 'retrieve' | 'feedback';
  content?: string;
  memoryId?: string;
  feedbackType?: 'positive' | 'negative' | 'very_negative';
  category?: string;
  limit?: number;
  threshold?: number;
}

export interface MastraToolOutput {
  success: boolean;
  data?: unknown;
  error?: string;
}

/**
 * Memory tool for Mastra agents
 */
export class MastraMemoryTool {
  private client: Hystersis;
  private userId?: string;
  private orgId?: string;
  private agentId?: string;
  private name: string;
  private description: string;

  constructor(
    config: MastraMemoryConfig & { name?: string; description?: string } = {
      baseUrl: 'http://localhost:8080',
    }
  ) {
    this.client = new Hystersis({
      baseUrl: config.baseUrl,
      apiKey: config.apiKey,
    });
    this.userId = config.userId;
    this.orgId = config.orgId;
    this.agentId = config.agentId;
    this.name = config.name ?? 'memory';
    this.description =
      config.description ??
      'Search and store memories for the user. Use search to find relevant past information, and store to save important facts.';
  }

  /**
   * Get the tool schema for Mastra
   */
  getSchema() {
    return {
      name: this.name,
      description: this.description,
      parameters: {
        type: 'object',
        properties: {
          action: {
            type: 'string',
            enum: ['search', 'store', 'retrieve', 'feedback'],
            description: 'The memory action to perform',
          },
          query: {
            type: 'string',
            description: 'Search query for memory retrieval',
          },
          content: {
            type: 'string',
            description: 'Content to store in memory',
          },
          memoryId: {
            type: 'string',
            description: 'Memory ID for feedback actions',
          },
          feedbackType: {
            type: 'string',
            enum: ['positive', 'negative', 'very_negative'],
            description: 'Type of feedback',
          },
          category: {
            type: 'string',
            description: 'Category for the memory',
          },
          limit: {
            type: 'number',
            description: 'Maximum number of results',
          },
          threshold: {
            type: 'number',
            description: 'Minimum relevance threshold',
          },
          memoryType: {
            type: 'string',
            enum: ['user', 'session', 'conversation', 'org'],
            description: 'Type of memory',
          },
        },
        required: ['action'],
      },
    };
  }

  /**
   * Execute the tool
   */
  async execute(input: MastraToolInput): Promise<MastraToolOutput> {
    try {
      switch (input.action) {
        case 'search':
          return this.executeSearch(input);
        case 'store':
          return this.executeStore(input);
        case 'retrieve':
          return this.executeRetrieve(input);
        case 'feedback':
          return this.executeFeedback(input);
        default:
          return { success: false, error: `Unknown action: ${input.action}` };
      }
    } catch (error) {
      return {
        success: false,
        error: error instanceof Error ? error.message : 'Unknown error',
      };
    }
  }

  private async executeSearch(input: MastraToolInput): Promise<MastraToolOutput> {
    if (!input.query) {
      return { success: false, error: 'Query is required for search' };
    }

    const results = await this.client.memories.search(input.query, {
      limit: input.limit ?? 10,
      threshold: input.threshold ?? 0.5,
      userId: this.userId,
      orgId: this.orgId,
      agentId: this.agentId,
      memoryType: input.memoryType,
      rerank: true,
    });

    return {
      success: true,
      data: {
        count: results.length,
        memories: results.map((r) => ({
          id: r.memoryId ?? r.entity.id,
          content: r.text,
          score: r.score,
          category: r.entity.properties?.category,
          createdAt: r.entity.properties?.created_at,
        })),
      },
    };
  }

  private async executeStore(input: MastraToolInput): Promise<MastraToolOutput> {
    if (!input.content) {
      return { success: false, error: 'Content is required for store' };
    }

    const memory = await this.client.memories.create({
      content: input.content,
      userId: this.userId,
      orgId: this.orgId,
      agentId: this.agentId,
      category: input.category,
      memoryType: input.memoryType ?? 'user',
    });

    return {
      success: true,
      data: {
        memoryId: memory.id,
        createdAt: memory.createdAt,
      },
    };
  }

  private async executeRetrieve(input: MastraToolInput): Promise<MastraToolOutput> {
    if (!input.memoryId) {
      return { success: false, error: 'Memory ID is required for retrieve' };
    }

    const memory = await this.client.memories.get(input.memoryId);

    return {
      success: true,
      data: memory,
    };
  }

  private async executeFeedback(input: MastraToolInput): Promise<MastraToolOutput> {
    if (!input.memoryId || !input.feedbackType) {
      return { success: false, error: 'Memory ID and feedback type are required' };
    }

    await this.client.feedback.add({
      memoryId: input.memoryId,
      feedbackType: input.feedbackType,
    });

    return {
      success: true,
      data: { status: 'feedback recorded' },
    };
  }
}

/**
 * Mastra memory storage for agent context
 */
export class MastraMemoryStorage {
  private client: Hystersis;
  private userId?: string;
  private orgId?: string;
  private agentId?: string;

  constructor(config: MastraMemoryConfig) {
    this.client = new Hystersis({
      baseUrl: config.baseUrl,
      apiKey: config.apiKey,
    });
    this.userId = config.userId;
    this.orgId = config.orgId;
    this.agentId = config.agentId;
  }

  /**
   * Store agent context for later retrieval
   */
  async storeContext(
    key: string,
    value: string,
    metadata?: Record<string, unknown>
  ): Promise<void> {
    await this.client.memories.create({
      content: `${key}: ${value}`,
      userId: this.userId,
      orgId: this.orgId,
      agentId: this.agentId,
      category: 'context',
      metadata: {
        ...metadata,
        context_key: key,
      },
    });
  }

  /**
   * Retrieve context by key
   */
  async retrieveContext(key: string): Promise<string | null> {
    const results = await this.client.memories.search(`context_key:${key}`, {
      limit: 1,
      threshold: 0.5,
      userId: this.userId,
      orgId: this.orgId,
      agentId: this.agentId,
    });

    if (results.length > 0) {
      const content = results[0].text;
      const colonIndex = content.indexOf(':');
      return colonIndex > 0 ? content.substring(colonIndex + 1).trim() : content;
    }

    return null;
  }

  /**
   * Get recent context entries
   */
  async getRecentContext(limit = 10): Promise<Array<{ key: string; value: string }>> {
    const results = await this.client.memories.list({
      userId: this.userId,
      orgId: this.orgId,
    });

    return results.memories
      .filter((m) => m.category === 'context')
      .slice(0, limit)
      .map((m) => {
        const colonIndex = m.content.indexOf(':');
        return {
          key: m.metadata?.context_key as string ?? '',
          value: colonIndex > 0 ? m.content.substring(colonIndex + 1).trim() : m.content,
        };
      });
  }
}

export default {
  MastraMemoryTool,
  MastraMemoryStorage,
};
