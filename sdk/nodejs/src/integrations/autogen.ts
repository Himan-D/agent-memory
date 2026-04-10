/**
 * AutoGen Integration for Agent Memory - Node.js SDK
 * 
 * Provides shared memory capabilities for AutoGen multi-agent systems.
 * 
 * @example
 * ```typescript
 * import { AutoGenMemory } from 'agent-memory/integrations/autogen';
 * 
 * const memory = new AutoGenMemory({
 *   groupId: 'research-team',
 *   userId: 'user-123',
 *   baseUrl: 'http://localhost:8080'
 * });
 * 
 * // All agents can share memories
 * await memory.addSharedMemory('Research shows AI will transform healthcare');
 * const memories = await memory.getSharedMemories();
 * ```
 */

import { AgentMemory, type Memory, type MemoryResult } from '../index.js';

export interface AutoGenMemoryConfig {
  groupId: string;
  userId?: string;
  orgId?: string;
  baseUrl: string;
  apiKey?: string;
}

export interface AgentContext {
  agentId: string;
  role?: string;
  goal?: string;
}

/**
 * Shared memory for AutoGen multi-agent systems
 */
export class AutoGenMemory {
  private client: AgentMemory;
  private groupId: string;
  private userId?: string;
  private orgId?: string;

  constructor(config: AutoGenMemoryConfig) {
    this.client = new AgentMemory({
      baseUrl: config.baseUrl,
      apiKey: config.apiKey,
    });
    this.groupId = config.groupId;
    this.userId = config.userId;
    this.orgId = config.orgId;
  }

  /**
   * Add a shared memory visible to all agents in the group
   */
  async addSharedMemory(
    content: string,
    category = 'agent-shared',
    metadata?: Record<string, unknown>
  ): Promise<Memory> {
    return this.client.memories.create({
      content,
      memoryType: 'org',
      category,
      userId: this.userId,
      orgId: this.orgId,
      metadata: {
        ...metadata,
        group_id: this.groupId,
        shared: true,
      },
    });
  }

  /**
   * Get all shared memories for this group
   */
  async getSharedMemories(category?: string, limit = 50): Promise<Memory[]> {
    const result = await this.client.memories.list({
      userId: this.userId,
      orgId: this.orgId,
    });

    return result.memories.filter((m) => {
      const meta = m.metadata as Record<string, unknown> | undefined;
      return meta?.group_id === this.groupId && meta?.shared === true;
    }).filter((m) => !category || m.category === category).slice(0, limit);
  }

  /**
   * Search shared memories across the group
   */
  async searchSharedMemories(
    query: string,
    limit = 10,
    threshold = 0.5
  ): Promise<MemoryResult[]> {
    const results = await this.client.memories.search(query, {
      limit,
      threshold,
      userId: this.userId,
      orgId: this.orgId,
    });

    return results.filter((r) => {
      const meta = r.metadata?.metadata as Record<string, unknown> | undefined;
      return meta?.group_id === this.groupId && meta?.shared === true;
    });
  }

  /**
   * Get a memory agent for a specific AutoGen agent
   */
  getAgentMemory(agentId: string, agentContext?: AgentContext): AutoGenAgentMemory {
    return new AutoGenAgentMemory({
      client: this.client,
      agentId,
      groupId: this.groupId,
      agentContext,
      userId: this.userId,
      orgId: this.orgId,
    });
  }
}

/**
 * Agent-specific memory for AutoGen
 */
export class AutoGenAgentMemory {
  private client: AgentMemory;
  private agentId: string;
  private groupId: string;
  private agentContext?: AgentContext;
  private userId?: string;
  private orgId?: string;

  constructor(config: {
    client: AgentMemory;
    agentId: string;
    groupId: string;
    agentContext?: AgentContext;
    userId?: string;
    orgId?: string;
  }) {
    this.client = config.client;
    this.agentId = config.agentId;
    this.groupId = config.groupId;
    this.agentContext = config.agentContext;
    this.userId = config.userId;
    this.orgId = config.orgId;
  }

  /**
   * Add an agent-specific memory
   */
  async addMemory(
    content: string,
    memoryType: 'user' | 'session' = 'user',
    category?: string
  ): Promise<Memory> {
    return this.client.memories.create({
      content,
      memoryType,
      category,
      userId: this.userId,
      orgId: this.orgId,
      agentId: this.agentId,
      metadata: {
        group_id: this.groupId,
        agent_context: this.agentContext,
      },
    });
  }

  /**
   * Get all memories for this agent
   */
  async getMemories(limit = 50): Promise<Memory[]> {
    const result = await this.client.memories.list({
      userId: this.userId,
      orgId: this.orgId,
    });

    return result.memories.filter((m) => m.agentId === this.agentId).slice(0, limit);
  }

  /**
   * Search agent memories
   */
  async search(query: string, limit = 10, threshold = 0.5): Promise<MemoryResult[]> {
    return this.client.memories.search(query, {
      limit,
      threshold,
      userId: this.userId,
      orgId: this.orgId,
      agentId: this.agentId,
    });
  }

  /**
   * Get shared memories visible to this agent
   */
  async getSharedMemories(limit = 50): Promise<Memory[]> {
    const result = await this.client.memories.list({
      userId: this.userId,
      orgId: this.orgId,
    });

    return result.memories.filter((m) => {
      const meta = m.metadata as Record<string, unknown> | undefined;
      return meta?.group_id === this.groupId && meta?.shared === true;
    }).slice(0, limit);
  }

  /**
   * Add feedback to improve future searches
   */
  async addFeedback(
    memoryId: string,
    feedbackType: 'positive' | 'negative' | 'very_negative',
    comment?: string
  ): Promise<void> {
    await this.client.feedback.add({
      memoryId,
      feedbackType,
      comment,
      userId: this.userId,
    });
  }
}

export default AutoGenMemory;
