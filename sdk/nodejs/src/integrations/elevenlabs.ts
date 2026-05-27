/**
 * ElevenLabs Conversational AI Integration for Hystersis - Node.js SDK
 *
 * Provides memory context for ElevenLabs conversational AI agents.
 *
 * @example
 * ```typescript
 * import { HystersisElevenLabsMemory } from 'hystersis/integrations/elevenlabs';
 *
 * const memory = new HystersisElevenLabsMemory({
 *   apiKey: 'your-api-key',
 *   baseUrl: 'https://api.hystersis.ai'
 * });
 *
 * // Get system prompt context
 * const context = await memory.getSystemPromptContext('user-123');
 *
 * // Store a conversation
 * await memory.storeConversation([
 *   { role: 'user', content: 'Tell me about my last order' },
 *   { role: 'assistant', content: 'Your last order was...' }
 * ], 'user-123');
 * ```
 */

import { HystersisClient as Hystersis } from '../index.js';

export interface ElevenLabsMemoryConfig {
  apiKey: string;
  baseUrl: string;
  contextLimit?: number;
}

/**
 * Memory context provider for ElevenLabs conversational AI.
 */
export class HystersisElevenLabsMemory {
  private client: Hystersis;
  private contextLimit: number;

  constructor(config: ElevenLabsMemoryConfig) {
    this.client = new Hystersis({
      baseUrl: config.baseUrl,
      apiKey: config.apiKey,
    });
    this.contextLimit = config.contextLimit ?? 5;
  }

  /**
   * Get memory context to prepend to the ElevenLabs system prompt.
   */
  async getSystemPromptContext(userId: string, query?: string): Promise<string> {
    try {
      let items: Array<Record<string, unknown>>;

      if (query) {
        const results = await this.client.memories.search(query, {
          user_id: userId,
          limit: this.contextLimit,
        });
        items = Array.isArray(results) ? results : [];
      } else {
        const results = await this.client.memories.list({
          user_id: userId,
          limit: this.contextLimit,
        });
        items = Array.isArray(results) ? results : [];
      }

      const texts = items
        .map((item) => (item.text as string) || (item.content as string) || '')
        .filter(Boolean)
        .map((t) => `- ${t}`);

      if (texts.length === 0) return '';

      return (
        'You have the following context about this user from previous conversations:\n' +
        texts.join('\n')
      );
    } catch {
      return '';
    }
  }

  /**
   * Store conversation messages in Hystersis memory.
   */
  async storeConversation(
    messages: Array<{ role: string; content: string }>,
    userId: string
  ): Promise<void> {
    for (const msg of messages) {
      if (!msg.content) continue;
      try {
        await this.client.memories.create({
          content: msg.content,
          user_id: userId,
          type: 'conversation',
          metadata: {
            source: 'elevenlabs',
            role: msg.role,
          },
        });
      } catch (error) {
        console.warn('Failed to store message:', error);
      }
    }
  }
}
