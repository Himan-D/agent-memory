/**
 * Hystersis integration for OpenAI Agents SDK.
 * Provides memory tools that OpenAI agents can call.
 */
import { HystersisClient } from '../index.js';
import type { Memory, MemoryResult } from '../types.js';

export interface OpenAIAgentsConfig {
  baseUrl: string;
  apiKey: string;
  userId?: string;
  agentId?: string;
}

/**
 * Creates OpenAI-compatible tool definitions for Hystersis memory operations.
 * Use with the OpenAI Agents SDK's tool_choice parameter.
 */
export class HystersisOpenAITools {
  private client: HystersisClient;
  private userId: string;
  private agentId: string;

  constructor(config: OpenAIAgentsConfig) {
    this.client = new HystersisClient({ baseUrl: config.baseUrl, apiKey: config.apiKey });
    this.userId = config.userId || 'default';
    this.agentId = config.agentId || 'default';
  }

  /** Returns tool definitions compatible with OpenAI's function calling schema */
  getToolDefinitions() {
    return [
      {
        type: 'function' as const,
        function: {
          name: 'store_memory',
          description: 'Store a piece of information in long-term memory for later retrieval',
          parameters: {
            type: 'object',
            properties: {
              content: { type: 'string', description: 'The information to remember' },
              category: { type: 'string', description: 'Category: preference, fact, decision, skill, goal' },
              importance: { type: 'string', enum: ['critical', 'high', 'medium', 'low'] },
            },
            required: ['content'],
          },
        },
      },
      {
        type: 'function' as const,
        function: {
          name: 'recall_memories',
          description: 'Search long-term memory for relevant information',
          parameters: {
            type: 'object',
            properties: {
              query: { type: 'string', description: 'What to search for' },
              limit: { type: 'number', description: 'Max results (default 5)' },
            },
            required: ['query'],
          },
        },
      },
      {
        type: 'function' as const,
        function: {
          name: 'memory_feedback',
          description: 'Rate whether a retrieved memory was helpful',
          parameters: {
            type: 'object',
            properties: {
              memory_id: { type: 'string', description: 'ID of the memory' },
              feedback: { type: 'string', enum: ['positive', 'negative'], description: 'Was the memory helpful?' },
            },
            required: ['memory_id', 'feedback'],
          },
        },
      },
    ];
  }

  /** Execute a tool call from an OpenAI agent response */
  async executeTool(name: string, args: Record<string, unknown>): Promise<string> {
    switch (name) {
      case 'store_memory': {
        const mem = await this.client.memories.create({
          content: args.content as string,
          user_id: this.userId,
          agent_id: this.agentId,
          category: args.category as string | undefined,
          importance: args.importance as string | undefined,
        });
        return JSON.stringify({ stored: true, id: (mem as Memory).id });
      }

      case 'recall_memories': {
        const results = await this.client.search(args.query as string, {
          limit: (args.limit as number) || 5,
          user_id: this.userId,
          agent_id: this.agentId,
        });
        return JSON.stringify((results as MemoryResult[]).map((r: MemoryResult) => ({
          id: r.memoryId,
          content: r.text,
          score: r.score,
        })));
      }

      case 'memory_feedback': {
        await this.client.feedback.add(args.memory_id as string, args.feedback as string);
        return JSON.stringify({ recorded: true });
      }

      default:
        return JSON.stringify({ error: `Unknown tool: ${name}` });
    }
  }
}
