/**
 * CrewAI Integration for Hystersis - Node.js SDK
 *
 * Provides shared memory capabilities for CrewAI crews and agents.
 *
 * @example
 * ```typescript
 * import { CrewMemory } from 'hystersis/integrations/crewai';
 *
 * const crewMemory = new CrewMemory({
 *   crewId: 'research-team',
 *   userId: 'user-123',
 *   baseUrl: 'http://localhost:8080'
 * });
 *
 * // Add shared memory for the crew
 * await crewMemory.addSharedMemory('We decided to use RAG for this project');
 *
 * // Get agent-specific memory
 * const agentMemory = crewMemory.getAgentMemory('researcher');
 * await agentMemory.addMemory('Found a great paper on transformers');
 * ```
 */

import { HystersisClient as Hystersis } from '../index.js';
import type { Memory, MemoryResult } from '../types.js';

export interface CrewMemoryConfig {
  crewId: string;
  userId?: string;
  orgId?: string;
  baseUrl: string;
  apiKey?: string;
}

export interface CrewAgentConfig {
  agentId: string;
  role?: string;
  goal?: string;
}

/**
 * Shared memory for CrewAI crews
 */
export class CrewMemory {
  private client: Hystersis;
  private crewId: string;
  private userId?: string;
  private orgId?: string;

  constructor(config: CrewMemoryConfig) {
    this.client = new Hystersis({
      baseUrl: config.baseUrl,
      apiKey: config.apiKey,
    });
    this.crewId = config.crewId;
    this.userId = config.userId;
    this.orgId = config.orgId;
  }

  /**
   * Add a shared memory visible to all crew members
   */
  async addSharedMemory(
    content: string,
    category = 'crew-shared',
    metadata?: Record<string, unknown>
  ): Promise<Memory> {
    return this.client.memories.create({
      content,
      type: 'org',
      category,
      user_id: this.userId,
      org_id: this.orgId,
      metadata: {
        ...metadata,
        crew_id: this.crewId,
        shared: true,
      },
    }) as Promise<Memory>;
  }

  /**
   * Get all shared memories for this crew
   */
  async getSharedMemories(category?: string, limit = 50): Promise<Memory[]> {
    const result = await this.client.memories.list({
      user_id: this.userId,
      org_id: this.orgId,
    });

    return (result.memories as Memory[]).filter((m: Memory) => {
      const meta = m.metadata as Record<string, unknown> | undefined;
      return meta?.crew_id === this.crewId && meta?.shared === true;
    }).filter((m: Memory) => !category || m.category === category).slice(0, limit);
  }

  /**
   * Search shared crew memories
   */
  async searchShared(
    query: string,
    limit = 10,
    threshold = 0.5
  ): Promise<MemoryResult[]> {
    const results = await this.client.memories.search(query, {
      limit,
      threshold,
      user_id: this.userId,
      org_id: this.orgId,
    }) as MemoryResult[];

    return results.filter((r: MemoryResult) => {
      const meta = r.metadata?.metadata as Record<string, unknown> | undefined;
      return meta?.crew_id === this.crewId && meta?.shared === true;
    });
  }

  /**
   * Add feedback to shared memory
   */
  async addFeedbackToShared(
    memoryId: string,
    feedbackType: 'positive' | 'negative' | 'very_negative',
    comment?: string
  ): Promise<void> {
    await this.client.feedback.add(memoryId, feedbackType, comment, this.userId);
  }

  /**
   * Get memory for a specific crew agent
   */
  getAgentMemory(agentId: string, context?: CrewAgentConfig): CrewAgentMemory {
    return new CrewAgentMemory({
      client: this.client,
      agentId,
      crewId: this.crewId,
      agentContext: context,
      userId: this.userId,
      orgId: this.orgId,
    });
  }
}

/**
 * Agent-specific memory for CrewAI
 */
export class CrewAgentMemory {
  private client: Hystersis;
  private agentId: string;
  private crewId: string;
  private agentContext?: CrewAgentConfig;
  private userId?: string;
  private orgId?: string;

  constructor(config: {
    client: Hystersis;
    agentId: string;
    crewId: string;
    agentContext?: CrewAgentConfig;
    userId?: string;
    orgId?: string;
  }) {
    this.client = config.client;
    this.agentId = config.agentId;
    this.crewId = config.crewId;
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
      type: memoryType,
      category,
      user_id: this.userId,
      org_id: this.orgId,
      agent_id: this.agentId,
      metadata: {
        crew_id: this.crewId,
        agent_context: this.agentContext,
      },
    }) as Promise<Memory>;
  }

  /**
   * Get all memories for this agent
   */
  async getMemories(limit = 50): Promise<Memory[]> {
    const result = await this.client.memories.list({
      user_id: this.userId,
      org_id: this.orgId,
    });

    return (result.memories as Memory[]).filter((m: Memory) => m.agentId === this.agentId).slice(0, limit);
  }

  /**
   * Search agent memories
   */
  async search(query: string, limit = 10, threshold = 0.5): Promise<MemoryResult[]> {
    return this.client.memories.search(query, {
      limit,
      threshold,
      user_id: this.userId,
      org_id: this.orgId,
      agent_id: this.agentId,
    }) as Promise<MemoryResult[]>;
  }

  /**
   * Get shared crew memories accessible to this agent
   */
  async getSharedMemories(limit = 50): Promise<Memory[]> {
    const result = await this.client.memories.list({
      user_id: this.userId,
      org_id: this.orgId,
    });

    return (result.memories as Memory[]).filter((m: Memory) => {
      const meta = m.metadata as Record<string, unknown> | undefined;
      return meta?.crew_id === this.crewId && meta?.shared === true;
    }).slice(0, limit);
  }

  /**
   * Add feedback to memory
   */
  async addFeedback(
    memoryId: string,
    feedbackType: 'positive' | 'negative' | 'very_negative',
    comment?: string
  ): Promise<void> {
    await this.client.feedback.add(memoryId, feedbackType, comment, this.userId);
  }
}

export default {
  CrewMemory,
  CrewAgentMemory,
};
