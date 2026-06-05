/**
 * LiveKit Voice Agent Memory Integration for Hystersis - Node.js SDK
 *
 * Provides memory capabilities for LiveKit voice agents, storing
 * conversation transcripts and retrieving relevant context during live calls.
 *
 * @example
 * ```typescript
 * import { HystersisLiveKitPlugin } from 'hystersis/integrations/livekit';
 *
 * const plugin = new HystersisLiveKitPlugin({
 *   apiKey: 'your-api-key',
 *   userId: 'caller-123',
 *   baseUrl: 'https://api.hystersis.ai'
 * });
 *
 * // Store transcribed speech
 * await plugin.onTranscript('I need help with my order', 'user');
 *
 * // Retrieve relevant context
 * const context = await plugin.getContext('order help');
 * ```
 */

import { HystersisClient as Hystersis } from '../index.js';

export interface LiveKitPluginConfig {
  apiKey: string;
  userId: string;
  baseUrl: string;
  sessionId?: string;
  agentId?: string;
}

export interface ToolDefinition {
  name: string;
  description: string;
  parameters: {
    type: string;
    properties: Record<string, unknown>;
    required: string[];
  };
}

/**
 * Memory plugin for LiveKit voice agents.
 */
export class HystersisLiveKitPlugin {
  private client: Hystersis;
  private userId: string;
  private sessionId?: string;
  private agentId?: string;

  constructor(config: LiveKitPluginConfig) {
    this.client = new Hystersis({
      baseUrl: config.baseUrl,
      apiKey: config.apiKey,
    });
    this.userId = config.userId;
    this.sessionId = config.sessionId;
    this.agentId = config.agentId;
  }

  /**
   * Store transcribed speech as memory.
   */
  async onTranscript(transcript: string, role: 'user' | 'assistant' = 'user'): Promise<void> {
    const metadata: Record<string, unknown> = {
      source: 'livekit',
      role,
    };
    if (this.sessionId) metadata.session_id = this.sessionId;
    if (this.agentId) metadata.agent_id = this.agentId;

    try {
      await this.client.memories.create({
        content: transcript,
        user_id: this.userId,
        type: 'conversation',
        metadata,
      });
    } catch (error) {
      console.warn('Failed to store transcript:', error);
    }
  }

  /**
   * Retrieve relevant memories for current conversation context.
   */
  async getContext(query: string, limit = 5): Promise<string[]> {
    try {
      const results = await this.client.memories.search(query, {
        user_id: this.userId,
        limit,
      });
      const items = Array.isArray(results) ? results : [];
      return items
        .map((r: Record<string, unknown>) => (r.text as string) || (r.content as string) || '')
        .filter(Boolean);
    } catch {
      return [];
    }
  }

  /**
   * Store a complete conversation turn.
   */
  async storeConversationTurn(userMessage: string, assistantResponse: string): Promise<void> {
    await this.onTranscript(userMessage, 'user');
    await this.onTranscript(assistantResponse, 'assistant');
  }

  /**
   * Return MCP-compatible tool definitions for LiveKit agents.
   */
  getToolDefinitions(): ToolDefinition[] {
    return [
      {
        name: 'remember',
        description: 'Store important information from the conversation',
        parameters: {
          type: 'object',
          properties: {
            content: { type: 'string', description: 'The information to remember' },
          },
          required: ['content'],
        },
      },
      {
        name: 'recall',
        description: 'Recall relevant information from past conversations',
        parameters: {
          type: 'object',
          properties: {
            query: { type: 'string', description: 'What to search for in memory' },
          },
          required: ['query'],
        },
      },
    ];
  }

  /**
   * Handle a tool call from the LiveKit agent.
   */
  async handleToolCall(toolName: string, args: Record<string, unknown>): Promise<string> {
    if (toolName === 'remember') {
      const content = args.content as string;
      await this.onTranscript(content, 'user');
      return 'Remembered.';
    } else if (toolName === 'recall') {
      const query = args.query as string;
      const memories = await this.getContext(query);
      return memories.length > 0
        ? memories.map((m) => `- ${m}`).join('\n')
        : 'No relevant memories found.';
    }
    return `Unknown tool: ${toolName}`;
  }
}
