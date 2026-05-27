/**
 * AWS Bedrock Integration for Hystersis - Node.js SDK
 *
 * Provides memory capabilities for AWS Bedrock agents.
 *
 * @example
 * ```typescript
 * import { HystersisBedrockMemory } from 'hystersis/integrations/bedrock';
 *
 * const memory = new HystersisBedrockMemory({
 *   apiKey: 'your-api-key',
 *   baseUrl: 'https://api.hystersis.ai'
 * });
 *
 * // Get session context
 * const context = await memory.getSessionContext('session-123');
 *
 * // Save a turn
 * await memory.saveTurn('session-123', 'What is AI?', 'AI is...');
 * ```
 */

import { HystersisClient as Hystersis } from '../index.js';

export interface BedrockMemoryConfig {
  apiKey: string;
  baseUrl: string;
  agentId?: string;
}

/**
 * Memory provider for AWS Bedrock agents.
 */
export class HystersisBedrockMemory {
  private client: Hystersis;
  private agentId?: string;

  constructor(config: BedrockMemoryConfig) {
    this.client = new Hystersis({
      baseUrl: config.baseUrl,
      apiKey: config.apiKey,
    });
    this.agentId = config.agentId;
  }

  /**
   * Get conversation context for a Bedrock session.
   */
  async getSessionContext(
    sessionId: string,
    query?: string,
    limit = 10
  ): Promise<Array<Record<string, unknown>>> {
    try {
      if (query) {
        const results = await this.client.memories.search(query, {
          user_id: sessionId,
          limit,
        });
        return Array.isArray(results) ? results : [];
      }

      const results = await this.client.memories.list({
        user_id: sessionId,
        limit,
      });
      return Array.isArray(results) ? results : [];
    } catch {
      return [];
    }
  }

  /**
   * Save a conversation turn from a Bedrock agent session.
   */
  async saveTurn(
    sessionId: string,
    inputText: string,
    outputText: string,
    metadata?: Record<string, unknown>
  ): Promise<void> {
    const baseMeta: Record<string, unknown> = {
      source: 'bedrock',
      session_id: sessionId,
    };
    if (this.agentId) baseMeta.agent_id = this.agentId;
    if (metadata) Object.assign(baseMeta, metadata);

    if (inputText) {
      try {
        await this.client.memories.create({
          content: inputText,
          user_id: sessionId,
          type: 'conversation',
          metadata: { ...baseMeta, role: 'user' },
        });
      } catch (error) {
        console.warn('Failed to store input:', error);
      }
    }

    if (outputText) {
      try {
        await this.client.memories.create({
          content: outputText,
          user_id: sessionId,
          type: 'conversation',
          metadata: { ...baseMeta, role: 'assistant' },
        });
      } catch (error) {
        console.warn('Failed to store output:', error);
      }
    }
  }

  /**
   * Get a formatted summary of the session context.
   */
  async getSessionSummary(sessionId: string): Promise<string> {
    const items = await this.getSessionContext(sessionId);
    if (items.length === 0) return 'No previous context available.';

    const lines = items
      .map((item) => {
        const role = (item.metadata as Record<string, unknown>)?.role ?? 'unknown';
        const content = (item.text as string) || (item.content as string) || '';
        return content ? `[${role}] ${content}` : '';
      })
      .filter(Boolean);

    return lines.length > 0 ? lines.join('\n') : 'No previous context available.';
  }
}
