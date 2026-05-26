/**
 * Hystersis integration for Vercel AI SDK.
 * Auto-injects relevant memories into system prompt for useChat/useCompletion.
 */
import { HystersisClient } from '../index.js';
import type { MemoryResult } from '../types.js';

export interface VercelAIConfig {
  baseUrl: string;
  apiKey: string;
  userId?: string;
  maxMemories?: number;
  minScore?: number;
}

export class HystersisVercelProvider {
  private client: HystersisClient;
  private userId: string;
  private maxMemories: number;
  private minScore: number;

  constructor(config: VercelAIConfig) {
    this.client = new HystersisClient({ baseUrl: config.baseUrl, apiKey: config.apiKey });
    this.userId = config.userId || 'default';
    this.maxMemories = config.maxMemories || 5;
    this.minScore = config.minScore || 0.3;
  }

  /**
   * Middleware for Vercel AI SDK's useChat.
   * Injects relevant memories into the system message before sending to the model.
   */
  async enhanceMessages(messages: Array<{ role: string; content: string }>): Promise<Array<{ role: string; content: string }>> {
    const lastUserMessage = [...messages].reverse().find((m) => m.role === 'user');
    if (!lastUserMessage) return messages;

    const memories = await this.client.search(lastUserMessage.content, {
      user_id: this.userId,
      limit: this.maxMemories,
    });

    const relevantMemories = (memories as MemoryResult[]).filter((m: MemoryResult) => m.score >= this.minScore);
    if (relevantMemories.length === 0) return messages;

    const memoryContext = relevantMemories
      .map((m: MemoryResult) => `- ${m.text}`)
      .join('\n');

    const systemMessage = {
      role: 'system' as const,
      content: `Relevant memories about this user:\n${memoryContext}\n\nUse these memories to personalize your response.`,
    };

    // Store the user message as a new memory (fire and forget)
    this.client.memories.create({
      content: lastUserMessage.content,
      user_id: this.userId,
      type: 'conversation',
    }).catch(() => {});

    return [systemMessage, ...messages];
  }

  /**
   * Post-response hook: store assistant responses as memories.
   */
  async storeResponse(response: string): Promise<void> {
    await this.client.memories.create({
      content: response,
      user_id: this.userId,
      type: 'conversation',
    });
  }
}
